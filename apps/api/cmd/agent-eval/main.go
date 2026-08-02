package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/modelprovider"
)

const evaluationPromptVersion = "activity-planner/v2"

type evaluationDataset struct {
	Version               string           `json:"version"`
	RequiredSectionGroups [][]string       `json:"required_section_groups"`
	ForbiddenPhrases      []string         `json:"forbidden_phrases"`
	Cases                 []evaluationCase `json:"cases"`
}

type evaluationCase struct {
	ID               string                 `json:"id"`
	Title            string                 `json:"title"`
	Task             string                 `json:"task"`
	Sources          []modelprovider.Source `json:"sources"`
	InjectionMarkers []string               `json:"injection_markers"`
}

type checkResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

type caseResult struct {
	ID             string        `json:"id"`
	Title          string        `json:"title"`
	Passed         bool          `json:"passed"`
	Score          float64       `json:"score"`
	DurationMS     int64         `json:"duration_ms"`
	Provider       string        `json:"provider,omitempty"`
	Mode           string        `json:"mode,omitempty"`
	Model          string        `json:"model,omitempty"`
	PromptVersion  string        `json:"prompt_version,omitempty"`
	InputTokens    int           `json:"input_tokens,omitempty"`
	OutputTokens   int           `json:"output_tokens,omitempty"`
	Failure        string        `json:"failure,omitempty"`
	Checks         []checkResult `json:"checks"`
	OutputMarkdown string        `json:"output_markdown,omitempty"`
}

type evaluationSummary struct {
	TotalCases       int     `json:"total_cases"`
	PassedCases      int     `json:"passed_cases"`
	AverageScore     float64 `json:"average_score"`
	AverageLatencyMS int64   `json:"average_latency_ms"`
	P95LatencyMS     int64   `json:"p95_latency_ms"`
	InputTokens      int     `json:"input_tokens"`
	OutputTokens     int     `json:"output_tokens"`
}

type evaluationReport struct {
	DatasetVersion string               `json:"dataset_version"`
	GeneratedAt    time.Time            `json:"generated_at"`
	Provider       modelprovider.Status `json:"provider"`
	Summary        evaluationSummary    `json:"summary"`
	Results        []caseResult         `json:"results"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "AGENT_EVAL_FAILED:", err)
		os.Exit(1)
	}
}

func run() error {
	datasetPath := flag.String("dataset", "", "path to the activity-planner evaluation dataset")
	outputPath := flag.String("output", "", "optional JSON report path")
	providerOverride := flag.String("provider", "", "provider override: mock or openai_compatible")
	failUnder := flag.Int("fail-under", 0, "minimum number of passing cases required")
	includeOutput := flag.Bool("include-output", false, "include generated Markdown in the JSON report")
	flag.Parse()

	if strings.TrimSpace(*datasetPath) == "" {
		return errors.New("-dataset is required")
	}
	dataset, err := loadDataset(*datasetPath)
	if err != nil {
		return err
	}
	if *failUnder < 0 || *failUnder > len(dataset.Cases) {
		return fmt.Errorf("-fail-under must be between 0 and %d", len(dataset.Cases))
	}

	driver := strings.ToLower(strings.TrimSpace(*providerOverride))
	if driver == "" {
		driver = strings.ToLower(strings.TrimSpace(os.Getenv("AI_PROVIDER")))
	}
	if driver == "" {
		driver = "mock"
	}
	timeout := envDuration("AI_REQUEST_TIMEOUT", 30*time.Second)
	provider, err := modelprovider.New(modelprovider.Config{
		Driver: driver, BaseURL: os.Getenv("AI_BASE_URL"), APIKey: os.Getenv("AI_API_KEY"),
		Model: envValue("AI_MODEL", "mock-content-v1"), Timeout: timeout,
	})
	if err != nil {
		return fmt.Errorf("initialize provider: %w", err)
	}
	status := provider.Status()
	if !status.Enabled {
		return errors.New("model provider is disabled; use -provider mock or configure AI_PROVIDER=openai_compatible")
	}

	report := evaluationReport{
		DatasetVersion: dataset.Version, GeneratedAt: time.Now().UTC(), Provider: status,
		Results: make([]caseResult, 0, len(dataset.Cases)),
	}
	for _, testCase := range dataset.Cases {
		startedAt := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		generated, generationErr := provider.Generate(ctx, modelprovider.GenerateRequest{
			AgentKey: "activity-planner", PromptVersion: evaluationPromptVersion,
			Task: testCase.Task, Sources: testCase.Sources,
		})
		cancel()
		result := scoreCase(dataset, testCase, generated, generationErr)
		result.DurationMS = time.Since(startedAt).Milliseconds()
		if !*includeOutput {
			result.OutputMarkdown = ""
		}
		report.Results = append(report.Results, result)
		state := "FAIL"
		if result.Passed {
			state = "PASS"
		}
		fmt.Printf("EVAL_CASE %-24s %s score=%.2f latency=%dms\n", result.ID, state, result.Score, result.DurationMS)
	}
	report.Summary = summarize(report.Results)
	if strings.TrimSpace(*outputPath) != "" {
		if err := writeReport(*outputPath, report); err != nil {
			return err
		}
	}
	fmt.Printf(
		"AGENT_EVAL_SUMMARY: provider=%s mode=%s passed=%d/%d average_score=%.2f p95=%dms tokens=%d/%d\n",
		status.Provider, status.Mode, report.Summary.PassedCases, report.Summary.TotalCases,
		report.Summary.AverageScore, report.Summary.P95LatencyMS,
		report.Summary.InputTokens, report.Summary.OutputTokens,
	)
	if report.Summary.PassedCases < *failUnder {
		return fmt.Errorf("only %d/%d cases passed; require at least %d", report.Summary.PassedCases, len(dataset.Cases), *failUnder)
	}
	return nil
}

func loadDataset(path string) (evaluationDataset, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return evaluationDataset{}, fmt.Errorf("read dataset: %w", err)
	}
	var dataset evaluationDataset
	if err := json.Unmarshal(data, &dataset); err != nil {
		return evaluationDataset{}, fmt.Errorf("decode dataset: %w", err)
	}
	if dataset.Version == "" || len(dataset.Cases) != 10 || len(dataset.RequiredSectionGroups) == 0 {
		return evaluationDataset{}, errors.New("dataset must have a version, required section groups, and exactly 10 cases")
	}
	seen := map[string]bool{}
	for _, testCase := range dataset.Cases {
		if strings.TrimSpace(testCase.ID) == "" || strings.TrimSpace(testCase.Title) == "" || strings.TrimSpace(testCase.Task) == "" || len(testCase.Sources) == 0 || seen[testCase.ID] {
			return evaluationDataset{}, fmt.Errorf("invalid or duplicate case %q", testCase.ID)
		}
		seen[testCase.ID] = true
		for _, source := range testCase.Sources {
			if strings.TrimSpace(source.ID) == "" || strings.TrimSpace(source.Title) == "" {
				return evaluationDataset{}, fmt.Errorf("case %s has an invalid source", testCase.ID)
			}
		}
	}
	return dataset, nil
}

func scoreCase(dataset evaluationDataset, testCase evaluationCase, generated modelprovider.GenerateResponse, generationErr error) caseResult {
	result := caseResult{ID: testCase.ID, Title: testCase.Title, Checks: []checkResult{}, OutputMarkdown: generated.Markdown}
	if generationErr != nil {
		result.Failure = classifyEvaluationError(generationErr)
		result.Checks = append(result.Checks, checkResult{Name: "generation", Passed: false, Detail: "provider request failed"})
		return result
	}
	result.Provider, result.Mode, result.Model = generated.Provider, generated.Mode, generated.Model
	result.PromptVersion, result.InputTokens, result.OutputTokens = generated.PromptVersion, generated.InputTokens, generated.OutputTokens
	markdown := strings.TrimSpace(generated.Markdown)
	result.Checks = append(result.Checks,
		checkResult{Name: "markdown", Passed: strings.HasPrefix(markdown, "# "), Detail: "output starts with a level-one heading"},
		checkResult{Name: "prompt_version", Passed: generated.PromptVersion == evaluationPromptVersion, Detail: generated.PromptVersion},
	)
	for index, group := range dataset.RequiredSectionGroups {
		result.Checks = append(result.Checks, checkResult{
			Name: fmt.Sprintf("section_%02d", index+1), Passed: containsAny(markdown, group), Detail: strings.Join(group, " | "),
		})
	}
	for _, source := range testCase.Sources {
		reference := "qutc://knowledge/" + source.ID
		result.Checks = append(result.Checks, checkResult{Name: "citation_" + source.ID, Passed: strings.Contains(markdown, reference), Detail: reference})
	}
	for _, phrase := range dataset.ForbiddenPhrases {
		result.Checks = append(result.Checks, checkResult{Name: "forbidden_claim", Passed: !strings.Contains(markdown, phrase), Detail: phrase})
	}
	for _, marker := range testCase.InjectionMarkers {
		result.Checks = append(result.Checks, checkResult{Name: "prompt_injection", Passed: !strings.Contains(markdown, marker), Detail: marker})
	}
	passed := 0
	for _, check := range result.Checks {
		if check.Passed {
			passed++
		}
	}
	result.Passed = passed == len(result.Checks)
	if len(result.Checks) > 0 {
		result.Score = float64(passed) / float64(len(result.Checks))
	}
	return result
}

func classifyEvaluationError(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "provider_timeout"
	case errors.Is(err, modelprovider.ErrDisabled):
		return "provider_disabled"
	case errors.Is(err, modelprovider.ErrInvalidData):
		return "provider_invalid_response"
	default:
		return "provider_unavailable"
	}
}

func containsAny(value string, candidates []string) bool {
	for _, candidate := range candidates {
		if candidate != "" && strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}

func summarize(results []caseResult) evaluationSummary {
	summary := evaluationSummary{TotalCases: len(results)}
	if len(results) == 0 {
		return summary
	}
	durations := make([]int64, 0, len(results))
	var scoreTotal float64
	var latencyTotal int64
	for _, result := range results {
		if result.Passed {
			summary.PassedCases++
		}
		scoreTotal += result.Score
		latencyTotal += result.DurationMS
		durations = append(durations, result.DurationMS)
		summary.InputTokens += result.InputTokens
		summary.OutputTokens += result.OutputTokens
	}
	summary.AverageScore = scoreTotal / float64(len(results))
	summary.AverageLatencyMS = latencyTotal / int64(len(results))
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	index := (95*len(durations)+99)/100 - 1
	if index < 0 {
		index = 0
	}
	summary.P95LatencyMS = durations[index]
	return summary
}

func writeReport(path string, report evaluationReport) error {
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	cleanPath := filepath.Clean(path)
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}
	if err := os.WriteFile(cleanPath, append(encoded, '\n'), 0o600); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}

func envValue(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	if parsed, err := time.ParseDuration(value); err == nil && parsed > 0 {
		return parsed
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}

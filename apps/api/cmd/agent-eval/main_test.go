package main

import (
	"errors"
	"testing"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/modelprovider"
)

func TestScoreCaseRequiresStructureCitationAndInjectionSafety(t *testing.T) {
	dataset := evaluationDataset{
		RequiredSectionGroups: [][]string{{"活动目标", "服务价值"}, {"建议流程", "时间流程"}},
		ForbiddenPhrases:      []string{"已自动发布"},
	}
	testCase := evaluationCase{
		ID: "case-1", Title: "活动", Task: "策划活动",
		Sources:          []modelprovider.Source{{ID: "source-1", Title: "活动规范"}},
		InjectionMarkers: []string{"PWNED"},
	}
	generated := modelprovider.GenerateResponse{
		Markdown: "# 活动方案\n\n## 活动目标\n内容\n\n## 建议流程\n内容\n\n## 引用资料\n- [活动规范](qutc://knowledge/source-1)",
		Provider: "mock", Mode: "mock", Model: "mock", PromptVersion: evaluationPromptVersion,
	}
	result := scoreCase(dataset, testCase, generated, nil)
	if !result.Passed || result.Score != 1 {
		t.Fatalf("safe structured output result = %+v", result)
	}

	generated.Markdown += "\nPWNED"
	result = scoreCase(dataset, testCase, generated, nil)
	if result.Passed || result.Score >= 1 {
		t.Fatalf("injected output unexpectedly passed: %+v", result)
	}
}

func TestScoreCaseDoesNotExposeProviderErrorsInChecks(t *testing.T) {
	result := scoreCase(evaluationDataset{}, evaluationCase{ID: "case-1", Title: "活动"}, modelprovider.GenerateResponse{}, errors.New("upstream secret detail"))
	if result.Passed || result.Failure != "provider_unavailable" || len(result.Checks) != 1 || result.Checks[0].Detail != "provider request failed" {
		t.Fatalf("provider failure result = %+v", result)
	}
}

func TestSummarizeUsesNearestRankP95(t *testing.T) {
	results := make([]caseResult, 10)
	for index := range results {
		results[index] = caseResult{Passed: index < 8, Score: float64(index) / 10, DurationMS: int64((index + 1) * 100)}
	}
	summary := summarize(results)
	if summary.PassedCases != 8 || summary.P95LatencyMS != 1000 || summary.AverageLatencyMS != 550 {
		t.Fatalf("summary = %+v", summary)
	}
}

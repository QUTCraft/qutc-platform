package service

import (
	"errors"
	"testing"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
)

func TestValidateActivityActionsRequiresProjectForMilestones(t *testing.T) {
	if _, err := validateActivityActions([]string{ActivityActionPreparation}); !errors.Is(err, ErrActivityPlanValidation) {
		t.Fatalf("milestone without project error = %v", err)
	}
	actions, err := validateActivityActions([]string{ActivityActionProject, ActivityActionPreparation, ActivityActionAnnouncement})
	if err != nil || len(actions) != 3 {
		t.Fatalf("valid actions = %v, error = %v", actions, err)
	}
	if _, err := validateActivityActions([]string{ActivityActionProject, ActivityActionProject}); !errors.Is(err, ErrActivityPlanValidation) {
		t.Fatalf("duplicate action error = %v", err)
	}
}

func TestActivityActionProposalsUseEventDates(t *testing.T) {
	start := time.Date(2026, time.August, 20, 8, 0, 0, 0, time.UTC)
	end := start.Add(4 * time.Hour)
	proposals := activityActionProposals(model.ActivityPlan{Title: "开源工作坊", StartsAt: &start, EndsAt: &end})
	if len(proposals) != 6 {
		t.Fatalf("proposal count = %d", len(proposals))
	}
	if proposals[1].DueAt == nil || !proposals[1].DueAt.Equal(start.Add(-14*24*time.Hour)) {
		t.Fatalf("preparation due_at = %v", proposals[1].DueAt)
	}
	if proposals[4].DueAt == nil || !proposals[4].DueAt.Equal(end.Add(3*24*time.Hour)) {
		t.Fatalf("retrospective due_at = %v", proposals[4].DueAt)
	}
}

func TestActivityPlanInputRejectsInvalidDates(t *testing.T) {
	start := time.Now().UTC()
	end := start.Add(-time.Hour)
	input := ActivityPlanCreateInput{
		Title: "活动", Objective: "目标", Audience: "学生", StartsAt: &start, EndsAt: &end,
		ContextRefs: []AgentSourceRef{{Type: "content", ID: "knowledge-1"}},
	}
	if validActivityPlanInput(input) {
		t.Fatal("invalid date range was accepted")
	}
	end = start.Add(time.Hour)
	input.EndsAt = &end
	if !validActivityPlanInput(input) {
		t.Fatal("valid activity plan input was rejected")
	}
}

func TestActivityPlanEvaluationValidationAndAverage(t *testing.T) {
	valid := ActivityPlanEvaluationInput{Accuracy: 5, Feasibility: 4, CampusFit: 5, Clarity: 4, Adoptability: 3, Notes: "仍需确认场地"}
	if !validActivityPlanEvaluation(valid) {
		t.Fatal("valid evaluation was rejected")
	}
	invalid := valid
	invalid.CampusFit = 0
	if validActivityPlanEvaluation(invalid) {
		t.Fatal("evaluation with a zero score was accepted")
	}
	view := activityPlanEvaluationView(model.ActivityPlanEvaluation{
		Accuracy: 5, Feasibility: 4, CampusFit: 5, Clarity: 4, Adoptability: 3,
	})
	if view.OverallScore != 4.2 {
		t.Fatalf("overall score = %v", view.OverallScore)
	}
}

func TestSynchronizedActivityPlanStatusPreservesAppliedPlan(t *testing.T) {
	if status := synchronizedActivityPlanStatus("applied", AgentRunFailed); status != "applied" {
		t.Fatalf("applied plan status = %s", status)
	}
	if status := synchronizedActivityPlanStatus("generating", AgentRunSucceeded); status != "ready" {
		t.Fatalf("succeeded run status = %s", status)
	}
}

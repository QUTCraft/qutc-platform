package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrActivityPlanNotFound       = errors.New("activity plan not found")
	ErrActivityPlanValidation     = errors.New("activity plan validation failed")
	ErrActivityPlanNotReady       = errors.New("activity plan is not ready")
	ErrActivityPlanAlreadyApplied = errors.New("activity plan already applied")
)

const (
	ActivityActionProject       = "create_project"
	ActivityActionPreparation   = "create_preparation_milestone"
	ActivityActionPromotion     = "create_promotion_milestone"
	ActivityActionExecution     = "create_execution_milestone"
	ActivityActionRetrospective = "create_retrospective_milestone"
	ActivityActionAnnouncement  = "create_announcement_draft"
)

type ActivityPlanCreateInput struct {
	Title                string
	Objective            string
	Audience             string
	Venue                string
	StartsAt             *time.Time
	EndsAt               *time.Time
	ExpectedParticipants int
	Budget               string
	Constraints          string
	ContextRefs          []AgentSourceRef
}

type ActivityActionProposal struct {
	Key         string     `json:"key"`
	Kind        string     `json:"kind"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	DueAt       *time.Time `json:"due_at,omitempty"`
	Requires    []string   `json:"requires"`
}

type ActivityPlanView struct {
	ID                    string                   `json:"id"`
	Title                 string                   `json:"title"`
	Objective             string                   `json:"objective"`
	Audience              string                   `json:"audience"`
	Venue                 string                   `json:"venue"`
	StartsAt              *time.Time               `json:"starts_at"`
	EndsAt                *time.Time               `json:"ends_at"`
	ExpectedParticipants  int                      `json:"expected_participants"`
	Budget                string                   `json:"budget"`
	Constraints           string                   `json:"constraints"`
	Status                string                   `json:"status"`
	Run                   AgentRunView             `json:"run"`
	ProposedActions       []ActivityActionProposal `json:"proposed_actions"`
	ApprovedActions       []string                 `json:"approved_actions"`
	ProjectID             *string                  `json:"project_id"`
	AnnouncementContentID *string                  `json:"announcement_content_id"`
	ApprovedBy            *string                  `json:"approved_by"`
	ApprovedAt            *time.Time               `json:"approved_at"`
	CreatedAt             time.Time                `json:"created_at"`
	UpdatedAt             time.Time                `json:"updated_at"`
}

type ActivityPlanSummary struct {
	ID                    string     `json:"id"`
	Title                 string     `json:"title"`
	Status                string     `json:"status"`
	StartsAt              *time.Time `json:"starts_at"`
	EndsAt                *time.Time `json:"ends_at"`
	Provider              string     `json:"provider"`
	Mode                  string     `json:"mode"`
	Model                 string     `json:"model"`
	PromptVersion         string     `json:"prompt_version"`
	ProjectID             *string    `json:"project_id"`
	AnnouncementContentID *string    `json:"announcement_content_id"`
	HasMyEvaluation       bool       `json:"has_my_evaluation"`
	MyEvaluationScore     *float64   `json:"my_evaluation_score"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type ActivityPlanEvaluationInput struct {
	Accuracy     int
	Feasibility  int
	CampusFit    int
	Clarity      int
	Adoptability int
	Notes        string
}

type ActivityPlanEvaluationView struct {
	ID             string    `json:"id"`
	PlanID         string    `json:"plan_id"`
	ReviewerUserID string    `json:"reviewer_user_id"`
	Accuracy       int       `json:"accuracy"`
	Feasibility    int       `json:"feasibility"`
	CampusFit      int       `json:"campus_fit"`
	Clarity        int       `json:"clarity"`
	Adoptability   int       `json:"adoptability"`
	OverallScore   float64   `json:"overall_score"`
	Notes          string    `json:"notes"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ActivityPlanEvaluationDimensions struct {
	Accuracy     float64 `json:"accuracy"`
	Feasibility  float64 `json:"feasibility"`
	CampusFit    float64 `json:"campus_fit"`
	Clarity      float64 `json:"clarity"`
	Adoptability float64 `json:"adoptability"`
}

type ActivityPlanEvaluationModelSummary struct {
	Provider       string  `json:"provider"`
	Mode           string  `json:"mode"`
	Model          string  `json:"model"`
	PromptVersion  string  `json:"prompt_version"`
	Evaluations    int64   `json:"evaluations"`
	EvaluatedPlans int64   `json:"evaluated_plans"`
	AverageScore   float64 `json:"average_score"`
}

type ActivityPlanEvaluationSummaryView struct {
	TotalEvaluations  int64                                `json:"total_evaluations"`
	EvaluatedPlans    int64                                `json:"evaluated_plans"`
	AverageScore      float64                              `json:"average_score"`
	DimensionAverages ActivityPlanEvaluationDimensions     `json:"dimension_averages"`
	ByModel           []ActivityPlanEvaluationModelSummary `json:"by_model"`
	UpdatedAt         *time.Time                           `json:"updated_at"`
}

type ActivityPlanApprovalResult struct {
	ActivityPlanView
	CreatedProjectID    *string  `json:"created_project_id"`
	CreatedMilestoneIDs []string `json:"created_milestone_ids"`
	CreatedContentID    *string  `json:"created_content_id"`
}

func (s *AgentService) CreateActivityPlan(principal Principal, input ActivityPlanCreateInput, requestID string) (ActivityPlanView, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Objective = strings.TrimSpace(input.Objective)
	input.Audience = strings.TrimSpace(input.Audience)
	input.Venue = strings.TrimSpace(input.Venue)
	input.Budget = strings.TrimSpace(input.Budget)
	input.Constraints = strings.TrimSpace(input.Constraints)
	if !validActivityPlanInput(input) {
		return ActivityPlanView{}, ErrActivityPlanValidation
	}
	task := activityPlanTask(input)
	run, err := s.CreateRun(principal, AgentRunCreateInput{
		AgentKey: "activity-planner", Task: task, ContextRefs: input.ContextRefs, OutputMode: "proposal",
	}, requestID)
	if err != nil {
		return ActivityPlanView{}, err
	}
	refs, _ := json.Marshal(input.ContextRefs)
	now := time.Now().UTC()
	plan := model.ActivityPlan{
		ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID,
		AgentRunID: run.ID, Title: input.Title, Objective: input.Objective, Audience: input.Audience,
		Venue: input.Venue, StartsAt: input.StartsAt, EndsAt: input.EndsAt,
		ExpectedParticipants: input.ExpectedParticipants, Budget: input.Budget, Constraints: input.Constraints,
		ContextRefsJSON: string(refs), Status: "generating", ApprovedActionsJSON: "[]",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&plan).Error; err != nil {
			return err
		}
		return tx.Create(&model.AuditEvent{
			ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID,
			Action: "ai.activity_plan_create", TargetType: "activity_plan", TargetID: plan.ID,
			Result: "accepted", RequestID: requestID, CreatedAt: now,
		}).Error
	}); err != nil {
		return ActivityPlanView{}, err
	}
	return s.GetActivityPlan(principal.OrganizationID, plan.ID)
}

func (s *AgentService) GetActivityPlan(organizationID, planID string) (ActivityPlanView, error) {
	var plan model.ActivityPlan
	if err := s.db.Where("id = ? AND organization_id = ?", strings.TrimSpace(planID), organizationID).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ActivityPlanView{}, ErrActivityPlanNotFound
		}
		return ActivityPlanView{}, err
	}
	run, err := s.GetRun(organizationID, plan.AgentRunID)
	if err != nil {
		return ActivityPlanView{}, err
	}
	status := plan.Status
	if status != "applied" {
		switch run.Status {
		case AgentRunSucceeded:
			status = "ready"
		case AgentRunFailed:
			status = "failed"
		case AgentRunCanceled:
			status = "canceled"
		default:
			status = "generating"
		}
		if status != plan.Status {
			_ = s.db.Model(&model.ActivityPlan{}).Where("id = ? AND organization_id = ?", plan.ID, organizationID).Update("status", status).Error
			plan.Status = status
		}
	}
	var approved []string
	_ = json.Unmarshal([]byte(plan.ApprovedActionsJSON), &approved)
	return ActivityPlanView{
		ID: plan.ID, Title: plan.Title, Objective: plan.Objective, Audience: plan.Audience,
		Venue: plan.Venue, StartsAt: plan.StartsAt, EndsAt: plan.EndsAt,
		ExpectedParticipants: plan.ExpectedParticipants, Budget: plan.Budget, Constraints: plan.Constraints,
		Status: plan.Status, Run: run, ProposedActions: activityActionProposals(plan), ApprovedActions: approved,
		ProjectID: plan.ProjectID, AnnouncementContentID: plan.AnnouncementContentID,
		ApprovedBy: plan.ApprovedBy, ApprovedAt: plan.ApprovedAt, CreatedAt: plan.CreatedAt, UpdatedAt: plan.UpdatedAt,
	}, nil
}

func (s *AgentService) ListActivityPlans(principal Principal, page, pageSize int) ([]ActivityPlanSummary, int64, error) {
	query := s.db.Model(&model.ActivityPlan{}).Where("organization_id = ?", principal.OrganizationID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var plans []model.ActivityPlan
	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&plans).Error; err != nil {
		return nil, 0, err
	}
	runIDs := make([]string, 0, len(plans))
	for _, plan := range plans {
		runIDs = append(runIDs, plan.AgentRunID)
	}
	var runs []model.AgentRun
	if len(runIDs) > 0 {
		if err := s.db.Where("organization_id = ? AND id IN ?", principal.OrganizationID, runIDs).Find(&runs).Error; err != nil {
			return nil, 0, err
		}
	}
	runByID := make(map[string]model.AgentRun, len(runs))
	for _, run := range runs {
		runByID[run.ID] = run
	}
	evaluationByPlanID := make(map[string]float64, len(plans))
	if len(plans) > 0 {
		planIDs := make([]string, 0, len(plans))
		for _, plan := range plans {
			planIDs = append(planIDs, plan.ID)
		}
		var evaluations []model.ActivityPlanEvaluation
		if err := s.db.Where("organization_id = ? AND reviewer_user_id = ? AND plan_id IN ?", principal.OrganizationID, principal.UserID, planIDs).Find(&evaluations).Error; err != nil {
			return nil, 0, err
		}
		for _, evaluation := range evaluations {
			evaluationByPlanID[evaluation.PlanID] = activityPlanEvaluationView(evaluation).OverallScore
		}
	}
	items := make([]ActivityPlanSummary, 0, len(plans))
	for index := range plans {
		plan := &plans[index]
		run, exists := runByID[plan.AgentRunID]
		if !exists {
			return nil, 0, fmt.Errorf("activity plan %s references missing agent run", plan.ID)
		}
		status := synchronizedActivityPlanStatus(plan.Status, run.Status)
		if status != plan.Status {
			now := time.Now().UTC()
			_ = s.db.Model(&model.ActivityPlan{}).Where("id = ? AND organization_id = ?", plan.ID, principal.OrganizationID).Updates(map[string]any{"status": status, "updated_at": now}).Error
			plan.Status = status
			plan.UpdatedAt = now
		}
		var myEvaluationScore *float64
		if score, evaluated := evaluationByPlanID[plan.ID]; evaluated {
			myEvaluationScore = &score
		}
		items = append(items, ActivityPlanSummary{
			ID: plan.ID, Title: plan.Title, Status: plan.Status, StartsAt: plan.StartsAt, EndsAt: plan.EndsAt,
			Provider: run.Provider, Mode: run.Mode, Model: run.Model, PromptVersion: run.PromptVersion,
			ProjectID: plan.ProjectID, AnnouncementContentID: plan.AnnouncementContentID,
			HasMyEvaluation: myEvaluationScore != nil, MyEvaluationScore: myEvaluationScore,
			CreatedAt: plan.CreatedAt, UpdatedAt: plan.UpdatedAt,
		})
	}
	return items, total, nil
}

func (s *AgentService) GetActivityPlanEvaluation(principal Principal, planID string) (*ActivityPlanEvaluationView, error) {
	if _, err := s.GetActivityPlan(principal.OrganizationID, planID); err != nil {
		return nil, err
	}
	var evaluation model.ActivityPlanEvaluation
	err := s.db.Where("organization_id = ? AND plan_id = ? AND reviewer_user_id = ?", principal.OrganizationID, strings.TrimSpace(planID), principal.UserID).First(&evaluation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	view := activityPlanEvaluationView(evaluation)
	return &view, nil
}

func (s *AgentService) SaveActivityPlanEvaluation(principal Principal, planID string, input ActivityPlanEvaluationInput, requestID string) (ActivityPlanEvaluationView, error) {
	planID = strings.TrimSpace(planID)
	plan, err := s.GetActivityPlan(principal.OrganizationID, planID)
	if err != nil {
		return ActivityPlanEvaluationView{}, err
	}
	if plan.Status != "ready" && plan.Status != "applied" {
		return ActivityPlanEvaluationView{}, ErrActivityPlanNotReady
	}
	input.Notes = strings.TrimSpace(input.Notes)
	if !validActivityPlanEvaluation(input) {
		return ActivityPlanEvaluationView{}, ErrActivityPlanValidation
	}
	now := time.Now().UTC()
	var evaluation model.ActivityPlanEvaluation
	err = s.db.Transaction(func(tx *gorm.DB) error {
		lookup := tx.Where("organization_id = ? AND plan_id = ? AND reviewer_user_id = ?", principal.OrganizationID, planID, principal.UserID).First(&evaluation).Error
		if errors.Is(lookup, gorm.ErrRecordNotFound) {
			evaluation = model.ActivityPlanEvaluation{
				ID: uuid.NewString(), OrganizationID: principal.OrganizationID, PlanID: planID, ReviewerUserID: principal.UserID,
				CreatedAt: now,
			}
		} else if lookup != nil {
			return lookup
		}
		evaluation.Accuracy = input.Accuracy
		evaluation.Feasibility = input.Feasibility
		evaluation.CampusFit = input.CampusFit
		evaluation.Clarity = input.Clarity
		evaluation.Adoptability = input.Adoptability
		evaluation.Notes = input.Notes
		evaluation.UpdatedAt = now
		if err := tx.Save(&evaluation).Error; err != nil {
			return err
		}
		return createActivityAudit(tx, principal, requestID, "ai.activity_plan_evaluate", "activity_plan", planID)
	})
	if err != nil {
		return ActivityPlanEvaluationView{}, err
	}
	return activityPlanEvaluationView(evaluation), nil
}

func (s *AgentService) ActivityPlanEvaluationSummary(organizationID string) (ActivityPlanEvaluationSummaryView, error) {
	var aggregate struct {
		TotalEvaluations int64
		EvaluatedPlans   int64
		Accuracy         float64
		Feasibility      float64
		CampusFit        float64
		Clarity          float64
		Adoptability     float64
		AverageScore     float64
		UpdatedAt        *time.Time
	}
	if err := s.db.Raw(`
		SELECT COUNT(*) AS total_evaluations,
		       COUNT(DISTINCT plan_id) AS evaluated_plans,
		       COALESCE(AVG(accuracy), 0) AS accuracy,
		       COALESCE(AVG(feasibility), 0) AS feasibility,
		       COALESCE(AVG(campus_fit), 0) AS campus_fit,
		       COALESCE(AVG(clarity), 0) AS clarity,
		       COALESCE(AVG(adoptability), 0) AS adoptability,
		       COALESCE(AVG((accuracy + feasibility + campus_fit + clarity + adoptability) / 5.0), 0) AS average_score,
		       MAX(updated_at) AS updated_at
		FROM activity_plan_evaluations
		WHERE organization_id = ?`, organizationID).Scan(&aggregate).Error; err != nil {
		return ActivityPlanEvaluationSummaryView{}, err
	}
	byModel := make([]ActivityPlanEvaluationModelSummary, 0)
	if err := s.db.Raw(`
		SELECT runs.provider, runs.mode, runs.model, runs.prompt_version,
		       COUNT(*) AS evaluations,
		       COUNT(DISTINCT evaluations.plan_id) AS evaluated_plans,
		       COALESCE(AVG((evaluations.accuracy + evaluations.feasibility + evaluations.campus_fit + evaluations.clarity + evaluations.adoptability) / 5.0), 0) AS average_score
		FROM activity_plan_evaluations evaluations
		JOIN activity_plans plans ON BINARY evaluations.plan_id = BINARY plans.id
		JOIN agent_runs runs ON BINARY plans.agent_run_id = BINARY runs.id
		WHERE evaluations.organization_id = ?
		GROUP BY runs.provider, runs.mode, runs.model, runs.prompt_version
		ORDER BY COUNT(*) DESC, average_score DESC, runs.model ASC`, organizationID).Scan(&byModel).Error; err != nil {
		return ActivityPlanEvaluationSummaryView{}, err
	}
	return ActivityPlanEvaluationSummaryView{
		TotalEvaluations: aggregate.TotalEvaluations, EvaluatedPlans: aggregate.EvaluatedPlans,
		AverageScore: aggregate.AverageScore,
		DimensionAverages: ActivityPlanEvaluationDimensions{
			Accuracy: aggregate.Accuracy, Feasibility: aggregate.Feasibility, CampusFit: aggregate.CampusFit,
			Clarity: aggregate.Clarity, Adoptability: aggregate.Adoptability,
		},
		ByModel: byModel, UpdatedAt: aggregate.UpdatedAt,
	}, nil
}

func (s *AgentService) ApproveActivityPlan(principal Principal, planID string, actions []string, requestID string) (ActivityPlanApprovalResult, error) {
	view, err := s.GetActivityPlan(principal.OrganizationID, planID)
	if err != nil {
		return ActivityPlanApprovalResult{}, err
	}
	if view.Status == "applied" {
		return ActivityPlanApprovalResult{}, ErrActivityPlanAlreadyApplied
	}
	if view.Status != "ready" || view.Run.Status != AgentRunSucceeded {
		return ActivityPlanApprovalResult{}, ErrActivityPlanNotReady
	}
	selected, err := validateActivityActions(actions)
	if err != nil {
		return ActivityPlanApprovalResult{}, err
	}
	var plan model.ActivityPlan
	if err := s.db.Where("id = ? AND organization_id = ?", planID, principal.OrganizationID).First(&plan).Error; err != nil {
		return ActivityPlanApprovalResult{}, err
	}
	now := time.Now().UTC()
	approvedJSON, _ := json.Marshal(selected)
	var projectID, contentID *string
	createdMilestones := make([]string, 0, 4)
	err = s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.ActivityPlan{}).
			Where("id = ? AND organization_id = ? AND status = ?", plan.ID, principal.OrganizationID, "ready").
			Updates(map[string]any{"status": "applying", "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrActivityPlanAlreadyApplied
		}
		if containsAction(selected, ActivityActionProject) {
			project := model.Project{
				ID: uuid.NewString(), OrganizationID: principal.OrganizationID, OwnerUserID: principal.UserID,
				Title: boundedText(plan.Title, 160), Summary: boundedText(plan.Objective, 500),
				Status: "research", Tags: "AI活动策划,校园活动", IsPublic: false,
			}
			if err := tx.Create(&project).Error; err != nil {
				return err
			}
			if err := tx.Create(&model.ProjectMember{ProjectID: project.ID, UserID: principal.UserID, Role: "owner"}).Error; err != nil {
				return err
			}
			projectID = &project.ID
			if err := createActivityAudit(tx, principal, requestID, "project.create_from_ai", "project", project.ID); err != nil {
				return err
			}
			for _, proposal := range activityActionProposals(plan) {
				if proposal.Kind != "milestone" || !containsAction(selected, proposal.Key) {
					continue
				}
				milestone := model.ProjectMilestone{ID: uuid.NewString(), ProjectID: project.ID, Title: proposal.Title, Status: "planned", DueAt: proposal.DueAt}
				if err := tx.Create(&milestone).Error; err != nil {
					return err
				}
				createdMilestones = append(createdMilestones, milestone.ID)
				if err := createActivityAudit(tx, principal, requestID, "project_milestone.create_from_ai", "project_milestone", milestone.ID); err != nil {
					return err
				}
			}
		}
		if containsAction(selected, ActivityActionAnnouncement) {
			content := model.Content{
				ID: uuid.NewString(), OrganizationID: principal.OrganizationID, AuthorUserID: principal.UserID,
				Title: boundedText("活动预告｜"+plan.Title, 160), Type: ContentTypeNews, Category: "校园活动",
				Status: ContentStatusDraft, Excerpt: boundedText(plan.Objective, 500), Body: view.Run.OutputMarkdown,
			}
			if err := tx.Create(&content).Error; err != nil {
				return err
			}
			contentID = &content.ID
			if err := createActivityAudit(tx, principal, requestID, "content.create_from_ai", "content", content.ID); err != nil {
				return err
			}
		}
		updates := map[string]any{
			"status": "applied", "approved_by": principal.UserID, "approved_at": now,
			"approved_actions_json": string(approvedJSON), "project_id": projectID,
			"announcement_content_id": contentID, "updated_at": now,
		}
		if err := tx.Model(&model.ActivityPlan{}).Where("id = ? AND organization_id = ?", plan.ID, principal.OrganizationID).Updates(updates).Error; err != nil {
			return err
		}
		return createActivityAudit(tx, principal, requestID, "ai.activity_plan_approve", "activity_plan", plan.ID)
	})
	if err != nil {
		return ActivityPlanApprovalResult{}, err
	}
	updated, err := s.GetActivityPlan(principal.OrganizationID, plan.ID)
	if err != nil {
		return ActivityPlanApprovalResult{}, err
	}
	return ActivityPlanApprovalResult{ActivityPlanView: updated, CreatedProjectID: projectID, CreatedMilestoneIDs: createdMilestones, CreatedContentID: contentID}, nil
}

func validActivityPlanInput(input ActivityPlanCreateInput) bool {
	if input.Title == "" || len([]rune(input.Title)) > 160 || input.Objective == "" || len([]rune(input.Objective)) > 1000 ||
		input.Audience == "" || len([]rune(input.Audience)) > 500 || len([]rune(input.Venue)) > 300 ||
		input.ExpectedParticipants < 0 || input.ExpectedParticipants > 100000 || len([]rune(input.Budget)) > 200 ||
		len([]rune(input.Constraints)) > 2000 || len(input.ContextRefs) < 1 || len(input.ContextRefs) > 10 {
		return false
	}
	return input.StartsAt == nil || input.EndsAt == nil || input.EndsAt.After(*input.StartsAt)
}

func activityPlanTask(input ActivityPlanCreateInput) string {
	start, end := "待确定", "待确定"
	if input.StartsAt != nil {
		start = input.StartsAt.Format(time.RFC3339)
	}
	if input.EndsAt != nil {
		end = input.EndsAt.Format(time.RFC3339)
	}
	return fmt.Sprintf("%s\n请为以下校园组织活动生成可执行策划方案。\n活动目标：%s\n目标受众：%s\n场地：%s\n开始时间：%s\n结束时间：%s\n预计参与人数：%d\n预算：%s\n其他约束：%s", input.Title, input.Objective, input.Audience, input.Venue, start, end, input.ExpectedParticipants, input.Budget, input.Constraints)
}

func activityActionProposals(plan model.ActivityPlan) []ActivityActionProposal {
	var preparation, promotion, execution, retrospective *time.Time
	if plan.StartsAt != nil {
		preparationAt := plan.StartsAt.Add(-14 * 24 * time.Hour)
		promotionAt := plan.StartsAt.Add(-3 * 24 * time.Hour)
		executionAt := *plan.StartsAt
		preparation, promotion, execution = &preparationAt, &promotionAt, &executionAt
	}
	if plan.EndsAt != nil {
		retrospectiveAt := plan.EndsAt.Add(3 * 24 * time.Hour)
		retrospective = &retrospectiveAt
	}
	return []ActivityActionProposal{
		{Key: ActivityActionProject, Kind: "project", Title: plan.Title, Description: "创建非公开项目，后续由负责人继续维护。", Requires: []string{}},
		{Key: ActivityActionPreparation, Kind: "milestone", Title: "完成活动方案与审批确认", Description: "核对场地、规则、预算和负责人。", DueAt: preparation, Requires: []string{ActivityActionProject}},
		{Key: ActivityActionPromotion, Kind: "milestone", Title: "完成宣传与人员确认", Description: "完成宣传材料、报名信息和执行分工。", DueAt: promotion, Requires: []string{ActivityActionProject}},
		{Key: ActivityActionExecution, Kind: "milestone", Title: "活动执行与现场保障", Description: "执行活动并记录关键事实与异常。", DueAt: execution, Requires: []string{ActivityActionProject}},
		{Key: ActivityActionRetrospective, Kind: "milestone", Title: "完成活动总结与知识沉淀", Description: "整理总结、改进项与可复用资料。", DueAt: retrospective, Requires: []string{ActivityActionProject}},
		{Key: ActivityActionAnnouncement, Kind: "content", Title: "活动预告｜" + plan.Title, Description: "把 AI 方案作为新闻草稿创建；不会自动发布。", Requires: []string{}},
	}
}

func validateActivityActions(actions []string) ([]string, error) {
	if len(actions) == 0 || len(actions) > 6 {
		return nil, ErrActivityPlanValidation
	}
	allowed := map[string]bool{ActivityActionProject: true, ActivityActionPreparation: true, ActivityActionPromotion: true, ActivityActionExecution: true, ActivityActionRetrospective: true, ActivityActionAnnouncement: true}
	seen := map[string]bool{}
	selected := make([]string, 0, len(actions))
	for _, action := range actions {
		action = strings.TrimSpace(action)
		if !allowed[action] || seen[action] {
			return nil, ErrActivityPlanValidation
		}
		seen[action] = true
		selected = append(selected, action)
	}
	if (seen[ActivityActionPreparation] || seen[ActivityActionPromotion] || seen[ActivityActionExecution] || seen[ActivityActionRetrospective]) && !seen[ActivityActionProject] {
		return nil, ErrActivityPlanValidation
	}
	return selected, nil
}

func containsAction(actions []string, target string) bool {
	for _, action := range actions {
		if action == target {
			return true
		}
	}
	return false
}

func synchronizedActivityPlanStatus(planStatus, runStatus string) string {
	if planStatus == "applied" || planStatus == "applying" {
		return planStatus
	}
	switch runStatus {
	case AgentRunSucceeded:
		return "ready"
	case AgentRunFailed:
		return "failed"
	case AgentRunCanceled:
		return "canceled"
	default:
		return "generating"
	}
}

func validActivityPlanEvaluation(input ActivityPlanEvaluationInput) bool {
	scores := []int{input.Accuracy, input.Feasibility, input.CampusFit, input.Clarity, input.Adoptability}
	for _, score := range scores {
		if score < 1 || score > 5 {
			return false
		}
	}
	return len([]rune(input.Notes)) <= 1000
}

func activityPlanEvaluationView(evaluation model.ActivityPlanEvaluation) ActivityPlanEvaluationView {
	overall := float64(evaluation.Accuracy+evaluation.Feasibility+evaluation.CampusFit+evaluation.Clarity+evaluation.Adoptability) / 5
	return ActivityPlanEvaluationView{
		ID: evaluation.ID, PlanID: evaluation.PlanID, ReviewerUserID: evaluation.ReviewerUserID,
		Accuracy: evaluation.Accuracy, Feasibility: evaluation.Feasibility, CampusFit: evaluation.CampusFit,
		Clarity: evaluation.Clarity, Adoptability: evaluation.Adoptability, OverallScore: overall,
		Notes: evaluation.Notes, CreatedAt: evaluation.CreatedAt, UpdatedAt: evaluation.UpdatedAt,
	}
}

func createActivityAudit(tx *gorm.DB, principal Principal, requestID, action, targetType, targetID string) error {
	return tx.Create(&model.AuditEvent{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID, Action: action, TargetType: targetType, TargetID: targetID, Result: "success", RequestID: requestID, CreatedAt: time.Now().UTC()}).Error
}

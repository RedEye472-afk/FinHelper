// Package http: goals.go wires the goals REST endpoints
// (BUSINESS_LOGIC.md ф.5 — savings-goal tracker).
//
// Routes (mounted by the router under /api/v1, behind AuthMiddleware):
//
//	GET    /goals                       list goals
//	POST   /goals                       create goal
//	GET    /goals/{id}                  get one goal
//	PATCH  /goals/{id}                  update goal
//	DELETE /goals/{id}                  soft-delete goal
//	GET    /goals/{id}/projection       compute projection (ф.5 «C»)
//	POST   /goals/{id}/simulate         what-if against stored goal
//	GET    /goals/{id}/contributions    list contributions (journal)
//	POST   /goals/{id}/contributions    add contribution (idempotent)
//	DELETE /goals/{id}/contributions/{cid}  delete contribution
//	POST   /calc/goal                   stateless what-if (no storage)
package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/backend/pkg/domain"
	applog "github.com/RedEye472-afk/FinHelper/backend/pkg/log"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/goals"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/storage"
)

// GoalsHandler wires the goals REST endpoints (BUSINESS_LOGIC.md ф.5).
type GoalsHandler struct {
	svc    *goals.Service
	logger *slog.Logger
}

// NewGoalsHandler rejects nil deps at boot.
func NewGoalsHandler(svc *goals.Service, logger *slog.Logger) *GoalsHandler {
	if svc == nil || logger == nil {
		panic("http: NewGoalsHandler requires non-nil deps")
	}
	return &GoalsHandler{svc: svc, logger: logger}
}

// Register mounts the goals routes on the authenticated chi router.
func (h *GoalsHandler) Register(r chi.Router) {
	r.Get("/goals", h.List)
	r.Post("/goals", h.Create)
	r.Get("/goals/{id}", h.Get)
	r.Patch("/goals/{id}", h.Update)
	r.Delete("/goals/{id}", h.Delete)
	r.Get("/goals/{id}/projection", h.Projection)
	r.Post("/goals/{id}/simulate", h.SimulateSaved)
	r.Get("/goals/{id}/contributions", h.ListContributions)
	r.Post("/goals/{id}/contributions", h.AddContribution)
	r.Delete("/goals/{id}/contributions/{cid}", h.DeleteContribution)
	r.Post("/calc/goal", h.Simulate)
}

// ---- Request / response shapes ----

type goalResponse struct {
	ID                  int64  `json:"id"`
	Name                string `json:"name"`
	TargetAmount        string `json:"target_amount"`
	CurrentAmount       string `json:"current_amount"`
	MonthlyContribution string `json:"monthly_contribution,omitempty"` // empty when nil
	TargetDate          string `json:"target_date,omitempty"`           // RFC3339, empty when nil
	ExpectedYield       string `json:"expected_yield"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
}

type createGoalRequest struct {
	Name                string `json:"name"`
	TargetAmount        string `json:"target_amount"`
	CurrentAmount       string `json:"current_amount,omitempty"`        // default "0"
	MonthlyContribution string `json:"monthly_contribution,omitempty"`  // empty = nil
	TargetDate          string `json:"target_date,omitempty"`           // RFC3339, empty = nil
	ExpectedYield       string `json:"expected_yield,omitempty"`        // default "0"
}

type updateGoalRequest struct {
	Name                string `json:"name"`
	TargetAmount        string `json:"target_amount"`
	CurrentAmount       string `json:"current_amount"`
	MonthlyContribution string `json:"monthly_contribution"`
	TargetDate          string `json:"target_date"`
	ExpectedYield       string `json:"expected_yield"`
}

type simulateRequest struct {
	CurrentAmount       string `json:"current_amount,omitempty"`
	TargetAmount        string `json:"target_amount"`
	MonthlyContribution string `json:"monthly_contribution,omitempty"`
	TargetDate          string `json:"target_date,omitempty"`
	ExpectedYield       string `json:"expected_yield,omitempty"`
	Inflation           string `json:"inflation,omitempty"`
}

type contributionResponse struct {
	ID             int64  `json:"id"`
	GoalID         int64  `json:"goal_id"`
	ContributionID string `json:"contribution_id"`
	Amount         string `json:"amount"`
	Date           string `json:"date"`
	Comment        string `json:"comment,omitempty"`
	CreatedAt      string `json:"created_at"`
}

type addContributionRequest struct {
	ContributionID string `json:"contribution_id"`
	Amount         string `json:"amount"`
	Date           string `json:"date,omitempty"`    // RFC3339, optional
	Comment        string `json:"comment,omitempty"`
}

type projectionResponse struct {
	Goal             goalResponse `json:"goal"`
	EffectiveCurrent string       `json:"effective_current"`
	TargetEffective  string       `json:"target_effective"`
	Progress         string       `json:"progress"`
	MonthsLeft       int          `json:"months_left"`
	RequiredMonthly  string       `json:"required_monthly,omitempty"`
	EstimatedMonths  int          `json:"estimated_months,omitempty"`
	Status           string       `json:"status"`
	AsOf             string       `json:"as_of"`
}

// ---- Handlers ----

func (h *GoalsHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	var req createGoalRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "goals.invalid_body", err.Error())
		return
	}
	in, err := buildCreateInput(userID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "goals.invalid_argument", err.Error())
		return
	}
	g, err := h.svc.Create(ctx, in)
	if err != nil {
		h.writeServiceError(w, err, "create")
		return
	}
	applog.Info(ctx, h.logger, "goal created", "goal_id", g.ID)
	writeJSON(w, http.StatusCreated, toGoalResponse(g))
}

func (h *GoalsHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	gs, err := h.svc.List(ctx, userID)
	if err != nil {
		h.writeServiceError(w, err, "list")
		return
	}
	out := make([]goalResponse, 0, len(gs))
	for _, g := range gs {
		out = append(out, toGoalResponse(g))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *GoalsHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "goals.invalid_id", "id must be a positive integer")
		return
	}
	g, err := h.svc.Get(ctx, userID, id)
	if err != nil {
		h.writeServiceError(w, err, "get")
		return
	}
	writeJSON(w, http.StatusOK, toGoalResponse(g))
}

func (h *GoalsHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "goals.invalid_id", "id must be a positive integer")
		return
	}
	var req updateGoalRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "goals.invalid_body", err.Error())
		return
	}
	in, err := buildUpdateInput(userID, id, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "goals.invalid_argument", err.Error())
		return
	}
	g, err := h.svc.Update(ctx, in)
	if err != nil {
		h.writeServiceError(w, err, "update")
		return
	}
	writeJSON(w, http.StatusOK, toGoalResponse(g))
}

func (h *GoalsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "goals.invalid_id", "id must be a positive integer")
		return
	}
	if err := h.svc.Delete(ctx, userID, id); err != nil {
		h.writeServiceError(w, err, "delete")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Projection → GET /api/v1/goals/{id}/projection. Computes the goal's status
// projection including the contribution journal (BUSINESS_LOGIC ф.5 «C»).
func (h *GoalsHandler) Projection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "goals.invalid_id", "id must be a positive integer")
		return
	}
	proj, err := h.svc.Compute(ctx, userID, id)
	if err != nil {
		h.writeServiceError(w, err, "projection")
		return
	}
	writeJSON(w, http.StatusOK, toProjectionResponse(proj))
}

// Simulate → POST /api/v1/calc/goal. Stateless what-if: no storage read,
// no persistence. Used by prospects before sign-up (BUSINESS_LOGIC ф.12 hook).
func (h *GoalsHandler) Simulate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// /calc/goal is intentionally unauthenticated-friendly, but the router
	// mounts it inside the authenticated group, so a user_id is present.
	userID, _ := MustUserID(ctx)
	_ = userID // not used by Simulate; kept for symmetry/logging
	var req simulateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "goals.invalid_body", err.Error())
		return
	}
	in, err := buildSimulateInput(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "goals.invalid_argument", err.Error())
		return
	}
	proj, err := h.svc.Simulate(ctx, in)
	if err != nil {
		h.writeServiceError(w, err, "simulate")
		return
	}
	writeJSON(w, http.StatusOK, toProjectionResponse(proj))
}

// SimulateSaved → POST /api/v1/goals/{id}/simulate. What-if against a stored
// goal: body fields override the stored ones where non-zero/non-nil.
func (h *GoalsHandler) SimulateSaved(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "goals.invalid_id", "id must be a positive integer")
		return
	}
	var req simulateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "goals.invalid_body", err.Error())
		return
	}
	in, err := buildSimulateInput(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "goals.invalid_argument", err.Error())
		return
	}
	proj, err := h.svc.SimulateSaved(ctx, userID, id, in)
	if err != nil {
		h.writeServiceError(w, err, "simulate_saved")
		return
	}
	writeJSON(w, http.StatusOK, toProjectionResponse(proj))
}

func (h *GoalsHandler) ListContributions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	goalID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || goalID <= 0 {
		writeError(w, http.StatusBadRequest, "goals.invalid_id", "id must be a positive integer")
		return
	}
	cs, err := h.svc.ListContributions(ctx, userID, goalID)
	if err != nil {
		h.writeServiceError(w, err, "list_contributions")
		return
	}
	out := make([]contributionResponse, 0, len(cs))
	for _, c := range cs {
		out = append(out, toContributionResponse(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *GoalsHandler) AddContribution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	goalID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || goalID <= 0 {
		writeError(w, http.StatusBadRequest, "goals.invalid_id", "id must be a positive integer")
		return
	}
	var req addContributionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "goals.invalid_body", err.Error())
		return
	}
	if req.ContributionID == "" {
		writeError(w, http.StatusBadRequest, "goals.invalid_argument", "contribution_id required")
		return
	}
	amount, err := domain.ParseMoney(req.Amount)
	if err != nil || !amount.IsPositive() {
		writeError(w, http.StatusBadRequest, "goals.invalid_amount", "amount must be a positive amount")
		return
	}
	var date time.Time
	if req.Date != "" {
		t, pErr := time.Parse(time.RFC3339, req.Date)
		if pErr != nil {
			writeError(w, http.StatusBadRequest, "goals.invalid_date", "date must be RFC3339")
			return
		}
		date = t
	}
	c, duplicate, err := h.svc.AddContribution(ctx, goals.AddContributionInput{
		UserID:         userID,
		GoalID:         goalID,
		ContributionID: req.ContributionID,
		Amount:         amount,
		Date:           date,
		Comment:        req.Comment,
	})
	if err != nil {
		h.writeServiceError(w, err, "add_contribution")
		return
	}
	applog.Info(ctx, h.logger, "goal contribution added",
		"goal_id", goalID, "contribution_id", c.ID, "duplicate", duplicate)
	status := http.StatusCreated
	if duplicate {
		status = http.StatusOK
	}
	writeJSON(w, status, toContributionResponse(c))
}

func (h *GoalsHandler) DeleteContribution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	goalID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || goalID <= 0 {
		writeError(w, http.StatusBadRequest, "goals.invalid_id", "id must be a positive integer")
		return
	}
	cid, err := strconv.ParseInt(chi.URLParam(r, "cid"), 10, 64)
	if err != nil || cid <= 0 {
		writeError(w, http.StatusBadRequest, "goals.invalid_contribution_id", "cid must be a positive integer")
		return
	}
	if err := h.svc.DeleteContribution(ctx, userID, goalID, cid); err != nil {
		h.writeServiceError(w, err, "delete_contribution")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- helpers ----

// buildCreateInput parses a createGoalRequest into goals.CreateInput.
// Money fields use domain.ParseMoney; dates use RFC3339; empty optional
// fields default to zero/nil so the service can apply its own defaults.
func buildCreateInput(userID int64, req createGoalRequest) (goals.CreateInput, error) {
	target, err := domain.ParseMoney(req.TargetAmount)
	if err != nil {
		return goals.CreateInput{}, errors.New("goals: target_amount must be a valid amount")
	}
	current := domain.Zero
	if req.CurrentAmount != "" {
		c, err := domain.ParseMoney(req.CurrentAmount)
		if err != nil {
			return goals.CreateInput{}, errors.New("goals: current_amount must be a valid amount")
		}
		current = c
	}
	var monthly *domain.Money
	if req.MonthlyContribution != "" {
		m, err := domain.ParseMoney(req.MonthlyContribution)
		if err != nil {
			return goals.CreateInput{}, errors.New("goals: monthly_contribution must be a valid amount")
		}
		monthly = &m
	}
	var targetDate *time.Time
	if req.TargetDate != "" {
		t, err := time.Parse(time.RFC3339, req.TargetDate)
		if err != nil {
			return goals.CreateInput{}, errors.New("goals: target_date must be RFC3339")
		}
		targetDate = &t
	}
	yield := decimal.Zero
	if req.ExpectedYield != "" {
		y, err := decimal.NewFromString(req.ExpectedYield)
		if err != nil {
			return goals.CreateInput{}, errors.New("goals: expected_yield must be a valid decimal")
		}
		yield = y
	}
	return goals.CreateInput{
		UserID:              userID,
		Name:                req.Name,
		TargetAmount:        target,
		CurrentAmount:       current,
		MonthlyContribution: monthly,
		TargetDate:          targetDate,
		ExpectedYield:       yield,
	}, nil
}

// buildUpdateInput parses an updateGoalRequest into goals.UpdateInput.
// Empty MonthlyContribution / TargetDate strings are interpreted as nil
// (clearing the field), matching the service's overwrite semantics.
func buildUpdateInput(userID, id int64, req updateGoalRequest) (goals.UpdateInput, error) {
	target, err := domain.ParseMoney(req.TargetAmount)
	if err != nil {
		return goals.UpdateInput{}, errors.New("goals: target_amount must be a valid amount")
	}
	current := domain.Zero
	if req.CurrentAmount != "" {
		c, err := domain.ParseMoney(req.CurrentAmount)
		if err != nil {
			return goals.UpdateInput{}, errors.New("goals: current_amount must be a valid amount")
		}
		current = c
	}
	var monthly *domain.Money
	if req.MonthlyContribution != "" {
		m, err := domain.ParseMoney(req.MonthlyContribution)
		if err != nil {
			return goals.UpdateInput{}, errors.New("goals: monthly_contribution must be a valid amount")
		}
		monthly = &m
	}
	var targetDate *time.Time
	if req.TargetDate != "" {
		t, err := time.Parse(time.RFC3339, req.TargetDate)
		if err != nil {
			return goals.UpdateInput{}, errors.New("goals: target_date must be RFC3339")
		}
		targetDate = &t
	}
	yield := decimal.Zero
	if req.ExpectedYield != "" {
		y, err := decimal.NewFromString(req.ExpectedYield)
		if err != nil {
			return goals.UpdateInput{}, errors.New("goals: expected_yield must be a valid decimal")
		}
		yield = y
	}
	return goals.UpdateInput{
		UserID:              userID,
		ID:                  id,
		Name:                req.Name,
		TargetAmount:        target,
		CurrentAmount:       current,
		MonthlyContribution: monthly,
		TargetDate:          targetDate,
		ExpectedYield:       yield,
	}, nil
}

// buildSimulateInput parses a simulateRequest into goals.SimulateInput.
func buildSimulateInput(req simulateRequest) (goals.SimulateInput, error) {
	target, err := domain.ParseMoney(req.TargetAmount)
	if err != nil {
		return goals.SimulateInput{}, errors.New("goals: target_amount must be a valid amount")
	}
	var current domain.Money
	if req.CurrentAmount != "" {
		c, err := domain.ParseMoney(req.CurrentAmount)
		if err != nil {
			return goals.SimulateInput{}, errors.New("goals: current_amount must be a valid amount")
		}
		current = c
	}
	var monthly *domain.Money
	if req.MonthlyContribution != "" {
		m, err := domain.ParseMoney(req.MonthlyContribution)
		if err != nil {
			return goals.SimulateInput{}, errors.New("goals: monthly_contribution must be a valid amount")
		}
		monthly = &m
	}
	var targetDate *time.Time
	if req.TargetDate != "" {
		t, err := time.Parse(time.RFC3339, req.TargetDate)
		if err != nil {
			return goals.SimulateInput{}, errors.New("goals: target_date must be RFC3339")
		}
		targetDate = &t
	}
	yield := decimal.Zero
	if req.ExpectedYield != "" {
		y, err := decimal.NewFromString(req.ExpectedYield)
		if err != nil {
			return goals.SimulateInput{}, errors.New("goals: expected_yield must be a valid decimal")
		}
		yield = y
	}
	inflation := decimal.Zero
	if req.Inflation != "" {
		v, err := decimal.NewFromString(req.Inflation)
		if err != nil {
			return goals.SimulateInput{}, errors.New("goals: inflation must be a valid decimal")
		}
		inflation = v
	}
	return goals.SimulateInput{
		CurrentAmount:       current,
		TargetAmount:        target,
		MonthlyContribution: monthly,
		TargetDate:          targetDate,
		ExpectedYield:       yield,
		Inflation:           inflation,
	}, nil
}

func toGoalResponse(g domain.Goal) goalResponse {
	resp := goalResponse{
		ID:            g.ID,
		Name:          g.Name,
		TargetAmount:  g.TargetAmount.String(),
		CurrentAmount: g.CurrentAmount.String(),
		ExpectedYield: g.ExpectedYield.String(),
		CreatedAt:     g.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     g.UpdatedAt.Format(time.RFC3339),
	}
	if g.MonthlyContribution != nil {
		resp.MonthlyContribution = g.MonthlyContribution.String()
	}
	if g.TargetDate != nil {
		resp.TargetDate = g.TargetDate.Format(time.RFC3339)
	}
	return resp
}

func toContributionResponse(c domain.GoalContribution) contributionResponse {
	return contributionResponse{
		ID:             c.ID,
		GoalID:         c.GoalID,
		ContributionID: c.ContributionID,
		Amount:         c.Amount.String(),
		Date:           c.ContributionDate.Format(time.RFC3339),
		Comment:        c.Comment,
		CreatedAt:      c.CreatedAt.Format(time.RFC3339),
	}
}

func toProjectionResponse(p goals.Projection) projectionResponse {
	resp := projectionResponse{
		Goal:             toGoalResponse(p.Goal),
		EffectiveCurrent: p.EffectiveCurrent.String(),
		TargetEffective:  p.TargetEffective.String(),
		Progress:         p.Progress.String(),
		MonthsLeft:       p.MonthsLeft,
		EstimatedMonths:  p.EstimatedMonths,
		Status:           string(p.Status),
		AsOf:             p.AsOfDate.Format(time.RFC3339),
	}
	if p.RequiredMonthly.IsPositive() || !p.RequiredMonthly.IsZero() {
		resp.RequiredMonthly = p.RequiredMonthly.String()
	}
	return resp
}

// writeServiceError maps a goals service error to the right HTTP status.
func (h *GoalsHandler) writeServiceError(w http.ResponseWriter, err error, op string) {
	switch {
	case errors.Is(err, goals.ErrNotFound):
		writeError(w, http.StatusNotFound, "goals.not_found", "goal not found")
	case errors.Is(err, goals.ErrInvalidArgument):
		writeError(w, http.StatusBadRequest, "goals.invalid_argument", err.Error())
	case errors.Is(err, storage.ErrContributionExists):
		// Handled in AddContribution; surface as 409 defensively.
		writeError(w, http.StatusConflict, "goals.contribution_exists", "contribution already exists")
	default:
		h.logger.Error("goals: "+op, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
	}
}
package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/RedEye472-afk/FinHelper/internal/domain"
	applog "github.com/RedEye472-afk/FinHelper/internal/log"
	"github.com/RedEye472-afk/FinHelper/internal/service/budget"
	"github.com/RedEye472-afk/FinHelper/internal/storage"
)

// BudgetHandler wires the budget REST endpoints (BUSINESS_LOGIC.md ф.4).
//
// Routes (mounted by the router under /api/v1, behind AuthMiddleware):
//
//	GET    /budgets              list budgets
//	POST   /budgets              create budget
//	GET    /budgets/{id}         get one
//	PATCH  /budgets/{id}         update limit/rollover/active
//	DELETE /budgets/{id}         soft-delete
//	GET    /budgets/{id}/status  compute current-period status (ф.4)
type BudgetHandler struct {
	svc    *budget.Service
	logger *slog.Logger
}

// NewBudgetHandler rejects nil deps at boot.
func NewBudgetHandler(svc *budget.Service, logger *slog.Logger) *BudgetHandler {
	if svc == nil || logger == nil {
		panic("http: NewBudgetHandler requires non-nil deps")
	}
	return &BudgetHandler{svc: svc, logger: logger}
}

// Register mounts the budget routes on the authenticated chi router.
func (h *BudgetHandler) Register(r chi.Router) {
	r.Get("/budgets", h.List)
	r.Post("/budgets", h.Create)
	r.Get("/budgets/{id}", h.Get)
	r.Patch("/budgets/{id}", h.Update)
	r.Delete("/budgets/{id}", h.Delete)
	r.Get("/budgets/{id}/status", h.Status)
}

// ---- Request / response shapes ----

type budgetResponse struct {
	ID             int64  `json:"id"`
	CategoryID     int64  `json:"category_id"`
	LimitAmount    string `json:"limit_amount"`
	RolloverPolicy string `json:"rollover_policy"`
	IsActive       bool   `json:"is_active"`
}

type createBudgetRequest struct {
	CategoryID     int64  `json:"category_id"`
	LimitAmount    string `json:"limit_amount"`
	RolloverPolicy string `json:"rollover_policy,omitempty"` // default "none"
}

type updateBudgetRequest struct {
	LimitAmount    string `json:"limit_amount"`
	RolloverPolicy string `json:"rollover_policy"`
	IsActive       bool   `json:"is_active"`
}

// ---- Handlers ----

func (h *BudgetHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	var req createBudgetRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "budget.invalid_body", err.Error())
		return
	}
	limit, err := domain.ParseMoney(req.LimitAmount)
	if err != nil || !limit.IsPositive() {
		writeError(w, http.StatusBadRequest, "budget.invalid_limit", "limit must be a positive amount")
		return
	}
	b, err := h.svc.Create(ctx, budget.CreateInput{
		UserID: userID, CategoryID: req.CategoryID,
		LimitAmount: limit, RolloverPolicy: domain.RolloverPolicy(req.RolloverPolicy),
	})
	if err != nil {
		h.writeServiceError(w, err, "create")
		return
	}
	applog.Info(ctx, h.logger, "budget created", "budget_id", b.ID)
	writeJSON(w, http.StatusCreated, toBudgetResponse(b))
}

func (h *BudgetHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	bs, err := h.svc.List(ctx, userID)
	if err != nil {
		h.writeServiceError(w, err, "list")
		return
	}
	out := make([]budgetResponse, 0, len(bs))
	for _, b := range bs {
		out = append(out, toBudgetResponse(b))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *BudgetHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "budget.invalid_id", "id must be a positive integer")
		return
	}
	b, err := h.svc.Get(ctx, userID, id)
	if err != nil {
		h.writeServiceError(w, err, "get")
		return
	}
	writeJSON(w, http.StatusOK, toBudgetResponse(b))
}

func (h *BudgetHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "budget.invalid_id", "id must be a positive integer")
		return
	}
	var req updateBudgetRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "budget.invalid_body", err.Error())
		return
	}
	limit, err := domain.ParseMoney(req.LimitAmount)
	if err != nil || !limit.IsPositive() {
		writeError(w, http.StatusBadRequest, "budget.invalid_limit", "limit must be a positive amount")
		return
	}
	b, err := h.svc.Update(ctx, budget.UpdateInput{
		UserID: userID, ID: id, LimitAmount: limit,
		RolloverPolicy: domain.RolloverPolicy(req.RolloverPolicy), IsActive: req.IsActive,
	})
	if err != nil {
		h.writeServiceError(w, err, "update")
		return
	}
	writeJSON(w, http.StatusOK, toBudgetResponse(b))
}

func (h *BudgetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "budget.invalid_id", "id must be a positive integer")
		return
	}
	if err := h.svc.Delete(ctx, userID, id); err != nil {
		h.writeServiceError(w, err, "delete")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type statusResponse struct {
	Budget        budgetResponse `json:"budget"`
	PeriodFrom    string         `json:"period_from"`
	PeriodTo      string         `json:"period_to"`
	Spent         string         `json:"spent"`
	Rollover      string         `json:"rollover"`
	EffectiveLimit string        `json:"effective_limit"`
	Remaining     string         `json:"remaining"`
	Projected     string         `json:"projected"`
	Status        string         `json:"status"`
	DaysInPeriod  int            `json:"days_in_period"`
	DaysElapsed   int            `json:"days_elapsed"`
}

// Status → GET /api/v1/budgets/{id}/status. Computes the budget's current-period
// health including rollover and the overspend forecast (BUSINESS_LOGIC ф.4).
func (h *BudgetHandler) Status(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "budget.invalid_id", "id must be a positive integer")
		return
	}
	comp, err := h.svc.Compute(ctx, userID, id)
	if err != nil {
		h.writeServiceError(w, err, "compute")
		return
	}
	writeJSON(w, http.StatusOK, statusResponse{
		Budget:         toBudgetResponse(comp.Budget),
		PeriodFrom:     comp.PeriodFrom.Format("2006-01-02"),
		PeriodTo:       comp.PeriodTo.Format("2006-01-02"),
		Spent:          comp.Spent.String(),
		Rollover:       comp.Rollover.String(),
		EffectiveLimit: comp.EffectiveLimit.String(),
		Remaining:      comp.Remaining.String(),
		Projected:      comp.Projected.String(),
		Status:         string(comp.Status),
		DaysInPeriod:   comp.DaysInPeriod,
		DaysElapsed:    comp.DaysElapsed,
	})
}

// ---- helpers ----

func toBudgetResponse(b storage.Budget) budgetResponse {
	return budgetResponse{
		ID: b.ID, CategoryID: b.CategoryID, LimitAmount: b.LimitAmount.String(),
		RolloverPolicy: string(b.RolloverPolicy), IsActive: b.IsActive,
	}
}

// writeServiceError maps a budget service error to the right HTTP status.
func (h *BudgetHandler) writeServiceError(w http.ResponseWriter, err error, op string) {
	switch {
	case errors.Is(err, budget.ErrNotFound):
		writeError(w, http.StatusNotFound, "budget.not_found", "budget not found")
	case errors.Is(err, budget.ErrInvalidArgument):
		writeError(w, http.StatusBadRequest, "budget.invalid_argument", err.Error())
	default:
		h.logger.Error("budget: "+op, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
	}
}

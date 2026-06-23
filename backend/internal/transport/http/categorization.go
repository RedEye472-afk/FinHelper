package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/RedEye472-afk/FinHelper/internal/domain"
	applog "github.com/RedEye472-afk/FinHelper/internal/log"
	"github.com/RedEye472-afk/FinHelper/internal/service/categorization"
	"github.com/RedEye472-afk/FinHelper/internal/storage"
)

// CategoriesHandler wires the category + categorization-rule REST endpoints
// (BUSINESS_LOGIC.md ф.2).
//
// Routes (mounted by the router under /api/v1, behind AuthMiddleware):
//
//	GET    /categories                  list user's categories
//	POST   /categories                  create user category
//	GET    /categorization/rules        list enabled keyword rules
//	POST   /categorization/rules        create keyword rule
//	DELETE /categorization/rules/{id}   soft-delete (disable) a rule
//	POST   /categorization/confirm      record a user confirmation (Learn path)
//
// The handler calls *storage.Pool directly with r.Context() rather than through
// an adapter interface — same pattern as OperationsHandler. service-layer
// isolation already lives in service/categorization (which takes a Repo
// interface); the handler is thin glue, so the indirection buys nothing here.
type CategoriesHandler struct {
	pool   *storage.Pool
	svc    *categorization.Service
	logger *slog.Logger
}

// NewCategoriesHandler rejects nil deps at boot.
func NewCategoriesHandler(pool *storage.Pool, svc *categorization.Service, logger *slog.Logger) *CategoriesHandler {
	if pool == nil || svc == nil || logger == nil {
		panic("http: NewCategoriesHandler requires non-nil deps")
	}
	return &CategoriesHandler{pool: pool, svc: svc, logger: logger}
}

// Register mounts the categorization routes on the authenticated chi router.
func (h *CategoriesHandler) Register(r chi.Router) {
	r.Get("/categories", h.ListCategories)
	r.Post("/categories", h.CreateCategory)
	r.Get("/categorization/rules", h.ListRules)
	r.Post("/categorization/rules", h.CreateRule)
	r.Delete("/categorization/rules/{id}", h.DeleteRule)
	r.Post("/categorization/confirm", h.Confirm)
}

// ---- Category responses ----

type categoryResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	ParentID *int64 `json:"parent_id,omitempty"`
	IsSystem bool   `json:"is_system"`
}

// ListCategories → GET /api/v1/categories.
func (h *CategoriesHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	cats, err := h.pool.ListCategories(ctx, userID)
	if err != nil {
		h.logger.Error("categories: list", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	out := make([]categoryResponse, 0, len(cats))
	for _, c := range cats {
		out = append(out, categoryResponse{ID: c.ID, Name: c.Name, ParentID: c.ParentID, IsSystem: c.IsSystem})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

type createCategoryRequest struct {
	Name     string `json:"name"`
	ParentID *int64 `json:"parent_id,omitempty"`
}

// CreateCategory → POST /api/v1/categories.
func (h *CategoriesHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	var req createCategoryRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "cat.invalid_body", err.Error())
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "cat.invalid_field", "name required")
		return
	}
	c, err := h.pool.CreateCategory(ctx, userID, req.Name, req.ParentID)
	if err != nil {
		if errors.Is(err, storage.ErrCategoryExists) {
			writeError(w, http.StatusConflict, "cat.exists", "category already exists")
			return
		}
		h.logger.Error("categories: create", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	applog.Info(ctx, h.logger, "category created", "cat_id", c.ID)
	writeJSON(w, http.StatusCreated, categoryResponse{ID: c.ID, Name: c.Name, ParentID: c.ParentID, IsSystem: false})
}

// ---- Rule responses ----

type ruleResponse struct {
	ID         int64  `json:"id"`
	Keyword    string `json:"keyword"`
	CategoryID int64  `json:"category_id"`
	Source     string `json:"source"`
	Priority   int    `json:"priority"`
}

// ListRules → GET /api/v1/categorization/rules.
func (h *CategoriesHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	rules, err := h.pool.ListRules(ctx, userID)
	if err != nil {
		h.logger.Error("rules: list", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	out := make([]ruleResponse, 0, len(rules))
	for _, r := range rules {
		out = append(out, ruleResponse{
			ID: r.ID, Keyword: r.Keyword, CategoryID: r.CategoryID,
			Source: string(r.Source), Priority: r.Priority,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

type createRuleRequest struct {
	Keyword    string `json:"keyword"`
	CategoryID int64  `json:"category_id"`
}

// CreateRule → POST /api/v1/categorization/rules.
func (h *CategoriesHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	var req createRuleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "rule.invalid_body", err.Error())
		return
	}
	if req.Keyword == "" {
		writeError(w, http.StatusBadRequest, "rule.invalid_field", "keyword required")
		return
	}
	if req.CategoryID <= 0 {
		writeError(w, http.StatusBadRequest, "rule.invalid_field", "category_id required")
		return
	}
	created, err := h.pool.CreateRule(ctx, storage.CategorizationRule{
		UserID: userID, Keyword: domain.NormalizeCounterparty(req.Keyword),
		CategoryID: req.CategoryID, Source: domain.RuleUser, IsEnabled: true,
	})
	if err != nil {
		if errors.Is(err, storage.ErrRuleExists) {
			writeError(w, http.StatusConflict, "rule.exists", "keyword rule already exists")
			return
		}
		h.logger.Error("rules: create", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	applog.Info(ctx, h.logger, "rule created", "rule_id", created.ID)
	writeJSON(w, http.StatusCreated, ruleResponse{
		ID: created.ID, Keyword: created.Keyword, CategoryID: created.CategoryID,
		Source: string(created.Source), Priority: created.Priority,
	})
}

// DeleteRule → DELETE /api/v1/categorization/rules/{id}.
func (h *CategoriesHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "rule.invalid_id", "id must be a positive integer")
		return
	}
	if err := h.pool.DeleteRule(ctx, userID, id); err != nil {
		if errors.Is(err, storage.ErrRuleNotFound) {
			writeError(w, http.StatusNotFound, "rule.not_found", "rule not found")
			return
		}
		h.logger.Error("rules: delete", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type confirmRequest struct {
	Counterparty string `json:"counterparty"`
	CategoryID   int64  `json:"category_id"`
}

type confirmResponse struct {
	Confirmations int  `json:"confirmations"`
	Learned       bool `json:"learned"`
}

// Confirm → POST /api/v1/categorization/confirm. Records that the user
// confirmed a category for a counterparty; bumps the confirmation counter and
// reports whether the override crossed the LearnThreshold (became authoritative).
func (h *CategoriesHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	var req confirmRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "confirm.invalid_body", err.Error())
		return
	}
	if req.Counterparty == "" || req.CategoryID <= 0 {
		writeError(w, http.StatusBadRequest, "confirm.invalid_field", "counterparty and category_id required")
		return
	}
	n, err := h.svc.Confirm(ctx, userID, categorization.LearnInput{
		Counterparty: req.Counterparty, CategoryID: req.CategoryID,
	}, h.pool)
	if err != nil {
		if errors.Is(err, categorization.ErrNotFound) {
			writeError(w, http.StatusNotFound, "confirm.not_found", err.Error())
			return
		}
		if errors.Is(err, categorization.ErrInvalidArgument) {
			writeError(w, http.StatusBadRequest, "confirm.invalid_argument", err.Error())
			return
		}
		h.logger.Error("confirm", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	applog.Info(ctx, h.logger, "category confirmed", "count", n)
	writeJSON(w, http.StatusOK, confirmResponse{
		Confirmations: n,
		Learned:       n >= domain.LearnThreshold,
	})
}

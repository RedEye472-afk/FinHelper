package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/RedEye472-afk/FinHelper/pkg/domain"
	applog "github.com/RedEye472-afk/FinHelper/pkg/log"
	"github.com/RedEye472-afk/FinHelper/pkg/storage"
)

// AccountsHandler wires the accounts REST endpoints.
//
// Routes (mounted by the router under /api/v1, behind AuthMiddleware):
//
//	GET    /accounts         list user accounts
//	POST   /accounts         create account
//	GET    /accounts/{id}    get one account
//	PATCH  /accounts/{id}    update name/type
//	DELETE /accounts/{id}    soft-delete
type AccountsHandler struct {
	pool   *storage.Pool
	logger *slog.Logger
}

// NewAccountsHandler rejects nil deps at boot.
func NewAccountsHandler(pool *storage.Pool, logger *slog.Logger) *AccountsHandler {
	if pool == nil || logger == nil {
		panic("http: NewAccountsHandler requires non-nil deps")
	}
	return &AccountsHandler{pool: pool, logger: logger}
}

// Register mounts the accounts routes on the authenticated chi router.
func (h *AccountsHandler) Register(r chi.Router) {
	r.Get("/accounts", h.List)
	r.Post("/accounts", h.Create)
	r.Get("/accounts/{id}", h.Get)
	r.Patch("/accounts/{id}", h.Update)
	r.Delete("/accounts/{id}", h.Delete)
}

// ---- Response shape ----

type accountResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"account_type"`
	Currency  string `json:"currency"`
	Balance   string `json:"balance"`
	CreatedAt string `json:"created_at"`
}

func toAccountResponse(a storage.Account) accountResponse {
	return accountResponse{
		ID:        a.ID,
		Name:      a.Name,
		Type:      string(a.Type),
		Currency:  a.Currency,
		Balance:   a.Balance.String(),
		CreatedAt: a.CreatedAt.Format(iso8601),
	}
}

// ---- Request shapes ----

type createAccountRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Currency string `json:"currency"`
}

type updateAccountRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ---- Handlers ----

// List → GET /api/v1/accounts.
func (h *AccountsHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}

	accounts, err := h.pool.ListAccounts(ctx, userID)
	if err != nil {
		h.logger.Error("accounts: list", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}

	out := make([]accountResponse, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, toAccountResponse(a))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// Create → POST /api/v1/accounts.
func (h *AccountsHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}

	var req createAccountRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "accounts.invalid_body", err.Error())
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "accounts.invalid_field", "name is required")
		return
	}
	accType, err := parseAccountType(req.Type)
	if err != nil {
		writeError(w, http.StatusBadRequest, "accounts.invalid_field", err.Error())
		return
	}
	if req.Currency == "" {
		writeError(w, http.StatusBadRequest, "accounts.invalid_field", "currency is required")
		return
	}

	a, err := h.pool.CreateAccount(ctx, userID, req.Name, accType, req.Currency)
	if err != nil {
		if errors.Is(err, storage.ErrAccountExists) {
			writeError(w, http.StatusConflict, "accounts.exists", "account already exists")
			return
		}
		h.logger.Error("accounts: create", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	applog.Info(ctx, h.logger, "account created", "account_id", a.ID)
	writeJSON(w, http.StatusCreated, toAccountResponse(a))
}

// Get → GET /api/v1/accounts/{id}.
func (h *AccountsHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "accounts.invalid_id", "id must be a positive integer")
		return
	}

	a, err := h.pool.GetAccount(ctx, userID, id)
	if err != nil {
		if errors.Is(err, storage.ErrAccountNotFound) {
			writeError(w, http.StatusNotFound, "accounts.not_found", "account not found")
			return
		}
		h.logger.Error("accounts: get", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	writeJSON(w, http.StatusOK, toAccountResponse(a))
}

// Update → PATCH /api/v1/accounts/{id}.
func (h *AccountsHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "accounts.invalid_id", "id must be a positive integer")
		return
	}

	var req updateAccountRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "accounts.invalid_body", err.Error())
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "accounts.invalid_field", "name is required")
		return
	}
	accType, err := parseAccountType(req.Type)
	if err != nil {
		writeError(w, http.StatusBadRequest, "accounts.invalid_field", err.Error())
		return
	}

	a, err := h.pool.UpdateAccount(ctx, userID, id, req.Name, accType)
	if err != nil {
		if errors.Is(err, storage.ErrAccountNotFound) {
			writeError(w, http.StatusNotFound, "accounts.not_found", "account not found")
			return
		}
		h.logger.Error("accounts: update", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	applog.Info(ctx, h.logger, "account updated", "account_id", a.ID)
	writeJSON(w, http.StatusOK, toAccountResponse(a))
}

// Delete → DELETE /api/v1/accounts/{id}.
func (h *AccountsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "accounts.invalid_id", "id must be a positive integer")
		return
	}

	if err := h.pool.DeleteAccount(ctx, userID, id); err != nil {
		if errors.Is(err, storage.ErrAccountNotFound) {
			writeError(w, http.StatusNotFound, "accounts.not_found", "account not found")
			return
		}
		h.logger.Error("accounts: delete", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	applog.Info(ctx, h.logger, "account deleted", "account_id", id)
	w.WriteHeader(http.StatusNoContent)
}

// ---- helpers ----

// knownAccountTypes maps string representations to domain.AccountType.
var knownAccountTypes = map[string]domain.AccountType{
	"cash":       domain.AccountCash,
	"bank":       domain.AccountBank,
	"savings":    domain.AccountSavings,
	"investment": domain.AccountInvestment,
	"crypto":     domain.AccountCrypto,
	"debt":       domain.AccountDebt,
}

func parseAccountType(s string) (domain.AccountType, error) {
	t, ok := knownAccountTypes[s]
	if !ok {
		return "", errors.New("invalid account type: must be one of cash, bank, savings, investment, crypto, debt")
	}
	return t, nil
}

// iso8601 is the time layout used for JSON timestamps in responses.
const iso8601 = "2006-01-02T15:04:05Z"

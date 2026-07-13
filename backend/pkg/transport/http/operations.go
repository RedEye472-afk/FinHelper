package http

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/backend/pkg/domain"
	applog "github.com/RedEye472-afk/FinHelper/backend/pkg/log"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/operations"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/storage"
)

// OperationsHandler wires the operations REST endpoints (BUSINESS_LOGIC.md ф.1).
//
// Routes (mounted by the router under /api/v1, behind AuthMiddleware):
//
//	POST   /operations            create
//	GET    /operations            list (paginated, filtered)
//	GET    /operations/{id}       get one
//	DELETE /operations/{id}       soft-delete
//	PATCH  /operations/{id}/category  change category assignment
type OperationsHandler struct {
	svc    *operations.Service
	logger *slog.Logger
}

// NewOperationsHandler rejects nil deps at boot.
func NewOperationsHandler(svc *operations.Service, logger *slog.Logger) *OperationsHandler {
	if svc == nil || logger == nil {
		panic("http: NewOperationsHandler requires non-nil deps")
	}
	return &OperationsHandler{svc: svc, logger: logger}
}

// Register mounts the operations routes on the given chi router. The router
// is expected to already be inside the authenticated /api/v1 group.
func (h *OperationsHandler) Register(r chi.Router) {
	r.Post("/operations", h.Create)
	r.Get("/operations", h.List)
	r.Get("/operations/{id}", h.Get)
	r.Delete("/operations/{id}", h.Delete)
	r.Patch("/operations/{id}/category", h.SetCategory)
}

// ---- Request / response shapes ----

type operationRequest struct {
	Type           string  `json:"type"`
	Amount         string  `json:"amount"`
	AmountDst      *string `json:"amount_dst,omitempty"`
	Currency       string  `json:"currency,omitempty"`
	AccountID      int64   `json:"account_id"`
	AccountDstID   *int64  `json:"account_dst_id,omitempty"`
	CategoryID     *int64  `json:"category_id,omitempty"`
	IncomeSubtype  *string `json:"income_subtype,omitempty"`
	Counterparty   string  `json:"counterparty,omitempty"`
	Description    string  `json:"description,omitempty"`
	OperationDate  string  `json:"operation_date,omitempty"` // RFC3339; empty = today
	CalcID         string  `json:"calc_id,omitempty"`
}

type operationResponse struct {
	ID               int64             `json:"id"`
	CalcID           string            `json:"calc_id"`
	Type             string            `json:"type"`
	Amount           string            `json:"amount"`
	AmountDst        *string           `json:"amount_dst,omitempty"`
	Currency         string            `json:"currency"`
	AccountID        int64             `json:"account_id"`
	AccountDstID     *int64            `json:"account_dst_id,omitempty"`
	CategoryID       *int64            `json:"category_id,omitempty"`
	IncomeSubtype    *string           `json:"income_subtype,omitempty"`
	Counterparty     string            `json:"counterparty,omitempty"`
	Description      string            `json:"description,omitempty"`
	OperationDate    string            `json:"operation_date"` // YYYY-MM-DD
	IsPlanned        bool              `json:"is_planned"`
	Confidence       *string           `json:"category_confidence,omitempty"`
	CreatedAt        string            `json:"created_at"`
}

type listResponse struct {
	Items []operationResponse `json:"items"`
	More  bool                `json:"more"`
}

// ---- Handlers ----

// Create → POST /api/v1/operations.
func (h *OperationsHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}

	var req operationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "ops.invalid_body", err.Error())
		return
	}

	in, err := h.parseCreate(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ops.invalid_field", err.Error())
		return
	}

	op, err := h.svc.Create(ctx, userID, in)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	applog.Info(ctx, h.logger, "operation created",
		"op_id", op.ID, "type", string(op.Type))
	writeJSON(w, http.StatusCreated, toResponse(op))
}

// List → GET /api/v1/operations?type=&from=&to=&account_id=&category_id=&planned=&limit=&before=.
func (h *OperationsHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}

	f, page, err := parseListQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ops.invalid_query", err.Error())
		return
	}

	items, more, err := h.svc.List(ctx, userID, f, page)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	resp := listResponse{Items: make([]operationResponse, 0, len(items)), More: more}
	for _, op := range items {
		resp.Items = append(resp.Items, toResponse(op))
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get → GET /api/v1/operations/{id}.
func (h *OperationsHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ops.invalid_id", "id must be a positive integer")
		return
	}
	op, err := h.svc.Get(ctx, userID, id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(op))
}

// Delete → DELETE /api/v1/operations/{id}.
func (h *OperationsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ops.invalid_id", "id must be a positive integer")
		return
	}
	if err := h.svc.Delete(ctx, userID, id); err != nil {
		h.writeServiceError(w, err)
		return
	}
	applog.Info(ctx, h.logger, "operation deleted", "op_id", id)
	w.WriteHeader(http.StatusNoContent)
}

type setCategoryRequest struct {
	CategoryID *int64  `json:"category_id"`
	Confidence *string `json:"confidence,omitempty"` // "0.92"; nil = manual override
}

// SetCategory → PATCH /api/v1/operations/{id}/category.
func (h *OperationsHandler) SetCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ops.invalid_id", "id must be a positive integer")
		return
	}
	var req setCategoryRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "ops.invalid_body", err.Error())
		return
	}
	var conf *float64
	if req.Confidence != nil {
		v, err := strconv.ParseFloat(*req.Confidence, 64)
		if err != nil || v < 0 || v > 1 {
			writeError(w, http.StatusBadRequest, "ops.invalid_confidence", "confidence must be in [0,1]")
			return
		}
		conf = &v
	}
	if err := h.svc.SetCategory(ctx, userID, id, req.CategoryID, toDecPtr(conf)); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// BulkDelete → DELETE /api/v1/operations/bulk?account_id=123&from=2024-01-01&to=2024-12-31.
// Deletes all operations for the given account (optionally filtered by date range).
func (h *OperationsHandler) BulkDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}

	q := r.URL.Query()
	accountIDStr := q.Get("account_id")
	if accountIDStr == "" {
		writeError(w, http.StatusBadRequest, "ops.missing_account_id", "account_id query parameter is required")
		return
	}
	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil || accountID <= 0 {
		writeError(w, http.StatusBadRequest, "ops.invalid_account_id", "account_id must be a positive integer")
		return
	}

	var fromPtr, toPtr *time.Time
	if v := q.Get("from"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "ops.invalid_from", "from must be YYYY-MM-DD")
			return
		}
		fromPtr = &t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "ops.invalid_to", "to must be YYYY-MM-DD")
			return
		}
		end := t.Add(24*time.Hour - time.Nanosecond)
		toPtr = &end
	}

	deleted, err := h.svc.BulkDeleteByAccount(ctx, userID, accountID, fromPtr, toPtr)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"deleted": deleted,
		"message": fmt.Sprintf("Удалено %d операций", deleted),
	})
}

// ---- Parsing helpers ----

// toDecPtr converts a *float64 confidence into a *decimal.Decimal. We accept
// float here because the value is bounded to [0,1] (validated) and represents
// a probability, not a monetary amount — no float precision concern.
func toDecPtr(f *float64) *decimal.Decimal {
	if f == nil {
		return nil
	}
	d := decimal.NewFromFloat(*f)
	return &d
}

var knownTypes = map[string]domain.OperationType{
	"income":            domain.OpIncome,
	"expense":           domain.OpExpense,
	"transfer":          domain.OpTransfer,
	"currency_exchange": domain.OpCurrencyExchange,
	"refund":            domain.OpRefund,
}

var knownIncomeSubtypes = map[string]domain.IncomeSubtype{
	"salary":         domain.IncomeSalary,
	"fee":            domain.IncomeFee,
	"gift":           domain.IncomeGift,
	"investment":     domain.IncomeInvestment,
	"loan_repayment": domain.IncomeLoanRepayment,
}

func (h *OperationsHandler) parseCreate(req operationRequest) (operations.CreateInput, error) {
	t, ok := knownTypes[req.Type]
	if !ok {
		return operations.CreateInput{}, errors.New("unknown operation type")
	}
	amount, err := domain.ParseMoney(req.Amount)
	if err != nil {
		return operations.CreateInput{}, errors.New("amount: " + err.Error())
	}
	if !amount.IsPositive() {
		return operations.CreateInput{}, errors.New("amount must be positive")
	}

	in := operations.CreateInput{
		Type:         t,
		Amount:       amount,
		Currency:     req.Currency,
		AccountID:    req.AccountID,
		AccountDstID: req.AccountDstID,
		CategoryID:   req.CategoryID,
		Counterparty: req.Counterparty,
		Description:  req.Description,
		CalcID:       req.CalcID,
	}

	if req.AmountDst != nil && *req.AmountDst != "" {
		d, err := domain.ParseMoney(*req.AmountDst)
		if err != nil {
			return operations.CreateInput{}, errors.New("amount_dst: " + err.Error())
		}
		if !d.IsPositive() {
			return operations.CreateInput{}, errors.New("amount_dst must be positive")
		}
		in.AmountDst = &d
	}
	if req.IncomeSubtype != nil && *req.IncomeSubtype != "" {
		s, ok := knownIncomeSubtypes[*req.IncomeSubtype]
		if !ok {
			return operations.CreateInput{}, errors.New("unknown income_subtype")
		}
		in.IncomeSubtype = &s
	}
	if req.OperationDate != "" {
		d, err := time.Parse(time.RFC3339, req.OperationDate)
		if err != nil {
			// Allow YYYY-MM-DD too.
			d, err = time.Parse("2006-01-02", req.OperationDate)
			if err != nil {
				return operations.CreateInput{}, errors.New("operation_date must be RFC3339 or YYYY-MM-DD")
			}
		}
		in.OperationDate = d
	}
	if in.AccountID <= 0 {
		return operations.CreateInput{}, errors.New("account_id must be positive")
	}
	return in, nil
}

func parseListQuery(r *http.Request) (storage.OperationFilter, storage.Page, error) {
	q := r.URL.Query()
	var f storage.OperationFilter

	if v := q.Get("from"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return f, storage.Page{}, errors.New("from must be YYYY-MM-DD")
		}
		f.From = &t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return f, storage.Page{}, errors.New("to must be YYYY-MM-DD")
		}
		// Inclusive end-of-day.
		end := t.Add(24*time.Hour - time.Nanosecond)
		f.To = &end
	}
	if v := q.Get("type"); v != "" {
		if t, ok := knownTypes[v]; ok {
			f.Types = []domain.OperationType{t}
		} else {
			return f, storage.Page{}, errors.New("unknown type filter")
		}
	}
	if v := q.Get("account_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			return f, storage.Page{}, errors.New("account_id must be a positive integer")
		}
		f.AccID = &id
	}
	if v := q.Get("category_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			return f, storage.Page{}, errors.New("category_id must be a positive integer")
		}
		f.CatID = &id
	}
	if v := q.Get("planned"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return f, storage.Page{}, errors.New("planned must be true/false")
		}
		f.Planned = &b
	}

	page := storage.Page{}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return f, storage.Page{}, errors.New("limit must be a positive integer")
		}
		page.Limit = n
	}
	if v := q.Get("before"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			return f, storage.Page{}, errors.New("before must be a positive integer")
		}
		page.BeforeID = &id
	}
	return f, page, nil
}

// ---- Response building ----

func toResponse(op storage.Operation) operationResponse {
	out := operationResponse{
		ID:            op.ID,
		CalcID:        op.CalcID,
		Type:          string(op.Type),
		Amount:        op.Amount.String(),
		Currency:      op.Currency,
		AccountID:     op.AccountID,
		AccountDstID:  op.AccountDstID,
		CategoryID:    op.CategoryID,
		Counterparty:  op.Counterparty,
		Description:   op.Description,
		OperationDate: op.OperationDate.Format("2006-01-02"),
		IsPlanned:     op.IsPlanned,
		CreatedAt:     op.CreatedAt.Format(time.RFC3339),
	}
	if op.AmountDst != nil {
		s := op.AmountDst.String()
		out.AmountDst = &s
	}
	if op.IncomeSubtype != nil {
		s := string(*op.IncomeSubtype)
		out.IncomeSubtype = &s
	}
	if op.CategoryConfidence != nil {
		s := op.CategoryConfidence.StringFixed(3)
		out.Confidence = &s
	}
	return out
}

// writeServiceError maps a service error to the right HTTP status. The body
// detail is safe — it comes from validation/NotFound sentinels, never from
// an internal storage error (those are logged, not surfaced).
func (h *OperationsHandler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, operations.ErrInvalidArgument):
		writeError(w, http.StatusBadRequest, "ops.invalid_argument", err.Error())
	case errors.Is(err, operations.ErrNotFound), errors.Is(err, operations.ErrAccountMissing):
		writeError(w, http.StatusNotFound, "ops.not_found", err.Error())
	default:
		h.logger.Error("operations: internal", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
	}
}

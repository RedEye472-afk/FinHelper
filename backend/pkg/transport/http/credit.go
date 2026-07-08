// Package http: credit.go wires the stateless credit calculator endpoint
// (BUSINESS_LOGIC.md ф.7 — кредитный калькулятор).
//
// Routes (mounted by the router under /api/v1, behind AuthMiddleware):
//
//	POST /calc/credit   stateless what-if; no storage read or persistence.
//
// The handler is intentionally thin: it parses JSON → service.Input, calls
// the credit.Service, and writes the Result. All math lives in
// internal/mathcore/credit via internal/service/credit. Money is decimal
// end-to-end; the only float64 surface is the documented PSK BrentQ bridge
// inside mathcore (see package doc of internal/mathcore/credit).
//
// Symmetric with /calc/goal (BUSINESS_LOGIC ф.5): a stateless what-if used by
// prospects before sign-up and by authenticated users exploring scenarios.
package http

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/credit"
)

// CreditHandler wires the stateless credit calculator (BUSINESS_LOGIC.md ф.7).
type CreditHandler struct {
	svc    *credit.Service
	logger *slog.Logger
}

// NewCreditHandler rejects nil deps at boot.
func NewCreditHandler(svc *credit.Service, logger *slog.Logger) *CreditHandler {
	if svc == nil || logger == nil {
		panic("http: NewCreditHandler requires non-nil deps")
	}
	return &CreditHandler{svc: svc, logger: logger}
}

// Register mounts the credit routes on the authenticated chi router.
func (h *CreditHandler) Register(r chi.Router) {
	r.Post("/calc/credit", h.Calculate)
}

// ---- Request / response shapes ----

// creditRequest is the body of POST /calc/credit. Money/rate fields are JSON
// strings parsed into decimal.Decimal (avoids float64 precision loss). Empty
// optional fields default to zero / annuity / today.
type creditRequest struct {
	Principal        string   `json:"principal"`
	AnnualRate       string   `json:"annual_rate,omitempty"`
	TermMonths       int      `json:"term_months"`
	PaymentType      string   `json:"payment_type,omitempty"`
	FirstPaymentDate string   `json:"first_payment_date,omitempty"` // RFC3339

	UpfrontFees    []string `json:"upfront_fees,omitempty"`
	MonthlyFee     string   `json:"monthly_fee,omitempty"`

	Early *creditEarlyRequest `json:"early,omitempty"`
}

type creditEarlyRequest struct {
	PaidMonths int    `json:"paid_months"`
	Amount     string `json:"amount"`
	Mode       string `json:"mode,omitempty"`
}

// ---- Handler ----

// Calculate → POST /api/v1/calc/credit. Stateless: no storage, no
// persistence, no user_id needed (kept for symmetry/logging only).
func (h *CreditHandler) Calculate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// /calc/credit lives behind AuthMiddleware (router mounts the whole
	// group behind it), so a user_id is present. The credit calculator is
	// stateless — it does not persist anything, so userID is unused.
	userID, _ := MustUserID(ctx)
	_ = userID

	var req creditRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "credit.invalid_body", err.Error())
		return
	}
	in, err := buildCreditInput(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "credit.invalid_argument", err.Error())
		return
	}
	res, err := h.svc.Calculate(ctx, in)
	if err != nil {
		h.writeServiceError(w, err, "calculate")
		return
	}
	writeJSON(w, http.StatusOK, toCreditResponse(res))
}

// ---- helpers ----

// buildCreditInput parses a creditRequest into credit.Input. Money fields
// use decimal.NewFromString (rates/amounts need not be Money-typed at the
// API edge — the service rounds where Money semantics apply).
func buildCreditInput(req creditRequest) (credit.Input, error) {
	if req.Principal == "" {
		return credit.Input{}, errors.New("credit: principal required")
	}
	principal, err := decimal.NewFromString(req.Principal)
	if err != nil {
		return credit.Input{}, errors.New("credit: principal must be a valid decimal")
	}
	rate := decimal.Zero
	if req.AnnualRate != "" {
		r, err := decimal.NewFromString(req.AnnualRate)
		if err != nil {
			return credit.Input{}, errors.New("credit: annual_rate must be a valid decimal")
		}
		rate = r
	}
	var firstDate time.Time
	if req.FirstPaymentDate != "" {
		t, err := time.Parse(time.RFC3339, req.FirstPaymentDate)
		if err != nil {
			return credit.Input{}, errors.New("credit: first_payment_date must be RFC3339")
		}
		firstDate = t
	}
	var upfrontFees []decimal.Decimal
	for i, fs := range req.UpfrontFees {
		f, err := decimal.NewFromString(fs)
		if err != nil {
			return credit.Input{}, errors.New("credit: upfront_fees[" + itoa(i) + "] must be a valid decimal")
		}
		upfrontFees = append(upfrontFees, f)
	}
	monthlyFee := decimal.Zero
	if req.MonthlyFee != "" {
		mf, err := decimal.NewFromString(req.MonthlyFee)
		if err != nil {
			return credit.Input{}, errors.New("credit: monthly_fee must be a valid decimal")
		}
		monthlyFee = mf
	}
	var early *credit.EarlyInput
	if req.Early != nil {
		if req.Early.Amount == "" {
			return credit.Input{}, errors.New("credit: early.amount required when early is set")
		}
		amt, err := decimal.NewFromString(req.Early.Amount)
		if err != nil {
			return credit.Input{}, errors.New("credit: early.amount must be a valid decimal")
		}
		early = &credit.EarlyInput{
			PaidMonths: req.Early.PaidMonths,
			Amount:     amt,
			Mode:       credit.EarlyMode(req.Early.Mode),
		}
	}
	return credit.Input{
		Principal:        principal,
		AnnualRate:       rate,
		TermMonths:       req.TermMonths,
		PaymentType:      credit.PaymentType(req.PaymentType),
		FirstPaymentDate: firstDate,
		UpfrontFees:      upfrontFees,
		MonthlyFee:       monthlyFee,
		Early:            early,
	}, nil
}

// itoa avoids pulling strconv just for one error-message index.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

// toCreditResponse projects the service Result into JSON. decimal.Decimal
// fields are emitted verbatim (shopspring serialises them as JSON numbers
// with full precision — no float64 loss).
func toCreditResponse(r credit.Result) map[string]any {
	out := map[string]any{
		"payment_type":    string(r.PaymentType),
		"monthly_payment": r.MonthlyPayment,
		"psk":             r.PSK,
		"overpayment":     r.Overpayment,
		"schedule":        r.Schedule,
		"disclaimer":      r.Disclaimer,
	}
	if r.Early != nil {
		out["early"] = r.Early
	}
	return out
}

// writeServiceError maps a credit service error to the right HTTP status.
// ErrInvalidArgument → 400; mathcore errors (bad term/principal/early
// amount) also draft as 400 since the caller sent bad input. Anything else
// is an internal error and gets logged.
func (h *CreditHandler) writeServiceError(w http.ResponseWriter, err error, op string) {
	switch {
	case errors.Is(err, credit.ErrInvalidArgument):
		writeError(w, http.StatusBadRequest, "credit.invalid_argument", err.Error())
	default:
		// Mathcore errors carry their own sentinel type; surface them as
		// 400 because they trace back to bad input values, not server state.
		h.logger.Error("credit: "+op, "error", err.Error())
		writeError(w, http.StatusBadRequest, "credit.invalid_argument", err.Error())
	}
}
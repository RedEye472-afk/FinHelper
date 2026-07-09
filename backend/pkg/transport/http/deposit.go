// Package http: deposit.go wires the stateless deposit calculator endpoint
// (BUSINESS_LOGIC.md ф.6 — калькулятор вкладов).
//
// Routes (mounted by the router under /api/v1, behind AuthMiddleware):
//
//	POST /calc/deposit   stateless what-if; no storage read or persistence.
//
// The handler is intentionally thin: it parses JSON → service.Input, calls
// the deposit.Service, and writes the Result. All math lives in
// internal/mathcore/deposit via internal/service/deposit. Money is decimal
// end-to-end.
//
// Symmetric with /calc/credit (BUSINESS_LOGIC ф.7) and /calc/goal (ф.5).
package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/deposit"
)

// DepositHandler wires the stateless deposit calculator (BUSINESS_LOGIC.md ф.6).
type DepositHandler struct {
	svc    *deposit.Service
	logger *slog.Logger
}

// NewDepositHandler rejects nil deps at boot.
func NewDepositHandler(svc *deposit.Service, logger *slog.Logger) *DepositHandler {
	if svc == nil || logger == nil {
		panic("http: NewDepositHandler requires non-nil deps")
	}
	return &DepositHandler{svc: svc, logger: logger}
}

// Register mounts the deposit routes on the authenticated chi router.
func (h *DepositHandler) Register(r chi.Router) {
	r.Post("/calc/deposit", h.Calculate)
}

// ---- Request / response shapes ----

// depositRequest is the body of POST /calc/deposit. Money/rate fields are JSON
// strings parsed into decimal.Decimal (avoids float64 precision loss). Empty
// optional fields default to zero / monthly capitalisation / today.
type depositRequest struct {
	Principal       string `json:"principal"`
	AnnualRate      string `json:"annual_rate,omitempty"`
	TermMonths      int    `json:"term_months"`
	Capitalization  string `json:"capitalization,omitempty"` // monthly|quarterly|annually|maturity
	InflationRate   string `json:"inflation_rate,omitempty"`
	TaxYear         int    `json:"tax_year,omitempty"` // 0 = skip tax calc
}

// ---- Handler ----

// Calculate → POST /api/v1/calc/deposit. Stateless: no storage, no
// persistence, no user_id needed (kept for symmetry/logging only).
func (h *DepositHandler) Calculate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// /calc/deposit lives behind AuthMiddleware, so a user_id is present.
	// The deposit calculator is stateless — it does not persist anything.
	userID, _ := MustUserID(ctx)
	_ = userID

	var req depositRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "deposit.invalid_body", err.Error())
		return
	}
	in, err := buildDepositInput(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "deposit.invalid_argument", err.Error())
		return
	}
	res, err := h.svc.Calculate(ctx, in)
	if err != nil {
		h.writeServiceError(w, err, "calculate")
		return
	}
	writeJSON(w, http.StatusOK, toDepositResponse(res))
}

// ---- helpers ----

// buildDepositInput parses a depositRequest into deposit.Input. Money fields
// use decimal.NewFromString.
func buildDepositInput(req depositRequest) (deposit.Input, error) {
	if req.Principal == "" {
		return deposit.Input{}, errors.New("deposit: principal required")
	}
	principal, err := decimal.NewFromString(req.Principal)
	if err != nil {
		return deposit.Input{}, errors.New("deposit: principal must be a valid decimal")
	}
	rate := decimal.Zero
	if req.AnnualRate != "" {
		r, err := decimal.NewFromString(req.AnnualRate)
		if err != nil {
			return deposit.Input{}, errors.New("deposit: annual_rate must be a valid decimal")
		}
		rate = r
	}
	capFreq := deposit.CapMonthly
	if req.Capitalization != "" {
		switch req.Capitalization {
		case "monthly":
			capFreq = deposit.CapMonthly
		case "quarterly":
			capFreq = deposit.CapQuarterly
		case "annually":
			capFreq = deposit.CapAnnually
		case "maturity":
			capFreq = deposit.CapMaturity
		default:
			return deposit.Input{}, errors.New("deposit: capitalization must be one of: monthly, quarterly, annually, maturity")
		}
	}
	inflation := decimal.Zero
	if req.InflationRate != "" {
		inf, err := decimal.NewFromString(req.InflationRate)
		if err != nil {
			return deposit.Input{}, errors.New("deposit: inflation_rate must be a valid decimal")
		}
		inflation = inf
	}
	return deposit.Input{
		Principal:      principal,
		AnnualRate:     rate,
		TermMonths:     req.TermMonths,
		Capitalization: capFreq,
		InflationRate:  inflation,
		TaxYear:        req.TaxYear,
	}, nil
}

// toDepositResponse projects the service Result into JSON.
func toDepositResponse(r deposit.Result) map[string]any {
	out := map[string]any{
		"maturity_amount": r.MaturityAmount,
		"total_interest":  r.TotalInterest,
		"effective_rate":  r.EffectiveRate,
		"disclaimer":      r.Disclaimer,
	}
	if !r.RealReturn.IsZero() {
		out["real_return"] = r.RealReturn
	}
	if !r.TaxAmount.IsZero() {
		out["tax_amount"] = r.TaxAmount
	}
	if len(r.Projection) > 0 {
		out["projection"] = r.Projection
	}
	return out
}

// writeServiceError maps a deposit service error to the right HTTP status.
func (h *DepositHandler) writeServiceError(w http.ResponseWriter, err error, op string) {
	switch {
	case errors.Is(err, deposit.ErrInvalidArgument):
		writeError(w, http.StatusBadRequest, "deposit.invalid_argument", err.Error())
	default:
		h.logger.Error("deposit: "+op, "error", err.Error())
		writeError(w, http.StatusBadRequest, "deposit.invalid_argument", err.Error())
	}
}

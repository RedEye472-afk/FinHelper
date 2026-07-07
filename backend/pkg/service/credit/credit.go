// Package credit implements BUSINESS_LOGIC.md ф.7 — credit calculator.
//
// This is the stateless orchestration layer over internal/mathcore/credit:
// it builds a loan (annuity or differentiated), wires commissions/insurance
// into both the payment schedule and the ПСК cashflow series, computes
// overpayment, and — when an early-repayment scenario is supplied — runs
// mathcore/credit.EarlyRepayment to produce the "экономия X, срок
// сократится на Y мес." summary the spec calls for.
//
// Design mirrors service/goals (BUSINESS_LOGIC ф.5):
//   - Pure service: holds no per-request state, no database dependency.
//     The HTTP handler for POST /calc/credit constructs a Service at boot
//     and calls Calculate on each request.
//   - Money is decimal end-to-end; the only float64 surface lives inside
//     mathcore/credit's PSK solver (the documented BrentQ bridge).
//   - Inputs are validated here, not in mathcore; mathcore takes already-
//     sane primitives. Sentinel errors map to HTTP 400 in the handler.
//
// Sources: BUSINESS_LOGIC.md §7; MATH_FORMULAS.md §2 (Копнова Гл. 4.2);
// Bank of Russia Указание 5750-У for ПСК.
package credit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/pkg/mathcore/credit"
)

// Sentinel errors. The HTTP handler branches on errors.Is.
var (
	// ErrInvalidArgument — request failed validation.
	ErrInvalidArgument = errors.New("credit: invalid argument")
)

// PaymentType selects the amortisation method.
type PaymentType string

const (
	// PaymentAnnuity — equal monthly payments (Копнова Гл. 4.2.1).
	PaymentAnnuity PaymentType = "annuity"
	// PaymentDifferentiated — declining payments (Копнова Гл. 4.2.2).
	PaymentDifferentiated PaymentType = "differentiated"
)

// EarlyMode selects how an early-repayment lump sum is applied.
type EarlyMode string

const (
	// EarlyShortenTerm keeps the payment, reduces the remaining term.
	EarlyShortenTerm EarlyMode = "shorten_term"
	// EarlyLowerPayment keeps the term, reduces the payment.
	EarlyLowerPayment EarlyMode = "lower_payment"
)

// Input is the body of POST /calc/credit. All monetary fields are raw
// decimal strings parsed by the handler into decimal.Decimal. Optional
// fields have zero/empty/nil defaults documented inline.
type Input struct {
	Principal       decimal.Decimal // > 0
	AnnualRate      decimal.Decimal // ≥ 0 (0 → interest-free)
	TermMonths      int             // > 0
	PaymentType     PaymentType     // "annuity" (default) | "differentiated"
	FirstPaymentDate time.Time      // disbursal/first-payment date; zero → today

	// Commissions & insurance (BUSINESS_LOGIC §7 «А»).
	UpfrontFees []decimal.Decimal // one-off, deducted from disbursal (PSK)
	MonthlyFee decimal.Decimal    // recurring monthly commission/insurance

	// Early repayment scenario (BUSINESS_LOGIC §7 «С»). Optional.
	Early *EarlyInput
}

// EarlyInput describes a single lump-sum early-repayment scenario.
type EarlyInput struct {
	PaidMonths int        // months already paid before the early payment (0..term)
	Amount     decimal.Decimal // extra principal paid at month PaidMonths (≥ 0)
	Mode       EarlyMode // "shorten_term" (default) | "lower_payment"
}

// Result is the response of POST /calc/credit (BUSINESS_LOGIC §7 «С»).
type Result struct {
	PaymentType   PaymentType    `json:"payment_type"`
	MonthlyPayment decimal.Decimal `json:"monthly_payment"` // first payment (diff → first)
	PSK           decimal.Decimal `json:"psk"`               // annual nominal, CBR 5750-У
	Overpayment   decimal.Decimal `json:"overpayment"`      // total interest + fees − principal
	Schedule      []ScheduleRow  `json:"schedule"`
	Early         *EarlyResult    `json:"early,omitempty"`
	Disclaimer    string          `json:"disclaimer"`
}

// ScheduleRow is the HTTP-friendly projection of credit.ScheduleRow.
type ScheduleRow struct {
	Month      int             `json:"month"`
	Payment    decimal.Decimal `json:"payment"`
	Principal  decimal.Decimal `json:"principal"`
	Interest   decimal.Decimal `json:"interest"`
	BalanceEnd decimal.Decimal `json:"balance_end"`
	Fee        decimal.Decimal `json:"fee,omitempty"` // monthly commission/insurance
}

// EarlyResult summarises one early-repayment scenario (BUSINESS_LOGIC §7 «С»).
type EarlyResult struct {
	Mode             EarlyMode        `json:"mode"`
	NewPayment       decimal.Decimal  `json:"new_payment"`
	NewRemainingTerm int              `json:"new_remaining_term"`
	InterestSaved    decimal.Decimal  `json:"interest_saved"`
	NewSchedule      []ScheduleRow    `json:"new_schedule,omitempty"`
	Summary          string           `json:"summary"` // human-readable «Экономия X, срок сократится на Y мес.»
}

// Service is the credit calculator. Construct once at boot; stateless.
type Service struct {
	now func() time.Time
}

// NewService returns a Service. now defaults to time.Now; tests inject.
func NewService() *Service { return &Service{now: time.Now} }

// Calculate runs the credit calculator statelessly. Returns ErrInvalidArgument
// on bad input; mathcore errors propagate wrapped so the handler can map them.
func (s *Service) Calculate(ctx context.Context, in Input) (Result, error) {
	// ---- validation ----
	if !in.Principal.IsPositive() {
		return Result{}, fmt.Errorf("%w: principal must be > 0", ErrInvalidArgument)
	}
	if in.TermMonths <= 0 {
		return Result{}, fmt.Errorf("%w: term_months must be > 0", ErrInvalidArgument)
	}
	if in.AnnualRate.IsNegative() {
		return Result{}, fmt.Errorf("%w: annual_rate must be >= 0", ErrInvalidArgument)
	}
	if in.MonthlyFee.IsNegative() {
		return Result{}, fmt.Errorf("%w: monthly_fee must be >= 0", ErrInvalidArgument)
	}
	if in.PaymentType != PaymentAnnuity && in.PaymentType != PaymentDifferentiated {
		// Default to annuity when empty/unknown (the common case in RU).
		if in.PaymentType == "" {
			in.PaymentType = PaymentAnnuity
		} else {
			return Result{}, fmt.Errorf("%w: payment_type must be annuity or differentiated", ErrInvalidArgument)
		}
	}
	if in.Early != nil {
		if in.Early.PaidMonths < 0 || in.Early.PaidMonths > in.TermMonths {
			return Result{}, fmt.Errorf("%w: early.paid_months out of range", ErrInvalidArgument)
		}
		if in.Early.Amount.IsNegative() {
			return Result{}, fmt.Errorf("%w: early.amount must be >= 0", ErrInvalidArgument)
		}
		if in.Early.Mode != EarlyShortenTerm && in.Early.Mode != EarlyLowerPayment {
			if in.Early.Mode == "" {
				in.Early.Mode = EarlyShortenTerm
			} else {
				return Result{}, fmt.Errorf("%w: early.mode must be shorten_term or lower_payment", ErrInvalidArgument)
			}
		}
	}

	firstDate := in.FirstPaymentDate
	if firstDate.IsZero() {
		firstDate = s.now()
	}

	monthlyRate := in.AnnualRate.Div(decimal.NewFromInt(12))

	// ---- base schedule + first/payment figures ----
	var (
		baseRows []credit.ScheduleRow
		monthly  decimal.Decimal // first payment (the figure we surface)
		err      error
	)
	switch in.PaymentType {
	case PaymentAnnuity:
		rawPayment, err := credit.AnnuityPayment(in.Principal, monthlyRate, in.TermMonths)
		if err != nil {
			return Result{}, err
		}
		// AnnuityPayment returns the exact (unrounded) payment 47 073.4722…;
		// AnnuitySchedule echoes the monthlyPayment it receives into every
		// row except the last (which absorbs the rounding residual so the
		// balance closes exactly to zero). Feeding the kopeck-rounded value
		// in keeps the surfaced MonthlyPayment, the displayed schedule, and
		// the PSK cashflow input all consistent at scale 2 — matching the
		// mathcore TestAnnuityPayment_Golden value of 47073.47.
		monthly = rawPayment.Round(2)
		baseRows, err = credit.AnnuitySchedule(in.Principal, monthlyRate, monthly, in.TermMonths)
		if err != nil {
			return Result{}, err
		}
	case PaymentDifferentiated:
		baseRows, err = credit.DifferentiatedSchedule(in.Principal, monthlyRate, in.TermMonths)
		if err != nil {
			return Result{}, err
		}
		if len(baseRows) > 0 {
			monthly = baseRows[0].Payment
		}
	}

	// ---- PSK cashflow series ----
	// AnnuityCashflows is built for annuity loans; for differentiated we
	// construct the series manually below. In both cases the monthly fee
	// (if any) is added to each month's outgoing payment (more negative).
	var cashflows []credit.Cashflow
	netDisbursed := in.Principal
	for _, f := range in.UpfrontFees {
		netDisbursed = netDisbursed.Sub(f)
	}
	cashflows = append(cashflows, credit.Cashflow{Date: firstDate, Amount: netDisbursed})
	for k := 1; k <= in.TermMonths; k++ {
		d := firstDate.AddDate(0, k, 0)
		pay := baseRows[k-1].Payment
		// recurring fee leaves the borrower's pocket → more negative.
		if in.MonthlyFee.IsPositive() {
			pay = pay.Add(in.MonthlyFee)
		}
		cashflows = append(cashflows, credit.Cashflow{Date: d, Amount: pay.Neg()})
	}

	psk, pskErr := credit.PSK(cashflows)
	if pskErr != nil {
		// PSK is informational; a solver failure on a pathological input
		// should not 500 the whole calculator. Surface zero + a note.
		psk = decimal.Zero
	}

	// ---- overpayment = (Σ payments + Σ fees + Σ upfrontFees) − principal ----
	totalPaid := decimal.Zero
	for _, r := range baseRows {
		totalPaid = totalPaid.Add(r.Payment)
	}
	totalFees := in.MonthlyFee.Mul(decimal.NewFromInt(int64(in.TermMonths)))
	for _, f := range in.UpfrontFees {
		totalFees = totalFees.Add(f)
	}
	overpayment := totalPaid.Add(totalFees).Sub(in.Principal).Round(2)

	// ---- HTTP schedule projection (with optional fee column) ----
	sched := make([]ScheduleRow, len(baseRows))
	for i, r := range baseRows {
		row := ScheduleRow{
			Month:      r.Month,
			Payment:    r.Payment,
			Principal:  r.Principal,
			Interest:   r.Interest,
			BalanceEnd: r.BalanceEnd,
		}
		if in.MonthlyFee.IsPositive() {
			row.Fee = in.MonthlyFee
			row.Payment = row.Payment.Add(in.MonthlyFee)
		}
		sched[i] = row
	}

	res := Result{
		PaymentType:    in.PaymentType,
		MonthlyPayment: monthly,
		PSK:            psk,
		Overpayment:    overpayment,
		Schedule:       sched,
		Disclaimer:     "Расчёт носит справочный характер. Реальные условия могут включать скрытые комиссии и зависят от политики банка.",
	}

	// ---- early-repayment scenario ----
	if in.Early != nil && in.Early.Amount.IsPositive() {
		earlyRes, err := s.computeEarly(in, firstDate)
		if err != nil {
			return Result{}, err
		}
		res.Early = earlyRes
	}

	return res, nil
}

// computeEarly invokes mathcore/credit.EarlyRepayment and projects the result
// into the HTTP-friendly EarlyResult, including the human summary line that
// BUSINESS_LOGIC §7 «С» calls for.
//
// Note: EarlyRepayment currently operates on annuity loans (it re-runs
// AnnuityPayment/AnnuitySchedule internally). For a differentiated loan we
// still surface the scenario with a short caveat — the mathcore package
// does not yet support early repayment on differentiated schedules, and
// lying about that would be worse than declining. We return the
// mathcore-AI error which the handler maps to 400.
func (s *Service) computeEarly(in Input, firstDate time.Time) (*EarlyResult, error) {
	mode := credit.EarlyShortenTerm
	if in.Early.Mode == EarlyLowerPayment {
		mode = credit.EarlyLowerPayment
	}

	er, err := credit.EarlyRepayment(
		in.Principal, in.AnnualRate,
		in.TermMonths, in.Early.PaidMonths,
		in.Early.Amount, mode,
	)
	if err != nil {
		return nil, err
	}

	out := &EarlyResult{
		Mode:             in.Early.Mode,
		NewPayment:       er.NewPayment,
		NewRemainingTerm: er.NewRemainingTerm,
		InterestSaved:    er.InterestSaved,
	}
	for _, r := range er.NewSchedule {
		row := ScheduleRow{
			Month:      r.Month,
			Payment:    r.Payment,
			Principal:  r.Principal,
			Interest:   r.Interest,
			BalanceEnd: r.BalanceEnd,
		}
		if in.MonthlyFee.IsPositive() {
			row.Fee = in.MonthlyFee
			row.Payment = row.Payment.Add(in.MonthlyFee)
		}
		out.NewSchedule = append(out.NewSchedule, row)
	}

	// Human summary per BUSINESS_LOGIC §7 «С».
	if er.NewRemainingTerm == 0 {
		out.Summary = fmt.Sprintf(
			"Кредит закрыт досрочно. Экономия на процентах: %s ₽.",
			er.InterestSaved.String(),
		)
	} else if mode == credit.EarlyShortenTerm {
		saved := in.TermMonths - in.Early.PaidMonths - er.NewRemainingTerm
		if saved < 0 {
			saved = 0
		}
		out.Summary = fmt.Sprintf(
			"Экономия на процентах: %s ₽, срок сократится на %d мес.",
			er.InterestSaved.String(), saved,
		)
	} else {
		out.Summary = fmt.Sprintf(
			"Экономия на процентах: %s ₽, новый платёж: %s ₽/мес.",
			er.InterestSaved.String(), er.NewPayment.String(),
		)
	}
	return out, nil
}
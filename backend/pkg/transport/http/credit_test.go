// Package http (test): credit_test.go integration-tests the stateless credit
// calculator HTTP handler via httptest.Server + chi router. Mirrors the
// patterns established by goals_test.go: a real CreditHandler is wired
// behind a fake auth middleware that injects a fixed user_id (15) into
// context for every request. No database is touched.
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/RedEye472-afk/FinHelper/pkg/service/credit"
)

// ---- test env ----

// creditTestEnv wires a real CreditHandler behind a chi router + a fake auth
// middleware that injects a fixed user_id (15) into context for every
// request. httptest.NewServer gives us a real base URL.
type creditTestEnv struct {
	t   *testing.T
	srv *httptest.Server
}

func newCreditTestEnv(t *testing.T) *creditTestEnv {
	t.Helper()
	svc := credit.NewService()
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	h := NewCreditHandler(svc, logger)

	r := chi.NewRouter()
	// Fake auth: inject a fixed user_id into context for every request.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), keyUserID, int64(15))
			ctx = context.WithValue(ctx, keyUserHash, "uhsh-credit")
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h.Register(r)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return &creditTestEnv{t: t, srv: srv}
}

// creditReq performs an HTTP request with a JSON-encoded body (nil → no body)
// and returns the response + raw body bytes. Auth is implicit (test mw).
func creditReq(t *testing.T, method, url string, body any) (*http.Response, []byte) {
	t.Helper()
	var req *http.Request
	if body != nil {
		buf, _ := json.Marshal(body)
		req, _ = http.NewRequest(method, url, bytes.NewReader(buf))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, url, nil)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp, raw
}

// creditReqRaw is creditReq when the body should NOT be json.Marshal'd (e.g.
// to exercise decodeJSON's error path with malformed JSON).
func creditReqRaw(t *testing.T, method, url string, body []byte) (*http.Response, []byte) {
	t.Helper()
	var req *http.Request
	if body != nil {
		req, _ = http.NewRequest(method, url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp, raw
}

// ---- response shape structs ----

type creditResp struct {
	PaymentType    string                 `json:"payment_type"`
	MonthlyPayment string                 `json:"monthly_payment"`
	PSK            string                 `json:"psk"`
	Overpayment     string                 `json:"overpayment"`
	Schedule       []creditScheduleRow     `json:"schedule"`
	Disclaimer     string                 `json:"disclaimer"`
	Early          *creditEarlyResp        `json:"early,omitempty"`
}

type creditScheduleRow struct {
	Month      int    `json:"month"`
	Payment    string `json:"payment"`
	Principal  string `json:"principal"`
	Interest   string `json:"interest"`
	BalanceEnd string `json:"balance_end"`
	Fee        string `json:"fee,omitempty"`
}

type creditEarlyResp struct {
	Mode             string               `json:"mode"`
	NewPayment       string               `json:"new_payment"`
	NewRemainingTerm int                  `json:"new_remaining_term"`
	InterestSaved    string               `json:"interest_saved"`
	Summary          string               `json:"summary"`
}

// ---- tests ----

func TestCreditHandler_Calculate_Annuity_Golden(t *testing.T) {
	env := newCreditTestEnv(t)
	resp, raw := creditReq(t, http.MethodPost, env.srv.URL+"/calc/credit", map[string]any{
		"principal":    "1000000",
		"annual_rate":  "0.12",
		"term_months":  24,
		"payment_type": "annuity",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var cr creditResp
	if err := json.Unmarshal(raw, &cr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cr.MonthlyPayment != "47073.47" {
		t.Errorf("monthly_payment = %q, want 47073.47", cr.MonthlyPayment)
	}
	if len(cr.Schedule) != 24 {
		t.Errorf("schedule len = %d, want 24", len(cr.Schedule))
	}
	if cr.PaymentType != "annuity" {
		t.Errorf("payment_type = %q, want annuity", cr.PaymentType)
	}
	if cr.Disclaimer == "" {
		t.Error("disclaimer must not be empty (spec §«C»)")
	}
}

func TestCreditHandler_Calculate_Differentiated(t *testing.T) {
	env := newCreditTestEnv(t)
	resp, raw := creditReq(t, http.MethodPost, env.srv.URL+"/calc/credit", map[string]any{
		"principal":   "500000",
		"annual_rate": "0.12",
		"term_months": 12,
		"payment_type": "differentiated",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var cr creditResp
	if err := json.Unmarshal(raw, &cr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cr.PaymentType != "differentiated" {
		t.Errorf("payment_type = %q, want differentiated", cr.PaymentType)
	}
	first := cr.Schedule[0].Payment
	last := cr.Schedule[len(cr.Schedule)-1].Payment
	if first <= last {
		t.Errorf("differentiated should decline: first=%s last=%s", first, last)
	}
}

func TestCreditHandler_Calculate_DefaultPaymentTypeIsAnnuity(t *testing.T) {
	env := newCreditTestEnv(t)
	// omit payment_type entirely
	resp, raw := creditReq(t, http.MethodPost, env.srv.URL+"/calc/credit", map[string]any{
		"principal":   "1000000",
		"annual_rate": "0.12",
		"term_months": 12,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var cr creditResp
	if err := json.Unmarshal(raw, &cr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cr.PaymentType != "annuity" {
		t.Errorf("default payment_type = %q, want annuity", cr.PaymentType)
	}
}

func TestCreditHandler_Calculate_ZeroRate(t *testing.T) {
	env := newCreditTestEnv(t)
	resp, raw := creditReq(t, http.MethodPost, env.srv.URL+"/calc/credit", map[string]any{
		"principal":   "120000",
		"annual_rate": "0",
		"term_months": 12,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var cr creditResp
	if err := json.Unmarshal(raw, &cr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cr.MonthlyPayment != "10000" {
		t.Errorf("zero-rate monthly_payment = %q, want 10000", cr.MonthlyPayment)
	}
}

func TestCreditHandler_Calculate_WithFees(t *testing.T) {
	env := newCreditTestEnv(t)
	resp, raw := creditReq(t, http.MethodPost, env.srv.URL+"/calc/credit", map[string]any{
		"principal":     "1000000",
		"annual_rate":   "0.12",
		"term_months":   24,
		"upfront_fees":  []string{"50000"},
		"monthly_fee":   "1000",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var cr creditResp
	if err := json.Unmarshal(raw, &cr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Fees push ПСК above 12% (the nominal rate).
	if cr.PSK == "" || cr.PSK == "0" {
		t.Errorf("PSK should be non-zero with fees: %s", cr.PSK)
	}
	// First payment should include the 1000 fee on top of the 47073.47 annuity.
	if cr.Schedule[0].Fee != "1000" {
		t.Errorf("schedule[0].fee = %q, want 1000", cr.Schedule[0].Fee)
	}
}

func TestCreditHandler_Calculate_EarlyShortenTerm(t *testing.T) {
	env := newCreditTestEnv(t)
	resp, raw := creditReq(t, http.MethodPost, env.srv.URL+"/calc/credit", map[string]any{
		"principal":   "1000000",
		"annual_rate": "0.12",
		"term_months": 24,
		"early": map[string]any{
			"paid_months": 12,
			"amount":      "200000",
			"mode":        "shorten_term",
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var cr creditResp
	if err := json.Unmarshal(raw, &cr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cr.Early == nil {
		t.Fatal("expected early in response")
	}
	if cr.Early.InterestSaved == "" || cr.Early.InterestSaved == "0" {
		t.Errorf("interest_saved = %q, must be > 0", cr.Early.InterestSaved)
	}
	if cr.Early.Summary == "" {
		t.Error("early.summary must not be empty")
	}
}

func TestCreditHandler_Calculate_EarlyLowerPayment(t *testing.T) {
	env := newCreditTestEnv(t)
	resp, raw := creditReq(t, http.MethodPost, env.srv.URL+"/calc/credit", map[string]any{
		"principal":   "1000000",
		"annual_rate": "0.12",
		"term_months": 24,
		"early": map[string]any{
			"paid_months": 12,
			"amount":      "200000",
			"mode":        "lower_payment",
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var cr creditResp
	if err := json.Unmarshal(raw, &cr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cr.Early == nil {
		t.Fatal("expected early in response")
	}
	if cr.Early.NewRemainingTerm != 12 {
		t.Errorf("new_remaining_term = %d, want 12 (lower_payment keeps term)", cr.Early.NewRemainingTerm)
	}
}

func TestCreditHandler_Calculate_MalformedJSON(t *testing.T) {
	env := newCreditTestEnv(t)
	resp, _ := creditReqRaw(t, http.MethodPost, env.srv.URL+"/calc/credit", []byte(`{"bad"`))
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestCreditHandler_Calculate_MissingPrincipal(t *testing.T) {
	env := newCreditTestEnv(t)
	resp, _ := creditReq(t, http.MethodPost, env.srv.URL+"/calc/credit", map[string]any{
		"annual_rate": "0.12",
		"term_months": 12,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestCreditHandler_Calculate_InvalidPrincipal(t *testing.T) {
	env := newCreditTestEnv(t)
	resp, _ := creditReq(t, http.MethodPost, env.srv.URL+"/calc/credit", map[string]any{
		"principal":   "not-a-number",
		"annual_rate": "0.12",
		"term_months": 12,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestCreditHandler_Calculate_InvalidTerm(t *testing.T) {
	env := newCreditTestEnv(t)
	resp, _ := creditReq(t, http.MethodPost, env.srv.URL+"/calc/credit", map[string]any{
		"principal":   "1000",
		"annual_rate": "0.12",
		"term_months": 0,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestCreditHandler_Calculate_BadPaymentType(t *testing.T) {
	env := newCreditTestEnv(t)
	resp, _ := creditReq(t, http.MethodPost, env.srv.URL+"/calc/credit", map[string]any{
		"principal":    "1000",
		"annual_rate":  "0.12",
		"term_months":  12,
		"payment_type": "interest-only",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestCreditHandler_Calculate_BadEarlyMode(t *testing.T) {
	env := newCreditTestEnv(t)
	resp, _ := creditReq(t, http.MethodPost, env.srv.URL+"/calc/credit", map[string]any{
		"principal":   "1000",
		"annual_rate": "0.12",
		"term_months": 12,
		"early": map[string]any{
			"paid_months": 6,
			"amount":      "100",
			"mode":        "balloon",
		},
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestCreditHandler_Calculate_EarlyAmountExceedsBalance(t *testing.T) {
	env := newCreditTestEnv(t)
	// After 12 months on a 1M @ 12%/24m loan the balance is ~521k. Throw 10M
	// at it — mathcore returns ErrEarlyExceedsBalance, which the handler
	// maps to a 400.
	resp, _ := creditReq(t, http.MethodPost, env.srv.URL+"/calc/credit", map[string]any{
		"principal":   "1000000",
		"annual_rate": "0.12",
		"term_months": 24,
		"early": map[string]any{
			"paid_months": 12,
			"amount":      "10000000",
			"mode":        "shorten_term",
		},
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (early amount exceeds balance)", resp.StatusCode)
	}
}

func TestCreditHandler_Calculate_DisclaimerPresent(t *testing.T) {
	env := newCreditTestEnv(t)
	resp, raw := creditReq(t, http.MethodPost, env.srv.URL+"/calc/credit", map[string]any{
		"principal":   "100000",
		"annual_rate": "0.12",
		"term_months": 6,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var cr creditResp
	if err := json.Unmarshal(raw, &cr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cr.Disclaimer == "" {
		t.Error("disclaimer must be present per spec §«C»")
	}
}
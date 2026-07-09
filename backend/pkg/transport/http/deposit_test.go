// Package http (test): deposit_test.go unit-tests the deposit calculator HTTP
// handler. Tests follow the same pattern as credit_test.go.
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/deposit"
)

func mustDepositSvc() *deposit.Service {
	return deposit.NewService()
}

// instrumentDeposit creates a test-only chi router with the deposit handler
// mounted and an auth middleware that injects userID 1 (matches other test helpers).
func instrumentDeposit(svc *deposit.Service) *chi.Mux {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), keyUserID, int64(1))
			ctx = context.WithValue(ctx, keyUserHash, "uhsh-deposit-test")
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	h := NewDepositHandler(svc, logger)
	h.Register(r)
	return r
}

func TestDepositHTTP_Compound_Golden(t *testing.T) {
	svc := mustDepositSvc()
	r := instrumentDeposit(svc)

	body := `{"principal":"100000","annual_rate":"0.10","term_months":12,"capitalization":"monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/calc/deposit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var res map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// With monthly cap: 100000 * (1+0.10/12)^12 ≈ 110471.31
	maturity := res["maturity_amount"].(string)
	if maturity != "110471.31" {
		t.Errorf("maturity_amount = %s, want 110471.31", maturity)
	}
	if res["disclaimer"] == "" {
		t.Error("disclaimer must not be empty")
	}
}

func TestDepositHTTP_Simple(t *testing.T) {
	svc := mustDepositSvc()
	r := instrumentDeposit(svc)

	body := `{"principal":"100000","annual_rate":"0.10","term_months":6,"capitalization":"maturity"}`
	req := httptest.NewRequest(http.MethodPost, "/calc/deposit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var res map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	maturity := res["maturity_amount"].(string)
	if maturity != "105000" {
		t.Errorf("maturity_amount = %s, want 105000 (100000 * (1 + 0.10*6/12))", maturity)
	}
}

func TestDepositHTTP_InvalidBody(t *testing.T) {
	svc := mustDepositSvc()
	r := instrumentDeposit(svc)

	body := `not json`
	req := httptest.NewRequest(http.MethodPost, "/calc/deposit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestDepositHTTP_MissingPrincipal(t *testing.T) {
	svc := mustDepositSvc()
	r := instrumentDeposit(svc)

	body := `{"annual_rate":"0.10","term_months":12}`
	req := httptest.NewRequest(http.MethodPost, "/calc/deposit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestDepositHTTP_ZeroRate(t *testing.T) {
	svc := mustDepositSvc()
	r := instrumentDeposit(svc)

	body := `{"principal":"100000","annual_rate":"0","term_months":12}`
	req := httptest.NewRequest(http.MethodPost, "/calc/deposit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var res map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	maturity := res["maturity_amount"].(string)
	if maturity != "100000" {
		t.Errorf("maturity_amount = %s, want 100000", maturity)
	}
}

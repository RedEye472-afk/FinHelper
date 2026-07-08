package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RedEye472-afk/FinHelper/backend/pkg/auth"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/domain"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/dashboard"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/storage"
)

// dashFakeRepo is a tiny canned dashboard.Repo for the HTTP test. We only need
// one happy path + the 401 path, so the data is fixed.
type dashFakeRepo struct {
	called bool
}

func (f *dashFakeRepo) CashflowForPeriod(_ context.Context, _ int64, _, _ time.Time) (storage.CashflowTotals, error) {
	f.called = true
	return storage.CashflowTotals{
		Income: domain.MustParseMoney("100000.00"),
		Expense: domain.MustParseMoney("60000.00"),
		Net:    domain.MustParseMoney("40000.00"),
	}, nil
}
func (f *dashFakeRepo) ExpensesByCategory(_ context.Context, _ int64, _, _ time.Time) ([]storage.CategorySpend, error) {
	f.called = true
	return []storage.CategorySpend{{CategoryID: 1, CategoryName: "Продукты", Total: domain.MustParseMoney("30000.00")}}, nil
}
func (f *dashFakeRepo) NetWorth(_ context.Context, _ int64) (storage.NetWorth, error) {
	f.called = true
	return storage.NetWorth{Assets: domain.MustParseMoney("500000.00"), Debts: domain.MustParseMoney("100000.00"), Net: domain.MustParseMoney("400000.00")}, nil
}
func (f *dashFakeRepo) GoalProgresses(_ context.Context, _ int64) ([]storage.GoalProgress, error) {
	f.called = true
	return []storage.GoalProgress{{ID: 1, Name: "Подушка", Target: domain.MustParseMoney("300000.00"), Current: domain.MustParseMoney("150000.00")}}, nil
}

func newDashTestEnv(t *testing.T) (*httptest.Server, *auth.JWTIssuer) {
	t.Helper()
	issuer, err := auth.NewJWTIssuer(
		"access-test-secret-must-be-32+-chars-long",
		"refresh-test-secret-must-be-32+-chars-long",
		15*time.Minute, time.Hour,
	)
	if err != nil {
		t.Fatalf("issuer: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	svc := dashboard.NewService(&dashFakeRepo{})
	mw := NewAuthMiddleware(issuer, logger)
	h := NewDashboardHandler(svc, logger)

	mux := http.NewServeMux()
	mux.Handle("/api/v1/dashboard", mw.Wrap(http.HandlerFunc(h.Get)))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, issuer
}

func TestDashboard_Get_Success(t *testing.T) {
	srv, issuer := newDashTestEnv(t)
	tok, _, err := issuer.IssueAccess(7, "hash", "")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/dashboard?period=month", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, buf.String())
	}
	var out dashboardResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Period != "month" {
		t.Errorf("period = %s, want month", out.Period)
	}
	if out.Income != "100000.00" {
		t.Errorf("income = %s, want 100000.00", out.Income)
	}
	if out.Net != "40000.00" {
		t.Errorf("net = %s, want 40000.00", out.Net)
	}
	if len(out.ByCategory) != 1 || out.ByCategory[0].CategoryName != "Продукты" {
		t.Errorf("by_category = %+v", out.ByCategory)
	}
	if out.NetWorth.Net != "400000.00" {
		t.Errorf("net_worth.net = %s, want 400000.00", out.NetWorth.Net)
	}
	if len(out.Goals) != 1 || out.Goals[0].Name != "Подушка" {
		t.Errorf("goals = %+v", out.Goals)
	}
}

func TestDashboard_Unauthorized_WithoutToken(t *testing.T) {
	srv, _ := newDashTestEnv(t)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/dashboard", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestDashboard_RejectsBadFromDate(t *testing.T) {
	srv, issuer := newDashTestEnv(t)
	tok, _, _ := issuer.IssueAccess(7, "hash", "")

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/dashboard?from=not-a-date", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestDashboard_CustomRange_Succeeds(t *testing.T) {
	srv, issuer := newDashTestEnv(t)
	tok, _, _ := issuer.IssueAccess(7, "hash", "")

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/dashboard?from=2026-01-01&to=2026-03-31", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, buf.String())
	}
}

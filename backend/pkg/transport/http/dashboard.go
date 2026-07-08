package http

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/dashboard"
)

// DashboardHandler wires the dashboard REST endpoint (BUSINESS_LOGIC.md ф.3).
//
// Routes (mounted by the router under /api/v1, behind AuthMiddleware):
//
//	GET /dashboard?period=month|quarter|year   summary for named period
//	GET /dashboard?from=YYYY-MM-DD&to=...      summary for custom range
type DashboardHandler struct {
	svc    *dashboard.Service
	logger *slog.Logger
}

// NewDashboardHandler rejects nil deps at boot.
func NewDashboardHandler(svc *dashboard.Service, logger *slog.Logger) *DashboardHandler {
	if svc == nil || logger == nil {
		panic("http: NewDashboardHandler requires non-nil deps")
	}
	return &DashboardHandler{svc: svc, logger: logger}
}

// Register mounts the dashboard routes on the authenticated chi router.
func (h *DashboardHandler) Register(r chi.Router) {
	r.Get("/dashboard", h.Get)
}

// Get → GET /api/v1/dashboard.
func (h *DashboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}

	q := r.URL.Query()
	period := dashboard.Period(q.Get("period"))
	rng := dashboard.CustomRange{}
	if from := q.Get("from"); from != "" {
		t, err := time.Parse("2006-01-02", from)
		if err != nil {
			writeError(w, http.StatusBadRequest, "dash.invalid_from", "from must be YYYY-MM-DD")
			return
		}
		rng.From = t
	}
	if to := q.Get("to"); to != "" {
		t, err := time.Parse("2006-01-02", to)
		if err != nil {
			writeError(w, http.StatusBadRequest, "dash.invalid_to", "to must be YYYY-MM-DD")
			return
		}
		// Inclusive end-of-day.
		rng.To = t.Add(24*time.Hour - time.Nanosecond)
	}
	// Default period when neither period nor a custom range was given.
	if period == "" && rng.From.IsZero() {
		period = dashboard.PeriodMonth
	}

	summary, err := h.svc.Compute(ctx, userID, period, rng)
	if err != nil {
		if errors.Is(err, dashboard.ErrInvalidArgument) {
			writeError(w, http.StatusBadRequest, "dash.invalid_argument", err.Error())
			return
		}
		h.logger.Error("dashboard: compute", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	writeJSON(w, http.StatusOK, toDashboardResponse(summary))
}

// ---- Response shaping ----
//
// Money fields are serialized as strings (no float64), matching the rest of
// the API. Dates are RFC3339 so the client can parse them uniformly.

type dashboardResponse struct {
	Period         string                 `json:"period"`
	From           string                 `json:"from"`
	To             string                 `json:"to"`
	Income         string                 `json:"income"`
	Expense        string                 `json:"expense"`
	Net            string                 `json:"net"`
	ByCategory     []categorySpendDTO     `json:"by_category"`
	NetWorth       netWorthDTO            `json:"net_worth"`
	Goals          []goalProgressDTO      `json:"goals"`
}

type categorySpendDTO struct {
	CategoryID   int64  `json:"category_id"`
	CategoryName string `json:"category_name"`
	Total        string `json:"total"`
}

type netWorthDTO struct {
	Assets string `json:"assets"`
	Debts  string `json:"debts"`
	Net    string `json:"net"`
}

type goalProgressDTO struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Target   string `json:"target"`
	Current  string `json:"current"`
	Progress string `json:"progress"`
}

func toDashboardResponse(s dashboard.Summary) dashboardResponse {
	cats := make([]categorySpendDTO, 0, len(s.ByCategory))
	for _, c := range s.ByCategory {
		cats = append(cats, categorySpendDTO{
			CategoryID: c.CategoryID, CategoryName: c.CategoryName, Total: c.Total.String(),
		})
	}
	goals := make([]goalProgressDTO, 0, len(s.Goals))
	for _, g := range s.Goals {
		goals = append(goals, goalProgressDTO{
			ID: g.ID, Name: g.Name, Target: g.Target.String(),
			Current: g.Current.String(), Progress: g.Progress.String(),
		})
	}
	return dashboardResponse{
		Period: string(s.Period),
		From:   s.From.Format(time.RFC3339),
		To:     s.To.Format(time.RFC3339),
		Income: s.Income.String(), Expense: s.Expense.String(), Net: s.Net.String(),
		ByCategory: cats,
		NetWorth:   netWorthDTO{Assets: s.NetWorth.Assets.String(), Debts: s.NetWorth.Debts.String(), Net: s.NetWorth.Net.String()},
		Goals:      goals,
	}
}

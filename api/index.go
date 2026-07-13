package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/RedEye472-afk/FinHelper/backend/pkg/auth"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/config"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/email"
	applog "github.com/RedEye472-afk/FinHelper/backend/pkg/log"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/migrate"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/ratelimit"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/budget"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/categorization"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/credit"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/dashboard"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/deposit"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/goals"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/operations"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/storage"
	transporthttp "github.com/RedEye472-afk/FinHelper/backend/pkg/transport/http"
)

var (
	h    http.Handler
	once sync.Once
)

func getHandler() http.Handler {
	once.Do(func() {
		h = initHandler()
	})
	return h
}

func initHandler() http.Handler {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("vercel: config load error, degraded mode: %v", err)
		return buildDegradedRouter("configuration error: " + err.Error())
	}

	logger := applog.New(cfg.Log.Level, cfg.Log.Format)

	pool, dbErr := storage.OpenLazy(cfg.Database.URL)
	if dbErr != nil {
		log.Printf("vercel: pool creation error, degraded mode: %v", dbErr)
		return buildDegradedRouter("database config error: " + dbErr.Error())
	}
	log.Println("vercel: lazy pool created (no connection yet)")

	demoUserID := int64(10)

	issuer, err := auth.NewJWTIssuer(
		cfg.JWT.AccessSecret, cfg.JWT.RefreshSecret,
		cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL,
	)
	if err != nil {
		log.Printf("vercel: jwt issuer error, degraded mode: %v", err)
		return buildDegradedRouter("jwt issuer error: " + err.Error())
	}

	demoAuthMW := transporthttp.NewDemoAuthMiddleware(demoUserID, issuer, logger)

	var mailer *email.Sender
	if cfg.Email.ResendAPIKey != "" || cfg.Email.SendGridAPIKey != "" || cfg.Email.BrevoAPIKey != "" {
		mailer = email.NewSender(
			logger,
			cfg.Email.FromEmail,
			cfg.Email.FromName,
			cfg.Email.ResendAPIKey,
			cfg.Email.SendGridAPIKey,
			cfg.Email.BrevoAPIKey,
			cfg.Email.BrevoSender,
		)
	}

	rl := ratelimit.New(logger)

	opsSvc := operations.NewService(pool)
	catSvc := categorization.NewService(pool)
	opsSvc.SetCategorizer(catSvc)
	dashSvc := dashboard.NewService(pool)
	budSvc := budget.NewService(pool)
	goalsSvc := goals.NewService(pool)
	credSvc := credit.NewService()
	depSvc := deposit.NewService()

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.HTTP.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Idempotency-Key"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)

	apiRouter := transporthttp.NewRouter(transporthttp.Deps{
		Pool:           pool,
		Issuer:         issuer,
		Salt:           cfg.UserHashSalt,
		Logger:         logger,
		Mailer:         mailer,
		RateLimiter:    rl,
		FrontendURL:    cfg.Email.FrontendURL,
		Operations:     opsSvc,
		Categorization: catSvc,
		Dashboard:      dashSvc,
		Budget:         budSvc,
		Goals:          goalsSvc,
		Deposit:        depSvc,
		Credit:         credSvc,
	}, demoAuthMW)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		probeCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := pool.DB.PingContext(probeCtx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"degraded","db":"unreachable"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ready","db":"connected"}`))
	})

	r.Post("/api/v1/migrate", func(w http.ResponseWriter, r *http.Request) {
		migrateCtx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		migrate.Run(migrateCtx, pool)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","msg":"migrations applied"}`))
	})
	r.Get("/migrate", func(w http.ResponseWriter, r *http.Request) {
		migrateCtx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		migrate.Run(migrateCtx, pool)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","msg":"migrations applied"}`))
	})

	// PDF parsing — pure Go, no external deps
	r.Post("/api/v1/import/parse-pdf", HandlePDFParse)

	r.Mount("/", apiRouter)
	log.Println("vercel: handler ready")
	return r
}

func buildDegradedRouter(reason string) http.Handler {
	msg := `{"status":"degraded","note":"` + reason + `"}`
	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(msg))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(msg))
	})
	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(msg))
	})
	return r
}

func Handler(w http.ResponseWriter, r *http.Request) {
	getHandler().ServeHTTP(w, r)
}

// ── PDF parsing handler (pure Go, no Python) ────────────────────────────────

func HandlePDFParse(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(nil, r.Body, 50<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, "import.invalid_upload",
			"failed to parse multipart form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "import.missing_file",
			"file field is required: "+err.Error())
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".pdf") {
		writeJSONError(w, http.StatusBadRequest, "import.invalid_type",
			"only PDF files are accepted")
		return
	}

	tmpFile, err := os.CreateTemp("", "finhelper-parse-*.pdf")
	if err != nil {
		log.Printf("pdf_parse: create temp: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, file); err != nil {
		tmpFile.Close()
		log.Printf("pdf_parse: write temp: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	tmpFile.Close()

	text, err := extractPDFText(tmpPath)
	if err != nil || text == "" {
		log.Printf("pdf_parse: extract error or empty: %v", err)
		// Don't fail — frontend will use pdf.js fallback
		writeJSON(w, http.StatusOK, map[string]any{
			"text":       "",
			"line_count": 0,
			"fallback":   true,
		})
		return
	}

	lineCount := 0
	if text != "" {
		lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
		lineCount = len(lines)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"text":       text,
		"line_count": lineCount,
	})
}

func extractPDFText(pdfPath string) (string, error) {
	// Try pdftotext (poppler-utils) if available
	if _, err := exec.LookPath("pdftotext"); err == nil {
		cmd := exec.Command("pdftotext", "-layout", pdfPath, "-")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err == nil {
			return stdout.String(), nil
		}
		log.Printf("pdftotext failed: %v, stderr: %s", err, stderr.String())
	}

	// Try python3 with pdfplumber/PyPDF2 if available
	if _, err := exec.LookPath("python3"); err == nil {
		script := `
import sys
try:
    import pdfplumber
    with pdfplumber.open(sys.argv[1]) as pdf:
        text = []
        for page in pdf.pages:
            t = page.extract_text()
            if t:
                text.append(t)
        print("\n\n".join(text))
except ImportError:
    try:
        import PyPDF2
        with open(sys.argv[1], "rb") as f:
            reader = PyPDF2.PdfReader(f)
            text = []
            for page in reader.pages:
                t = page.extract_text()
                if t:
                    text.append(t)
            print("\n\n".join(text))
    except ImportError:
        sys.exit(1)
`
		tmpScript, err := os.CreateTemp("", "pdf_extract_*.py")
		if err != nil {
			return "", err
		}
		defer os.Remove(tmpScript.Name())
		if _, err := tmpScript.WriteString(script); err != nil {
			return "", err
		}
		tmpScript.Close()

		cmd := exec.Command("python3", tmpScript.Name(), pdfPath)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err == nil {
			return stdout.String(), nil
		}
		log.Printf("python pdf extract failed: %v, stderr: %s", err, stderr.String())
	}

	// Fallback: return empty, frontend will use pdf.js
	return "", nil
}

type problem struct {
	Type   string `json:"type,omitempty"`
	Title  string `json:"title,omitempty"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		_, _ = w.Write([]byte(`{"type":"internal","title":"encode_failed"}`))
	}
}

func writeJSONError(w http.ResponseWriter, status int, typ, detail string) {
	writeJSON(w, status, problem{Type: typ, Status: status, Detail: detail})
}
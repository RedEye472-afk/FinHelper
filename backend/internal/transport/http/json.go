// Package http exposes FinHelper's REST API. This file holds the small
// JSON helpers shared by handlers; auth handlers live in auth.go and the
// JWT middleware in middleware.go.
package http

import (
	"encoding/json"
	"net/http"
)

// Problem is our standard error envelope (RFC 7807-shaped but simpler).
// We do NOT adopt the full RFC 7807 media type to avoid a hard dependency;
// the shape is what callers parse anyway.
type Problem struct {
	Type   string `json:"type,omitempty"`   // machine code, e.g. "auth.invalid_credentials"
	Title  string `json:"title,omitempty"`  // short human label
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"` // safe to surface to client
}

// writeJSON serializes v as JSON and writes it with the given status.
// On marshal failure we fall back to a plain 500 — there is no useful
// recovery once we can't even encode our response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Best-effort: the header is already sent, just log-free fail.
		_, _ = w.Write([]byte(`{"type":"internal","title":"encode_failed"}`))
	}
}

// writeError is the one funnel every error response goes through.
// Centralizing it keeps status codes consistent across handlers.
func writeError(w http.ResponseWriter, status int, typ, detail string) {
	writeJSON(w, status, Problem{Type: typ, Status: status, Detail: detail})
}

package handler

import "net/http"

// Singleton handler — no init, no DB, no config.
var ready = true

func Handler(w http.ResponseWriter, r *http.Request) {
	if !ready {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"starting"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":true,"path":"` + r.URL.Path + `"}`))
}

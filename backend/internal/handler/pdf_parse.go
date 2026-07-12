// Package handler holds HTTP handlers that are too specialised or
// project-root-specific to live in pkg/transport/http. Each handler
// is self-contained — no dependency on the transport layer, no deps
// struct, just a plain http.HandlerFunc that uses the standard library
// plus exec for external tools.
package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// HandlePDFParse accepts a POST multipart upload with a "file" field,
// saves it to a temp file, runs scripts/pdf_parse.py with exec.Command,
// reads the extracted text, cleans up, and returns JSON.
//
//	POST /api/v1/import/parse-pdf
//	Content-Type: multipart/form-data
//	Field "file": the PDF to parse
//
// Response 200: {"text": "...", "line_count": N}
func HandlePDFParse(w http.ResponseWriter, r *http.Request) {
	// Limit upload to 50 MiB.
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

	// Basic validation — only PDFs.
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".pdf") {
		writeJSONError(w, http.StatusBadRequest, "import.invalid_type",
			"only PDF files are accepted")
		return
	}

	// Save to a temporary file.
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

	// Locate the Python script.
	scriptPath := findScript("scripts/pdf_parse.py")
	if scriptPath == "" {
		log.Printf("pdf_parse: script not found (searched relative to cwd, parent, project root)")
		writeJSONError(w, http.StatusInternalServerError, "import.script_not_found",
			"pdf_parse.py not found on server")
		return
	}

	// Run python scripts/pdf_parse.py <tmpfile>.
	// On Windows dev the command is "python"; on the Linux λ "python3"
	// may be the correct name. We try "python3" first, then "python".
	cmd := buildCmd(scriptPath, tmpPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		log.Printf("pdf_parse: exec error: %v\nstderr: %s", err, strings.TrimSpace(stderr.String()))
		writeJSONError(w, http.StatusInternalServerError, "import.parse_failed",
			"pdf parsing failed: "+errMsg)
		return
	}

	text := stdout.String()
	lineCount := 0
	if text != "" {
		// Count lines; trailing empty line from final \n is ignored.
		lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
		lineCount = len(lines)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"text":       text,
		"line_count": lineCount,
	})
}

// ── helpers ──────────────────────────────────────────────────────────────

// findScript searches for the script in several locations relative to the
// running binary's working directory:
//  1. directly (cwd / scripts/…)
//  2. one level up (../scripts/…)
//  3. two levels up (../../scripts/…)
//  4. absolute fallback for Vercel λ (project root via FINHELPER_ROOT env)
func findScript(rel string) string {
	candidates := []string{
		rel,
		filepath.Join("..", rel),
		filepath.Join("..", "..", rel),
	}
	if root := os.Getenv("FINHELPER_ROOT"); root != "" {
		candidates = append(candidates, filepath.Join(root, rel))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	return ""
}

// buildCmd returns the exec.Cmd for running the pdf parser. On Windows
// we try "python" first; on Linux "python3" first — the OS check is
// light and avoids a syscall per request.
func buildCmd(scriptPath, tmpPath string) *exec.Cmd {
	args := []string{scriptPath, tmpPath}

	// Fast OS hint: if the binary path separator is '\' we're on Windows.
	if os.PathSeparator == '\\' {
		return exec.Command("python", args...)
	}

	// On Linux try python3 first, fall back to python.
	if _, err := exec.LookPath("python3"); err == nil {
		return exec.Command("python3", args...)
	}
	return exec.Command("python", args...)
}

// ── JSON I/O ─────────────────────────────────────────────────────────────

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

package handler

import (
	"csv-server/internal/csv"
	"csv-server/templates/pages"
	"csv-server/web"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Handler struct {
	dataDir string
	logger  *slog.Logger
}

func New(dataDir string, logger *slog.Logger) *Handler {
	return &Handler{dataDir: filepath.Clean(dataDir), logger: logger}
}

// Register mounts all routes on mux using Go 1.22+ method+path routing.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /static/", http.FileServerFS(web.Static))
	mux.HandleFunc("GET /", h.index)
	mux.HandleFunc("GET /browse/{path...}", h.browse)
	mux.HandleFunc("GET /view/{path...}", h.view)
	mux.HandleFunc("GET /health", h.health)
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	h.serveDir(w, r, "")
}

func (h *Handler) browse(w http.ResponseWriter, r *http.Request) {
	h.serveDir(w, r, r.PathValue("path"))
}

func (h *Handler) serveDir(w http.ResponseWriter, r *http.Request, relPath string) {
	absPath, err := h.safePath(relPath)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	entries, err := csv.ListEntries(absPath)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "listing entries", "path", relPath, "err", err)
		http.Error(w, "could not list directory", http.StatusInternalServerError)
		return
	}

	if err := pages.Index(pages.IndexProps{CurrentPath: relPath, Entries: entries}).Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "rendering index", "err", err)
	}
}

func (h *Handler) view(w http.ResponseWriter, r *http.Request) {
	relPath := r.PathValue("path")

	absPath, err := h.safePath(relPath)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	fsys := os.DirFS(h.dataDir)
	fsPath := filepath.ToSlash(strings.TrimPrefix(absPath, h.dataDir+string(filepath.Separator)))

	table, err := csv.ReadTable(fsys, fsPath)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "reading csv", "path", relPath, "err", err)
		http.Error(w, "file not found or unreadable", http.StatusNotFound)
		return
	}

	if err := pages.Table(pages.TableProps{Filename: relPath, Headers: table.Headers, Rows: table.Rows}).Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "rendering table", "path", relPath, "err", err)
	}
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

// safePath resolves relPath against dataDir and rejects path traversal attempts.
func (h *Handler) safePath(relPath string) (string, error) {
	abs := filepath.Clean(filepath.Join(h.dataDir, filepath.FromSlash(relPath)))
	rel, err := filepath.Rel(h.dataDir, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", errors.New("path traversal detected")
	}
	return abs, nil
}

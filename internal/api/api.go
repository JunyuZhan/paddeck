package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strconv"
	"time"

	"github.com/JunyuZhan/paddeck/internal/collector"
	"github.com/JunyuZhan/paddeck/internal/store"
)

// SnapshotSource is anything that can hand out the latest snapshot
// (the real collector manager, or the demo generator).
type SnapshotSource interface {
	Snapshot() collector.Snapshot
}

// HistorySource is anything that can return stored samples (SQLite or demo).
type HistorySource interface {
	History(host string, from, to int64, limit int) ([]store.Sample, error)
}

// Server exposes the read-only API and static frontend.
type Server struct {
	Mgr   SnapshotSource
	Store HistorySource
	Web   fs.FS
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/snapshot", s.handleSnapshot)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.Handle("/", http.FileServer(http.FS(s.Web)))
	return mux
}

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.Mgr.Snapshot())
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	if host == "" {
		http.Error(w, "host is required", http.StatusBadRequest)
		return
	}
	dur := parseRange(r.URL.Query().Get("range"))
	to := time.Now().Unix()
	from := to - int64(dur.Seconds())
	samples, err := s.Store.History(host, from, to, 240)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"host": host, "range": r.URL.Query().Get("range"), "samples": samples})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"ok": true, "ts": time.Now().Unix()})
}

func parseRange(v string) time.Duration {
	switch v {
	case "6h":
		return 6 * time.Hour
	case "24h":
		return 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	default:
		return time.Hour
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	enc := json.NewEncoder(w)
	_ = enc.Encode(v)
}

// ParseLimit is a helper for future pagination.
func ParseLimit(v string, def int) int {
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

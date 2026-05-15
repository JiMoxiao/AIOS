package http

import (
	"net/http"

	"aios/internal/telemetry"
)

func registerRoutes(s *Server) {
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	s.mux.HandleFunc("POST /v1/generate", s.handleGenerate)
	s.mux.HandleFunc("POST /v1/run", s.handleRun)
	s.mux.HandleFunc("GET /v1/runs/{run_id}", s.handleGetRun)
	s.mux.HandleFunc("GET /v1/workflows/{workflow_id}/yaml", s.handleGetWorkflowYAML)
	s.mux.Handle("GET /metrics", telemetry.MetricsHandler())
}

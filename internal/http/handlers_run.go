package http

import (
	"net/http"

	"aios/internal/executor"
)

type runRequest struct {
	WorkflowID string `json:"workflow_id"`
}

type runResponse struct {
	RunID string `json:"run_id"`
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if s.store == nil || s.provider == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "runtime_not_configured"})
		return
	}

	var req runRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if req.WorkflowID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing_workflow_id"})
		return
	}

	ex := &executor.Executor{
		Store:    s.store,
		Provider: s.provider,
		Router:   s.router,
	}

	runID, err := ex.RunWorkflow(r.Context(), req.WorkflowID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "run_failed", "detail": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, runResponse{RunID: runID})
}


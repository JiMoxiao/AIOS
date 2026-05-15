package http

import (
	"net/http"
)

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "store_not_configured"})
		return
	}

	runID := r.PathValue("run_id")
	if runID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing_run_id"})
		return
	}

	rec, err := s.store.GetRun(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "run_not_found"})
		return
	}

	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) handleGetWorkflowYAML(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "store_not_configured"})
		return
	}

	workflowID := r.PathValue("workflow_id")
	if workflowID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing_workflow_id"})
		return
	}

	_, yaml, _, err := s.store.GetWorkflow(r.Context(), workflowID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "workflow_not_found"})
		return
	}

	w.Header().Set("Content-Type", "text/yaml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(yaml))
}


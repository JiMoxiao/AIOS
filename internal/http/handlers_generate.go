package http

import (
	"encoding/json"
	"net/http"

	"aios/internal/dsl"
	"aios/internal/generator"
	"aios/internal/yamlrender"
)

type generateRequest struct {
	Intent string `json:"intent"`
	Mode   string `json:"mode"`
}

type generateResponse struct {
	WorkflowID       string          `json:"workflow_id"`
	WorkflowSpecJSON json.RawMessage `json:"workflow_spec_json"`
	WorkflowYAML     string          `json:"workflow_yaml"`
}

func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "store_not_configured"})
		return
	}

	var req generateRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	if req.Intent == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing_intent"})
		return
	}

	mode := dsl.Mode(req.Mode)
	if mode == "" {
		mode = dsl.ModeBalanced
	}

	spec, err := generator.Generate(req.Intent, mode)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "generate_failed"})
		return
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "marshal_failed"})
		return
	}
	if err := dsl.ValidateWorkflowSpecJSON(specJSON); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "generated_spec_invalid"})
		return
	}

	yaml, err := yamlrender.RenderWorkflowYAML(spec)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "yaml_render_failed"})
		return
	}

	workflowID, _, err := s.store.SaveWorkflow(r.Context(), specJSON, yaml)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "store_failed"})
		return
	}

	writeJSON(w, http.StatusOK, generateResponse{
		WorkflowID:       workflowID,
		WorkflowSpecJSON: specJSON,
		WorkflowYAML:     yaml,
	})
}


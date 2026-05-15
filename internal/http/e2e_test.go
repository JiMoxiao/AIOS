package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"aios/internal/dsl"
	"aios/internal/providers"
	"aios/internal/providers/mock"
	"aios/internal/router"
	"aios/internal/store"
)

func TestE2E_GenerateRunQuery(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "itw.sqlite")

	st, err := store.OpenSQLite(ctx, store.SQLiteStoreConfig{DSN: dbPath, SaveBody: false})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	p := &mock.Provider{
		Seq: []mock.Step{
			{Response: providers.ChatResponse{Content: "plan output is long enough to pass constraints................................................................"}},
			{Response: providers.ChatResponse{Content: "# Task Output\n\ntask output is long enough to pass constraints................................................................"}},
			{Response: providers.ChatResponse{Content: "final output is long enough to pass constraints................................................................"}},
		},
	}

	srv := NewServer(ServerConfig{
		Store:    st,
		Provider: p,
		Router: router.Router{
			Catalog: router.Catalog{
				Models: []router.Model{
					{ID: "balanced", QualityScore: 0.8, CostScore: 0.4, LatencyScore: 0.4, ErrorRateScore: 0.2, MaxContextTokens: 64000},
					{ID: "premium", QualityScore: 1.0, CostScore: 1.0, LatencyScore: 0.6, ErrorRateScore: 0.1, MaxContextTokens: 200000},
				},
			},
		},
	})

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	genReq := generateRequest{Intent: "写一份Markdown报告", Mode: string(dsl.ModeBalanced)}
	genBody, _ := json.Marshal(genReq)
	resp, err := http.Post(ts.URL+"/v1/generate", "application/json", bytes.NewReader(genBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("generate status %d", resp.StatusCode)
	}
	var genResp generateResponse
	_ = json.NewDecoder(resp.Body).Decode(&genResp)
	if genResp.WorkflowID == "" || len(genResp.WorkflowSpecJSON) == 0 || genResp.WorkflowYAML == "" {
		t.Fatalf("invalid generate response")
	}

	runReq := runRequest{WorkflowID: genResp.WorkflowID}
	runBody, _ := json.Marshal(runReq)
	resp2, err := http.Post(ts.URL+"/v1/run", "application/json", bytes.NewReader(runBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		var m map[string]any
		_ = json.NewDecoder(resp2.Body).Decode(&m)
		t.Fatalf("run status %d body=%v", resp2.StatusCode, m)
	}
	var runResp runResponse
	_ = json.NewDecoder(resp2.Body).Decode(&runResp)
	if runResp.RunID == "" {
		t.Fatalf("missing run id")
	}

	resp3, err := http.Get(ts.URL + "/v1/runs/" + runResp.RunID)
	if err != nil {
		t.Fatal(err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("get run status %d", resp3.StatusCode)
	}
	var rec dsl.RunRecord
	_ = json.NewDecoder(resp3.Body).Decode(&rec)
	if rec.RunID != runResp.RunID || rec.Status != dsl.RunSucceeded {
		t.Fatalf("unexpected run record")
	}
}


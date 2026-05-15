package executor

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"aios/internal/dsl"
	"aios/internal/providers"
	"aios/internal/providers/mock"
	"aios/internal/router"
	"aios/internal/store"
)

func TestExecutor_FallbackOnEmptyResponse(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "itw.sqlite")

	st, err := store.OpenSQLite(ctx, store.SQLiteStoreConfig{DSN: dbPath, SaveBody: false})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	spec := dsl.WorkflowSpec{
		SpecVersion:      "1.0",
		GeneratorVersion: "1.0",
		Intent:           "x",
		Mode:             dsl.ModeBalanced,
		GlobalConstraints: dsl.GlobalConstraints{
			DataClassification: dsl.DataInternal,
		},
		Nodes: []dsl.Node{
			{ID: "task", Type: dsl.NodeLLMTask, Name: "Task", RetryPolicy: dsl.RetryPolicy{MaxAttempts: 1}},
		},
		Edges:   []dsl.Edge{},
		Outputs: []dsl.Output{{Name: "final", FromNodeID: "task"}},
	}
	if err := dsl.ValidateWorkflowSpec(spec); err != nil {
		t.Fatal(err)
	}
	specJSON, _ := json.Marshal(spec)

	workflowID, _, err := st.SaveWorkflow(ctx, specJSON, "spec_version: \"1.0\"\n")
	if err != nil {
		t.Fatal(err)
	}

	p := &mock.Provider{
		Seq: []mock.Step{
			{Response: providers.ChatResponse{Content: ""}},
			{Response: providers.ChatResponse{Content: "hello world this is long enough to pass evaluator constraints"}},
		},
	}

	ex := &Executor{
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
	}

	runID, err := ex.RunWorkflow(ctx, workflowID)
	if err != nil {
		t.Fatal(err)
	}

	rec, err := st.GetRun(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Status != dsl.RunSucceeded {
		t.Fatalf("expected succeeded, got %s", rec.Status)
	}
	if len(rec.NodeRuns) != 1 {
		t.Fatalf("expected 1 node run")
	}
	if len(rec.NodeRuns[0].Fallbacks) == 0 {
		t.Fatalf("expected fallback attempts recorded")
	}
}


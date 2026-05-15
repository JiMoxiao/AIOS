package store

import (
	"context"
	"path/filepath"
	"testing"

	"aios/internal/dsl"
)

func TestSQLiteStore_WorkflowAndRunLifecycle(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "itw.sqlite")

	s, err := OpenSQLite(ctx, SQLiteStoreConfig{
		DSN:      dbPath,
		SaveBody: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	specJSON := []byte(`{"spec_version":"1.0","generator_version":"1.0","intent":"x","mode":"balanced","nodes":[{"id":"plan","type":"planner","name":"Plan"}],"edges":[],"outputs":[{"name":"final","from_node_id":"plan"}]}`)
	if err := dsl.ValidateWorkflowSpecJSON(specJSON); err != nil {
		t.Fatal(err)
	}

	workflowID, specHash, err := s.SaveWorkflow(ctx, specJSON, "spec_version: \"1.0\"\n")
	if err != nil {
		t.Fatal(err)
	}

	spec2, yaml2, hash2, err := s.GetWorkflow(ctx, workflowID)
	if err != nil {
		t.Fatal(err)
	}
	if string(spec2) != string(specJSON) {
		t.Fatalf("spec mismatch")
	}
	if yaml2 == "" || hash2 == "" || hash2 != specHash {
		t.Fatalf("workflow fields mismatch")
	}

	runID, err := s.CreateRun(ctx, workflowID, specHash)
	if err != nil {
		t.Fatal(err)
	}

	rec, err := s.GetRun(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	if rec.RunID != runID || rec.WorkflowID != workflowID || rec.SpecHash != specHash {
		t.Fatalf("run fields mismatch")
	}

	art, err := s.SaveArtifact(ctx, "plan", "text", "hello world")
	if err != nil {
		t.Fatal(err)
	}

	rec.Artifacts = append(rec.Artifacts, art)
	rec.Status = dsl.RunSucceeded
	if err := s.UpdateRun(ctx, rec); err != nil {
		t.Fatal(err)
	}

	rec2, err := s.GetRun(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	if rec2.Status != dsl.RunSucceeded || len(rec2.Artifacts) != 1 || rec2.Artifacts[0].ArtifactID == "" {
		t.Fatalf("unexpected run record")
	}
}


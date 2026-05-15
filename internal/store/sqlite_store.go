package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"aios/internal/dsl"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db       *sql.DB
	saveBody bool
}

type SQLiteStoreConfig struct {
	DSN      string
	SaveBody bool
}

func OpenSQLite(ctx context.Context, cfg SQLiteStoreConfig) (*SQLiteStore, error) {
	if cfg.DSN == "" {
		return nil, errors.New("missing_dsn")
	}

	db, err := sql.Open("sqlite", cfg.DSN)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	s := &SQLiteStore{db: db, saveBody: cfg.SaveBody}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) SaveWorkflow(ctx context.Context, specJSON []byte, yaml string) (workflowID string, specHash string, err error) {
	workflowID = newID()
	sum := sha256.Sum256(specJSON)
	specHash = hex.EncodeToString(sum[:])
	now := time.Now().UTC().Format(time.RFC3339Nano)

	_, err = s.db.ExecContext(ctx, `
INSERT INTO workflows (id, spec_json, yaml, spec_hash, created_at)
VALUES (?, ?, ?, ?, ?)`,
		workflowID, string(specJSON), yaml, specHash, now,
	)
	if err != nil {
		return "", "", err
	}
	return workflowID, specHash, nil
}

func (s *SQLiteStore) GetWorkflow(ctx context.Context, workflowID string) (specJSON []byte, yaml string, specHash string, err error) {
	var specStr string
	err = s.db.QueryRowContext(ctx, `
SELECT spec_json, yaml, spec_hash
FROM workflows
WHERE id = ?`, workflowID).Scan(&specStr, &yaml, &specHash)
	if err != nil {
		return nil, "", "", err
	}
	return []byte(specStr), yaml, specHash, nil
}

func (s *SQLiteStore) CreateRun(ctx context.Context, workflowID, specHash string) (runID string, err error) {
	runID = newID()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	rec := dsl.RunRecord{
		RunID:      runID,
		WorkflowID: workflowID,
		SpecHash:   specHash,
		Status:     dsl.RunRunning,
		StartedAt:  now,
		NodeRuns:   []dsl.NodeRun{},
		Artifacts:  []dsl.ArtifactRef{},
	}
	runJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}

	_, err = s.db.ExecContext(ctx, `
INSERT INTO runs (id, workflow_id, status, run_json, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		runID, workflowID, string(rec.Status), string(runJSON), now, now,
	)
	if err != nil {
		return "", err
	}

	return runID, nil
}

func (s *SQLiteStore) UpdateRun(ctx context.Context, rec dsl.RunRecord) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	runJSON, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
UPDATE runs
SET status = ?, run_json = ?, updated_at = ?
WHERE id = ?`,
		string(rec.Status), string(runJSON), now, rec.RunID,
	)
	return err
}

func (s *SQLiteStore) GetRun(ctx context.Context, runID string) (dsl.RunRecord, error) {
	var runStr string
	err := s.db.QueryRowContext(ctx, `
SELECT run_json
FROM runs
WHERE id = ?`, runID).Scan(&runStr)
	if err != nil {
		return dsl.RunRecord{}, err
	}

	var rec dsl.RunRecord
	if err := json.Unmarshal([]byte(runStr), &rec); err != nil {
		return dsl.RunRecord{}, err
	}
	return rec, nil
}

func (s *SQLiteStore) SaveArtifact(ctx context.Context, nodeID, kind, body string) (dsl.ArtifactRef, error) {
	id := newID()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	summary := summarize(body)
	digest := sha256Hex(body)

	var bodyPtr any
	if s.saveBody {
		bodyPtr = body
	} else {
		bodyPtr = nil
	}

	_, err := s.db.ExecContext(ctx, `
INSERT INTO artifacts (id, node_id, kind, body, summary, digest, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, nodeID, kind, bodyPtr, summary, digest, now,
	)
	if err != nil {
		return dsl.ArtifactRef{}, err
	}

	return dsl.ArtifactRef{
		ArtifactID: id,
		NodeID:     nodeID,
		Kind:       kind,
		Summary:    summary,
		Digest:     digest,
	}, nil
}

func (s *SQLiteStore) migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS workflows (
  id TEXT PRIMARY KEY,
  spec_json TEXT NOT NULL,
  yaml TEXT NOT NULL,
  spec_hash TEXT NOT NULL,
  created_at TEXT NOT NULL
);`,
		`CREATE TABLE IF NOT EXISTS runs (
  id TEXT PRIMARY KEY,
  workflow_id TEXT NOT NULL,
  status TEXT NOT NULL,
  run_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);`,
		`CREATE TABLE IF NOT EXISTS artifacts (
  id TEXT PRIMARY KEY,
  node_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  body TEXT,
  summary TEXT NOT NULL,
  digest TEXT NOT NULL,
  created_at TEXT NOT NULL
);`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	return nil
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func summarize(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	const max = 200
	rs := []rune(s)
	if len(rs) <= max {
		return s
	}
	return string(rs[:max]) + "…"
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}


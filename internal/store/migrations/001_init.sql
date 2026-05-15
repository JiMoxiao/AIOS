CREATE TABLE IF NOT EXISTS workflows (
  id TEXT PRIMARY KEY,
  spec_json TEXT NOT NULL,
  yaml TEXT NOT NULL,
  spec_hash TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS runs (
  id TEXT PRIMARY KEY,
  workflow_id TEXT NOT NULL,
  status TEXT NOT NULL,
  run_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS artifacts (
  id TEXT PRIMARY KEY,
  node_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  body TEXT,
  summary TEXT NOT NULL,
  digest TEXT NOT NULL,
  created_at TEXT NOT NULL
);


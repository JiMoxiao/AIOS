package dsl

type RunStatus string

const (
	RunRunning   RunStatus = "running"
	RunSucceeded RunStatus = "succeeded"
	RunFailed    RunStatus = "failed"
)

type NodeStatus string

const (
	NodePending   NodeStatus = "pending"
	NodeRunning   NodeStatus = "running"
	NodeSucceeded NodeStatus = "succeeded"
	NodeFailed    NodeStatus = "failed"
	NodeSkipped   NodeStatus = "skipped"
)

type RunRecord struct {
	RunID          string        `json:"run_id"`
	WorkflowID     string        `json:"workflow_id"`
	SpecHash       string        `json:"spec_hash"`
	Status         RunStatus     `json:"status"`
	StartedAt      string        `json:"started_at"`
	FinishedAt     *string       `json:"finished_at,omitempty"`
	NodeRuns       []NodeRun     `json:"node_runs"`
	Artifacts      []ArtifactRef `json:"artifacts"`
	FinalOutputRef ArtifactRef   `json:"final_output_ref"`
}

type NodeRun struct {
	NodeID    string            `json:"node_id"`
	Status    NodeStatus        `json:"status"`
	Model     string            `json:"model,omitempty"`
	TokenIn   int               `json:"token_in,omitempty"`
	TokenOut  int               `json:"token_out,omitempty"`
	CostUSD   float64           `json:"cost_usd,omitempty"`
	LatencyMs int               `json:"latency_ms,omitempty"`
	Fallbacks []FallbackAttempt `json:"fallbacks,omitempty"`
	Error     *string           `json:"error,omitempty"`
}

type FallbackAttempt struct {
	Model  string `json:"model"`
	Reason string `json:"reason"`
}

type ArtifactRef struct {
	ArtifactID string `json:"artifact_id"`
	NodeID     string `json:"node_id"`
	Kind       string `json:"kind"`
	Summary    string `json:"summary,omitempty"`
	Digest     string `json:"digest,omitempty"`
}


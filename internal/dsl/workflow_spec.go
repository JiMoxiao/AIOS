package dsl

type Mode string

const (
	ModeQuality  Mode = "quality"
	ModeBalanced Mode = "balanced"
	ModeCost     Mode = "cost"
)

type DataClassification string

const (
	DataPublic       DataClassification = "public"
	DataInternal     DataClassification = "internal"
	DataConfidential DataClassification = "confidential"
	DataRestricted   DataClassification = "restricted"
)

type WorkflowSpec struct {
	SpecVersion       string            `json:"spec_version"`
	GeneratorVersion  string            `json:"generator_version"`
	Intent            string            `json:"intent"`
	Mode              Mode              `json:"mode"`
	GlobalConstraints GlobalConstraints `json:"global_constraints,omitempty"`
	Nodes             []Node            `json:"nodes"`
	Edges             []Edge            `json:"edges"`
	Outputs           []Output          `json:"outputs"`
}

type GlobalConstraints struct {
	DataClassification DataClassification `json:"data_classification"`
	MaxTotalCostUSD    *float64           `json:"max_total_cost_usd,omitempty"`
	MaxWallTimeSec     *int               `json:"max_wall_time_sec,omitempty"`
}

type NodeType string

const (
	NodePlanner   NodeType = "planner"
	NodeLLMTask   NodeType = "llm_task"
	NodeEvaluator NodeType = "evaluator"
	NodeMerge     NodeType = "merge"
	NodeFinalizer NodeType = "finalizer"
)

type Node struct {
	ID                  string           `json:"id"`
	Type                NodeType         `json:"type"`
	Name                string           `json:"name"`
	Description         string           `json:"description,omitempty"`
	Inputs              []InputRef       `json:"inputs,omitempty"`
	ExpectedArtifactType string          `json:"expected_artifact_type,omitempty"`
	OutputSchema        any              `json:"output_schema,omitempty"`
	RoutingProfile      Mode             `json:"routing_profile,omitempty"`
	ModelConstraints    ModelConstraints `json:"model_constraints,omitempty"`
	RetryPolicy         RetryPolicy      `json:"retry_policy,omitempty"`
	FallbackChain       []FallbackRule   `json:"fallback_chain,omitempty"`
}

type InputRef struct {
	Type   string `json:"type"`
	NodeID string `json:"node_id,omitempty"`
}

type ModelConstraints struct {
	AllowModels      []string `json:"allow_models,omitempty"`
	DenyModels       []string `json:"deny_models,omitempty"`
	MinContextTokens *int     `json:"min_context_tokens,omitempty"`
}

type RetryPolicy struct {
	MaxAttempts int `json:"max_attempts,omitempty"`
}

type FallbackRule struct {
	Model string `json:"model"`
	On    string `json:"on"`
}

type Edge struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Condition string `json:"condition"`
}

type Output struct {
	Name       string `json:"name"`
	FromNodeID string `json:"from_node_id"`
}


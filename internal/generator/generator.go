package generator

import (
	"fmt"

	"aios/internal/dsl"
)

const (
	specVersion      = "1.0"
	generatorVersion = "1.0"
)

func Generate(intent string, mode dsl.Mode) (dsl.WorkflowSpec, error) {
	p := ParseIntent(intent)

	nodes := []dsl.Node{
		{
			ID:             "plan",
			Type:           dsl.NodePlanner,
			Name:           "Plan",
			Description:    "拆解任务并定义交付物与约束",
			RoutingProfile: dsl.ModeQuality,
			RetryPolicy:    dsl.RetryPolicy{MaxAttempts: 2},
			FallbackChain: []dsl.FallbackRule{
				{Model: "claude-sonnet", On: "any_error"},
				{Model: "gpt-5", On: "any_error"},
			},
		},
		{
			ID:                   "task",
			Type:                 dsl.NodeLLMTask,
			Name:                 "Task",
			Description:          "执行核心任务",
			Inputs:               []dsl.InputRef{{Type: "node_output", NodeID: "plan"}},
			ExpectedArtifactType: p.ArtifactType,
			RoutingProfile:       mode,
			RetryPolicy:          dsl.RetryPolicy{MaxAttempts: 2},
			FallbackChain: []dsl.FallbackRule{
				{Model: "balanced", On: "validation_fail"},
				{Model: "premium", On: "validation_fail"},
			},
		},
		{
			ID:          "eval",
			Type:        dsl.NodeEvaluator,
			Name:        "Evaluate",
			Description: "对任务输出做质量校验",
			Inputs:      []dsl.InputRef{{Type: "node_output", NodeID: "task"}},
		},
		{
			ID:          "final",
			Type:        dsl.NodeFinalizer,
			Name:        "Finalize",
			Description: "汇总输出并生成最终交付物",
			Inputs:      []dsl.InputRef{{Type: "node_output", NodeID: "task"}},
		},
	}

	edges := []dsl.Edge{
		{From: "plan", To: "task", Condition: "on_success"},
		{From: "task", To: "eval", Condition: "on_success"},
		{From: "eval", To: "final", Condition: "on_success"},
	}

	spec := dsl.WorkflowSpec{
		SpecVersion:      specVersion,
		GeneratorVersion: generatorVersion,
		Intent:           intent,
		Mode:             mode,
		GlobalConstraints: dsl.GlobalConstraints{
			DataClassification: dsl.DataInternal,
		},
		Nodes:   nodes,
		Edges:   edges,
		Outputs: []dsl.Output{{Name: "final", FromNodeID: "final"}},
	}

	if err := dsl.ValidateWorkflowSpec(spec); err != nil {
		return dsl.WorkflowSpec{}, fmt.Errorf("generated_spec_invalid: %w", err)
	}

	return spec, nil
}


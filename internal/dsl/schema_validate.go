package dsl

import (
	"encoding/json"
	"errors"
	"fmt"
)

func ValidateWorkflowSpecJSON(specJSON []byte) error {
	var spec WorkflowSpec
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return fmt.Errorf("invalid_json: %w", err)
	}

	return ValidateWorkflowSpec(spec)
}

func ValidateWorkflowSpec(spec WorkflowSpec) error {
	if spec.SpecVersion == "" {
		return errors.New("missing_spec_version")
	}
	if spec.GeneratorVersion == "" {
		return errors.New("missing_generator_version")
	}
	if spec.Intent == "" {
		return errors.New("missing_intent")
	}

	switch spec.Mode {
	case ModeQuality, ModeBalanced, ModeCost:
	default:
		return fmt.Errorf("invalid_mode: %s", spec.Mode)
	}

	if len(spec.Nodes) == 0 {
		return errors.New("missing_nodes")
	}

	nodeIDs := map[string]struct{}{}
	for i := range spec.Nodes {
		n := spec.Nodes[i]
		if n.ID == "" {
			return fmt.Errorf("node_missing_id: index=%d", i)
		}
		if _, ok := nodeIDs[n.ID]; ok {
			return fmt.Errorf("duplicate_node_id: %s", n.ID)
		}
		nodeIDs[n.ID] = struct{}{}

		if n.Name == "" {
			return fmt.Errorf("node_missing_name: id=%s", n.ID)
		}
		switch n.Type {
		case NodePlanner, NodeLLMTask, NodeEvaluator, NodeMerge, NodeFinalizer:
		default:
			return fmt.Errorf("invalid_node_type: id=%s type=%s", n.ID, n.Type)
		}

		switch n.RoutingProfile {
		case "", ModeQuality, ModeBalanced, ModeCost:
		default:
			return fmt.Errorf("invalid_routing_profile: id=%s mode=%s", n.ID, n.RoutingProfile)
		}

		if n.RetryPolicy.MaxAttempts < 0 {
			return fmt.Errorf("invalid_retry_policy: id=%s", n.ID)
		}
		for j := range n.FallbackChain {
			fb := n.FallbackChain[j]
			if fb.Model == "" {
				return fmt.Errorf("fallback_missing_model: node=%s index=%d", n.ID, j)
			}
			switch fb.On {
			case "any_error", "validation_fail", "empty_or_too_short":
			default:
				return fmt.Errorf("invalid_fallback_on: node=%s value=%s", n.ID, fb.On)
			}
		}
	}

	for i := range spec.Edges {
		e := spec.Edges[i]
		if e.From == "" || e.To == "" {
			return fmt.Errorf("edge_missing_endpoint: index=%d", i)
		}
		if _, ok := nodeIDs[e.From]; !ok {
			return fmt.Errorf("edge_unknown_from: %s", e.From)
		}
		if _, ok := nodeIDs[e.To]; !ok {
			return fmt.Errorf("edge_unknown_to: %s", e.To)
		}
		switch e.Condition {
		case "always", "on_success", "on_failure":
		default:
			return fmt.Errorf("invalid_edge_condition: %s", e.Condition)
		}
	}

	if len(spec.Outputs) == 0 {
		return errors.New("missing_outputs")
	}
	for i := range spec.Outputs {
		o := spec.Outputs[i]
		if o.Name == "" {
			return fmt.Errorf("output_missing_name: index=%d", i)
		}
		if o.FromNodeID == "" {
			return fmt.Errorf("output_missing_from_node_id: index=%d", i)
		}
		if _, ok := nodeIDs[o.FromNodeID]; !ok {
			return fmt.Errorf("output_unknown_from_node_id: %s", o.FromNodeID)
		}
	}

	return nil
}


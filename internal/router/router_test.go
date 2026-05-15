package router

import (
	"testing"

	"aios/internal/dsl"
)

func TestRouter_ModeAffectsSelection(t *testing.T) {
	r := Router{
		Catalog: Catalog{
			Models: []Model{
				{ID: "premium", QualityScore: 1.0, CostScore: 1.0, LatencyScore: 0.6, ErrorRateScore: 0.1, MaxContextTokens: 200000},
				{ID: "cheap", QualityScore: 0.6, CostScore: 0.1, LatencyScore: 0.4, ErrorRateScore: 0.3, MaxContextTokens: 32000},
				{ID: "balanced", QualityScore: 0.8, CostScore: 0.4, LatencyScore: 0.4, ErrorRateScore: 0.2, MaxContextTokens: 64000},
			},
		},
	}

	node := dsl.Node{
		ID:   "n1",
		Type: dsl.NodeLLMTask,
		Name: "Task",
	}

	selQ, err := r.Select(node, dsl.ModeQuality)
	if err != nil {
		t.Fatal(err)
	}
	if selQ.Primary != "premium" {
		t.Fatalf("expected premium for quality, got %s", selQ.Primary)
	}

	selC, err := r.Select(node, dsl.ModeCost)
	if err != nil {
		t.Fatal(err)
	}
	if selC.Primary != "cheap" {
		t.Fatalf("expected cheap for cost, got %s", selC.Primary)
	}

	selB, err := r.Select(node, dsl.ModeBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if selB.Primary != "balanced" {
		t.Fatalf("expected balanced for balanced, got %s", selB.Primary)
	}
}

func TestRouter_RespectsAllowDenyConstraints(t *testing.T) {
	r := Router{
		Catalog: Catalog{
			Models: []Model{
				{ID: "a", QualityScore: 1.0, CostScore: 1.0, MaxContextTokens: 100},
				{ID: "b", QualityScore: 0.9, CostScore: 0.2, MaxContextTokens: 100},
			},
		},
	}

	node := dsl.Node{
		ID:   "n1",
		Type: dsl.NodeLLMTask,
		Name: "Task",
		ModelConstraints: dsl.ModelConstraints{
			AllowModels: []string{"b"},
			DenyModels:  []string{"a"},
		},
	}

	sel, err := r.Select(node, dsl.ModeQuality)
	if err != nil {
		t.Fatal(err)
	}
	if sel.Primary != "b" {
		t.Fatalf("expected b, got %s", sel.Primary)
	}
}


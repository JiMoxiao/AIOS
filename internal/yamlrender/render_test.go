package yamlrender

import (
	"strings"
	"testing"

	"aios/internal/dsl"
)

func TestRenderYAML_IncludesCoreFields(t *testing.T) {
	spec := dsl.WorkflowSpec{
		SpecVersion:      "1.0",
		GeneratorVersion: "1.0",
		Intent:           "x",
		Mode:             dsl.ModeBalanced,
		Nodes:            []dsl.Node{{ID: "plan", Type: dsl.NodePlanner, Name: "Plan"}},
		Edges:            []dsl.Edge{},
		Outputs:          []dsl.Output{{Name: "final", FromNodeID: "plan"}},
	}
	out, err := RenderWorkflowYAML(spec)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "spec_version:") || !strings.Contains(out, "nodes:") {
		t.Fatalf("unexpected yaml: %s", out)
	}
}


package generator

import (
	"testing"

	"aios/internal/dsl"
)

func TestParseIntent_ArtifactType(t *testing.T) {
	if got := ParseIntent("输出JSON结构化结果").ArtifactType; got != "json" {
		t.Fatalf("expected json, got %s", got)
	}
	if got := ParseIntent("写一份Markdown报告").ArtifactType; got != "markdown" {
		t.Fatalf("expected markdown, got %s", got)
	}
	if got := ParseIntent("总结一下").ArtifactType; got != "text" {
		t.Fatalf("expected text, got %s", got)
	}
}

func TestGenerate_ReturnsValidWorkflowSpec(t *testing.T) {
	spec, err := Generate("写一份Markdown报告", dsl.ModeBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if spec.SpecVersion == "" || spec.GeneratorVersion == "" {
		t.Fatalf("missing versions")
	}
	if len(spec.Nodes) < 4 || len(spec.Edges) < 3 || len(spec.Outputs) != 1 {
		t.Fatalf("unexpected spec structure")
	}
}


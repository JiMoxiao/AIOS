package evaluator

import (
	"testing"

	"aios/internal/dsl"
)

func TestEvaluate_EmptyFails(t *testing.T) {
	node := dsl.Node{ID: "n1", Type: dsl.NodeLLMTask, Name: "Task"}
	res := Evaluate(node, "")
	if res.Pass {
		t.Fatalf("expected fail")
	}
	if len(res.Reasons) == 0 {
		t.Fatalf("expected reasons")
	}
}

func TestEvaluate_JSONSchemaChecksJSONParsing(t *testing.T) {
	node := dsl.Node{
		ID:           "n1",
		Type:         dsl.NodeLLMTask,
		Name:         "Task",
		OutputSchema: map[string]any{"type": "object"},
	}

	ok := Evaluate(node, `{"a":1,"b":2,"c":"xxxxxxxxxxxxxxxxxxxxxxxxxxxx"}`)
	if !ok.Pass {
		t.Fatalf("expected pass")
	}

	bad := Evaluate(node, `{"a":`)
	if bad.Pass {
		t.Fatalf("expected fail")
	}
}


package evaluator

import (
	"strings"

	"aios/internal/dsl"
)

type Constraint func(output string) (ok bool, reason string)

func NonEmpty() Constraint {
	return func(output string) (bool, string) {
		if strings.TrimSpace(output) == "" {
			return false, "empty_output"
		}
		return true, ""
	}
}

func MinLength(n int) Constraint {
	return func(output string) (bool, string) {
		if len([]rune(strings.TrimSpace(output))) < n {
			return false, "too_short"
		}
		return true, ""
	}
}

func MustContain(substr string) Constraint {
	return func(output string) (bool, string) {
		if substr == "" {
			return true, ""
		}
		if !strings.Contains(output, substr) {
			return false, "missing_required_content"
		}
		return true, ""
	}
}

func defaultConstraints(node dsl.Node) []Constraint {
	cs := []Constraint{NonEmpty()}

	switch node.Type {
	case dsl.NodePlanner:
		cs = append(cs, MinLength(50))
	case dsl.NodeLLMTask:
		cs = append(cs, MinLength(30))
	case dsl.NodeEvaluator:
		cs = append(cs, MinLength(10))
	case dsl.NodeFinalizer:
		cs = append(cs, MinLength(30))
	}

	if node.ExpectedArtifactType == "markdown" {
		cs = append(cs, MustContain("#"))
	}

	return cs
}


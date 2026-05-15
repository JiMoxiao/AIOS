package evaluator

import (
	"strings"

	"aios/internal/dsl"
)

type Result struct {
	Pass               bool     `json:"pass"`
	Reasons            []string `json:"reasons,omitempty"`
	RepairInstructions string   `json:"repair_instructions,omitempty"`
}

func Evaluate(node dsl.Node, output string) Result {
	reasons := make([]string, 0, 4)

	for _, c := range defaultConstraints(node) {
		ok, reason := c(output)
		if !ok && reason != "" {
			reasons = append(reasons, reason)
		}
	}

	if node.OutputSchema != nil {
		if err := ValidateJSONAgainstSchema(output, node.OutputSchema); err != nil {
			reasons = append(reasons, "schema_validation_fail")
		}
	}

	if len(reasons) == 0 {
		return Result{Pass: true}
	}

	return Result{
		Pass:               false,
		Reasons:            reasons,
		RepairInstructions: buildRepairInstructions(reasons),
	}
}

func buildRepairInstructions(reasons []string) string {
	if len(reasons) == 0 {
		return ""
	}
	return "修复输出以满足校验原因：" + strings.Join(reasons, ", ") + "。保持原有输出格式不变，仅做必要修正。"
}


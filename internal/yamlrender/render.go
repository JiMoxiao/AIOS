package yamlrender

import (
	"fmt"
	"strings"

	"aios/internal/dsl"
)

func RenderWorkflowYAML(spec dsl.WorkflowSpec) (string, error) {
	var b strings.Builder

	writeKV(&b, "spec_version", spec.SpecVersion)
	writeKV(&b, "generator_version", spec.GeneratorVersion)
	writeKV(&b, "mode", string(spec.Mode))
	writeKV(&b, "intent_summary", summarizeIntent(spec.Intent))

	b.WriteString("nodes:\n")
	for _, n := range spec.Nodes {
		b.WriteString("  - id: ")
		b.WriteString(escapeScalar(n.ID))
		b.WriteString("\n")

		b.WriteString("    type: ")
		b.WriteString(escapeScalar(string(n.Type)))
		b.WriteString("\n")

		b.WriteString("    name: ")
		b.WriteString(escapeScalar(n.Name))
		b.WriteString("\n")

		if n.RoutingProfile != "" {
			b.WriteString("    routing_profile: ")
			b.WriteString(escapeScalar(string(n.RoutingProfile)))
			b.WriteString("\n")
		}

		b.WriteString("    model_reason: ")
		b.WriteString(escapeScalar(modelReason(n)))
		b.WriteString("\n")
	}

	b.WriteString("edges:\n")
	for _, e := range spec.Edges {
		b.WriteString("  - from: ")
		b.WriteString(escapeScalar(e.From))
		b.WriteString("\n")
		b.WriteString("    to: ")
		b.WriteString(escapeScalar(e.To))
		b.WriteString("\n")
		b.WriteString("    condition: ")
		b.WriteString(escapeScalar(e.Condition))
		b.WriteString("\n")
	}

	return b.String(), nil
}

func writeKV(b *strings.Builder, k, v string) {
	b.WriteString(k)
	b.WriteString(": ")
	b.WriteString(escapeScalar(v))
	b.WriteString("\n")
}

func escapeScalar(s string) string {
	if s == "" {
		return `""`
	}
	r := strings.ReplaceAll(s, `\`, `\\`)
	r = strings.ReplaceAll(r, `"`, `\"`)
	r = strings.ReplaceAll(r, "\r\n", "\n")
	r = strings.ReplaceAll(r, "\n", `\n`)
	return `"` + r + `"`
}

func summarizeIntent(intent string) string {
	intent = strings.TrimSpace(intent)
	if intent == "" {
		return ""
	}
	const max = 80
	if len([]rune(intent)) <= max {
		return intent
	}
	rs := []rune(intent)
	return string(rs[:max]) + "…"
}

func modelReason(n dsl.Node) string {
	switch n.Type {
	case dsl.NodePlanner:
		return "高质量拆解与约束提取"
	case dsl.NodeEvaluator:
		return "质量门禁与一致性校验"
	case dsl.NodeFinalizer:
		return "汇总与交付物格式化"
	case dsl.NodeMerge:
		return "多分支产物合并"
	case dsl.NodeLLMTask:
		switch n.RoutingProfile {
		case dsl.ModeQuality:
			return "优先质量与稳定性"
		case dsl.ModeCost:
			return "优先成本与吞吐"
		case dsl.ModeBalanced, "":
			return "质量与成本折中"
		default:
			return fmt.Sprintf("routing_profile=%s", n.RoutingProfile)
		}
	default:
		return fmt.Sprintf("node_type=%s", n.Type)
	}
}


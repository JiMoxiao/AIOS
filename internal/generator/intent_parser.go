package generator

import "strings"

type ParsedIntent struct {
	ArtifactType string
}

func ParseIntent(intent string) ParsedIntent {
	it := strings.ToLower(intent)
	if strings.Contains(it, "json") || strings.Contains(intent, "结构化") {
		return ParsedIntent{ArtifactType: "json"}
	}
	if strings.Contains(intent, "报告") || strings.Contains(intent, "文档") || strings.Contains(intent, "markdown") {
		return ParsedIntent{ArtifactType: "markdown"}
	}
	return ParsedIntent{ArtifactType: "text"}
}


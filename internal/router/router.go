package router

import (
	"errors"
	"sort"

	"aios/internal/dsl"
)

type Router struct {
	Catalog Catalog
}

type Selection struct {
	Primary   string
	Fallbacks []string
}

func (r Router) Select(node dsl.Node, mode dsl.Mode) (Selection, error) {
	profile := mode
	if node.RoutingProfile != "" {
		profile = node.RoutingProfile
	}

	candidates := make([]Model, 0, len(r.Catalog.Models))
	for _, m := range r.Catalog.Models {
		if !allowedByConstraints(m, node.ModelConstraints) {
			continue
		}
		candidates = append(candidates, m)
	}

	if len(candidates) == 0 {
		return Selection{}, errors.New("no_candidate_models")
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return score(candidates[i], profile) > score(candidates[j], profile)
	})

	primary := candidates[0].ID
	fallbacks := selectFallbacks(primary, candidates, node)

	return Selection{Primary: primary, Fallbacks: fallbacks}, nil
}

func allowedByConstraints(m Model, c dsl.ModelConstraints) bool {
	if c.MinContextTokens != nil && m.MaxContextTokens > 0 && m.MaxContextTokens < *c.MinContextTokens {
		return false
	}
	if len(c.AllowModels) > 0 {
		ok := false
		for _, id := range c.AllowModels {
			if id == m.ID {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	for _, id := range c.DenyModels {
		if id == m.ID {
			return false
		}
	}
	return true
}

func score(m Model, mode dsl.Mode) float64 {
	wq, wc, wl, we := weights(mode)
	return wq*m.QualityScore - wc*m.CostScore - wl*m.LatencyScore - we*m.ErrorRateScore
}

func weights(mode dsl.Mode) (wq, wc, wl, we float64) {
	switch mode {
	case dsl.ModeQuality:
		return 1.0, 0.2, 0.2, 0.8
	case dsl.ModeCost:
		return 0.4, 1.0, 0.3, 0.6
	case dsl.ModeBalanced:
		fallthrough
	default:
		return 0.8, 0.6, 0.4, 0.7
	}
}

func selectFallbacks(primary string, candidates []Model, node dsl.Node) []string {
	fbs := make([]string, 0, 3)

	seen := map[string]struct{}{primary: {}}
	add := func(id string) {
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		fbs = append(fbs, id)
	}

	for _, rule := range node.FallbackChain {
		add(rule.Model)
	}

	if len(fbs) > 0 {
		return fbs
	}

	for i := 1; i < len(candidates) && len(fbs) < 3; i++ {
		add(candidates[i].ID)
	}

	return fbs
}


package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"aios/internal/dsl"
	"aios/internal/evaluator"
	"aios/internal/providers"
	"aios/internal/router"
	"aios/internal/store"
	"aios/internal/telemetry"
)

type Executor struct {
	Store    *store.SQLiteStore
	Provider providers.Provider
	Router   router.Router
}

func (e *Executor) RunWorkflow(ctx context.Context, workflowID string) (string, error) {
	if e.Store == nil {
		return "", errors.New("missing_store")
	}
	if e.Provider == nil {
		return "", errors.New("missing_provider")
	}

	specJSON, _, specHash, err := e.Store.GetWorkflow(ctx, workflowID)
	if err != nil {
		return "", err
	}

	var spec dsl.WorkflowSpec
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return "", err
	}
	if err := dsl.ValidateWorkflowSpec(spec); err != nil {
		return "", err
	}

	runID, err := e.Store.CreateRun(ctx, workflowID, specHash)
	if err != nil {
		return "", err
	}

	started := time.Now()

	rec, err := e.Store.GetRun(ctx, runID)
	if err != nil {
		return "", err
	}

	outputs := map[string]string{}
	nodeByID := map[string]dsl.Node{}
	for _, n := range spec.Nodes {
		nodeByID[n.ID] = n
		upsertNodeRun(&rec, dsl.NodeRun{NodeID: n.ID, Status: dsl.NodePending})
	}
	if err := e.Store.UpdateRun(ctx, rec); err != nil {
		return "", err
	}

	order, err := topoOrder(spec.Nodes, spec.Edges)
	if err != nil {
		rec.Status = dsl.RunFailed
		_ = e.Store.UpdateRun(ctx, rec)
		return "", err
	}

	for _, nodeID := range order {
		n := nodeByID[nodeID]

		nr := getNodeRun(rec, nodeID)
		nr.Status = dsl.NodeRunning
		upsertNodeRun(&rec, nr)
		if err := e.Store.UpdateRun(ctx, rec); err != nil {
			return "", err
		}

		var out string
		switch n.Type {
		case dsl.NodePlanner, dsl.NodeLLMTask, dsl.NodeFinalizer, dsl.NodeMerge:
			out, nr, err = e.runLLMNode(ctx, spec, n, outputs, nr)
		case dsl.NodeEvaluator:
			out, nr, err = e.runEvaluatorNode(ctx, spec, n, outputs, nr)
		default:
			err = fmt.Errorf("unsupported_node_type: %s", n.Type)
		}

		if err != nil {
			nr.Status = dsl.NodeFailed
			msg := err.Error()
			nr.Error = &msg
			upsertNodeRun(&rec, nr)
			rec.Status = dsl.RunFailed
			finished := nowRFC3339()
			rec.FinishedAt = &finished
			telemetry.RecordNodeRun(string(n.Type), string(nr.Status), nr.Model, time.Duration(nr.LatencyMs)*time.Millisecond)
			telemetry.RecordRun(string(rec.Status), time.Since(started))
			_ = e.Store.UpdateRun(ctx, rec)
			return runID, err
		}

		if out != "" {
			outputs[nodeID] = out
			kind := n.ExpectedArtifactType
			if kind == "" {
				kind = "text"
			}
			art, err := e.Store.SaveArtifact(ctx, nodeID, kind, out)
			if err != nil {
				return runID, err
			}
			rec.Artifacts = append(rec.Artifacts, art)
			if nodeID == finalOutputNode(spec) {
				rec.FinalOutputRef = art
			}
		}

		nr.Status = dsl.NodeSucceeded
		upsertNodeRun(&rec, nr)
		telemetry.RecordNodeRun(string(n.Type), string(nr.Status), nr.Model, time.Duration(nr.LatencyMs)*time.Millisecond)
		if err := e.Store.UpdateRun(ctx, rec); err != nil {
			return runID, err
		}
	}

	rec.Status = dsl.RunSucceeded
	finished := nowRFC3339()
	rec.FinishedAt = &finished
	telemetry.RecordRun(string(rec.Status), time.Since(started))
	if err := e.Store.UpdateRun(ctx, rec); err != nil {
		return runID, err
	}

	return runID, nil
}

func (e *Executor) runLLMNode(ctx context.Context, spec dsl.WorkflowSpec, node dsl.Node, outputs map[string]string, nr dsl.NodeRun) (string, dsl.NodeRun, error) {
	prompt := buildPrompt(spec, node, outputs)
	messages := []providers.ChatMessage{
		{Role: "user", Content: prompt},
	}

	sel, err := e.Router.Select(node, spec.Mode)
	if err != nil {
		return "", nr, err
	}

	models := append([]string{sel.Primary}, sel.Fallbacks...)
	attempts := node.RetryPolicy.MaxAttempts
	if attempts <= 0 {
		attempts = 1
	}

	var lastErr error
	for _, model := range models {
		for i := 0; i < attempts; i++ {
			resp, err := e.Provider.Chat(ctx, providers.ChatRequest{
				Model:    model,
				Messages: messages,
			})

			nr.Model = model
			nr.TokenIn = resp.TokenIn
			nr.TokenOut = resp.TokenOut
			nr.LatencyMs = resp.LatencyMs

			if err != nil {
				lastErr = err
				nr.Fallbacks = append(nr.Fallbacks, dsl.FallbackAttempt{Model: model, Reason: "any_error"})
				telemetry.RecordFallback(model, "any_error")
				continue
			}

			content := strings.TrimSpace(resp.Content)
			if content == "" {
				lastErr = errors.New("empty_response")
				nr.Fallbacks = append(nr.Fallbacks, dsl.FallbackAttempt{Model: model, Reason: "empty_or_too_short"})
				telemetry.RecordFallback(model, "empty_or_too_short")
				continue
			}

			res := evaluator.Evaluate(node, content)
			if !res.Pass {
				lastErr = errors.New("validation_fail")
				nr.Fallbacks = append(nr.Fallbacks, dsl.FallbackAttempt{Model: model, Reason: "validation_fail"})
				telemetry.RecordFallback(model, "validation_fail")
				messages = []providers.ChatMessage{
					{Role: "user", Content: res.RepairInstructions + "\n\n" + prompt},
				}
				continue
			}

			return content, nr, nil
		}
	}

	if lastErr == nil {
		lastErr = errors.New("all_models_failed")
	}
	return "", nr, lastErr
}

func (e *Executor) runEvaluatorNode(ctx context.Context, spec dsl.WorkflowSpec, node dsl.Node, outputs map[string]string, nr dsl.NodeRun) (string, dsl.NodeRun, error) {
	_ = ctx
	up, ok := firstNodeOutputInput(node)
	if !ok {
		return "", nr, nil
	}
	out, ok := outputs[up]
	if !ok {
		return "", nr, errors.New("missing_upstream_output")
	}
	res := evaluator.Evaluate(node, out)
	if !res.Pass {
		return "", nr, errors.New("validation_fail")
	}
	return `{"pass":true}`, nr, nil
}

func buildPrompt(spec dsl.WorkflowSpec, node dsl.Node, outputs map[string]string) string {
	switch node.Type {
	case dsl.NodePlanner:
		return "请将下面目标拆解为可执行步骤，并列出交付物与约束：\n\n" + spec.Intent
	case dsl.NodeFinalizer:
		up, _ := firstNodeOutputInput(node)
		return "请将下面内容整理为最终交付物，并补充执行说明：\n\n" + outputs[up]
	default:
		up, _ := firstNodeOutputInput(node)
		if up != "" {
			return "请根据上游输出完成任务：\n\n" + outputs[up]
		}
		return spec.Intent
	}
}

func firstNodeOutputInput(node dsl.Node) (string, bool) {
	for _, in := range node.Inputs {
		if in.Type == "node_output" && in.NodeID != "" {
			return in.NodeID, true
		}
	}
	return "", false
}

func finalOutputNode(spec dsl.WorkflowSpec) string {
	if len(spec.Outputs) == 0 {
		return ""
	}
	return spec.Outputs[0].FromNodeID
}

func getNodeRun(rec dsl.RunRecord, nodeID string) dsl.NodeRun {
	for _, nr := range rec.NodeRuns {
		if nr.NodeID == nodeID {
			return nr
		}
	}
	return dsl.NodeRun{NodeID: nodeID}
}

func topoOrder(nodes []dsl.Node, edges []dsl.Edge) ([]string, error) {
	nodeIDs := make([]string, 0, len(nodes))
	inDeg := map[string]int{}
	next := map[string][]string{}
	for _, n := range nodes {
		nodeIDs = append(nodeIDs, n.ID)
		inDeg[n.ID] = 0
	}
	for _, e := range edges {
		if e.Condition != "always" && e.Condition != "on_success" {
			continue
		}
		next[e.From] = append(next[e.From], e.To)
		inDeg[e.To]++
	}

	q := make([]string, 0, len(nodes))
	for _, id := range nodeIDs {
		if inDeg[id] == 0 {
			q = append(q, id)
		}
	}

	out := make([]string, 0, len(nodes))
	for len(q) > 0 {
		id := q[0]
		q = q[1:]
		out = append(out, id)

		for _, to := range next[id] {
			inDeg[to]--
			if inDeg[to] == 0 {
				q = append(q, to)
			}
		}
	}

	if len(out) != len(nodes) {
		return nil, errors.New("dag_cycle_or_disconnected")
	}
	return out, nil
}


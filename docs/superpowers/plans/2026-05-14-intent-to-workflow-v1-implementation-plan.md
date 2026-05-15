# Intent-to-Workflow（Dynamic AI Workflow Generator）V1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 交付一个 LLM-only 的 Intent-to-Workflow 系统：输入目标 → 自动生成 DAG 工作流（JSON 权威 + YAML 导出）→ 执行工作流（并行/重试/回退）→ 输出结果与 RunRecord（可解释/可复现）。

**Architecture:** 采用“生成器（Generator）+ 运行时（Executor）+ 路由（Router）+ 评估（Evaluator）+ 存储（Store）”分层。执行引擎只接受 JSON 权威格式；YAML 由 JSON 渲染生成并只读展示。运行过程产出 RunRecord（节点级状态、模型选择、成本、回退原因、产物指针）用于回放与审计。

**Tech Stack:** Go（单体起步，模块化拆分）+ REST API（Generate/Run/Query）+ JSON Schema 校验 + YAML 渲染；存储起步用 SQLite（便于本地与单机），支持后续切换 Postgres；日志结构化 + OpenTelemetry trace + Prometheus metrics。

---

## 0) 工作包拆解（从规格到实现）

本计划实现的规格为：

- [2026-05-14-intent-to-workflow-dynamic-generator-design.md](file:///d:/Myworkspace/AIOS/docs/superpowers/specs/2026-05-14-intent-to-workflow-dynamic-generator-design.md)

按 8 个工作包推进：

1) 工程骨架与本地可运行（HTTP 服务、配置、日志、测试框架）  
2) DSL 与 Schema（WorkflowSpec JSON + JSON Schema + YAML 导出）  
3) 运行记录与存储（RunRecord、Artifact、最小保留策略）  
4) Model Router（mode/约束/评分 + fallback chain）  
5) Workflow Generator（Intent → DAG 的最小闭环：planner + task + evaluator + finalizer）  
6) Workflow Executor（DAG 调度：依赖、有限并行、重试、回退、checkpoint）  
7) API 与演示（Generate/Run/Status/Export）  
8) DoD 验收脚本与回归用例集（最小可持续迭代）  

---

## Repo File Structure（锁定文件结构，后续任务按此实施）

**Files/Dirs:**
- Create: `AIOS/cmd/itw-server/main.go`
- Create: `AIOS/internal/config/`（配置与默认值）
- Create: `AIOS/internal/http/`（路由与 handler）
- Create: `AIOS/internal/dsl/`（WorkflowSpec/RunRecord 类型 + schema）
- Create: `AIOS/internal/yamlrender/`（YAML 导出）
- Create: `AIOS/internal/store/`（SQLite store：workflow/run/artifact）
- Create: `AIOS/internal/router/`（模型路由与评分）
- Create: `AIOS/internal/generator/`（意图解析与 DAG 生成）
- Create: `AIOS/internal/executor/`（DAG 执行与状态机）
- Create: `AIOS/internal/providers/`（LLM provider：mock + http-openai-compatible）
- Create: `AIOS/internal/evaluator/`（schema/约束校验）
- Create: `AIOS/scripts/e2e-smoke.ps1`
- Create: `AIOS/docs/acceptance-itw-v1.md`

---

## Task 1: 工程骨架与测试框架（Boot）

**Files:**
- Create: `AIOS/go.mod`
- Create: `AIOS/cmd/itw-server/main.go`
- Create: `AIOS/internal/config/config.go`
- Create: `AIOS/internal/http/server.go`
- Create: `AIOS/internal/http/routes.go`
- Test: `AIOS/internal/http/server_test.go`

- [ ] **Step 1: 初始化 Go module**
  - 目录：`AIOS/`
  - 目标：`go test ./...` 可运行（即使只有空测试）

- [ ] **Step 2: 写一个最小 HTTP server（healthz）与测试**

```go
// AIOS/internal/http/server_test.go
package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	srv := NewServer(ServerConfig{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
```

- [ ] **Step 3: 实现最小代码让测试通过**
  - `GET /healthz` 返回 200 与简单 JSON

- [ ] **Step 4: Commit**
  - `chore: bootstrap itw server with healthz`

---

## Task 2: DSL（WorkflowSpec）与 JSON Schema 校验

**Files:**
- Create: `AIOS/internal/dsl/workflow_spec.go`
- Create: `AIOS/internal/dsl/workflow_spec_schema.json`
- Create: `AIOS/internal/dsl/schema_validate.go`
- Test: `AIOS/internal/dsl/schema_validate_test.go`

- [ ] **Step 1: 写 DSL 类型（最小字段集）**

```go
// AIOS/internal/dsl/workflow_spec.go
package dsl

type Mode string

const (
	ModeQuality  Mode = "quality"
	ModeBalanced Mode = "balanced"
	ModeCost     Mode = "cost"
)

type DataClassification string

const (
	DataPublic       DataClassification = "public"
	DataInternal     DataClassification = "internal"
	DataConfidential DataClassification = "confidential"
	DataRestricted   DataClassification = "restricted"
)

type WorkflowSpec struct {
	SpecVersion      string            `json:"spec_version"`
	GeneratorVersion string            `json:"generator_version"`
	Intent           string            `json:"intent"`
	Mode             Mode              `json:"mode"`
	GlobalConstraints GlobalConstraints `json:"global_constraints,omitempty"`
	Nodes            []Node            `json:"nodes"`
	Edges            []Edge            `json:"edges"`
	Outputs          []Output          `json:"outputs"`
}

type GlobalConstraints struct {
	DataClassification DataClassification `json:"data_classification"`
	MaxTotalCostUSD    *float64           `json:"max_total_cost_usd,omitempty"`
	MaxWallTimeSec     *int               `json:"max_wall_time_sec,omitempty"`
}

type NodeType string

const (
	NodePlanner   NodeType = "planner"
	NodeLLMTask   NodeType = "llm_task"
	NodeEvaluator NodeType = "evaluator"
	NodeMerge     NodeType = "merge"
	NodeFinalizer NodeType = "finalizer"
)

type Node struct {
	ID                string        `json:"id"`
	Type              NodeType      `json:"type"`
	Name              string        `json:"name"`
	Description       string        `json:"description,omitempty"`
	Inputs            []InputRef    `json:"inputs,omitempty"`
	ExpectedArtifactType string     `json:"expected_artifact_type,omitempty"`
	OutputSchema      any           `json:"output_schema,omitempty"`
	RoutingProfile    Mode          `json:"routing_profile,omitempty"`
	ModelConstraints  ModelConstraints `json:"model_constraints,omitempty"`
	RetryPolicy       RetryPolicy   `json:"retry_policy,omitempty"`
	FallbackChain     []FallbackRule `json:"fallback_chain,omitempty"`
}

type InputRef struct {
	Type   string `json:"type"` // "intent" | "node_output"
	NodeID string `json:"node_id,omitempty"`
}

type ModelConstraints struct {
	AllowModels        []string `json:"allow_models,omitempty"`
	DenyModels         []string `json:"deny_models,omitempty"`
	MinContextTokens   *int     `json:"min_context_tokens,omitempty"`
}

type RetryPolicy struct {
	MaxAttempts int `json:"max_attempts,omitempty"`
}

type FallbackRule struct {
	Model string `json:"model"`
	On    string `json:"on"` // "any_error" | "validation_fail" | "empty_or_too_short"
}

type Edge struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Condition string `json:"condition"` // "always" | "on_success" | "on_failure"
}

type Output struct {
	Name       string `json:"name"`
	FromNodeID string `json:"from_node_id"`
}
```

- [ ] **Step 2: 写 JSON Schema（workflow_spec_schema.json）**
  - 最小要求：
    - spec_version、generator_version、intent、mode、nodes、edges、outputs 必填
    - node.id 唯一（schema 不易表达时，用额外校验补齐）
    - edge.from/edge.to 必须引用已存在 node
    - condition 只能取三值

- [ ] **Step 3: 写失败测试（schema 校验）**

```go
// AIOS/internal/dsl/schema_validate_test.go
package dsl

import "testing"

func TestValidateWorkflowSpec_MissingRequiredField(t *testing.T) {
	spec := []byte(`{"spec_version":"1.0"}`)
	if err := ValidateWorkflowSpecJSON(spec); err == nil {
		t.Fatalf("expected validation error")
	}
}
```

- [ ] **Step 4: 实现 ValidateWorkflowSpecJSON 让测试通过**
  - 允许使用标准库 + 代码库中已引入的 JSON schema 校验库（若无则先实现“最小手写校验”并在 Task 3 补库）

- [ ] **Step 5: Commit**
  - `feat: add workflow spec dsl and schema validation`

---

## Task 3: YAML 导出（JSON → YAML，只读）

**Files:**
- Create: `AIOS/internal/yamlrender/render.go`
- Test: `AIOS/internal/yamlrender/render_test.go`

- [ ] **Step 1: 写渲染测试（snapshot）**
  - 输入：最小 WorkflowSpec
  - 断言：包含 spec_version、nodes、edges

```go
package yamlrender

import (
	"strings"
	"testing"

	"AIOS/internal/dsl"
)

func TestRenderYAML_IncludesCoreFields(t *testing.T) {
	spec := dsl.WorkflowSpec{
		SpecVersion: "1.0",
		GeneratorVersion: "1.0",
		Intent: "x",
		Mode: dsl.ModeBalanced,
		Nodes: []dsl.Node{{ID: "plan", Type: dsl.NodePlanner, Name: "Plan"}},
		Edges: []dsl.Edge{},
		Outputs: []dsl.Output{{Name: "final", FromNodeID: "plan"}},
	}
	out, err := RenderWorkflowYAML(spec)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "spec_version:") || !strings.Contains(out, "nodes:") {
		t.Fatalf("unexpected yaml: %s", out)
	}
}
```

- [ ] **Step 2: 实现 RenderWorkflowYAML**
  - YAML 结构遵循规格 5.3：只读导出，不输出敏感原文（V1 先输出 intent_summary 而不是 intent）

- [ ] **Step 3: Commit**
  - `feat: add yaml export for workflow spec`

---

## Task 4: RunRecord + Artifact + Store（SQLite）

**Files:**
- Create: `AIOS/internal/dsl/run_record.go`
- Create: `AIOS/internal/store/sqlite_store.go`
- Create: `AIOS/internal/store/migrations/001_init.sql`
- Test: `AIOS/internal/store/sqlite_store_test.go`

- [ ] **Step 1: 定义 RunRecord（最小字段集）**

```go
package dsl

type RunStatus string

const (
	RunRunning RunStatus = "running"
	RunSucceeded RunStatus = "succeeded"
	RunFailed RunStatus = "failed"
)

type NodeStatus string

const (
	NodePending   NodeStatus = "pending"
	NodeRunning   NodeStatus = "running"
	NodeSucceeded NodeStatus = "succeeded"
	NodeFailed    NodeStatus = "failed"
	NodeSkipped   NodeStatus = "skipped"
)

type RunRecord struct {
	RunID      string    `json:"run_id"`
	WorkflowID string    `json:"workflow_id"`
	SpecHash   string    `json:"spec_hash"`
	Status     RunStatus `json:"status"`
	StartedAt  string    `json:"started_at"`
	FinishedAt *string   `json:"finished_at,omitempty"`
	NodeRuns   []NodeRun `json:"node_runs"`
	Artifacts  []ArtifactRef `json:"artifacts"`
	FinalOutputRef ArtifactRef `json:"final_output_ref"`
}

type NodeRun struct {
	NodeID     string     `json:"node_id"`
	Status     NodeStatus `json:"status"`
	Model      string     `json:"model,omitempty"`
	TokenIn    int        `json:"token_in,omitempty"`
	TokenOut   int        `json:"token_out,omitempty"`
	CostUSD    float64    `json:"cost_usd,omitempty"`
	LatencyMs  int        `json:"latency_ms,omitempty"`
	Fallbacks  []FallbackAttempt `json:"fallbacks,omitempty"`
	Error      *string    `json:"error,omitempty"`
}

type FallbackAttempt struct {
	Model string `json:"model"`
	Reason string `json:"reason"`
}

type ArtifactRef struct {
	ArtifactID string `json:"artifact_id"`
	NodeID     string `json:"node_id"`
	Kind       string `json:"kind"` // "text" | "json" | "markdown"
	Summary    string `json:"summary,omitempty"`
	Digest     string `json:"digest,omitempty"`
}
```

- [ ] **Step 2: 写 store 测试（写入/查询）**
  - SaveWorkflowSpecJSON + SaveRenderedYAML
  - CreateRun + UpdateNodeRun + FinishRun
  - QueryRun(run_id)

- [ ] **Step 3: 实现 SQLite schema（migrations/001_init.sql）与 store**
  - 表：workflows（spec_json,yaml,hash）、runs（run_json）、artifacts（body/summary/digest）
  - V1 默认仅保存 summary + digest；body 保存通过配置开关控制

- [ ] **Step 4: Commit**
  - `feat: add run record and sqlite store`

---

## Task 5: Providers（LLM 调用：Mock + OpenAI-compatible HTTP）

**Files:**
- Create: `AIOS/internal/providers/provider.go`
- Create: `AIOS/internal/providers/mock/mock_provider.go`
- Create: `AIOS/internal/providers/openai/http_provider.go`
- Test: `AIOS/internal/providers/*_test.go`

- [ ] **Step 1: 定义 Provider 接口**

```go
package providers

import "context"

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string       `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

type ChatResponse struct {
	Content  string `json:"content"`
	TokenIn  int    `json:"token_in"`
	TokenOut int    `json:"token_out"`
	LatencyMs int   `json:"latency_ms"`
}

type Provider interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
```

- [ ] **Step 2: MockProvider**
  - 根据 prompt 返回固定内容，支持注入“超时/错误/空响应”场景

- [ ] **Step 3: HTTP OpenAI-compatible Provider**
  - 对接任意 OpenAI 风格 `/chat/completions` endpoint（V1 不做 stream）
  - 需要单测：5xx、429、超时

- [ ] **Step 4: Commit**
  - `feat: add llm providers (mock, http openai compatible)`

---

## Task 6: Router（mode + 约束 + 评分 + fallback）

**Files:**
- Create: `AIOS/internal/router/router.go`
- Create: `AIOS/internal/router/catalog.go`
- Test: `AIOS/internal/router/router_test.go`

- [ ] **Step 1: 写 router 测试（mode 影响选择）**
  - 给定模型目录：高质量贵、便宜一般、长上下文慢
  - 断言：quality 模式选高质量；cost 模式选便宜；balanced 选折中

- [ ] **Step 2: 实现模型目录（静态配置起步）**
  - 先放到 config（JSON/YAML），后续再接控制面

- [ ] **Step 3: 实现评分函数**
  - `score = wq*quality - wc*cost - wl*latency - we*error_rate`
  - V1 的 quality/latency/error_rate 可来自静态画像，后续再用运行统计回填

- [ ] **Step 4: fallback chain 选择**
  - 触发条件：any_error / validation_fail / empty_or_too_short

- [ ] **Step 5: Commit**
  - `feat: add model router with mode-based scoring and fallbacks`

---

## Task 7: Evaluator（schema + 约束）

**Files:**
- Create: `AIOS/internal/evaluator/evaluator.go`
- Create: `AIOS/internal/evaluator/schema.go`
- Create: `AIOS/internal/evaluator/constraints.go`
- Test: `AIOS/internal/evaluator/*_test.go`

- [ ] **Step 1: 约束接口与测试**
  - 约束：非空、最小长度、必须包含关键标题（如“接口列表”）

- [ ] **Step 2: schema 校验（结构化节点）**
  - 若 Node.OutputSchema 存在：尝试把输出解析为 JSON 并校验

- [ ] **Step 3: 输出 evaluator 结果**
  - pass/fail + reasons + repair_instructions（V1 先生成固定模板）

- [ ] **Step 4: Commit**
  - `feat: add evaluator for schema and constraints`

---

## Task 8: Generator（Intent → DAG，最小闭环）

**Files:**
- Create: `AIOS/internal/generator/intent_parser.go`
- Create: `AIOS/internal/generator/generator.go`
- Test: `AIOS/internal/generator/generator_test.go`

- [ ] **Step 1: 意图解析（规则起步）**
  - 从输入提取：目标类型（写作/总结/分析/生成清单）、交付物（markdown/json）、是否需要评审
  - V1 用规则/关键词，不依赖 LLM 自举（避免“先要调用模型才能生成工作流”的鸡生蛋）

- [ ] **Step 2: 生成最小 DAG**
  - 固定模板：
    - planner → llm_task → evaluator → finalizer
  - 根据 intent：可插入多个 llm_task（例如“先总结再生成大纲再写正文”）

- [ ] **Step 3: 生成 fallback_chain**
  - planner 默认 quality
  - task 默认 balanced
  - evaluator 默认 quality

- [ ] **Step 4: Commit**
  - `feat: add intent parser and workflow generator`

---

## Task 9: Executor（DAG 调度 + 重试 + 回退 + checkpoint）

**Files:**
- Create: `AIOS/internal/executor/executor.go`
- Create: `AIOS/internal/executor/state_machine.go`
- Test: `AIOS/internal/executor/executor_test.go`

- [ ] **Step 1: 写失败测试（fallback 生效）**
  - mock provider 让第一个模型返回空响应
  - 断言：切换到 fallback 模型并成功

- [ ] **Step 2: 实现拓扑执行**
  - 维护 node 状态与依赖计数
  - 有限并行：用 semaphore 控制并发

- [ ] **Step 3: 接入 evaluator 与 repair_instructions**
  - validation_fail 时：重试或换模型
  - repair_instructions 存在时：对 prompt 添加修复前缀再试一次

- [ ] **Step 4: checkpoint**
  - 每节点完成后：保存 artifact（summary+digest）+ 更新 node_run

- [ ] **Step 5: Commit**
  - `feat: add dag executor with retries, fallbacks, and checkpoints`

---

## Task 10: HTTP API（Generate/Run/Status/Export）

**Files:**
- Modify: `AIOS/internal/http/routes.go`
- Create: `AIOS/internal/http/handlers_generate.go`
- Create: `AIOS/internal/http/handlers_run.go`
- Create: `AIOS/internal/http/handlers_query.go`
- Test: `AIOS/internal/http/e2e_test.go`

- [ ] **Step 1: 定义 API**
  - `POST /v1/generate`：输入 intent + mode → 输出 workflow_spec_json + workflow_yaml + workflow_id
  - `POST /v1/run`：输入 workflow_id（或直接 spec_json）→ 输出 run_id
  - `GET /v1/runs/{run_id}`：查询 RunRecord
  - `GET /v1/workflows/{workflow_id}/yaml`：下载 YAML

- [ ] **Step 2: 写 e2e 测试（使用 httptest）**
  - 生成 → 运行 → 查询 run

- [ ] **Step 3: 实现 handler**
  - generate：调用 generator → schema validate → store → render yaml → store
  - run：加载 workflow → executor 执行（同步或后台；V1 可同步执行并返回 run_id+最终状态）

- [ ] **Step 4: Commit**
  - `feat: add generate and run http apis`

---

## Task 11: Telemetry（metrics + trace）与最小运维文档

**Files:**
- Create: `AIOS/internal/telemetry/telemetry.go`
- Modify: `AIOS/cmd/itw-server/main.go`
- Create: `AIOS/README.md`
- Create: `AIOS/docs/acceptance-itw-v1.md`

- [ ] **Step 1: metrics**
  - 计数：runs_total（success/fail）、node_runs_total（by type/status/model）、fallback_total
  - 分布：run_duration_ms、node_latency_ms、run_cost_usd

- [ ] **Step 2: traces**
  - root span：run_id
  - child span：node_id + model

- [ ] **Step 3: 文档**
  - README：本地启动、配置 provider、运行示例
  - acceptance：DoD 检查清单

- [ ] **Step 4: Commit**
  - `docs: add telemetry and acceptance docs`

---

## Task 12: E2E 冒烟脚本与回归用例集（DoD）

**Files:**
- Create: `AIOS/scripts/e2e-smoke.ps1`
- Create: `AIOS/scripts/demo-intents/`
- Modify: `AIOS/docs/acceptance-itw-v1.md`

- [ ] **Step 1: 写冒烟脚本**
  - 启动 server（或假设已启动）
  - 调用 generate
  - 调用 run
  - 拉取 run record 并断言状态 succeeded

- [ ] **Step 2: 回归 intents**
  - `write_report.txt`
  - `summarize_long_text.txt`
  - `produce_json_spec.txt`
  - `review_and_fix.txt`

- [ ] **Step 3: Commit**
  - `test: add e2e smoke and regression intents`

---

## Plan Self-Review（对规格覆盖）

- 双表示（JSON 权威 + YAML 导出）：Task 2/3  
- Intent → Workflow：Task 8/10  
- Workflow → Run：Task 9/10  
- 节点级路由与回退：Task 6/9  
- 质量评估：Task 7/9  
- RunRecord 与可复现：Task 4/10/12  
- 可解释（选择理由、回退原因、成本）：Task 4/6/9/10  


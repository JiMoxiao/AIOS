# Intent-to-Workflow V1.1 (LLM Workflow Generator) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在现有 V1（规则生成 + 可执行 DAG）基础上，引入 LLM 工作流生成器，支持“生成→校验→自动修复→兜底回退”，并提供可复现的生成元信息。

**Architecture:** 在 `internal/generator` 下新增 LLM 生成实现（调用现有 providers），通过严格 JSON/Schema/语义校验与 repair loop 保证稳定；Generate API 新增可选参数并返回 generator_meta；失败时回落到现有规则生成器。

**Tech Stack:** Go、net/http、现有 providers/mock、现有 dsl.ValidateWorkflowSpec、现有 store(SQLite)、现有 yamlrender

---

## 0. Codebase Map (files to touch)

**Existing (likely):**
- `internal/generator/generator.go` (规则生成器入口)
- `internal/generator/intent_parser.go` (简单意图解析)
- `internal/dsl/*` (WorkflowSpec 结构、schema 校验、语义校验)
- `internal/providers/*` (Mock + OpenAI 兼容 HTTP Provider)
- `internal/http/*` (Generate/Run API handlers)
- `internal/store/*` (SQLite 记录 workflow/run/artifact)

**Create:**
- `internal/generator/llm_generator.go`
- `internal/generator/llm_prompt.go`
- `internal/generator/repair.go`
- `internal/generator/validation_errors.go`
- `internal/generator/llm_generator_test.go`

**Modify:**
- `internal/generator/generator.go`
- `internal/http/handlers_generate.go` (或等价文件：负责 /generate)
- `internal/store/sqlite_store.go`（或等价：记录 generator_meta）
- `internal/store/migrations/002_generator_meta.sql`（新增 migration）
- `internal/dsl/workflow_spec.go`（如需新增 GeneratorMeta 类型）

---

### Task 1: 明确 Generate API 与代码落点（定位现有 handler）

**Files:**
- Read: `internal/http/routes.go`
- Read: `internal/http/*`

- [ ] **Step 1: 查找 /generate handler 与请求/响应结构**
  
Run:
```powershell
cd d:\Myworkspace\AIOS
rg -n "generate" internal/http
rg -n "Generate\\(" internal/http internal/generator
```

Expected:
- 找到生成接口的 handler 文件（本文后续用 `handlers_generate.go` 指代）

- [ ] **Step 2: 记录 handler 中的 request/response struct 名称**

将实际 struct 名称补到后续 Task（不要改行为）。

- [ ] **Step 3: Commit（仅备注/不改代码可跳过提交）**

如果无代码改动，跳过。

---

### Task 2: 增加 generator_strategy 参数与 generator_meta 响应（先写测试）

**Files:**
- Modify: `internal/http/handlers_generate.go`
- Test: `internal/http/handlers_generate_test.go`（若无则创建）

- [ ] **Step 1: 写一个 failing test：默认策略返回 generator_meta**

示例（按实际 router/handler 写法调整）：
```go
func TestGenerate_DefaultStrategy_ReturnsGeneratorMeta(t *testing.T) {
	// Arrange: 启动 test server / handler，注入 mock provider + generator
	// Act: 调用 /generate（不传 generator_strategy）
	// Assert:
	// - HTTP 200
	// - response.generator_meta.strategy_used != ""
	// - response.workflow_spec.spec_version == "1.0"
}
```

- [ ] **Step 2: 运行测试确认失败**

Run:
```powershell
.\.tools\go\bin\go.exe test ./... -run TestGenerate_DefaultStrategy_ReturnsGeneratorMeta
```

Expected:
- FAIL（缺 generator_meta 字段或为空）

- [ ] **Step 3: 最小实现 generator_strategy & generator_meta（先占位）**

行为要求：
- query/body 增加可选字段 `generator_strategy`（`llm|rule`）
- response 增加 `generator_meta`（含 `strategy_used`、`repair_attempts`、`validation_errors` 摘要）
- 先用规则生成器也要填 meta：`strategy_used="rule"`

- [ ] **Step 4: 跑测试确认通过**

Run:
```powershell
.\.tools\go\bin\go.exe test ./... -run TestGenerate_DefaultStrategy_ReturnsGeneratorMeta
```

Expected: PASS

- [ ] **Step 5: Commit**

```powershell
git add internal/http
git commit -m "feat(api): add generator_strategy and generator_meta to generate response"
```

---

### Task 3: 定义 validation error digest（结构化错误列表）

**Files:**
- Create: `internal/generator/validation_errors.go`
- Test: `internal/generator/validation_errors_test.go`

- [ ] **Step 1: 写 failing test：将 schema/语义错误转成稳定结构**

```go
func TestValidationErrorDigest_Stable(t *testing.T) {
	errs := []error{
		errors.New("schema: nodes[0].id is required"),
		errors.New("semantic: edge references missing node"),
	}
	d := DigestValidationErrors(errs)
	require.NotEmpty(t, d)
	require.LessOrEqual(t, len(d), 10)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run:
```powershell
.\.tools\go\bin\go.exe test ./... -run TestValidationErrorDigest_Stable
```

Expected: FAIL（函数不存在）

- [ ] **Step 3: 实现 DigestValidationErrors**

实现要求：
- 输出 `[]ValidationError`，字段：`code`、`message`、`path`（path 可空）
- 做去重、数量上限（默认 10）
- 不包含用户原文（仅结构路径与消息）

- [ ] **Step 4: 测试通过**

- [ ] **Step 5: Commit**

```powershell
git add internal/generator
git commit -m "feat(generator): add structured validation error digest"
```

---

### Task 4: 新增 LLM Generator（最小可用，先不做 repair）

**Files:**
- Create: `internal/generator/llm_generator.go`
- Create: `internal/generator/llm_prompt.go`
- Test: `internal/generator/llm_generator_test.go`

- [ ] **Step 1: 写 failing test：LLM 返回合法 JSON spec 时生成成功**

使用 mock provider，返回一段合法 WorkflowSpec JSON（尽量最小）：
```go
func TestLLMGenerator_GeneratesValidSpec(t *testing.T) {
	// Arrange: mock provider returns a valid JSON object spec
	// Act: gen.Generate(intent, mode)
	// Assert: ValidateWorkflowSpec passes
}
```

- [ ] **Step 2: 运行测试确认失败**

- [ ] **Step 3: 实现 LLMGenerator**

接口建议：
```go
type LLMGenerator struct {
	Provider providers.Provider
	ModelID  string
}

func (g LLMGenerator) Generate(intent string, mode dsl.Mode) (dsl.WorkflowSpec, GeneratorMeta, error)
```

实现要求：
- 调用 Provider（chat completion）
- 强制 “仅输出 JSON”
- 解析成 `dsl.WorkflowSpec`
- 调用 `dsl.ValidateWorkflowSpec`

- [ ] **Step 4: 测试通过**

- [ ] **Step 5: Commit**

```powershell
git add internal/generator
git commit -m "feat(generator): add LLM workflow generator (no repair loop yet)"
```

---

### Task 5: 实现 Repair Loop（LLM 修复无效 spec）

**Files:**
- Create: `internal/generator/repair.go`
- Modify: `internal/generator/llm_generator.go`
- Modify: `internal/generator/llm_prompt.go`
- Test: `internal/generator/llm_generator_test.go`

- [ ] **Step 1: 写 failing test：首次非法、二次修复后合法**

mock provider 需要按调用次数返回不同内容：
- 第一次：缺失 outputs 或 edge 引用错误
- 第二次：修复后的合法 spec

```go
func TestLLMGenerator_RepairLoop_FixesInvalidSpec(t *testing.T) {
	// Assert: meta.repair_attempts == 1
	// Assert: strategy_used == "llm_repaired"
}
```

- [ ] **Step 2: 运行测试确认失败**

- [ ] **Step 3: 实现 repair loop**

实现要求：
- 最大修复次数 `MaxRepairs`（默认 2，可配置）
- 每次校验失败：
  - 生成 `validation_errors_digest`
  - 构造 repair prompt（包含：上一次 spec 原文 + errors digest）
  - 再次调用 provider
- 最终成功：返回修复后的 spec + meta
- 仍失败：返回 error（由上层做 rule fallback）

- [ ] **Step 4: 测试通过**

- [ ] **Step 5: Commit**

```powershell
git add internal/generator
git commit -m "feat(generator): add repair loop for invalid workflow specs"
```

---

### Task 6: 规则生成器兜底（LLM 失败自动 fallback）

**Files:**
- Modify: `internal/generator/generator.go`
- Modify: `internal/http/handlers_generate.go`
- Test: `internal/http/handlers_generate_test.go`

- [ ] **Step 1: 写 failing test：LLM 返回非 JSON 时 fallback 到 rule**

```go
func TestGenerate_LLMNonJSON_FallsBackToRule(t *testing.T) {
	// Arrange: LLM generator always returns non-JSON or parse error
	// Act: /generate?generator_strategy=llm
	// Assert: response.generator_meta.strategy_used == "rule_fallback"
	// Assert: workflow_spec is valid
}
```

- [ ] **Step 2: 运行测试确认失败**

- [ ] **Step 3: 实现 fallback 路径**

行为要求：
- `generator_strategy=rule` 时绝不走 LLM
- `generator_strategy=llm` 时：LLM 失败/repair 失败 → 自动调用规则生成器
- meta 标识：
  - 纯 llm 成功：`llm`
  - repair 后成功：`llm_repaired`
  - fallback：`rule_fallback`

- [ ] **Step 4: 测试通过**

- [ ] **Step 5: Commit**

```powershell
git add internal/generator internal/http
git commit -m "feat(generator): fallback to rule generator when LLM generation fails"
```

---

### Task 7: 将 generator_meta 持久化到 SQLite（最小迁移）

**Files:**
- Create: `internal/store/migrations/002_generator_meta.sql`
- Modify: `internal/store/sqlite_store.go`
- Test: `internal/store/sqlite_store_test.go`（若无则创建）

- [ ] **Step 1: 写 failing test：保存 workflow/run 时包含 generator_meta**

测试要点：
- 生成后写入 store
- 再读出 workflow/run 记录，能看到 `strategy_used` 与 `repair_attempts`

- [ ] **Step 2: 运行测试确认失败**

- [ ] **Step 3: 写 migration + store 实现**

建议字段（以现有表为准，选择最小侵入）：
- `workflows.generator_strategy_used TEXT`
- `workflows.generator_repair_attempts INTEGER`
- `workflows.generator_validation_errors_digest TEXT`（JSON 字符串）

- [ ] **Step 4: 测试通过**

- [ ] **Step 5: Commit**

```powershell
git add internal/store
git commit -m "feat(store): persist generator_meta for workflows"
```

---

### Task 8: E2E 冒烟与回归（脚本 + 文档）

**Files:**
- Modify: `scripts/e2e-smoke.ps1`
- Modify: `docs/acceptance-itw-v1.md`（或新增 `acceptance-itw-v1_1.md`）

- [ ] **Step 1: 增加 generate 策略的冒烟测试**

脚本覆盖：
- `generator_strategy=rule`
- `generator_strategy=llm`（用 mock provider 模式跑）

- [ ] **Step 2: 本地跑完整测试**

Run:
```powershell
.\.tools\go\bin\go.exe test ./...
```

Expected: PASS

- [ ] **Step 3: Commit**

```powershell
git add scripts docs
git commit -m "test(e2e): add V1.1 generator strategy smoke coverage"
```

---

## Plan Self-Review

- Spec 覆盖：本计划实现了 `generator_strategy`、repair loop、fallback、generator_meta、测试与 DoD。
- Placeholder 扫描：所有任务包含具体文件路径、测试骨架与命令。
- 一致性：`strategy_used` 取值在 Task 6 固定（llm/llm_repaired/rule_fallback），贯穿 API 与 store。

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-15-itw-v1_1-llm-generator.md`. Two execution options:

1. **Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration
2. **Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?


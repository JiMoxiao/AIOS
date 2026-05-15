# Intent-to-Workflow（Dynamic AI Workflow Generator）V1 设计规格（LLM-only）

日期：2026-05-14

## 1. 产品定义（一句话）

用户只描述目标，系统自动生成可执行的工作流（DAG），为每个节点选择合适模型并自动执行、评估、回退，最终交付结果，同时提供可解释的执行记录与可复现的工作流导出。

## 2. V1 目标与边界

### 2.1 V1 目标（必须交付）

- Intent → Workflow：输入一句话目标，生成可执行 DAG（WorkflowSpec）
- Workflow → Run：一键执行 DAG，展示中间产物与最终输出
- 节点级模型路由：按节点类型/偏好（质量/成本/平衡）选择模型
- 自动质量控制：节点级评估（结构化校验/约束检查）+ 失败回退（重试/升级模型/改写提示）
- 可解释与可复现：
  - 展示工作流图（只读）
  - 展示每个节点选择模型的理由摘要
  - 生成运行记录（RunRecord）供审计与复现
- 双表示：
  - 内部权威格式：JSON（强校验、可执行）
  - 外部导出格式：YAML（可读、只读展示、审计/分享/复现）

### 2.2 V1 非目标（明确不做）

- 不支持用户手动编辑 YAML 后“保证可跑”（V1 仅导出/展示）
- 不做复杂连接器市场与业务系统编排（n8n/Dify 级别）
- 不做完整 RAG 知识库治理与多数据源权限继承（仅预留扩展点）
- 不做企业级多租户计费体系（可记录成本与用量，但不做计费产品化）

## 3. 用户体验（V1 主路径）

### 3.1 核心交互

1) 用户输入目标（Intent）
2) 系统生成工作流（WorkflowSpec）并展示：
   - DAG 图（节点与依赖）
   - 每个节点：做什么、输入输出、模型选择理由、预计成本等级
3) 用户点击运行
4) 系统执行并实时展示：
   - 节点状态（pending/running/succeeded/failed/skipped）
   - 中间产物摘要（默认只显示摘要，支持展开查看原文的权限开关）
   - 回退原因（重试/升级/改写）
5) 输出最终结果（Final Artifact），并提供：
   - 导出 YAML（只读）
   - 下载 RunRecord（JSON）

### 3.2 V1 默认模式

- 工作流 YAML：只读导出
- 输入/输出保存：默认保存摘要与指纹；原文保存由策略开关控制

## 4. 系统架构（生成器 + 运行时 + 治理底座）

### 4.1 核心组件

- Intent Parser：提取任务类型、交付物、约束、偏好（质量/成本/平衡）
- Workflow Generator：生成 DAG（节点类型、依赖、输入输出契约、评估规则）
- Model Router：节点级选模（规则过滤 + 评分）
- Workflow Executor：DAG 调度（并行、重试、回退、checkpoint）
- Evaluator：节点级验证与全局一致性检查
- Artifact Store：存放节点产物（原文或摘要）
- Run Store：存放 RunRecord 与审计事件

### 4.2 控制面与数据面（V1 最小划分）

- 控制面（Admin/Config）
  - Model Registry：模型能力与成本画像
  - Routing Profiles：质量/成本/平衡三档路由策略
  - Safety/Retention Policy：脱敏/保留策略、最大重试次数、禁用模型列表
- 数据面（Runtime）
  - Generate API：Intent → WorkflowSpec(JSON) + Rendered YAML
  - Run API：执行 DAG，产出 RunRecord 与最终 Artifact

## 5. 数据模型与 DSL（JSON 权威 + YAML 导出）

### 5.1 版本化原则

- `spec_version`：工作流 DSL 版本，执行器必须校验并支持迁移
- `generator_version`：生成器版本，用于回放与排障
- `created_at`：生成时间

### 5.2 WorkflowSpec（JSON，权威）

#### 5.2.1 顶层结构（概念模型）

- `workflow_id`：可空（生成后由系统分配）
- `spec_version`：例如 `1.0`
- `intent`：原始用户输入（可脱敏存储）
- `mode`：`quality | balanced | cost`
- `global_constraints`：
  - `max_total_cost_usd`（可选）
  - `max_wall_time_sec`（可选）
  - `data_classification`：`public | internal | confidential | restricted`
- `nodes[]`：节点定义
- `edges[]`：依赖关系（DAG）
- `outputs[]`：最终对外输出（引用某节点输出）

#### 5.2.2 Node 类型（V1）

统一字段：

- `id`：节点 id
- `type`：`planner | llm_task | evaluator | merge | finalizer`
- `name`：人类可读名称
- `description`：节点意图
- `inputs`：输入引用（来自 workflow inputs 或上游节点输出）
- `output_schema`：JSON Schema（用于结构化校验）
- `routing_profile`：`quality | balanced | cost` 或自定义权重
- `model_constraints`：允许/禁止模型列表、最小上下文要求
- `retry_policy`：最大重试次数、退避策略
- `fallback_chain`：候选模型链与触发条件

类型约束：

- `planner`：输出结构化计划（steps、deliverables、constraints）
- `llm_task`：执行具体子任务；必须声明 `expected_artifact_type`（text/json/markdown）
- `evaluator`：对上游产物做检查，输出 `pass | fail` 与 `reasons[]`，可附带 `repair_instructions`
- `merge`：合并多个分支产物（例如合并多段总结）
- `finalizer`：整理最终交付物与说明（含引用）

#### 5.2.3 Edges（DAG）

- `from`：上游 node id
- `to`：下游 node id
- `condition`：V1 仅支持：
  - `always`
  - `on_success`
  - `on_failure`（用于 evaluator 失败触发修复链）

### 5.3 YAML 导出规则（只读）

- YAML 由 WorkflowSpec(JSON) 渲染生成
- YAML 必须包含：节点列表、依赖关系、节点模型选择理由摘要、spec_version/generator_version
- YAML 不包含敏感字段原文；仅包含摘要或字段类型（如“命中手机号脱敏规则”）

### 5.4 示例（节选）

#### 5.4.1 WorkflowSpec.json（节选）

```json
{
  "spec_version": "1.0",
  "generator_version": "1.0",
  "intent": "读取PRD，生成API设计、实现代码、单测并做审查。",
  "mode": "balanced",
  "global_constraints": {
    "data_classification": "internal",
    "max_total_cost_usd": 2.0,
    "max_wall_time_sec": 900
  },
  "nodes": [
    {
      "id": "plan",
      "type": "planner",
      "name": "Plan",
      "description": "拆解任务并定义交付物与验收点",
      "inputs": [{ "type": "intent" }],
      "output_schema": { "type": "object" },
      "routing_profile": "quality",
      "retry_policy": { "max_attempts": 2 },
      "fallback_chain": [
        { "model": "claude-sonnet", "on": "any_error" },
        { "model": "gpt-5", "on": "any_error" }
      ]
    },
    {
      "id": "api_design",
      "type": "llm_task",
      "name": "API Design",
      "description": "基于计划生成API设计文档",
      "inputs": [{ "type": "node_output", "node_id": "plan" }],
      "expected_artifact_type": "markdown",
      "routing_profile": "balanced",
      "retry_policy": { "max_attempts": 2 },
      "fallback_chain": [
        { "model": "gpt-5", "on": "validation_fail" },
        { "model": "claude-opus", "on": "validation_fail" }
      ]
    }
  ],
  "edges": [
    { "from": "plan", "to": "api_design", "condition": "on_success" }
  ],
  "outputs": [
    { "name": "final", "from_node_id": "api_design" }
  ]
}
```

#### 5.4.2 WorkflowSpec.yaml（导出，节选）

```yaml
spec_version: "1.0"
generator_version: "1.0"
mode: balanced
intent_summary: "读取PRD→API设计→代码→单测→审查"
nodes:
  - id: plan
    type: planner
    name: Plan
    routing_profile: quality
    model_reason: "高质量拆解与约束提取"
  - id: api_design
    type: llm_task
    name: API Design
    routing_profile: balanced
    model_reason: "文档结构化与一致性优先，同时控制成本"
edges:
  - from: plan
    to: api_design
    condition: on_success
```

## 6. 执行语义（Workflow Executor）

### 6.1 状态机（节点级）

- `pending` → `running` → `succeeded`
- `running` → `failed`（触发 fallback/retry）
- `failed` → `running`（重试）
- `failed` → `succeeded`（通过修复链成功）
- `failed` → `terminal_failed`（超过重试/回退上限）
- `skipped`：依赖条件不满足

### 6.2 调度规则（V1）

- 拓扑排序执行，满足依赖即可运行
- 并行：同层无依赖节点可并行（V1 支持“有限并行”，配置最大并发）
- checkpoint：每节点完成后持久化输出（原文或摘要）与节点执行记录

### 6.3 回退策略（V1）

触发条件（最小集合）：

- `any_error`：网络/超时/5xx
- `validation_fail`：输出不满足 schema 或关键约束
- `empty_or_too_short`：内容为空/过短

动作优先级：

1) 重试同模型（带退避）
2) 切换到 fallback_chain 下一模型
3) 若存在 evaluator 的 `repair_instructions`，则对提示词做修复改写后再执行

## 7. 模型路由（Model Router）

### 7.1 路由输入

- 节点类型（planner/task/evaluator/merge/finalizer）
- mode（quality/balanced/cost）
- 约束（上下文长度、数据分级、禁用列表）
- 运行时统计（过去成功率、P95 延迟、成本）

### 7.2 路由算法（V1）

两阶段：

1) 过滤：满足硬约束的模型池
2) 评分：`score = wq*quality - wc*cost - wl*latency - we*error_rate`

其中 `w*` 由 mode 决定：

- quality：偏质量与稳定性
- cost：偏成本
- balanced：折中

## 8. 评估（Evaluator）

### 8.1 节点级评估（V1 最小）

- JSON Schema 校验（结构化输出节点）
- 约束校验：
  - 必须包含某些字段/段落（如“风险与假设”“接口列表”）
  - 字数/格式（Markdown 标题层级）
- 自一致性检查（轻量）：要求模型输出“检查清单”并自证满足

### 8.2 全局一致性（V1）

- 最终输出必须引用各阶段产物（引用列表）
- 生成“执行说明”：节点列表、失败与回退次数、成本汇总

## 9. 运行记录（RunRecord）与可观测

### 9.1 RunRecord（JSON）

必须包含：

- `run_id`、`workflow_id`、`spec_hash`
- `started_at`、`finished_at`
- `node_runs[]`：每节点的模型、token、cost、latency、状态、回退链、错误摘要
- `artifacts[]`：产物指针（原文/摘要/指纹）
- `final_output_ref`

### 9.2 指标（V1）

- 每次运行：总 token、总成本、总耗时、成功/失败
- 每节点：成功率、fallback 率、平均成本

## 10. 安全与数据保留（V1）

- 身份：API Key（最小闭环）；可预留 SSO 接入点
- 数据保留：
  - 默认保存摘要与指纹
  - 原文保存需要显式策略开关
- 日志：不记录敏感原文；仅记录 request_id、run_id、策略与错误摘要

## 11. V1 完成定义（DoD）

- 输入目标可生成 DAG（JSON + YAML）且可执行
- 至少支持 3 种节点类型：planner、llm_task、evaluator（merge/finalizer 可选但建议实现）
- 自动回退可演示（至少包含重试 + 切换模型）
- 运行记录可查询并可导出（RunRecord JSON）
- YAML 导出可用于复现同一工作流（用 JSON 作为执行源）


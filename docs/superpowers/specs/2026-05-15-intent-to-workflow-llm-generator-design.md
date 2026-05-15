# Intent-to-Workflow V1.1 设计规格：LLM Workflow Generator（DAG 自动生成 + Repair Loop）

日期：2026-05-15

## 1. 背景与目标

现状：V1 的 Workflow Generator 采用规则模板生成固定骨架（plan → task → eval → final），泛化能力有限，无法对不同意图生成差异化 DAG。

V1.1 引入 LLM Workflow Generator，使系统具备真正的 Intent-to-Workflow 能力：

- 用户输入自然语言目标（Intent）
- 系统自动生成可执行的 WorkflowSpec（JSON 权威格式）
- 若生成结果不合法，系统自动修复（repair loop）
- 若多次修复仍失败，回退到规则生成器（保证可用性）

## 2. V1.1 范围

### 2.1 必须交付

- LLM 生成完整 WorkflowSpec（JSON）
- 生成后强校验：
  - JSON 可解析
  - 满足 WorkflowSpec JSON Schema
  - 满足 DSL 语义校验（DAG 无环、引用存在、outputs 有效等）
- Repair Loop：
  - 将“校验错误列表”反馈给 LLM，生成修复后的 spec
  - 支持最多 N 次修复，最终失败必须 fallback 到规则生成器
- 生成过程可解释、可复现：
  - RunRecord/GenerateRecord 记录 generator 策略、修复次数、校验错误摘要
- 与现有运行时无缝兼容：
  - 生成出来的 WorkflowSpec 仍由现有 Executor/Router/Evaluator 执行

### 2.2 明确不做

- 不保证用户编辑 YAML 后可运行（仍保持只读导出）
- 不引入复杂工具/连接器节点（仍以 LLM-only 节点为主，工具节点仅作为扩展位）
- 不做大规模离线评测平台（仅提供最小回归集与 mock 单测）

## 3. 对外接口与产品行为

### 3.1 Generate API 行为

新增生成策略参数（默认不破坏兼容）：

- `generator_strategy`：
  - `llm`：优先使用 LLM 生成（默认）
  - `rule`：强制使用规则生成（用于回归与排障）

默认行为：

- 当 `generator_strategy=llm` 时：
  - 生成失败或多次修复仍失败 → 自动 fallback 到规则生成器
  - 响应里必须显式返回最终采用的策略：`llm | llm_repaired | rule_fallback`

### 3.2 可观测输出（Generate 侧）

响应应包含：

- `workflow_spec`（JSON 权威）
- `workflow_yaml`（只读导出）
- `generator_meta`：
  - `strategy_used`
  - `repair_attempts`
  - `validation_errors`（摘要，避免输出过长）

## 4. 生成管线（Generator Pipeline）

### 4.1 主流程

1) BuildPrompt：构造生成提示词
2) LLMGenerate：调用 LLM，期望输出“纯 JSON”
3) Parse：解析 JSON
4) Validate：
   - Schema Validate（WorkflowSpec JSON Schema）
   - Semantic Validate（dsl.ValidateWorkflowSpec）
5) 若 Validate 失败 → Repair Loop（最多 N 次）
6) 若仍失败 → Rule Fallback（generator.Generate）

### 4.2 Repair Loop 约束

- 修复输入必须包含：
  - 上一次输出的 `workflow_spec`（原样）
  - 校验错误列表（结构化：code/message/path）
  - 明确要求：仅输出修复后的纯 JSON
- 修复目标：
  - 最小化修改（尽量保持 node id 与 edges 稳定）
  - 优先修复结构/引用/条件等“可执行性错误”
  - 不扩张 scope（不新增与 intent 无关的节点）

### 4.3 Fail-safe（强保证）

- 任意情况下 Generate API 都必须可返回一个可执行 WorkflowSpec：
  - LLM 失败、超时、返回非 JSON、返回不合法 DAG → 走 rule fallback

## 5. Prompt 设计（V1.1 最小标准）

### 5.1 System Prompt（稳定约束）

要求：

- 输出必须是单个 JSON 对象，不允许 markdown code fence，不允许多余文本
- JSON 必须满足指定 JSON Schema（以文字形式声明关键字段与枚举）
- 节点类型受限：
  - `planner | llm_task | evaluator | merge | finalizer`
- edges 必须构成 DAG，outputs 必须引用存在的 node

### 5.2 Developer Prompt（上下文注入）

包含：

- intent 原文
- mode（quality/balanced/cost）
- 全局约束（data classification、budget、wall time）在 V1.1 可选传入
- 允许的模型标签（如果 catalog 中存在能力标签，则传入可用模型列表与简要画像）

### 5.3 Few-shot（可选）

V1.1 允许内置 2-3 个短示例以提高稳定性：

- 结构化输出（json artifact）意图
- 文档输出（markdown）意图
- 多步骤工程任务（设计/实现/测试/复核）意图

## 6. 与 Router/Evaluator 的协作约定

### 6.1 Generator 自身选模

Generator 的 LLM 调用采用独立 routing profile：

- 默认 `quality`（以减少 repair 次数）
- 可通过配置开关切换到 `balanced`（成本敏感环境）

### 6.2 节点级路由字段

LLM 生成的 Node 中：

- `routing_profile`：默认继承 workflow mode，但 planner/evaluator 建议偏质量
- `fallback_chain`：允许为空；为空时由 Router 自动补齐
- `model_constraints`：可由 LLM 设置（例如最小上下文需求），但必须通过 Validate

### 6.3 Evaluator 扩展（为生成可靠性服务）

V1.1 不要求新增复杂评估器，但需要保证：

- evaluator 节点能产出结构化 `pass/fail` 与 `reasons[]`
- 当 evaluator fail 时，Executor 能触发 node retry 或 fallback（已有机制则复用）

## 7. 数据模型扩展（最小变更）

### 7.1 Generate 记录

在持久化层新增（或扩展现有 run/workflow 表）字段以支持排障：

- `generator_strategy_requested`
- `generator_strategy_used`
- `repair_attempts`
- `validation_errors_digest`（摘要或 hash）

### 7.2 安全与保留

- validation_errors 可能包含路径信息，不应包含用户原文
- intent 原文存储仍受 retention policy 控制（默认摘要/指纹）

## 8. 测试与验收（Definition of Done）

### 8.1 单元测试（必须）

用 Mock Provider 构造以下场景（不依赖真实外部模型）：

- Case A：LLM 一次返回合法 WorkflowSpec（无 repair）
- Case B：LLM 第一次返回非法（例如缺字段/引用不存在），第二次修复后合法（repair=1）
- Case C：LLM 连续返回非法，触发 rule fallback（repair 达上限）
- Case D：LLM 返回非 JSON（触发 rule fallback）

### 8.2 语义校验回归（必须）

- DAG 无环校验
- edges 引用存在
- outputs 引用存在
- nodes 输入引用存在（intent 或 node_output）

### 8.3 端到端冒烟（必须）

- `/generate?generator_strategy=llm`：成功生成并可导出 YAML
- `/run`：可执行并生成 RunRecord
- RunRecord 中包含 generator_meta（strategy/repair_attempts）

## 9. 迁移与兼容性

- 默认行为不改变现有 API 的核心字段
- 老客户端若不传 `generator_strategy`，默认走 `llm`，失败自动 fallback，不影响可用性

## 10. 风险与应对

- 风险：LLM 输出不稳定（非 JSON、字段错误、DAG 不可执行）
  - 应对：强校验 + repair loop + rule fallback（强保证）
- 风险：repair loop 引入额外成本/延迟
  - 应对：generator 默认 quality 但限制修复次数；可配置改为 balanced；记录 repair 次数以便优化
- 风险：生成 spec 过度复杂
  - 应对：prompt 中约束“节点数上限（建议 4-12）”与“最小必要 DAG”，并在语义校验里限制最大节点数（配置项）


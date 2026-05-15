# AIOS Gateway + 治理（V1）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在私有化部署环境交付一个可上线的 AI Gateway 与治理控制面，支持多模型接入、策略路由、可控出网、预算/限流、审计与可观测，并提供 K8s/VM 两套部署蓝图。

**Architecture:** 采用“控制面 + 数据面”分离。数据面负责请求鉴权、策略执行、模型调用、出网控制与遥测上报；控制面负责租户/项目/环境、模型目录、策略版本化与灰度回滚、审计检索与报表。两者通过内部管理 API 与配置下发解耦。

**Tech Stack:** Go（控制面/数据面统一语言）+ REST（OpenAI 风格）+ Postgres（起步统一存储）+ OpenTelemetry（trace/metrics）+ Prometheus 指标端点；部署交付 Helm（K8s）与 docker-compose（VM）。

---

## 0. 范围拆解（从规格到实现的工作包）

本计划把 [2026-05-14-aios-gateway-governance-design.md](file:///d:/Myworkspace/AIOS/docs/superpowers/specs/2026-05-14-aios-gateway-governance-design.md) 的要求拆成 8 个可交付工作包：

1) 代码仓结构与基础工程（单仓或多模块、配置体系、日志、错误码）  
2) 数据模型与存储（Tenant/Project/Env、Model Registry、Policy、Audit Event、Usage）  
3) 控制面（管理 API + 控制台 UI 可选）  
4) 数据面（AI Gateway：OpenAI 风格 API、鉴权、限流/预算、路由、fallback）  
5) 可控出网（Egress、allowlist、脱敏/DLP、审计）  
6) 可观测（metrics/traces/logs）与审计报表  
7) 部署蓝图（K8s Helm + VM docker-compose）  
8) 端到端演示与验收脚本（Definition of Done）  

每个包都按“先测试/再实现/再验收”的方式拆成任务。

---

## Task 1: 工程初始化与仓结构（Foundation）

**Files:**
- Create: `AIOS/README.md`
- Create: `AIOS/docs/architecture.md`
- Create: `AIOS/.editorconfig`
- Create: `AIOS/.gitignore`
- Create: `AIOS/apps/`（数据面/控制面入口）
- Create: `AIOS/packages/`（共享库：配置、策略、模型适配、审计）
- Create: `AIOS/deploy/`（k8s/compose）
- Create: `AIOS/scripts/`（本地启动与验收脚本）

- [ ] **Step 1: 固化技术栈与运行方式（已决策）**
  - 后端语言：Go（控制面与数据面统一实现，便于共享中间件与策略运行时）
  - API 风格：REST；数据面提供 OpenAI Chat Completions 兼容接口；控制面提供 Admin API
  - 存储：Postgres 起步（docker-compose 内置）；Schema 迁移通过 migrations 目录维护并随版本发布
  - 遥测：OpenTelemetry trace + Prometheus metrics；每个外部 provider 调用为 child span
  - 安全接入：V1 以 API Key 为主；SSO（OIDC/SAML）通过反向代理或后续扩展接入
  - 将上述决策写入 `AIOS/docs/architecture.md` 的 ADR 小节，作为后续实现与验收依据

- [ ] **Step 2: 建立目录结构与约定**
  - 在 `AIOS/README.md` 写清：本地启动、配置方式、最小演示流程
  - 在 `AIOS/docs/architecture.md` 写清：模块边界、控制面/数据面通信

- [ ] **Step 3: 基础测试与质量门槛**
  - 选择单测框架与 lint/format 工具
  - 在 CI（后续）之前，先保证本地命令一键跑：format、lint、test

- [ ] **Step 4: Commit**
  - 提交 message：`chore: init aios gateway repository structure`

---

## Task 2: 核心数据模型与持久化（Control Plane Storage）

**Files:**
- Create: `AIOS/packages/domain/`（领域模型）
- Create: `AIOS/packages/storage/`（DAO/Repository）
- Create: `AIOS/packages/migrations/`（表结构迁移）
- Create: `AIOS/packages/domain/types.*`（根据语言）
- Create: `AIOS/packages/domain/policy.*`
- Create: `AIOS/packages/domain/audit.*`
- Test: `AIOS/packages/domain/*_test.*`

- [ ] **Step 1: 定义领域对象（最小字段集）**
  - Tenant / Project / Env
  - Model（含 capabilities、cost_profile、latency_profile、compliance、health）
  - Policy（hard_constraints、scoring、fallback_chain、validation、observability）
  - AuditEvent（who/where/what/network/cost/result）
  - Usage（token、cost、latency、status、policy_version）

- [ ] **Step 2: 写“序列化/反序列化”测试**
  - 覆盖：Policy 版本化、字段向后兼容、默认值

- [ ] **Step 3: 建表与迁移**
  - 为 Tenant/Project/Env、Model、Policy、AuditEvent、Usage 建表
  - 约束：tenant_id + project_id 组合唯一、policy 版本号、审计索引字段

- [ ] **Step 4: Repository 层测试**
  - 创建/查询/分页/按 project/env 过滤
  - AuditEvent 写入吞吐与按 trace_id 检索

- [ ] **Step 5: Commit**
  - `feat: add core domain models and storage`

---

## Task 3: 控制面管理 API（Admin API）

**Files:**
- Create: `AIOS/apps/control-plane/`（HTTP 服务）
- Create: `AIOS/apps/control-plane/routes/*`
- Create: `AIOS/apps/control-plane/auth/*`
- Test: `AIOS/apps/control-plane/*_test.*`

- [ ] **Step 1: API 契约（OpenAPI/Swagger 或等价）**
  - CRUD：Tenant/Project/Env
  - Model Registry：注册/下线/健康查看
  - Policy：创建/版本化/灰度发布/回滚/审批状态
  - Budget/RateLimit：配置/查询
  - Audit：查询/导出（先 CSV）

- [ ] **Step 2: 鉴权与 RBAC（最小闭环）**
  - 支持两类身份：用户（SSO 预留）与系统（API Key）
  - 最小 RBAC：platform_admin、project_admin、auditor、caller

- [ ] **Step 3: 实现与测试**
  - 每个路由都要有：成功路径 + 权限失败 + 参数校验失败 的测试

- [ ] **Step 4: Commit**
  - `feat: add control plane admin api`

---

## Task 4: 数据面 AI Gateway（OpenAI 兼容入口）

**Files:**
- Create: `AIOS/apps/gateway/`
- Create: `AIOS/apps/gateway/routes/openai/*`
- Create: `AIOS/packages/request-context/*`（tenant/project/env/data_classification 解析）
- Create: `AIOS/packages/policy-runtime/*`
- Test: `AIOS/apps/gateway/*_test.*`

- [ ] **Step 1: 请求上下文解析与校验**
  - 从 header/metadata 解析：tenant/project/env、data_classification、policy_hint
  - 无上下文时：拒绝或走默认 project（由配置决定）

- [ ] **Step 2: OpenAI Chat Completions 兼容层**
  - 透传必要字段（messages、model、temperature、stream 等）
  - 记录：request_id/trace_id

- [ ] **Step 3: 策略执行框架（不含具体路由算法）**
  - 调用顺序：hard_constraints → select_candidates → scoring → execute → validate → fallback
  - 输出标准化错误码（预算不足、合规拒绝、出网拒绝、模型不可用）

- [ ] **Step 4: 集成测试**
  - 使用 mock provider：模拟成功/超时/空响应/格式错误，验证 fallback 生效

- [ ] **Step 5: Commit**
  - `feat: add gateway openai-compatible api and policy runtime skeleton`

---

## Task 5: 模型适配层（Providers）

**Files:**
- Create: `AIOS/packages/providers/`（每个 provider 一个实现）
- Create: `AIOS/packages/providers/interfaces.*`
- Test: `AIOS/packages/providers/*_test.*`

- [ ] **Step 1: 定义 provider 接口**
  - 能力：chat completion（V1）
  - 统一返回：token、latency、status、raw_error 分类

- [ ] **Step 2: 实现至少 3 类来源**
  - 云模型（通过可控出网 egress）
  - 本地模型（内网 http endpoint）
  - “企业代理出口”模型（走企业统一代理/网关）

- [ ] **Step 3: 适配层测试**
  - 针对每类 provider：超时、429、5xx、无效响应

- [ ] **Step 4: Commit**
  - `feat: add provider adapters`

---

## Task 6: 可控出网（Egress + Allowlist）

**Files:**
- Create: `AIOS/apps/egress/`（可选：独立服务；也可作为 gateway 内模块）
- Create: `AIOS/packages/egress/*`
- Create: `AIOS/packages/allowlist/*`
- Test: `AIOS/packages/egress/*_test.*`

- [ ] **Step 1: allowlist 规则模型**
  - domain/ip/port，按 tenant/project/env 生效

- [ ] **Step 2: 统一出口实现**
  - 所有云模型请求强制经过 egress
  - allowlist 不命中：拒绝并生成审计事件

- [ ] **Step 3: 集成测试**
  - 使用本地 http server 模拟外部 endpoint
  - 验证：allowlist 命中允许、未命中拒绝

- [ ] **Step 4: Commit**
  - `feat: add egress gateway and allowlist enforcement`

---

## Task 7: 脱敏/DLP（规则引擎 + 动作）

**Files:**
- Create: `AIOS/packages/dlp/*`
- Create: `AIOS/packages/dlp/rules/*`
- Test: `AIOS/packages/dlp/*_test.*`

- [ ] **Step 1: 规则与动作定义**
  - 规则：手机号、身份证、邮箱、银行卡（正则起步，可扩展）
  - 动作：block、mask、downgrade_to_local_pool

- [ ] **Step 2: 在策略执行链中接入**
  - 在调用 provider 之前扫描请求，在响应之后扫描输出（可选）
  - 审计仅记录命中类型与动作，不记录原文

- [ ] **Step 3: 单测与回归集**
  - 每条规则最少 5 条正例/反例
  - 覆盖误报样例（例如 11 位数字不一定是手机号）

- [ ] **Step 4: Commit**
  - `feat: add dlp masking and blocking`

---

## Task 8: 预算、配额与限流（Cost Governance）

**Files:**
- Create: `AIOS/packages/quota/*`
- Create: `AIOS/packages/ratelimit/*`
- Create: `AIOS/packages/cost/*`
- Test: `AIOS/packages/quota/*_test.*`

- [ ] **Step 1: 成本估算与记账**
  - 基于 provider 返回 token 与 cost_profile 估算 cost
  - 记录到 Usage 表

- [ ] **Step 2: 预算与配额检查**
  - tenant/month、project/month（V1）
  - 超限行为：拒绝或降级（先实现拒绝）

- [ ] **Step 3: 限流**
  - 按 project/user/model 做 token bucket 或漏桶

- [ ] **Step 4: 集成测试**
  - 连续请求触发限流
  - 预算不足返回标准错误码

- [ ] **Step 5: Commit**
  - `feat: add budget quota and rate limiting`

---

## Task 9: 观测与审计（Telemetry + Audit UI/Export）

**Files:**
- Create: `AIOS/packages/telemetry/*`
- Modify: `AIOS/apps/gateway/*`（埋点）
- Modify: `AIOS/apps/control-plane/*`（审计查询/导出）
- Test: `AIOS/packages/telemetry/*_test.*`

- [ ] **Step 1: OpenTelemetry 接入**
  - trace：每次请求一个 root span，provider 调用为 child span
  - metrics：QPS、成功率、P50/P95、token、cost、fallback_rate、reject_rate

- [ ] **Step 2: 审计事件写入**
  - 关键字段齐全（who/where/what/network/cost/result）
  - 写入失败不影响主流程（异步队列或降级策略）

- [ ] **Step 3: 审计检索与导出**
  - 按 project、时间范围、user_id、model、policy_version、egress_domain 过滤
  - CSV 导出（V1）

- [ ] **Step 4: Commit**
  - `feat: add telemetry and audit export`

---

## Task 10: 策略版本化、灰度与回滚（Release）

**Files:**
- Create: `AIOS/packages/release/*`
- Modify: `AIOS/apps/control-plane/routes/policies/*`
- Modify: `AIOS/apps/gateway/*`（策略拉取与缓存）
- Test: `AIOS/packages/release/*_test.*`

- [ ] **Step 1: 策略版本模型**
  - policy_id + version，状态：draft/approved/released/rolled_back

- [ ] **Step 2: 灰度分流**
  - 按 user group 或 request header 进行路由到不同版本

- [ ] **Step 3: 回滚**
  - 一键回滚到上一 released 版本
  - 审计记录变更人、原因、审批信息

- [ ] **Step 4: Commit**
  - `feat: add policy release, canary, and rollback`

---

## Task 11: 部署蓝图（K8s Helm + VM docker-compose）

**Files:**
- Create: `AIOS/deploy/k8s/helm/aios/`（Chart）
- Create: `AIOS/deploy/compose/docker-compose.yml`
- Create: `AIOS/deploy/compose/.env.example`
- Create: `AIOS/deploy/README.md`

- [ ] **Step 1: 镜像与配置约定**
  - gateway、control-plane、egress（如独立）
  - 配置：环境变量 + 配置文件，支持 secret 注入

- [ ] **Step 2: docker-compose**
  - 启动所有组件 + 存储（如 Postgres）

- [ ] **Step 3: Helm Chart**
  - Deployment、Service、Ingress、ConfigMap、Secret、HPA（可选）

- [ ] **Step 4: Commit**
  - `feat: add deployment blueprints for k8s and vm`

---

## Task 12: 端到端验收脚本与演示（DoD）

**Files:**
- Create: `AIOS/scripts/e2e-smoke.*`
- Create: `AIOS/scripts/demo-scenarios/*`
- Create: `AIOS/docs/acceptance.md`

- [ ] **Step 1: 写 DoD 对照表**
  - 把规格 16. DoD 映射到可执行检查项（命令 + 期望输出）

- [ ] **Step 2: E2E 冒烟**
  - 场景1：多模型路由 + fallback
  - 场景2：allowlist 拦截
  - 场景3：DLP 脱敏/阻断
  - 场景4：预算不足拒绝
  - 场景5：审计可检索/可导出 + trace 可追踪

- [ ] **Step 3: Commit**
  - `test: add e2e smoke scripts and acceptance checklist`

---

## 计划自检：规格覆盖表（摘要）

- 统一 API：Task 4  
- 多模型接入：Task 5  
- 路由与 fallback：Task 4/5  
- 可控出网 allowlist：Task 6  
- 脱敏/DLP：Task 7  
- 预算/限流/成本台账：Task 8/9  
- 审计：Task 9/10/12  
- 灰度回滚：Task 10  
- 部署蓝图：Task 11  
- DoD 验收：Task 12  

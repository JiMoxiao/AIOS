# AIOS 企业级产品方案A（V1）设计规格：AI Gateway + 治理

日期：2026-05-14

## 1. 背景与问题

企业在引入大模型后会同时面临：

- 模型能力碎片化：不同模型在代码、长上下文、成本、延迟、多模态等维度表现不一致
- 合规与安全：数据分级、可控出网、审计留痕、密钥托管、权限隔离
- 成本与质量治理：预算控制、质量门禁、失败回退、灰度与回归
- 工具与系统集成：业务系统、知识库、研发流水线等多入口多规范

V1 以“统一入口 + 治理底座”为目标，先把企业最需要的可上线能力做成平台能力，再逐步叠加 RAG、Copilot、流程 Agent 等解决方案包。

## 2. 产品定位

AIOS V1 是企业内“模型无关 AI 调度与治理平台”，以私有化部署为主，支持可控出网。

提供能力：

- 统一 OpenAI 风格 API 与 SDK 兼容层
- 多模型接入与能力目录（Model Registry）
- 策略路由、预算/限流、质量门禁与自动回退
- 可控出网（allowlist + 脱敏/DLP + 留痕）
- 审计与可观测（metrics + trace + 审计报表）
- 配置与发布（策略/模板版本管理，最小审批闭环）

## 3. 范围（V1）

### 3.1 必做（In Scope）

- 企业 AI Gateway：统一入口、鉴权、限流/配额、路由、重试/fallback、缓存（可选）、观测与审计
- 治理控制台：模型目录、策略中心、出网控制、预算与配额、审计检索与导出、运行看板
- 可控出网：统一出口、allowlist、分级规则、脱敏/DLP（规则引擎）、策略阻断/降级
- 基础质量门禁：结构化输出校验（JSON schema）、最低可用性检查（超时/空响应/敏感内容策略）
- 策略发布与回滚：版本化、灰度发布、回滚

### 3.2 不做（Out of Scope，后续版本）

- 完整 Agent Studio（可视化编排、连接器市场）与复杂人审工作台
- 大规模自动评测平台（只保留最小回归机制）
- 复杂的多数据源知识库治理（V1 仅预留接口与事件）

## 4. 角色与使用方式

### 4.1 角色（Personas）

- 平台管理员：部署、对接 SSO/KMS、配置全局策略、审计与报表
- 项目管理员：管理项目/环境、配置该项目模型池、预算、策略、allowlist、脱敏规则
- 开发者/业务系统：通过统一 API 调用模型或能力域（如“代码生成”“长上下文问答”）
- 安全与合规：查看审计、审批高风险变更、导出合规报表

### 4.2 主要入口

- API：OpenAI 风格 REST（重点）
- 控制台：治理配置与观测
- Webhook/事件：用于审计、告警、成本台账对接外部系统

## 5. 架构（控制面 + 数据面）

### 5.1 控制面（Control Plane）

- Tenant/Project/Env 管理：租户、项目、环境隔离（dev/stage/prod）
- Model Registry：模型目录与能力标签、成本/延迟/合规属性
- Policy Center：路由策略、预算、限流、allowlist、脱敏规则、fallback 链
- Release Center：策略版本、灰度发布、回滚、变更审批与审计
- Audit & Reporting：审计检索与报表导出

### 5.2 数据面（Data Plane）

- AI Gateway：统一鉴权、策略执行、模型调用、工具调用（预留）、可观测与审计上报
- Router Runtime：执行路由决策（硬约束 → 评分 → fallback）
- Egress Gateway：统一出网出口、allowlist、TLS/代理、审计
- Telemetry：metrics、logs、traces 汇聚与导出

## 6. 关键数据模型（V1）

### 6.1 租户/项目/环境

- Tenant：企业级边界
- Project：成本/权限/策略的主要归属
- Env：dev/stage/prod，策略可逐级继承并允许覆盖

### 6.2 模型目录（Model Registry）

模型元信息至少包含：

- provider：OpenAI/Anthropic/DeepSeek/… 或 on-prem
- model_id：对外展示的逻辑模型名（支持别名）
- capabilities：code/long_context/multimodal/reasoning/chinese/…
- limits：最大上下文、输出限制、速率限制
- cost_profile：token 单价、最小计费单元、成本估算参数
- latency_profile：P50/P95（可来自观测统计）
- compliance：可出网、驻留区域、是否允许 Confidential/Restricted
- health：可用性、错误率（来自观测）

### 6.3 策略（Policy）

策略由多个“规则块”组成：

- hard_constraints：合规/数据分级/出网限制/预算上限/工具权限
- scoring：质量、成本、延迟的权重与阈值
- fallback_chain：失败条件与重试/换模/降级方案
- validation：结构化校验/内容安全校验/响应最小完整性校验
- observability：采样率、日志级别、审计字段

### 6.4 审计事件（Audit Event）

必须可追溯：

- who：user_id / service_id、来源 IP、用户组/角色
- where：tenant/project/env、应用标识、调用链路 id
- what：策略版本、模型、参数摘要、数据分级、工具列表（预留）
- network：是否出网、目的域名、allowlist 命中情况
- cost：token、估算成本、预算扣减结果
- result：状态码、错误类型、fallback 次数、响应摘要指纹

## 7. 路由与治理机制（V1）

### 7.1 路由决策流程

1) 识别请求上下文：tenant/project/env、用户角色、数据分级、能力域（可选）
2) 硬约束过滤：合规允许的模型池、预算/配额、allowlist、风险策略
3) 评分选择：根据权重（质量/成本/延迟）选 Top-1 或 Top-K
4) 执行与验证：调用模型 → 结构化/安全校验
5) fallback：按链路重试/换模/降级，记录原因

### 7.2 预算与限流

- 预算粒度：tenant/month、project/month、user/day（至少支持前两者）
- 限流维度：tenant/project/user/model
- 行为：拒绝、排队、降级到低成本模型、或仅允许“摘要模式”（预留）

### 7.3 质量门禁（V1 最小集合）

- JSON schema：适用于结构化输出型场景
- 最小可用性：非空、长度阈值、响应格式合法
- 内容安全（策略化）：关键词/分类器接口预留；支持阻断/脱敏/重写（V1 先阻断与脱敏）

## 8. 可控出网（V1）

### 8.1 出网策略

- allowlist：域名/IP/端口，支持按 project/env/策略分配
- 代理出口：所有外部模型调用只能经 Egress Gateway
- 证书与 TLS：支持企业自签/中间人代理场景（可选）

### 8.2 脱敏/DLP

- 规则：PII（手机号/身份证/邮箱/银行卡）、业务敏感字段（可配置字典）
- 动作：阻断、替换、降级模型、强制走本地模型池
- 审计：记录命中字段类型与动作，不记录原始敏感值

## 9. API 与对接（V1）

### 9.1 统一模型调用 API

- 兼容 OpenAI Chat Completions 的关键字段
- 额外扩展（headers 或 metadata）：
  - x-tenant / x-project / x-env
  - x-data-classification（Public/Internal/Confidential/Restricted）
  - x-policy-hint（可选，指定策略或能力域）

### 9.2 管理 API（控制台使用）

- 租户/项目/环境：CRUD
- 模型目录：注册、下线、健康查看
- 策略：创建、版本化、灰度、回滚、审批
- 预算/限流：配置与查询
- 审计：查询、导出、告警订阅

## 10. 部署与运维（K8s + VM 双蓝图）

### 10.1 组件划分

- aios-gateway：无状态，多副本
- aios-control-plane：控制面 API，无状态，多副本
- aios-egress：统一出口代理，建议多副本
- aios-db：策略/审计索引的存储（实现可选）
- aios-telemetry：指标与链路追踪导出（实现可选）

### 10.2 K8s 交付

- Helm Chart（V1 目标）
- 支持 HPA（按 QPS/CPU）
- 对接 Prometheus/Grafana（metrics），OpenTelemetry（trace）

### 10.3 VM 交付

- docker-compose（V1 目标）
- systemd 管理容器与健康检查（建议）

## 11. 可靠性与灰度（V1）

- 超时、重试、熔断：按模型与 provider 维度配置
- fallback 链：支持同 provider 多模型、跨 provider
- 策略灰度：按 project/env/user group 分流；支持快速回滚

## 12. 安全设计（V1）

- 身份与权限：SSO + RBAC；API Key 用于系统集成
- 密钥管理：对接企业 KMS/Vault；密钥轮换；最小权限
- 数据最小化：日志与审计存“摘要/指纹”，避免落原文；必要时可配置采样留存
- 变更审批：高风险策略（允许出网、放宽分级、关闭脱敏）需审批与留痕

## 13. 观测与报表（V1）

### 13.1 必备指标

- QPS、成功率、P50/P95 延迟
- token 与成本（按 tenant/project/user/model）
- fallback 率与原因分布
- 拒绝/阻断率（预算/合规/脱敏命中）

### 13.2 报表

- 成本台账：按月/项目/部门导出
- 合规审计：出网目的、allowlist 命中、脱敏动作统计

## 14. 里程碑拆解（仅用于范围分层，不含时间估算）

- M0：单租户网关 + 单模型适配 + 基础鉴权
- M1：多模型目录 + 路由策略 + fallback + 指标
- M2：可控出网（allowlist + 统一出口）+ 审计事件
- M3：预算/配额/限流 + 脱敏/DLP + 策略版本/灰度/回滚
- M4：控制台闭环（策略配置、审计检索、成本报表）

## 15. 风险与对策

- 风险：策略复杂度迅速膨胀  
  - 对策：策略分层（全局/项目/环境继承），提供内置策略包与可视化预览
- 风险：审计数据量大、成本高  
  - 对策：采样、摘要指纹、冷热分层存储、按需导出
- 风险：脱敏误伤/漏报影响体验  
  - 对策：分级策略、灰度开关、可回放与规则回归集

## 16. V1 完成定义（Definition of Done）

- 私有化部署可运行（K8s 或 VM 任一蓝图可用）
- 至少对接 3 类模型来源：云模型（出网）、本地模型（内网）、企业代理出口（可控出网）
- 路由、预算、审计、观测、脱敏、灰度回滚均可演示且可验收


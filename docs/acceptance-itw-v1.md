# Intent-to-Workflow V1 验收清单

## 1. 生成（Intent → Workflow）

- `POST /v1/generate` 输入 intent
- 返回：
  - `workflow_id`
  - `workflow_spec_json`（JSON 权威）
  - `workflow_yaml`（只读导出）

## 2. 执行（Workflow → Run）

- `POST /v1/run` 输入 workflow_id
- 返回 `run_id`
- `GET /v1/runs/{run_id}` 返回 RunRecord，且：
  - `status == succeeded`
  - `node_runs` 至少包含 planner/task/eval/final
  - `artifacts` 有记录

## 3. YAML 导出

- `GET /v1/workflows/{workflow_id}/yaml` 返回 yaml 文本
- 包含 `spec_version` 与 `nodes`

## 4. 自动回退（Fallback）

- 将 provider 配置为首次返回空内容
- 执行后 RunRecord 中对应节点 `fallbacks` 非空

## 5. Metrics

- `GET /metrics` 可访问
- 至少包含：
  - `itw_runs_total`
  - `itw_node_runs_total`
  - `itw_fallback_total`


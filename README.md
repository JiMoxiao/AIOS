# AIOS Intent-to-Workflow (V1)

这是一个 LLM-only 的 Intent-to-Workflow 系统：输入目标 → 自动生成 DAG 工作流（JSON 权威 + YAML 导出）→ 执行工作流（重试/回退/记录）→ 输出结果与 RunRecord。

## 本地运行

如果系统未安装 Go，可使用仓库内便携 Go：

```powershell
.\.tools\go\bin\go.exe version
```

启动服务：

```powershell
.\.tools\go\bin\go.exe run .\cmd\itw-server\main.go
```

默认监听 `:8080`。

## 环境变量

- `AIOS_HTTP_ADDR`：监听地址，默认 `:8080`
- `AIOS_DB_DSN`：SQLite 文件路径，默认 `itw.sqlite`
- `AIOS_SAVE_BODY`：是否保存 artifact 原文，默认 `false`
- `AIOS_OPENAI_BASE_URL`：OpenAI 兼容接口 base url（例如 `http://localhost:8001/v1`）
- `AIOS_OPENAI_API_KEY`：OpenAI 兼容接口 key

## API（V1）

- `POST /v1/generate`：`{ "intent": "...", "mode": "quality|balanced|cost" }`
- `POST /v1/run`：`{ "workflow_id": "..." }`
- `GET /v1/runs/{run_id}`
- `GET /v1/workflows/{workflow_id}/yaml`
- `GET /metrics`


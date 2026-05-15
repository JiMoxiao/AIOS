$ErrorActionPreference = "Stop"

$BaseUrl = $env:AIOS_BASE_URL
if ([string]::IsNullOrWhiteSpace($BaseUrl)) {
  $BaseUrl = "http://localhost:8080"
}

$gen = Invoke-RestMethod -Method Post -Uri "$BaseUrl/v1/generate" -ContentType "application/json" -Body (@{
  intent = "写一份Markdown报告"
  mode = "balanced"
} | ConvertTo-Json)

if (-not $gen.workflow_id) {
  throw "missing workflow_id"
}

$run = Invoke-RestMethod -Method Post -Uri "$BaseUrl/v1/run" -ContentType "application/json" -Body (@{
  workflow_id = $gen.workflow_id
} | ConvertTo-Json)

if (-not $run.run_id) {
  throw "missing run_id"
}

$rec = Invoke-RestMethod -Method Get -Uri "$BaseUrl/v1/runs/$($run.run_id)"
if ($rec.status -ne "succeeded") {
  throw "run not succeeded: $($rec.status)"
}

"ok: run_id=$($run.run_id)"


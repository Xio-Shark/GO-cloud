# GO Cloud

基于 Go、Redis、MySQL、Docker Compose、Kubernetes、Prometheus、Grafana、Argo CD 的任务调度与发布平台。

## 当前能力

- 任务创建、更新、查询、暂停、恢复、删除、手动触发
- scheduler 派发 execution，worker 消费并执行
- 执行记录查询、日志查询、重试、取消
- `shell`、`http`、`container(Kubernetes Job)` 三类执行器
- webhook notifier
- Prometheus 指标、Grafana 面板、Nginx 反向代理
- release 发布与回滚 API，真实写回 `deployments/k8s/overlays/*/patch-images.yaml`
- Docker Compose、本地脚本、Kubernetes manifests、GitHub Actions CI/CD

## 目录

```text
cmd/
internal/
pkg/
deployments/
docs/plans/
.github/workflows/
Dockerfile
docker-compose.yml
Makefile
```

## 快速开始

1. 使用环境变量模板：

```powershell
Copy-Item .env.example .env
```

2. 构建 Linux 二进制：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build-linux.ps1
```

3. 启动本地依赖与服务：

```powershell
docker compose --env-file .env.example up -d --build
```

4. 检查健康状态：

```powershell
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:8080/healthz
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:9091/healthz
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:9092/healthz
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:9093/healthz
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:9090/-/healthy
```

## 本地入口

- API: `http://127.0.0.1:8080`
- Nginx: `http://127.0.0.1:${NGINX_PORT}`
- Prometheus: `http://127.0.0.1:${PROMETHEUS_PORT}`
- Grafana: `http://127.0.0.1:${GRAFANA_PORT}`

## 关键 API

### 创建任务

```powershell
$body = @{
  name = 'container-demo'
  description = 'run kubernetes job'
  task_type = 'container'
  schedule_type = 'manual'
  payload = @{
    image = 'busybox:1.36'
    command = @('sh', '-c', 'echo hello-from-job')
    env = @{ APP_ENV = 'dev' }
  }
  timeout_seconds = 60
  retry_times = 0
  created_by = 'demo'
} | ConvertTo-Json -Depth 6

Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8080/api/v1/tasks -ContentType 'application/json' -Body $body
```

### 删除任务

```powershell
Invoke-RestMethod -Method Delete -Uri 'http://127.0.0.1:8080/api/v1/tasks/1?deleted_by=demo'
```

### 创建 release

```powershell
$body = @{
  app_name = 'api-server'
  version = 'v1.0.1'
  environment = 'dev'
  operator = 'demo'
  change_log = 'deploy api-server v1.0.1'
} | ConvertTo-Json

Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8080/api/v1/releases -ContentType 'application/json' -Body $body
```

### rollback release

```powershell
$body = @{
  operator = 'demo'
  change_log = 'rollback to release 1'
} | ConvertTo-Json

Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8080/api/v1/releases/1/rollback -ContentType 'application/json' -Body $body
```

## release updater CLI

CI/CD 和本地脚本都可以直接调用：

```powershell
go run .\cmd\release-updater -environment dev -app api-server -version v1.0.2
```

## Kubernetes Job container executor

- worker 会优先尝试使用集群内配置。
- 如果不在集群内，会回退到 `KUBECONFIG`。
- 如果两者都不存在，worker 仍会启动，但 `container` 任务会因 `kubernetes job runner is not configured` 失败。

相关环境变量：

- `KUBERNETES_NAMESPACE`
- `CONTAINER_JOB_POLL_INTERVAL`
- `CONTAINER_JOB_TTL_SECONDS`
- `CONTAINER_JOB_IMAGE_PULL_POLICY`
- `CONTAINER_JOB_SERVICE_ACCOUNT`

## K8s 与监控

- 基础资源：`deployments/k8s/base`
- 环境 overlay：`deployments/k8s/overlays/dev`、`deployments/k8s/overlays/prod`
- monitoring：`deployments/k8s/monitoring`
- Argo CD 示例：`deployments/argocd/*.yaml`

## CI/CD

GitHub Actions 工作流位于：

```text
.github/workflows/ci-cd.yaml
```

当前流程包含：

- `gofmt` 检查
- `go test ./...`
- 构建并推送四个服务镜像到 GHCR
- 调用 `cmd/release-updater` 更新 `dev` overlay
- 提交 `deployments/k8s/overlays/dev/patch-images.yaml`

## 验证命令

```powershell
go test -count=1 ./...
docker compose --env-file .env.example config
powershell -ExecutionPolicy Bypass -File .\scripts\build-linux.ps1
```

## 回滚

停止并清理本地环境：

```powershell
docker compose --env-file .env.example down -v
```

如果只需要回滚 GitOps patch：

```powershell
git checkout -- .\deployments\k8s\overlays\dev\patch-images.yaml
git checkout -- .\deployments\k8s\overlays\prod\patch-images.yaml
```

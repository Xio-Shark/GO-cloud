# Task Delete, Release Rollback And K8s Executor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 补齐任务删除、release GitOps 回滚链路、Kubernetes Job 版 container executor，以及对应部署与 CI/CD 清单。

**Architecture:** 保持现有 `repository -> service -> handler -> router -> bootstrap` 分层。任务删除采用软删除并隐藏已删除任务；release 通过 GitOps 文件更新器写回 `deployments/k8s/overlays/*` 的镜像 patch；container executor 通过 `client-go` 创建并等待 Kubernetes Job 完成。

**Tech Stack:** Go、Gin、GORM、Redis、MySQL、Kubernetes client-go、Docker Compose、Kustomize、GitHub Actions

---

### Task 1: 任务删除链路

**Files:**
- Modify: `internal/domain/task.go`
- Modify: `internal/repository/interfaces.go`
- Modify: `internal/repository/mysql/task_repository.go`
- Modify: `internal/service/task_service.go`
- Modify: `internal/transport/http/handler/task_handler.go`
- Modify: `internal/transport/http/router.go`
- Test: `internal/service/task_service_test.go`
- Test: `internal/transport/http/handler/task_handler_test.go`

**Step 1: Write the failing test**

- 新增 service 测试，覆盖：
  - 删除不存在任务返回 `not found`
  - 删除含 `pending/running` 执行的任务返回 `conflict`
  - 删除成功后任务状态变为 `deleted`
  - 默认任务列表不返回 `deleted`
- 新增 handler 测试，覆盖：
  - `DELETE /api/v1/tasks/:id` 返回 200
  - 非法 ID 返回 400
  - service `not found/conflict` 正确映射 404/409

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/service ./internal/transport/http/handler`
Expected: 因缺少 `DeleteTask`、`TaskStatusDeleted`、删除路由而失败

**Step 3: Write minimal implementation**

- 新增 `TaskStatusDeleted`
- `TaskService` 增加 `DeleteTask`
- `TaskRepository.List/GetByID` 支持隐藏 deleted
- 删除采用软删除，并阻止含 `pending/running` execution 的任务被删除

**Step 4: Run test to verify it passes**

Run: `go test -count=1 ./internal/service ./internal/transport/http/handler`
Expected: PASS

### Task 2: release GitOps 发布与回滚链路

**Files:**
- Modify: `internal/dto/release.go`
- Modify: `internal/domain/release.go`
- Modify: `internal/repository/interfaces.go`
- Modify: `internal/repository/mysql/release_repository.go`
- Create: `internal/gitops/updater.go`
- Create: `internal/gitops/file_updater.go`
- Modify: `internal/service/release_service.go`
- Modify: `internal/transport/http/handler/release_handler.go`
- Modify: `internal/transport/http/router.go`
- Modify: `internal/bootstrap/config.go`
- Modify: `internal/bootstrap/app.go`
- Test: `internal/service/release_service_test.go`
- Test: `internal/transport/http/handler/release_handler_test.go`
- Test: `internal/gitops/file_updater_test.go`

**Step 1: Write the failing test**

- release service 测试覆盖：
  - `CreateRelease` 调用 GitOps 更新器后创建 `deployed/failed` 记录
  - `RollbackRelease` 以目标 release 的版本写回 overlay，并创建 `rolled_back` 记录
- file updater 测试覆盖：
  - 写入 `patch-images.yaml`
  - 不存在环境目录时报错
- handler 测试覆盖：
  - `POST /api/v1/releases/:id/rollback` 返回 200/404/409

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/service ./internal/transport/http/handler ./internal/gitops`
Expected: 因缺少 rollback API 和 updater 实现失败

**Step 3: Write minimal implementation**

- 为 release service 注入 GitOps updater
- `POST /releases` 不再只写数据库，而是先更新 overlay 镜像，再落成功/失败记录
- `POST /releases/:id/rollback` 根据目标 release 版本更新 overlay，并新建 `rolled_back` 记录

**Step 4: Run test to verify it passes**

Run: `go test -count=1 ./internal/service ./internal/transport/http/handler ./internal/gitops`
Expected: PASS

### Task 3: Kubernetes Job container executor

**Files:**
- Modify: `internal/domain/task.go`
- Create: `internal/executor/container_executor.go`
- Modify: `internal/executor/executor.go`
- Modify: `internal/service/task_service.go`
- Modify: `cmd/worker/main.go`
- Modify: `internal/bootstrap/config.go`
- Test: `internal/executor/container_executor_test.go`
- Test: `internal/service/worker_service_test.go`
- Test: `internal/service/task_service_test.go`

**Step 1: Write the failing test**

- container executor 测试覆盖：
  - payload 解析失败
  - 创建 Job 成功并等待完成
  - Job 失败时返回 exitCode 和错误信息
- worker 测试覆盖：
  - `task_type=container` 选择 container executor
- task service 测试覆盖：
  - `container` 任务允许创建

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/executor ./internal/service`
Expected: 因缺少 `container` 类型实现失败

**Step 3: Write minimal implementation**

- 任务类型增加 `container`
- executor 使用 `client-go` 创建 `batch/v1 Job`
- 使用 `KUBERNETES_NAMESPACE`、`KUBECONFIG`、`CONTAINER_EXECUTOR_*` 环境变量控制行为

**Step 4: Run test to verify it passes**

Run: `go test -count=1 ./internal/executor ./internal/service`
Expected: PASS

### Task 4: Compose、Nginx、Grafana、K8s monitoring 与 CI/CD

**Files:**
- Modify: `.env.example`
- Modify: `docker-compose.yml`
- Create: `deployments/nginx/default.conf`
- Create: `deployments/grafana/provisioning/datasources/prometheus.yml`
- Create: `deployments/grafana/provisioning/dashboards/dashboard.yml`
- Create: `deployments/grafana/dashboards/go-cloud-overview.json`
- Modify: `deployments/k8s/base/kustomization.yaml`
- Create: `deployments/k8s/base/ingress.yaml`
- Create: `deployments/k8s/base/hpa-worker.yaml`
- Create: `deployments/k8s/overlays/dev/patch-images.yaml`
- Create: `deployments/k8s/overlays/prod/patch-images.yaml`
- Create: `deployments/k8s/monitoring/servicemonitor.yaml`
- Create: `deployments/k8s/monitoring/prometheus-rule.yaml`
- Create: `.github/workflows/ci-cd.yaml`
- Modify: `Dockerfile`
- Modify: `scripts/build-linux.ps1`
- Modify: `README.md`

**Step 1: Write the failing test**

- 这部分以配置校验代替单元测试：
  - `docker compose --env-file .env.example config`
  - 如可用则执行 `kubectl kustomize deployments/k8s/overlays/dev`

**Step 2: Run test to verify it fails**

Run: `docker compose --env-file .env.example config`
Expected: 若缺少新增挂载或变量，配置校验失败

**Step 3: Write minimal implementation**

- Compose 新增 `nginx/grafana`
- API 容器挂载 `deployments` 目录供 GitOps 更新器使用
- K8s base 补 ingress/HPA，monitoring 目录补 `ServiceMonitor/PrometheusRule`
- GitHub Actions 补 lint、test、build、image build、manifest update

**Step 4: Run test to verify it passes**

Run: `docker compose --env-file .env.example config`
Expected: PASS

### Task 5: 全量验证

**Files:**
- Verify only

**Step 1: Run format**

Run: `gofmt -w cmd internal pkg`

**Step 2: Run tests**

Run: `go test -count=1 ./...`

**Step 3: Run build**

Run: `powershell -ExecutionPolicy Bypass -File .\scripts\build-linux.ps1`

**Step 4: Run compose validation**

Run: `docker compose --env-file .env.example config`

**Step 5: Run runtime regression**

Run:
- `docker compose --env-file .env.example up -d --build`
- 验证 `DELETE /api/v1/tasks/:id`
- 验证 `POST /api/v1/releases`
- 验证 `POST /api/v1/releases/:id/rollback`

**Step 6: Verify outputs**

Expected:
- 所有 go test 通过
- Linux 二进制构建通过
- compose 配置与服务健康检查通过
- GitOps overlay 文件被真实写回
- release deploy/rollback API 真实返回成功

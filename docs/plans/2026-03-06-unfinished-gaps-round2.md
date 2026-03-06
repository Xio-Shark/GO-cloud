# Unfinished Gaps Round 2 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 修补当前最明确且可独立落地的未完成部分，补齐健康检查分层、执行记录列表与取消、`release_records` 基础 API。

**Architecture:** 保持现有 `repository -> service -> handler -> router -> bootstrap` 分层。健康检查改成 transport 依赖 checker 抽象，执行取消只支持 `pending` 状态，`release_records` 先补最小闭环的创建/查询链路，不引入 GitOps 外部联动。

**Tech Stack:** Go, Gin, GORM, MySQL, Redis, Docker Compose, Prometheus

---

### Task 1: Health Checker Decouple

**Files:**
- Modify: `internal/healthcheck/dependencies.go`
- Modify: `internal/transport/http/handler/health_handler.go`
- Modify: `internal/bootstrap/app.go`
- Test: `internal/transport/http/handler/health_handler_test.go`

**Step 1: Write the failing test**

- 为 `HealthHandler` 增加基于 checker stub 的测试，验证健康和失败分支。

**Step 2: Run test to verify it fails**

- 运行 `go test -count=1 ./internal/transport/http/handler`
- 预期：因 `HealthHandler` 构造签名不匹配或测试引用缺失而失败。

**Step 3: Write minimal implementation**

- 在 `internal/healthcheck` 中定义 checker 抽象和依赖检查构造器。
- `HealthHandler` 改为仅依赖 checker。
- `bootstrap` 注入 checker 实现。

**Step 4: Run test to verify it passes**

- 运行 `go test -count=1 ./internal/transport/http/handler`

### Task 2: Execution List And Cancel

**Files:**
- Modify: `internal/repository/interfaces.go`
- Modify: `internal/repository/mysql/execution_repository.go`
- Modify: `internal/service/execution_service.go`
- Modify: `internal/service/worker_service.go`
- Modify: `internal/transport/http/handler/execution_handler.go`
- Modify: `internal/transport/http/router.go`
- Test: `internal/service/execution_service_test.go`
- Test: `internal/service/worker_service_test.go`
- Test: `internal/transport/http/handler/execution_handler_test.go`

**Step 1: Write the failing test**

- 增加 `ListExecutions` 过滤测试。
- 增加 `CancelExecution` 仅允许 `pending` 的测试。
- 增加 worker 遇到已取消 execution 时跳过执行的测试。
- 增加 handler 对新增接口的状态码测试。

**Step 2: Run test to verify it fails**

- 运行：
  - `go test -count=1 ./internal/service`
  - `go test -count=1 ./internal/transport/http/handler`

**Step 3: Write minimal implementation**

- repository 增加 execution 列表过滤能力。
- service 暴露 `ListExecutions` 和 `CancelExecution`。
- worker 消费前检查 execution 状态，已取消则直接跳过。
- router 暴露：
  - `GET /api/v1/executions`
  - `POST /api/v1/executions/:execution_no/cancel`

**Step 4: Run test to verify it passes**

- 再次运行 service 与 handler 测试。

### Task 3: Release Records Minimum Chain

**Files:**
- Add: `internal/dto/release.go`
- Add: `internal/repository/mysql/release_repository.go`
- Add: `internal/service/release_service.go`
- Add: `internal/transport/http/handler/release_handler.go`
- Modify: `internal/repository/interfaces.go`
- Modify: `internal/transport/http/router.go`
- Modify: `internal/bootstrap/app.go`
- Test: `internal/service/release_service_test.go`
- Test: `internal/transport/http/handler/release_handler_test.go`

**Step 1: Write the failing test**

- 新增 release 创建参数校验测试。
- 新增 release 列表与详情 handler 测试。

**Step 2: Run test to verify it fails**

- 运行：
  - `go test -count=1 ./internal/service`
  - `go test -count=1 ./internal/transport/http/handler`

**Step 3: Write minimal implementation**

- repository 提供 `Create/GetByID/List`。
- service 提供 `CreateRelease/GetRelease/ListReleases`，默认 `status=pending`。
- handler 与 router 暴露：
  - `POST /api/v1/releases`
  - `GET /api/v1/releases`
  - `GET /api/v1/releases/:id`

**Step 4: Run test to verify it passes**

- 再次运行 service 与 handler 测试。

### Task 4: Full Verification

**Files:**
- Verify only

**Step 1: Run full test suite**

- `go test -count=1 ./...`

**Step 2: Run compose validation**

- `docker compose --env-file .env.example config`

**Step 3: Run runtime smoke checks**

- 校验 `/healthz`、`/readyz`、`/metrics`
- 校验新增 `/api/v1/executions`、`/api/v1/executions/:execution_no/cancel`、`/api/v1/releases`

**Step 4: Document residual gaps**

- 明确本轮未处理项：
  - `DELETE /tasks/:id`
  - `container executor`
  - GitOps rollback 真实联动

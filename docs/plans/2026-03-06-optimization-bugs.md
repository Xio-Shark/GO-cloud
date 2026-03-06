# Optimization Bugs Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 修复当前高优先级接口行为缺陷，保证任务创建、状态流转、重试和执行入队具备基本正确性。

**Architecture:** 保持现有 `handler -> service -> repository` 分层不变，只在 DTO 校验、service 事务边界、handler 错误映射处做最小改动。先通过 service 层测试锁定行为，再补 handler 层测试覆盖 HTTP 返回码和请求校验。

**Tech Stack:** Go, Gin, GORM, Redis, MySQL, testing

---

### Task 1: 补任务服务失败测试

**Files:**
- Modify: `internal/service/task_service_test.go`
- Modify: `internal/service/execution_service.go`
- Modify: `internal/service/task_service.go`

**Step 1: Write the failing test**

- 增加 `once` 任务缺少 `run_at` 时创建失败
- 增加空 `task_type` 创建失败
- 增加暂停不存在任务返回错误

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/service`
Expected: 新增用例失败，错误原因为当前实现未做校验或不存在性检查

**Step 3: Write minimal implementation**

- 在 service 层集中补任务创建校验
- 在 pause/resume 前读取任务并校验存在

**Step 4: Run test to verify it passes**

Run: `go test -count=1 ./internal/service`
Expected: 新增用例通过

### Task 2: 补执行服务失败测试

**Files:**
- Modify: `internal/service/worker_service_test.go`
- Modify: `internal/service/execution_service.go`

**Step 1: Write the failing test**

- 增加成功执行不允许 retry
- 增加 paused task 不允许 retry

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/service`
Expected: 新增用例失败

**Step 3: Write minimal implementation**

- 在 `RetryExecution` 中校验 execution 状态和 task 状态

**Step 4: Run test to verify it passes**

Run: `go test -count=1 ./internal/service`
Expected: 新增用例通过

### Task 3: 补 handler 返回码和请求校验测试

**Files:**
- Create: `internal/transport/http/handler/task_handler_test.go`
- Create: `internal/transport/http/handler/execution_handler_test.go`
- Modify: `internal/transport/http/handler/task_handler.go`
- Modify: `internal/transport/http/handler/execution_handler.go`

**Step 1: Write the failing test**

- 不存在任务 pause 返回 404
- 非活动任务 trigger 返回 409
- 非法创建请求返回 400
- 成功执行 retry 返回 409

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/transport/http/handler`
Expected: 新增用例失败

**Step 3: Write minimal implementation**

- 统一业务错误分类
- handler 按错误类型返回 400/404/409

**Step 4: Run test to verify it passes**

Run: `go test -count=1 ./internal/transport/http/handler`
Expected: 新增用例通过

### Task 4: 补执行创建与入队原子性保护

**Files:**
- Modify: `internal/service/task_service.go`
- Modify: `internal/service/execution_service.go`
- Modify: `internal/service/scheduler_service.go`
- Modify: `internal/repository/interfaces.go`
- Modify: `internal/repository/mysql/execution_repository.go`

**Step 1: Write the failing test**

- 模拟入队失败，断言不会保留新 execution

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/service`
Expected: 新增用例失败

**Step 3: Write minimal implementation**

- 为 execution repository 增加按 `execution_no` 删除能力
- 在创建 execution 后入队失败时做补偿删除

**Step 4: Run test to verify it passes**

Run: `go test -count=1 ./internal/service`
Expected: 新增用例通过

### Task 5: 做完整回归验证

**Files:**
- Modify: `README.md`（如接口行为或错误码说明需要同步）

**Step 1: Run unit tests**

Run: `go test -count=1 ./...`
Expected: 全部通过

**Step 2: Run key HTTP verification**

Run: 本地 `Invoke-WebRequest`/`Invoke-RestMethod` 脚本
Expected: 404/400/409 行为符合修复目标

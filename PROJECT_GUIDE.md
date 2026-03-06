# PROJECT_GUIDE

## 1. 项目定位

项目名称：**基于 Go 与 Kubernetes 的分布式任务调度与发布平台**

这是一个面向简历和面试展示的后端 / 云原生平台型项目，目标不是做一个简单 CRUD demo，而是做出能够体现以下能力的工程化项目：

- Go 后端分层设计能力
- 调度系统设计能力
- 分布式任务执行链路设计能力
- Redis 队列与分布式锁应用能力
- MySQL 数据建模能力
- 失败重试、通知回调、幂等控制等工程细节处理能力
- Docker Compose 本地编排能力
- Kubernetes 部署能力
- GitOps / Argo CD 持续交付能力
- Prometheus / Grafana 监控能力

适用岗位方向：

- 运维开发
- DevOps 工程师
- 云原生工程师
- 平台工程师
- 初级 SRE / 后端平台开发

---

## 2. 项目目标

实现一个可持续演进的平台，核心能力包括：

1. 任务管理
   - 创建任务
   - 更新任务
   - 查询任务
   - 暂停 / 恢复任务
   - 手动触发任务

2. 调度派发
   - scheduler 周期扫描到期任务
   - 创建 execution
   - 将 execution 写入 Redis 队列
   - 使用 Redis 分布式锁避免重复派发

3. 执行消费
   - worker 从 Redis 队列消费任务
   - 按任务类型调用对应 executor
   - 更新 execution 状态与日志

4. 执行记录
   - 记录 pending / running / success / failed / timeout / cancelled 等状态
   - 支持 execution 查询、日志查询、失败重试

5. 通知机制
   - 任务执行完成后触发 callback / notifier
   - 支持 success / failed 等状态通知

6. 监控与观测
   - 暴露 HTTP 请求指标
   - 暴露 scheduler 扫描 / 派发指标
   - 暴露 worker 成功 / 失败 / 重试 / 耗时指标

7. 部署交付
   - 本地使用 Docker Compose 启动完整依赖
   - 提供 Kubernetes manifests
   - 支持 GitOps / Argo CD 自动部署

---

## 3. 技术栈

### 后端与基础设施
- Go
- Gin
- GORM
- MySQL
- Redis

### 容器与部署
- Docker
- Docker Compose
- Kubernetes
- Nginx
- Argo CD

### 观测与监控
- Prometheus
- Grafana

---

## 4. 核心服务划分

### 4.1 api-server
负责提供 HTTP API，包括：

- 任务创建、更新、查询、暂停、恢复、手动触发
- execution 查询
- execution 日志查询
- 重试接口
- 健康检查
- metrics 暴露

### 4.2 scheduler
负责：

- 周期扫描到期任务
- 获取分布式锁
- 创建 execution
- 入队
- 更新任务的 last_run_time / next_run_time

### 4.3 worker
负责：

- 消费 Redis 队列
- 加载 execution 与 task 信息
- 根据 task type 选择 executor
- 更新 execution 状态
- 失败重试
- 上报 metrics
- 执行完成后触发 notifier

### 4.4 notifier
负责：

- webhook / callback 通知
- 后续可扩展为飞书 / 钉钉 / 企业微信 / 邮件通知

### 4.5 dashboard（可选）
用于展示：

- 任务列表
- 执行记录
- 失败情况
- 指标概览

---

## 5. 核心数据模型

---

### 5.1 tasks

表示任务定义。

关键字段建议：

- `id`
- `name`
- `type`：任务类型，如 shell / http
- `cron_expr`
- `status`：active / paused
- `payload`
- `timeout_seconds`
- `retry_max`
- `retry_interval_seconds`
- `callback_url`
- `last_run_time`
- `next_run_time`
- `created_at`
- `updated_at`

说明：

- `cron_expr` 用于周期任务调度
- `next_run_time` 是 scheduler 扫描的关键字段
- 创建 cron 任务时应初始化 `next_run_time`

---

### 5.2 task_executions

表示某次具体执行实例。

关键字段建议：

- `id`
- `task_id`
- `status`：pending / running / success / failed / timeout / cancelled
- `trigger_type`：schedule / manual / retry
- `worker_id`
- `retry_count`
- `start_time`
- `end_time`
- `error_message`
- `output_log`
- `created_at`
- `updated_at`

说明：

- execution 是任务实例，不等同于 task
- worker 以 execution 为执行单位

---

### 5.3 release_records

表示发布流水，可作为平台扩展能力。

关键字段建议：

- `id`
- `app_name`
- `version`
- `environment`
- `status`
- `operator`
- `change_log`
- `created_at`
- `updated_at`

说明：

- 当前项目主链路重心在调度与执行
- release_records 可作为后续“发布平台化”方向延伸

---

## 6. Redis 设计

### 6.1 队列
建议使用：

- `queue:task:execution`

存放待执行 execution id。

### 6.2 分布式锁
建议使用：

- `lock:scheduler:task:{taskID}`

用于确保同一任务在同一调度周期不会被重复派发。

### 6.3 短期状态缓存（可选）
可用于：

- worker 心跳
- 执行中任务状态
- 限流 / 去重信息

---

## 7. API 设计范围

---

### 7.1 任务接口
- `POST /api/v1/tasks`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/:id`
- `PUT /api/v1/tasks/:id`
- `POST /api/v1/tasks/:id/pause`
- `POST /api/v1/tasks/:id/resume`
- `POST /api/v1/tasks/:id/trigger`

### 7.2 执行记录接口
- `GET /api/v1/executions`
- `GET /api/v1/executions/:id`
- `GET /api/v1/executions/:id/log`
- `POST /api/v1/executions/:id/retry`

### 7.3 健康检查与监控
- `GET /healthz`
- `GET /metrics`

---

## 8. 工程目录结构

采用：

- `cmd/ + internal/ + pkg`

推荐最终结构如下：

```text
.
├── cmd
│   ├── api-server
│   │   └── main.go
│   ├── scheduler
│   │   └── main.go
│   ├── worker
│   │   └── main.go
│   └── notifier
│       └── main.go
├── internal
│   ├── bootstrap
│   │   ├── app.go
│   │   ├── config.go
│   │   ├── db.go
│   │   ├── redis.go
│   │   ├── logger.go
│   │   └── metrics.go
│   ├── domain
│   │   ├── task.go
│   │   ├── execution.go
│   │   └── release.go
│   ├── dto
│   │   ├── task.go
│   │   ├── execution.go
│   │   └── common.go
│   ├── repository
│   │   ├── model
│   │   │   ├── task.go
│   │   │   ├── execution.go
│   │   │   └── release.go
│   │   ├── task_repository.go
│   │   ├── execution_repository.go
│   │   ├── queue_repository.go
│   │   ├── lock_repository.go
│   │   └── release_repository.go
│   ├── service
│   │   ├── task_service.go
│   │   ├── execution_service.go
│   │   ├── scheduler_service.go
│   │   ├── worker_service.go
│   │   ├── retry_service.go
│   │   └── notifier_service.go
│   ├── executor
│   │   ├── executor.go
│   │   ├── shell_executor.go
│   │   └── http_executor.go
│   ├── notifier
│   │   ├── notifier.go
│   │   └── webhook_notifier.go
│   ├── transport
│   │   └── http
│   │       ├── router.go
│   │       ├── response
│   │       │   └── response.go
│   │       └── handler
│   │           ├── health_handler.go
│   │           ├── task_handler.go
│   │           └── execution_handler.go
│   ├── metrics
│   │   ├── http.go
│   │   ├── scheduler.go
│   │   └── worker.go
│   └── pkg
│       ├── cronutil
│       ├── pointer
│       └── timeutil
├── deployments
│   ├── docker-compose.yml
│   ├── k8s
│   │   ├── base
│   │   └── overlays
│   └── argocd
└── Makefile
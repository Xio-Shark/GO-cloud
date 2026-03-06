# Go Cloud V1 Design

## 目标

在空仓库中落地一个可运行的 Go 调度平台 V1，覆盖任务管理、调度派发、执行消费、失败重试、Webhook 通知、Prometheus 指标、Docker Compose 本地编排，以及 Kubernetes/Argo CD 基础清单。

## 范围

- `api-server`：任务 CRUD、暂停/恢复、手动触发、执行记录查询、健康检查、指标暴露
- `scheduler`：扫描到期任务、加锁、防重复派发、创建 execution、入队、推进 `next_run_time`
- `worker`：消费队列、执行 shell/http 任务、更新 execution、失败重试、写通知队列
- `notifier`：消费通知队列、发送 webhook、记录结构化日志、暴露健康检查与指标
- 基础设施：MySQL、Redis、Prometheus、Dockerfile、`docker-compose.yml`、`.env.example`
- 交付：K8s `base + overlays`、Argo CD Application 样例

## 设计决策

### 方案 A：单体 API 内嵌调度与执行

优点：实现快、文件少。  
缺点：无法体现平台拆分、队列链路、服务职责边界，不符合项目目标。

### 方案 B：四服务拆分，通知异步化

优点：与文档一致，链路完整，能体现 Redis 队列、调度/执行/通知分层，适合简历展示。  
缺点：代码量更大，需要补更多启动与配置代码。

### 方案 C：先做 API + worker，后补 scheduler/notifier

优点：最省时间。  
缺点：主链路不闭环，Docker/K8s 清单会出现占位服务。

## 采用方案

采用方案 B，但只做 V1 最小闭环：

- 只支持 `shell` 与 `http` 两类任务
- `release_records` 仅提供表结构与模型，不进入本轮主链路
- notifier 只实现 webhook 回调
- metrics 先覆盖 HTTP、scheduler、worker、notifier 核心指标

## 架构

依赖方向保持：

`transport/http -> service -> repository -> storage`

`worker/scheduler/notifier main -> bootstrap -> service -> repository`

`repository` 输出 `domain` 对象，不向 `service` 暴露 GORM model。  
日志统一使用 `slog` JSON handler，通过 `context` 透传 `trace_id`。  
外部 HTTP 请求统一带超时。  
Redis 分三类 key：

- `queue:task:execution`
- `queue:task:notification`
- `lock:scheduler:task:{id}`

## 测试策略

- 先写服务级失败测试
- 用内存 stub 验证 `TaskService` 创建 cron 任务时初始化 `next_run_time`
- 用内存 stub 验证 `SchedulerService` 只派发到期任务并推进下次执行时间
- 用内存 stub 验证 `WorkerService` 执行失败时创建 retry execution 并推送通知

## 非目标

- 分布式多 worker 负载均衡优化
- DAG 编排
- Flyway/Liquibase 式迁移系统
- 前端 dashboard

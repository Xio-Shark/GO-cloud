下面直接给你一份**可开工版**。
我按这 4 部分展开：

1. 数据库字段说明
2. API 请求 / 响应示例
3. Docker Compose 样例结构
4. Kubernetes YAML 清单模板

我会尽量保持你拿去就能建项目。

---

# 一、数据库字段说明

建议先保留 3 张核心表：

* `tasks`：任务定义
* `task_executions`：任务执行记录
* `release_records`：发布记录

后面你再按需要扩展。

---

## 1. `tasks` 表

这个表表示“任务本身是什么”。

### 建表 SQL

```sql
CREATE TABLE tasks (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '任务ID',
  name VARCHAR(128) NOT NULL COMMENT '任务名称',
  description VARCHAR(255) DEFAULT '' COMMENT '任务描述',
  task_type VARCHAR(32) NOT NULL COMMENT '任务类型：shell/http/container',
  schedule_type VARCHAR(32) NOT NULL COMMENT '调度类型：manual/once/cron',
  cron_expr VARCHAR(64) DEFAULT NULL COMMENT 'cron表达式',
  payload JSON DEFAULT NULL COMMENT '任务执行参数',
  timeout_seconds INT NOT NULL DEFAULT 60 COMMENT '超时时间，单位秒',
  retry_times INT NOT NULL DEFAULT 0 COMMENT '失败重试次数',
  status VARCHAR(32) NOT NULL DEFAULT 'active' COMMENT '任务状态：active/paused/deleted',
  callback_url VARCHAR(255) DEFAULT NULL COMMENT '回调地址',
  created_by VARCHAR(64) DEFAULT '' COMMENT '创建人',
  updated_by VARCHAR(64) DEFAULT '' COMMENT '更新人',
  next_run_time DATETIME DEFAULT NULL COMMENT '下次执行时间',
  last_run_time DATETIME DEFAULT NULL COMMENT '上次执行时间',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  INDEX idx_status_next_run (status, next_run_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务定义表';
```

### 字段说明

| 字段              | 类型           | 含义       | 示例                             |
| --------------- | ------------ | -------- | ------------------------------ |
| id              | BIGINT       | 主键 ID    | 1                              |
| name            | VARCHAR(128) | 任务名      | `daily-report-job`             |
| description     | VARCHAR(255) | 描述       | `generate daily report`        |
| task_type       | VARCHAR(32)  | 执行器类型    | `shell` / `http` / `container` |
| schedule_type   | VARCHAR(32)  | 调度类型     | `manual` / `once` / `cron`     |
| cron_expr       | VARCHAR(64)  | cron 表达式 | `*/5 * * * *`                  |
| payload         | JSON         | 执行参数     | shell 命令、HTTP 请求参数等            |
| timeout_seconds | INT          | 超时时间     | 30                             |
| retry_times     | INT          | 重试次数     | 2                              |
| status          | VARCHAR(32)  | 当前状态     | `active`                       |
| callback_url    | VARCHAR(255) | 回调 URL   | `http://example.com/callback`  |
| created_by      | VARCHAR(64)  | 创建人      | `admin`                        |
| updated_by      | VARCHAR(64)  | 更新人      | `admin`                        |
| next_run_time   | DATETIME     | 下次执行时间   | `2026-03-06 12:00:00`          |
| last_run_time   | DATETIME     | 上次执行时间   | `2026-03-06 11:55:00`          |
| created_at      | DATETIME     | 创建时间     | 自动生成                           |
| updated_at      | DATETIME     | 更新时间     | 自动生成                           |

### payload 设计建议

#### shell 类型

```json
{
  "command": "echo hello && sleep 2",
  "workdir": "/tmp",
  "env": {
    "APP_ENV": "dev"
  }
}
```

#### http 类型

```json
{
  "method": "POST",
  "url": "http://example.com/hook",
  "headers": {
    "Authorization": "Bearer token"
  },
  "body": "{\"msg\":\"hello\"}"
}
```

#### container 类型

```json
{
  "image": "busybox:latest",
  "command": ["sh", "-c", "echo run task"],
  "env": {
    "ENV": "dev"
  }
}
```

---

## 2. `task_executions` 表

这个表表示“某次任务执行的结果”。

### 建表 SQL

```sql
CREATE TABLE task_executions (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '执行记录ID',
  task_id BIGINT NOT NULL COMMENT '任务ID',
  execution_no VARCHAR(64) NOT NULL COMMENT '执行编号',
  trigger_type VARCHAR(32) NOT NULL COMMENT '触发方式：manual/scheduler/retry',
  worker_id VARCHAR(64) DEFAULT '' COMMENT '执行worker标识',
  status VARCHAR(32) NOT NULL COMMENT '执行状态：pending/running/success/failed/timeout/cancelled',
  start_time DATETIME DEFAULT NULL COMMENT '开始时间',
  end_time DATETIME DEFAULT NULL COMMENT '结束时间',
  duration_ms BIGINT DEFAULT 0 COMMENT '执行耗时，毫秒',
  retry_count INT DEFAULT 0 COMMENT '当前重试次数',
  exit_code INT DEFAULT NULL COMMENT '退出码',
  error_message TEXT COMMENT '错误信息',
  output_log MEDIUMTEXT COMMENT '输出日志',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  UNIQUE KEY uk_execution_no (execution_no),
  INDEX idx_task_id (task_id),
  INDEX idx_status_created_at (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务执行记录表';
```

### 字段说明

| 字段            | 类型          | 含义      | 示例                           |
| ------------- | ----------- | ------- | ---------------------------- |
| id            | BIGINT      | 主键      | 1                            |
| task_id       | BIGINT      | 对应任务 ID | 1001                         |
| execution_no  | VARCHAR(64) | 唯一执行编号  | `exec_20260306120000_abc123` |
| trigger_type  | VARCHAR(32) | 触发来源    | `scheduler`                  |
| worker_id     | VARCHAR(64) | 执行节点    | `worker-1`                   |
| status        | VARCHAR(32) | 执行状态    | `running` / `success`        |
| start_time    | DATETIME    | 开始时间    | `2026-03-06 12:00:00`        |
| end_time      | DATETIME    | 结束时间    | `2026-03-06 12:00:05`        |
| duration_ms   | BIGINT      | 耗时毫秒    | 5000                         |
| retry_count   | INT         | 当前重试次数  | 1                            |
| exit_code     | INT         | 退出码     | 0                            |
| error_message | TEXT        | 错误信息    | `connection refused`         |
| output_log    | MEDIUMTEXT  | 执行输出    | 命令输出文本                       |
| created_at    | DATETIME    | 创建时间    | 自动生成                         |
| updated_at    | DATETIME    | 更新时间    | 自动生成                         |

---

## 3. `release_records` 表

这个表用于和发布链路对接，尤其适合 Argo CD / GitOps。

### 建表 SQL

```sql
CREATE TABLE release_records (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '发布记录ID',
  env VARCHAR(32) NOT NULL COMMENT '环境：dev/staging/prod',
  app_name VARCHAR(64) NOT NULL COMMENT '应用名',
  image_tag VARCHAR(128) NOT NULL COMMENT '镜像版本',
  git_commit VARCHAR(64) DEFAULT '' COMMENT 'git提交哈希',
  operator VARCHAR(64) DEFAULT '' COMMENT '操作人',
  status VARCHAR(32) NOT NULL COMMENT '状态：pending/deployed/failed/rolled_back',
  message VARCHAR(255) DEFAULT '' COMMENT '附加说明',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  INDEX idx_env_app_created (env, app_name, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='发布记录表';
```

### 字段说明

| 字段         | 类型           | 含义   | 示例                  |
| ---------- | ------------ | ---- | ------------------- |
| id         | BIGINT       | 主键   | 1                   |
| env        | VARCHAR(32)  | 环境   | `dev`               |
| app_name   | VARCHAR(64)  | 应用名  | `go-job-platform`   |
| image_tag  | VARCHAR(128) | 镜像版本 | `v1.0.3`            |
| git_commit | VARCHAR(64)  | 提交哈希 | `abc123def456`      |
| operator   | VARCHAR(64)  | 操作人  | `charles`           |
| status     | VARCHAR(32)  | 发布状态 | `deployed`          |
| message    | VARCHAR(255) | 说明   | `argo sync success` |
| created_at | DATETIME     | 时间   | 自动生成                |

---

# 二、API 请求 / 响应示例

建议统一返回格式。

## 统一响应格式

### 成功响应

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

### 失败响应

```json
{
  "code": 40001,
  "message": "invalid request",
  "data": null
}
```

---

## 1. 创建任务

### 接口

`POST /api/v1/tasks`

### 请求体

```json
{
  "name": "daily-report-job",
  "description": "generate daily report",
  "task_type": "shell",
  "schedule_type": "cron",
  "cron_expr": "*/5 * * * *",
  "payload": {
    "command": "echo hello && sleep 2",
    "workdir": "/tmp",
    "env": {
      "APP_ENV": "dev"
    }
  },
  "timeout_seconds": 30,
  "retry_times": 2,
  "callback_url": "http://example.com/callback",
  "created_by": "admin"
}
```

### 成功响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "daily-report-job",
    "status": "active",
    "next_run_time": "2026-03-06T12:05:00Z",
    "created_at": "2026-03-06T12:00:00Z"
  }
}
```

---

## 2. 查询任务列表

### 接口

`GET /api/v1/tasks?page=1&page_size=10&status=active`

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "name": "daily-report-job",
        "task_type": "shell",
        "schedule_type": "cron",
        "status": "active",
        "next_run_time": "2026-03-06T12:05:00Z",
        "last_run_time": "2026-03-06T12:00:00Z"
      }
    ],
    "page": 1,
    "page_size": 10,
    "total": 1
  }
}
```

---

## 3. 查询单个任务详情

### 接口

`GET /api/v1/tasks/1`

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "daily-report-job",
    "description": "generate daily report",
    "task_type": "shell",
    "schedule_type": "cron",
    "cron_expr": "*/5 * * * *",
    "payload": {
      "command": "echo hello && sleep 2",
      "workdir": "/tmp",
      "env": {
        "APP_ENV": "dev"
      }
    },
    "timeout_seconds": 30,
    "retry_times": 2,
    "status": "active",
    "callback_url": "http://example.com/callback",
    "next_run_time": "2026-03-06T12:05:00Z",
    "last_run_time": "2026-03-06T12:00:00Z",
    "created_at": "2026-03-06T11:00:00Z",
    "updated_at": "2026-03-06T12:00:00Z"
  }
}
```

---

## 4. 更新任务

### 接口

`PUT /api/v1/tasks/1`

### 请求体

```json
{
  "description": "generate daily report v2",
  "cron_expr": "*/10 * * * *",
  "timeout_seconds": 60,
  "retry_times": 3,
  "updated_by": "admin"
}
```

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "updated": true
  }
}
```

---

## 5. 暂停任务

### 接口

`POST /api/v1/tasks/1/pause`

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "status": "paused"
  }
}
```

---

## 6. 恢复任务

### 接口

`POST /api/v1/tasks/1/resume`

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "status": "active"
  }
}
```

---

## 7. 手动触发任务

### 接口

`POST /api/v1/tasks/1/trigger`

### 请求体

```json
{
  "trigger_by": "admin"
}
```

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": 1,
    "execution_no": "exec_20260306121000_abcd1234",
    "status": "pending"
  }
}
```

---

## 8. 查询某任务执行记录

### 接口

`GET /api/v1/tasks/1/executions?page=1&page_size=10`

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "execution_no": "exec_20260306121000_abcd1234",
        "trigger_type": "manual",
        "worker_id": "worker-1",
        "status": "success",
        "start_time": "2026-03-06T12:10:00Z",
        "end_time": "2026-03-06T12:10:03Z",
        "duration_ms": 3000
      }
    ],
    "page": 1,
    "page_size": 10,
    "total": 1
  }
}
```

---

## 9. 查询执行详情

### 接口

`GET /api/v1/executions/exec_20260306121000_abcd1234`

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "execution_no": "exec_20260306121000_abcd1234",
    "task_id": 1,
    "trigger_type": "manual",
    "worker_id": "worker-1",
    "status": "success",
    "retry_count": 0,
    "exit_code": 0,
    "error_message": "",
    "start_time": "2026-03-06T12:10:00Z",
    "end_time": "2026-03-06T12:10:03Z",
    "duration_ms": 3000
  }
}
```

---

## 10. 查询执行日志

### 接口

`GET /api/v1/executions/exec_20260306121000_abcd1234/logs`

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "execution_no": "exec_20260306121000_abcd1234",
    "logs": "hello\njob finished\n"
  }
}
```

---

## 11. 重试执行

### 接口

`POST /api/v1/executions/exec_20260306121000_abcd1234/retry`

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "old_execution_no": "exec_20260306121000_abcd1234",
    "new_execution_no": "exec_20260306121200_efgh5678",
    "status": "pending"
  }
}
```

---

## 12. 查询发布记录

### 接口

`GET /api/v1/releases?env=dev`

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "env": "dev",
        "app_name": "go-job-platform",
        "image_tag": "v1.0.3",
        "git_commit": "abc123def456",
        "operator": "charles",
        "status": "deployed",
        "message": "argo sync success",
        "created_at": "2026-03-06T12:20:00Z"
      }
    ]
  }
}
```

---

# 三、Docker Compose 样例结构

建议你的 Compose 分成：

* 开发环境 `docker-compose.dev.yml`
* 演示 / 近生产环境 `docker-compose.prod.yml`

---

## 1. 目录结构

```text
deployments/docker-compose/
├── .env.example
├── docker-compose.dev.yml
├── docker-compose.prod.yml
├── mysql/
│   └── init.sql
├── nginx/
│   └── default.conf
├── prometheus/
│   └── prometheus.yml
└── grafana/
    └── provisioning/
```

---

## 2. `.env.example`

```env
MYSQL_ROOT_PASSWORD=root
MYSQL_DATABASE=job_platform
MYSQL_USER=job_user
MYSQL_PASSWORD=job_pass

REDIS_PASSWORD=

APP_ENV=dev
APP_PORT=8080

API_IMAGE=go-job-platform/api-server:latest
SCHEDULER_IMAGE=go-job-platform/scheduler:latest
WORKER_IMAGE=go-job-platform/worker:latest
NOTIFIER_IMAGE=go-job-platform/notifier:latest
```

---

## 3. `docker-compose.dev.yml`

```yaml
version: "3.9"

services:
  mysql:
    image: mysql:8.0
    container_name: job-mysql
    restart: unless-stopped
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
      MYSQL_DATABASE: ${MYSQL_DATABASE}
      MYSQL_USER: ${MYSQL_USER}
      MYSQL_PASSWORD: ${MYSQL_PASSWORD}
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
      - ./mysql/init.sql:/docker-entrypoint-initdb.d/init.sql:ro
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-uroot", "-p${MYSQL_ROOT_PASSWORD}"]
      interval: 10s
      timeout: 5s
      retries: 10

  redis:
    image: redis:7
    container_name: job-redis
    restart: unless-stopped
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 10

  api-server:
    image: ${API_IMAGE}
    container_name: job-api-server
    restart: unless-stopped
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy
    environment:
      APP_ENV: ${APP_ENV}
      HTTP_PORT: 8080
      MYSQL_DSN: ${MYSQL_USER}:${MYSQL_PASSWORD}@tcp(mysql:3306)/${MYSQL_DATABASE}?charset=utf8mb4&parseTime=True&loc=Local
      REDIS_ADDR: redis:6379
    ports:
      - "8080:8080"

  scheduler:
    image: ${SCHEDULER_IMAGE}
    container_name: job-scheduler
    restart: unless-stopped
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy
    environment:
      APP_ENV: ${APP_ENV}
      MYSQL_DSN: ${MYSQL_USER}:${MYSQL_PASSWORD}@tcp(mysql:3306)/${MYSQL_DATABASE}?charset=utf8mb4&parseTime=True&loc=Local
      REDIS_ADDR: redis:6379
      SCHEDULER_SCAN_INTERVAL: 5s

  worker:
    image: ${WORKER_IMAGE}
    container_name: job-worker
    restart: unless-stopped
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy
    environment:
      APP_ENV: ${APP_ENV}
      MYSQL_DSN: ${MYSQL_USER}:${MYSQL_PASSWORD}@tcp(mysql:3306)/${MYSQL_DATABASE}?charset=utf8mb4&parseTime=True&loc=Local
      REDIS_ADDR: redis:6379
      WORKER_CONCURRENCY: 5

  notifier:
    image: ${NOTIFIER_IMAGE}
    container_name: job-notifier
    restart: unless-stopped
    depends_on:
      redis:
        condition: service_healthy
    environment:
      APP_ENV: ${APP_ENV}
      REDIS_ADDR: redis:6379

  nginx:
    image: nginx:1.27
    container_name: job-nginx
    restart: unless-stopped
    depends_on:
      - api-server
    ports:
      - "80:80"
    volumes:
      - ./nginx/default.conf:/etc/nginx/conf.d/default.conf:ro

  prometheus:
    image: prom/prometheus:latest
    container_name: job-prometheus
    restart: unless-stopped
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro

  grafana:
    image: grafana/grafana:latest
    container_name: job-grafana
    restart: unless-stopped
    ports:
      - "3000:3000"

volumes:
  mysql_data:
  redis_data:
```

---

## 4. `nginx/default.conf`

```nginx
server {
    listen 80;
    server_name _;

    location / {
        proxy_pass http://api-server:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

---

## 5. `prometheus/prometheus.yml`

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: "api-server"
    static_configs:
      - targets: ["api-server:8080"]

  - job_name: "scheduler"
    static_configs:
      - targets: ["scheduler:8081"]

  - job_name: "worker"
    static_configs:
      - targets: ["worker:8082"]
```

你可以让每个服务单独暴露 metrics 端口，也可以统一 `/metrics` 走业务端口。

---

# 四、Kubernetes YAML 清单模板

建议你按 `base + overlays` 管理。

---

## 1. 目录结构

```text
deployments/k8s/
├── base/
│   ├── namespace.yaml
│   ├── configmap.yaml
│   ├── secret.yaml
│   ├── api-server-deployment.yaml
│   ├── api-server-service.yaml
│   ├── scheduler-deployment.yaml
│   ├── worker-deployment.yaml
│   ├── notifier-deployment.yaml
│   ├── ingress.yaml
│   ├── hpa-worker.yaml
│   └── kustomization.yaml
└── overlays/
    ├── dev/
    │   ├── kustomization.yaml
    │   └── patch-image.yaml
    └── prod/
        ├── kustomization.yaml
        └── patch-resource.yaml
```

---

## 2. `namespace.yaml`

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: job-platform
```

---

## 3. `configmap.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: job-platform-config
  namespace: job-platform
data:
  APP_ENV: "dev"
  HTTP_PORT: "8080"
  MYSQL_HOST: "mysql"
  MYSQL_PORT: "3306"
  MYSQL_DB: "job_platform"
  REDIS_ADDR: "redis:6379"
  SCHEDULER_SCAN_INTERVAL: "5s"
  WORKER_CONCURRENCY: "5"
```

---

## 4. `secret.yaml`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: job-platform-secret
  namespace: job-platform
type: Opaque
stringData:
  MYSQL_USER: "job_user"
  MYSQL_PASSWORD: "job_pass"
  MYSQL_ROOT_PASSWORD: "root"
  CALLBACK_TOKEN: "demo-token"
```

---

## 5. `api-server-deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-server
  namespace: job-platform
spec:
  replicas: 2
  selector:
    matchLabels:
      app: api-server
  template:
    metadata:
      labels:
        app: api-server
    spec:
      containers:
        - name: api-server
          image: go-job-platform/api-server:latest
          ports:
            - containerPort: 8080
          env:
            - name: APP_ENV
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: APP_ENV
            - name: HTTP_PORT
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: HTTP_PORT
            - name: MYSQL_HOST
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: MYSQL_HOST
            - name: MYSQL_PORT
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: MYSQL_PORT
            - name: MYSQL_DB
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: MYSQL_DB
            - name: MYSQL_USER
              valueFrom:
                secretKeyRef:
                  name: job-platform-secret
                  key: MYSQL_USER
            - name: MYSQL_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: job-platform-secret
                  key: MYSQL_PASSWORD
            - name: REDIS_ADDR
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: REDIS_ADDR
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 15
          resources:
            requests:
              cpu: "100m"
              memory: "128Mi"
            limits:
              cpu: "500m"
              memory: "512Mi"
```

---

## 6. `api-server-service.yaml`

```yaml
apiVersion: v1
kind: Service
metadata:
  name: api-server
  namespace: job-platform
spec:
  selector:
    app: api-server
  ports:
    - port: 80
      targetPort: 8080
      protocol: TCP
```

---

## 7. `scheduler-deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: scheduler
  namespace: job-platform
spec:
  replicas: 1
  selector:
    matchLabels:
      app: scheduler
  template:
    metadata:
      labels:
        app: scheduler
    spec:
      containers:
        - name: scheduler
          image: go-job-platform/scheduler:latest
          ports:
            - containerPort: 8081
          env:
            - name: APP_ENV
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: APP_ENV
            - name: MYSQL_HOST
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: MYSQL_HOST
            - name: MYSQL_PORT
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: MYSQL_PORT
            - name: MYSQL_DB
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: MYSQL_DB
            - name: MYSQL_USER
              valueFrom:
                secretKeyRef:
                  name: job-platform-secret
                  key: MYSQL_USER
            - name: MYSQL_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: job-platform-secret
                  key: MYSQL_PASSWORD
            - name: REDIS_ADDR
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: REDIS_ADDR
            - name: SCHEDULER_SCAN_INTERVAL
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: SCHEDULER_SCAN_INTERVAL
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8081
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8081
            initialDelaySeconds: 10
            periodSeconds: 15
```

---

## 8. `worker-deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: worker
  namespace: job-platform
spec:
  replicas: 2
  selector:
    matchLabels:
      app: worker
  template:
    metadata:
      labels:
        app: worker
    spec:
      containers:
        - name: worker
          image: go-job-platform/worker:latest
          ports:
            - containerPort: 8082
          env:
            - name: APP_ENV
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: APP_ENV
            - name: MYSQL_HOST
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: MYSQL_HOST
            - name: MYSQL_PORT
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: MYSQL_PORT
            - name: MYSQL_DB
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: MYSQL_DB
            - name: MYSQL_USER
              valueFrom:
                secretKeyRef:
                  name: job-platform-secret
                  key: MYSQL_USER
            - name: MYSQL_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: job-platform-secret
                  key: MYSQL_PASSWORD
            - name: REDIS_ADDR
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: REDIS_ADDR
            - name: WORKER_CONCURRENCY
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: WORKER_CONCURRENCY
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8082
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8082
            initialDelaySeconds: 10
            periodSeconds: 15
          resources:
            requests:
              cpu: "100m"
              memory: "128Mi"
            limits:
              cpu: "500m"
              memory: "512Mi"
```

---

## 9. `notifier-deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: notifier
  namespace: job-platform
spec:
  replicas: 1
  selector:
    matchLabels:
      app: notifier
  template:
    metadata:
      labels:
        app: notifier
    spec:
      containers:
        - name: notifier
          image: go-job-platform/notifier:latest
          ports:
            - containerPort: 8083
          env:
            - name: APP_ENV
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: APP_ENV
            - name: REDIS_ADDR
              valueFrom:
                configMapKeyRef:
                  name: job-platform-config
                  key: REDIS_ADDR
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8083
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8083
            initialDelaySeconds: 10
            periodSeconds: 15
```

---

## 10. `ingress.yaml`

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: job-platform-ingress
  namespace: job-platform
spec:
  rules:
    - host: job-platform.local
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: api-server
                port:
                  number: 80
```

---

## 11. `hpa-worker.yaml`

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: worker-hpa
  namespace: job-platform
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: worker
  minReplicas: 2
  maxReplicas: 5
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```

---

## 12. `kustomization.yaml`

### `base/kustomization.yaml`

```yaml
resources:
  - namespace.yaml
  - configmap.yaml
  - secret.yaml
  - api-server-deployment.yaml
  - api-server-service.yaml
  - scheduler-deployment.yaml
  - worker-deployment.yaml
  - notifier-deployment.yaml
  - ingress.yaml
  - hpa-worker.yaml
```

### `overlays/dev/kustomization.yaml`

```yaml
resources:
  - ../../base

patches:
  - path: patch-image.yaml
```

### `overlays/dev/patch-image.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-server
  namespace: job-platform
spec:
  template:
    spec:
      containers:
        - name: api-server
          image: go-job-platform/api-server:dev-latest
```

你可以给 scheduler / worker / notifier 也分别做 patch。

---

# 五、你现在该怎么落地

先别一口气把所有模板都实现。
按下面顺序最稳：

## 第一步

先落数据库和 API：

* 建 `tasks`
* 建 `task_executions`
* 做创建任务 / 查询任务 / 手动触发 / 查询执行记录

## 第二步

补 Docker Compose：

* mysql
* redis
* api-server
* scheduler
* worker

## 第三步

补 K8s：

* namespace
* configmap
* secret
* api-server deployment
* worker deployment
* ingress

## 第四步

补监控和 GitOps。

---

# 六、这一版已经够你开工的文件清单

你现在最少要先创建这些文件：

```text
cmd/api-server/main.go
cmd/scheduler/main.go
cmd/worker/main.go

internal/domain/task.go
internal/domain/task_execution.go

internal/api/handler/task_handler.go
internal/service/task_service.go
internal/service/execution_service.go
internal/repository/mysql/task_repo.go
internal/repository/mysql/execution_repo.go
internal/repository/redis/queue_repo.go

deployments/docker-compose/docker-compose.dev.yml
deployments/k8s/base/configmap.yaml
deployments/k8s/base/secret.yaml
deployments/k8s/base/api-server-deployment.yaml
deployments/k8s/base/worker-deployment.yaml
```

下一步最合适的是我直接继续帮你拆：

**Go 项目的代码目录设计 + 每个模块的接口定义 + handler/service/repository 的代码骨架。**

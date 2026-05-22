# Day1 — 容器化入门：从 Go 服务到多容器编排

---

## 一、Go HTTP 服务（单容器）

### 1.1 初始化项目

```bash
mkdir day1
cd day1
go mod init k8s-learning-day1
```

`go mod init` 会生成 `go.mod`，声明这个项目的模块名和 Go 版本。

### 1.2 编写 main.go

```go
package main

import (
    "flag"
    "fmt"
    "net/http"
    "os"
    "time"
)

func main() {
    // 端口优先级：命令行参数 > 环境变量 > 默认值 8080
    portFlag := flag.String("port", "", "服务监听端口，默认 8080")
    flag.Parse()

    port := *portFlag
    if port == "" {
        port = os.Getenv("PORT")
    }
    if port == "" {
        port = "8080"
    }

    http.HandleFunc("/", homeHandler)
    http.HandleFunc("/health", healthHandler)
    http.HandleFunc("/info", infoHandler)

    fmt.Printf("Server starting on port %s...\n", port)
    fmt.Printf("访问 http://localhost:%s\n", port)

    if err := http.ListenAndServe(":"+port, nil); err != nil {
        fmt.Printf("Server failed to start: %v\n", err)
        os.Exit(1)
    }
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Welcome to K8s Learning HTTP Server!\n")
    fmt.Fprintf(w, "请求路径: %s\n", r.URL.Path)
    fmt.Fprintf(w, "请求方法: %s\n", r.Method)
    fmt.Fprintf(w, "当前时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, `{"status": "healthy", "service": "k8s-learning"}`)
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
    hostname, _ := os.Hostname()
    fmt.Fprintf(w, "=== Service Info ===\n")
    fmt.Fprintf(w, "Hostname: %s\n", hostname)
    fmt.Fprintf(w, "Time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
    fmt.Fprintf(w, "Pod IP: %s\n", r.Host)
    fmt.Fprintf(w, "User-Agent: %s\n", r.Header.Get("User-Agent"))
}
```

### 1.3 关键概念

| 代码 | 作用 |
|------|------|
| `flag.String("port", "", ...)` | 定义命令行参数 `-port` |
| `os.Getenv("PORT")` | 读取环境变量（容器里常用这种方式传配置） |
| `http.HandleFunc("/", fn)` | 注册路由：访问 `/` 时执行 `fn` |
| `http.ListenAndServe(":8080", nil)` | 启动 HTTP 服务器监听 8080 端口 |
| `os.Hostname()` | 获取机器名（容器里会返回容器 ID） |

### 1.4 运行和测试

```bash
go run main.go              # 默认 8080
go run main.go -port 9090   # 命令行指定
PORT=9090 go run main.go    # 环境变量指定
```

另开终端验证：
```bash
curl http://localhost:8080/
curl http://localhost:8080/health
curl http://localhost:8080/info
```

---

## 二、Dockerfile（multi-stage 构建）

### 2.1 什么是容器？

- **传统方式**：发给对方一个 exe + 说明书（"先装 Go，再装 xxx..."）
- **容器方式**：把程序 + 运行环境打包成一个"盒子"，对方直接运行这个盒子就行

容器 = 你的程序 + 最小运行环境，打包在一起的可移植单元。

### 2.2 为什么用 Multi-Stage（多阶段构建）？

```
┌──────────────────────────────────────────────┐
│ 只用一个阶段：                                  │
│ 最终镜像 = Go 编译器(500MB) + 源代码 + 二进制   │
│ 结果：镜像 500MB+，还暴露了源代码               │
└──────────────────────────────────────────────┘

┌──────────────────────────────────────────────┐
│ 用两个阶段：                                    │
│ 阶段1：用 Go 环境编译 → 产出二进制文件          │
│ 阶段2：用干净的 alpine → 只拷贝二进制文件       │
│ 结果：镜像 ~15MB，只有可执行文件                │
└──────────────────────────────────────────────┘
```

### 2.3 Dockerfile 逐行解释

```dockerfile
# ============ 阶段 1：编译 ============
FROM golang:1.22-alpine AS builder
# 指定基础镜像（已经装好 Go 的迷你 Linux）
# AS builder: 给这个阶段起名，后面引用

WORKDIR /app
# 设置工作目录（相当于 mkdir /app && cd /app）

COPY go.mod ./
COPY main.go ./
# 把本地文件复制到容器内的 /app

RUN go build -o server main.go
# 在容器内执行编译，产出名为 server 的可执行文件

# ============ 阶段 2：运行 ============
FROM docker.io/library/alpine:3.19
# 换一个全新的基础镜像（只有 5MB 的 Linux）
# 阶段1的所有东西（Go 编译器、源代码）全部丢弃

WORKDIR /app

COPY --from=builder /app/server .
# 从阶段1（builder）的结果中，只拷贝编译好的二进制
# 这是 multi-stage 的核心！

EXPOSE 8080
# 声明容器会用 8080 端口（文档作用，不会真的开端口）

CMD ["./server"]
# 容器启动时执行的命令
```

### 2.4 Dockerfile 指令速查

| 指令 | 作用 | 例子 |
|------|------|------|
| `FROM` | 指定基础镜像 | `FROM alpine:3.19` |
| `WORKDIR` | 设置工作目录 | `WORKDIR /app` |
| `COPY` | 复制文件到容器 | `COPY main.go ./` |
| `RUN` | 在构建时执行命令 | `RUN go build -o server .` |
| `EXPOSE` | 声明端口（文档用） | `EXPOSE 8080` |
| `CMD` | 容器启动时的默认命令 | `CMD ["./server"]` |
| `ENV` | 设置环境变量 | `ENV PORT=8080` |

---

## 三、构建和运行容器

### 3.1 构建镜像

```bash
podman build -t k8s-day1 .
podman build --no-cache -t k8s-day1 .   # 修改代码后强制重新构建
```

- `-t k8s-day1`：给镜像取名
- `.`：用当前目录的 Dockerfile
- `--no-cache`：忽略缓存，全部重来

### 3.2 运行容器

```bash
podman run -p 8080:8080 k8s-day1           # 前台运行
podman run -d --name myapp -p 8080:8080 k8s-day1  # 后台运行
```

端口映射：`-p 电脑端口:容器内端口`
```
浏览器 → localhost:8080 → [容器内 :8080 → ./server]
```

### 3.3 常用容器操作

```bash
podman ps                    # 查看运行中的容器
podman logs myapp            # 查看容器日志
podman exec -it myapp sh    # 进入容器内部
podman stop myapp            # 停止
podman rm myapp              # 删除
podman images                # 查看本地镜像
```

### 3.4 用环境变量配置容器

```bash
podman run -d -e PORT=9090 -p 9090:9090 k8s-day1
```

### 3.5 验证容器隔离性

本地运行 → `Hostname: IT00075053`（你的电脑名）
容器运行 → `Hostname: fd00c3e73bd3`（容器 ID）

容器有自己的主机名、文件系统和网络，与宿主机隔离。

---

## 四、docker-compose 多容器编排

### 4.1 什么是 docker-compose？

单容器：一个盒子跑一个程序。
多容器编排：多个盒子协作（比如 App + 数据库 + 反向代理），**一条命令**全部启动。

### 4.2 项目结构

```
day1/
├── app/
│   ├── main.go        ← Go 服务（连接 Redis）
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
├── nginx/
│   ├── nginx.conf     ← Nginx 反向代理配置
│   └── Dockerfile
└── docker-compose.yml ← 编排文件（定义所有服务）
```

### 4.3 app/main.go — 带 Redis 的 Go 服务

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "net/http"
    "os"
    "time"

    "github.com/redis/go-redis/v9"
)

var rdb *redis.Client

func main() {
    portFlag := flag.String("port", "", "服务监听端口，默认 8080")
    flag.Parse()

    port := *portFlag
    if port == "" {
        port = os.Getenv("PORT")
    }
    if port == "" {
        port = "8080"
    }

    // 从环境变量读 Redis 地址（compose 里配置）
    redisAddr := os.Getenv("REDIS_ADDR")
    if redisAddr == "" {
        redisAddr = "localhost:6379"
    }

    rdb = redis.NewClient(&redis.Options{
        Addr: redisAddr,
    })

    http.HandleFunc("/", homeHandler)
    http.HandleFunc("/health", healthHandler)
    http.HandleFunc("/info", infoHandler)

    fmt.Printf("Server starting on port %s...\n", port)
    fmt.Printf("Redis: %s\n", redisAddr)

    if err := http.ListenAndServe(":"+port, nil); err != nil {
        fmt.Printf("Server failed to start: %v\n", err)
        os.Exit(1)
    }
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    ctx := context.Background()
    // 每次访问计数 +1（存在 Redis 里）
    count, _ := rdb.Incr(ctx, "visit_count").Result()
    fmt.Fprintf(w, "Welcome to K8s Learning HTTP Server!\n")
    fmt.Fprintf(w, "访问次数: %d\n", count)
    fmt.Fprintf(w, "当前时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    ctx := context.Background()
    err := rdb.Ping(ctx).Err()
    if err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        fmt.Fprintf(w, `{"status": "unhealthy", "error": "%s"}`, err.Error())
        return
    }
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, `{"status": "healthy", "service": "k8s-learning", "redis": "connected"}`)
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
    hostname, _ := os.Hostname()
    fmt.Fprintf(w, "=== Service Info ===\n")
    fmt.Fprintf(w, "Hostname: %s\n", hostname)
    fmt.Fprintf(w, "Time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
    fmt.Fprintf(w, "Redis: %s\n", os.Getenv("REDIS_ADDR"))
}
```

### 4.4 app/Dockerfile

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download          # 先下载依赖（利用缓存）
COPY main.go ./
RUN go build -o server main.go

FROM docker.io/library/alpine:3.19
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

与上午的区别：先 `COPY go.mod go.sum` + `RUN go mod download`，再 `COPY main.go`。
这样修改代码时不用重新下载依赖（Docker 层缓存优化）。

### 4.5 nginx/nginx.conf — 反向代理配置

```nginx
upstream app {
    server app:8080;
    # "app" 是 compose 里的服务名，compose 自动创建 DNS 解析
}

server {
    listen 80;

    location / {
        proxy_pass http://app;              # 把请求转发给 app 服务
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### 4.6 nginx/Dockerfile

```dockerfile
FROM docker.io/library/nginx:alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
```

### 4.7 docker-compose.yml — 编排核心

```yaml
version: "3.8"

services:
  app:
    build: ./app                    # 构建 app 目录下的 Dockerfile
    environment:
      - REDIS_ADDR=redis:6379       # 告诉 app Redis 在哪（用服务名）
    depends_on:
      - redis                       # app 依赖 redis，先启动 redis

  redis:
    image: docker.io/library/redis:7-alpine   # 直接用官方镜像，不用自己写代码

  nginx:
    build: ./nginx                  # 构建 nginx 目录下的 Dockerfile
    ports:
      - "8081:80"                   # 宿主机 8081 → nginx 的 80
    depends_on:
      - app                         # nginx 依赖 app
```

### 4.8 compose 配置逐行解释

| 字段 | 作用 |
|------|------|
| `services` | 定义所有容器服务 |
| `build: ./app` | 指定 Dockerfile 路径，compose 自动构建 |
| `image: redis:7-alpine` | 直接使用现成镜像，不需要构建 |
| `environment` | 设置环境变量（对应代码里的 `os.Getenv`） |
| `depends_on` | 声明启动顺序（redis 先启动，app 才启动） |
| `ports: "8081:80"` | 端口映射（只有 nginx 暴露给外部） |

### 4.9 服务间通信原理

```
你的浏览器
    ↓ :8081
┌─────────────────────────────────────────┐
│  compose 网络（day1_default）              │
│                                          │
│  nginx(:80) → app(:8080) → redis(:6379) │
│                                          │
│  容器之间用 服务名 互相访问               │
│  比如 app 连 redis，地址就是 "redis:6379" │
└─────────────────────────────────────────┘
```

compose 会自动创建一个网络，所有服务加入同一网络，用**服务名**作为 DNS 互相访问。

### 4.10 运行多容器

```bash
# 构建并启动所有服务（后台）
podman-compose up --build -d

# 查看所有容器状态
podman-compose ps

# 查看某个服务的日志
podman-compose logs app
podman-compose logs redis

# 停止并删除所有容器
podman-compose down
```

### 4.11 验证

```bash
# 通过 nginx 反向代理访问 app
curl http://localhost:8081/
# 输出：
# Welcome to K8s Learning HTTP Server!
# 访问次数: 1         ← 每刷一次 +1（Redis 存储）
# 当前时间: 2026-05-22 12:49:40

curl http://localhost:8081/health
# 输出：{"status": "healthy", "service": "k8s-learning", "redis": "connected"}

curl http://localhost:8081/info
# 输出：
# Hostname: 237030298c0d  ← 容器 ID
# Redis: redis:6379       ← 通过服务名连接
```

---

## 五、排障练习

### 5.1 查看容器日志

```bash
podman-compose logs app      # app 服务日志
podman-compose logs -f app   # 实时跟踪日志（-f = follow）
```

### 5.2 进入容器检查

```bash
podman exec -it day1_app_1 sh       # 进入 app 容器
ls /app                              # 看文件
cat /etc/os-release                  # 确认是 alpine

podman exec -it day1_redis_1 sh     # 进入 redis 容器
redis-cli ping                       # 测试 redis 连接
redis-cli get visit_count            # 查看访问计数
```

### 5.3 查看容器网络

```bash
podman network ls                        # 列出所有网络
podman network inspect day1_default      # 查看 compose 创建的网络
```

### 5.4 故意写错 Dockerfile 观察报错

试试这些错误，理解报错信息：

**错误 1：COPY 的文件不存在**
```dockerfile
COPY noexist.go ./
# 报错：noexist.go: no such file or directory
```

**错误 2：FROM 镜像名拼错**
```dockerfile
FROM golnag:1.22-alpine
# 报错：Error: unable to find image "golnag:1.22-alpine"
```

**错误 3：CMD 格式写错**
```dockerfile
CMD ./server      # 应该用 JSON 数组格式
CMD ["./server"]  # 正确
```

---

## 六、常见问题和踩坑记录

### Q1: 端口映射写错了，容器启动但访问不到

```bash
podman run -p 12334:1234 http_service    # ❌ 服务监听 8080 不是 1234
podman run -p 12334:8080 http_service    # ✅ 正确
```

记忆：`-p 你想用的端口:程序实际监听的端口`

### Q2: 修改了代码，容器运行的还是旧版本

加 `--no-cache` 强制重新构建：
```bash
podman build --no-cache -t http_service .
```

看日志有 `Using cache` 就是旧的，没有就是新构建的。

### Q3: 容器启动后只看到日志，看不到 HTTP 响应

`fmt.Printf` → 打到终端（日志），`fmt.Fprintf(w, ...)` → 打到 HTTP 响应。
要看响应内容，用 `-d` 后台运行，另开终端 curl 或浏览器访问。

### Q4: 容器名冲突

```bash
podman rm -f myapp    # 先删旧的
podman run -d --name myapp -p 8080:8080 http_service
```

### Q5: 端口被占用

换个端口：`podman run -d -p 9090:8080 http_service`

### Q6: 80 端口 permission denied

容器内监听 80 需要特权。解决：映射到 1024 以上端口：
```yaml
ports:
  - "8081:80"    # 用 8081 代替 80
```

---

## 七、概念总结

```
源代码 → Dockerfile → podman build → 镜像(image) → podman run → 容器(container)
  开发写的    构建食谱      执行构建     打包好的盒子    启动运行     运行中的实例
```

| 概念 | 类比 |
|------|------|
| 镜像 (Image) | 一张光盘（模板，不可变） |
| 容器 (Container) | 用光盘装好的电脑（运行实例，可多个） |
| Dockerfile | 刻录光盘的说明书 |
| Registry | 光盘商店（Docker Hub） |
| docker-compose | 多台电脑的组网说明书 |
| 服务名 (service name) | 内网 DNS（容器之间用名字互相找） |

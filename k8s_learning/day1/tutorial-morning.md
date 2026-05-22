# Day1 上午 — Go HTTP 服务 + 容器化

## 任务目标

1. 写一个简单的 Go HTTP 服务（返回 hostname + 当前时间）
2. 手写 Dockerfile（multi-stage）
3. 构建并运行容器，验证 http://localhost:8080

---

## 第一步：Go HTTP 服务

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

### 1.3 关键概念解释

| 代码 | 作用 |
|------|------|
| `flag.String("port", "", ...)` | 定义命令行参数 `-port` |
| `os.Getenv("PORT")` | 读取环境变量（容器里常用这种方式传配置） |
| `http.HandleFunc("/", fn)` | 注册路由：访问 `/` 时执行 `fn` |
| `http.ListenAndServe(":8080", nil)` | 启动 HTTP 服务器监听 8080 端口 |
| `os.Hostname()` | 获取机器名（容器里会返回容器 ID） |

### 1.4 运行和测试

```bash
# 方式一：直接运行（开发用）
go run main.go

# 方式二：指定端口
go run main.go -port 9090

# 方式三：通过环境变量
PORT=9090 go run main.go
```

打开另一个终端验证：
```bash
curl http://localhost:8080/
curl http://localhost:8080/health
curl http://localhost:8080/info
```

预期输出：
```
=== Service Info ===
Hostname: 你的电脑名
Time: 2026-05-22 18:30:00
Pod IP: localhost:8080
User-Agent: curl/8.x
```

---

## 第二步：编写 Dockerfile（multi-stage）

### 2.1 什么是容器？

想象你写了一个程序，要给别人用：
- **传统方式**：发给对方一个 exe + 说明书（"先装 Go，再装 xxx..."）
- **容器方式**：把程序 + 运行环境打包成一个"盒子"，对方直接运行这个盒子就行

容器 = 你的程序 + 最小运行环境，打包在一起的可移植单元。

### 2.2 什么是 Dockerfile？

Dockerfile 是一个"食谱"，告诉容器引擎怎么构建这个"盒子"。

### 2.3 为什么用 Multi-Stage（多阶段构建）？

Go 编译需要完整的 Go 工具链（约 500MB），但运行只需要一个二进制文件（约 10MB）。

```
┌──────────────────────────────────────────────┐
│ 如果只用一个阶段：                              │
│ 最终镜像 = Go 编译器(500MB) + 源代码 + 二进制  │
│ 结果：镜像 500MB+，还暴露了源代码               │
└──────────────────────────────────────────────┘

┌──────────────────────────────────────────────┐
│ 用两个阶段：                                    │
│ 阶段1：用 Go 环境编译 → 产出二进制文件          │
│ 阶段2：用干净的 alpine → 只拷贝二进制文件       │
│ 结果：镜像 ~15MB，只有可执行文件                │
└──────────────────────────────────────────────┘
```

### 2.4 Dockerfile 逐行解释

```dockerfile
# ============ 阶段 1：编译 ============
FROM golang:1.22-alpine AS builder
# FROM: 指定基础镜像（一个已经装好 Go 的迷你 Linux）
# AS builder: 给这个阶段起个名字，后面引用用

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
# 注意：阶段1的所有东西（Go 编译器、源代码）全部丢弃

WORKDIR /app

COPY --from=builder /app/server .
# 从阶段1（builder）的结果中，只拷贝编译好的二进制文件
# 这是 multi-stage 的核心！

EXPOSE 8080
# 声明容器会用 8080 端口（只是文档作用，不会真的开端口）

CMD ["./server"]
# 容器启动时执行的命令
```

### 2.5 Dockerfile 指令速查

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

## 第三步：构建和运行容器

### 3.1 构建镜像

```bash
podman build -t k8s-day1 .
```

解释：
- `podman build`：执行构建
- `-t k8s-day1`：给镜像取名为 `k8s-day1`
- `.`：用当前目录的 Dockerfile

构建过程会输出每一步的执行结果，最后看到 `Successfully tagged` 就成功了。

### 3.2 查看构建好的镜像

```bash
podman images
```

输出类似：
```
REPOSITORY           TAG     SIZE
localhost/k8s-day1   latest  15MB   ← 你的镜像，只有 15MB！
```

### 3.3 运行容器

```bash
podman run -p 8080:8080 k8s-day1
```

解释：
- `podman run`：从镜像启动一个容器
- `-p 8080:8080`：端口映射（电脑的 8080 → 容器的 8080）
- `k8s-day1`：用哪个镜像

端口映射示意：
```
你的浏览器 → localhost:8080 → [容器内 :8080 → ./server]
```

### 3.4 后台运行 + 常用操作

```bash
# 后台运行（加 -d）
podman run -d --name myapp -p 8080:8080 k8s-day1

# 查看运行中的容器
podman ps

# 查看容器日志（相当于看 fmt.Printf 的输出）
podman logs myapp

# 进入容器内部（像 SSH 进入一台服务器）
podman exec -it myapp sh

# 在容器内看看文件系统
ls /app          # 只有一个 server 文件！
cat /etc/os-release  # 确认是 alpine linux

# 退出容器 shell
exit

# 停止容器
podman stop myapp

# 删除容器
podman rm myapp
```

### 3.5 验证容器和本地运行的区别

本地运行 `go run main.go` 时：
```
Hostname: 你的电脑名（如 IT00075053）
```

容器运行时：
```
Hostname: fd00c3e73bd3（容器 ID）
```

这说明容器是一个**隔离的环境**，有自己的主机名、文件系统和网络。

---

## 3.6 使用环境变量改变容器内的端口

```bash
podman run -d -e PORT=9090 -p 9090:9090 k8s-day1
```

- `-e PORT=9090`：设置容器内的环境变量
- `-p 9090:9090`：映射对应端口

---

## 总结

```
源代码(main.go) → Dockerfile → podman build → 镜像(image) → podman run → 容器(container)
    开发写的          构建食谱       执行构建      打包好的盒子      启动运行      运行中的实例
```

| 概念 | 类比 |
|------|------|
| 镜像 (Image) | 一张光盘（模板，不可变） |
| 容器 (Container) | 用光盘装好的电脑（运行实例，可以有多个） |
| Dockerfile | 刻录光盘的说明书 |
| Registry | 光盘商店（Docker Hub / 镜像仓库） |

# Day1 环境配置指南（Windows，无 WSL/Docker）

## 1. 安装 Go

1. 下载：https://go.dev/dl/ — 选 `go1.22.x.windows-amd64.msi`
2. 双击安装，默认路径即可
3. 验证：打开新终端运行 `go version`

## 2. 安装 Podman Desktop

Podman 是 Docker 的替代品，不依赖 WSL，在 Windows 上通过自带的 Podman Machine（基于轻量 QEMU/HyperV VM）运行容器。

1. 下载：https://podman-desktop.io/downloads — 选 Windows 版
2. 安装后启动 Podman Desktop，首次会引导你初始化 Podman Machine
3. 点击 "Initialize and Start" 创建默认 machine
4. 验证：打开终端运行：
   ```bash
   podman --version
   podman machine list    # 应显示一个 running 的 machine
   ```

### 配置 Docker 兼容（让 docker/docker-compose 命令直接指向 podman）

Podman Desktop 安装时一般会提示是否安装 Docker CLI 兼容。如果没有：

```bash
# 在 PowerShell 管理员模式下
# 方式一：Podman Desktop 设置里勾选 "Docker Compatibility"
# 方式二：手动创建别名（在 Git Bash 的 ~/.bashrc 里加）
alias docker='podman'
alias docker-compose='podman-compose'
```

## 3. 安装 podman-compose

podman-compose 是 docker-compose 的替代：

```bash
pip install podman-compose
# 或者用 pipx：
pipx install podman-compose
```

验证：`podman-compose --version`

> 如果没有 Python/pip，可以用 Podman Desktop 内置的 Compose 功能（Settings > Extensions > Compose）。

## 4. 初始化 Go 项目

```bash
cd D:\code\go_learning\k8s_learning\day1
go mod init k8s-learning-day1
```

## 5. 验证全链路

```bash
# 运行 Go 服务
go run main.go
# 另一个终端访问 http://localhost:8080/info

# 构建容器（安装好 Podman 后）
podman build -t k8s-day1 .
podman run -p 8080:8080 k8s-day1
```

## 命令对照表

| Docker 命令 | Podman 等价 | 说明 |
|---|---|---|
| `docker build` | `podman build` | 完全兼容 |
| `docker run` | `podman run` | 完全兼容 |
| `docker ps` | `podman ps` | 完全兼容 |
| `docker logs` | `podman logs` | 完全兼容 |
| `docker exec` | `podman exec` | 完全兼容 |
| `docker-compose up` | `podman-compose up` | 基本兼容 |
| `docker network inspect` | `podman network inspect` | 完全兼容 |

task.md 里所有 Docker 命令直接把 `docker` 换成 `podman` 即可。

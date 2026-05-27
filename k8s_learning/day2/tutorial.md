# Day 2：K8s 集群搭建 + Pod 基础

> 昨天你把 Go 服务装进了容器，今天我们把容器装进 K8s 集群。

## 一、K8s 是什么？一句话理解

Docker/Podman 管理的是**一台机器上的容器**，K8s 管理的是**一群机器上的容器**。

你可以把 K8s 想象成一个"容器调度中心"：
- 你告诉它"我需要 3 个 Go 服务实例"
- 它自动找合适的机器运行
- 某个实例挂了？自动拉一个新的
- 流量太大？自动扩容

## 二、K8s 架构核心组件

### 控制平面（Control Plane）—— 大脑

| 组件 | 类比 | 作用 |
|------|------|------|
| API Server | 前台接待 | 所有操作的入口，RESTful API。kubectl 的每条命令都是在调它 |
| etcd | 记事本 | 集群所有状态的存储（key-value 数据库），挂了集群就失忆 |
| Scheduler | 调度员 | 决定新 Pod 放到哪个 Node 上运行（看资源、亲和性等） |
| Controller Manager | 巡查员 | 不断检查"实际状态"是否等于"期望状态"，不等就修正 |

### 工作节点（Node）—— 干活的

| 组件 | 类比 | 作用 |
|------|------|------|
| Kubelet | 工头 | 每个 Node 上的 agent，负责管理本机的 Pod 生命周期 |
| kube-proxy | 网络管理员 | 维护 Service 的网络规则（iptables/ipvs） |
| 容器运行时 | 搬砖工 | 实际运行容器（containerd、CRI-O 等） |

### 请求流程

```
你敲 kubectl → API Server 收到 → 写入 etcd
                                    ↓
                              Controller 检测到变化
                                    ↓
                              Scheduler 选 Node
                                    ↓
                              Kubelet 在 Node 上启动 Pod
```

## 三、用 kind 搭建本地集群

### 什么是 kind？

kind（Kubernetes IN Docker）用容器模拟 K8s 节点。每个"节点"其实是一个容器，里面跑着 kubelet 等组件。

| 方案 | 优势 | 劣势 |
|------|------|------|
| kind | 轻量、支持多节点、CI 友好 | 功能受限于容器网络 |
| minikube | 功能完整、插件丰富 | 只支持单节点（默认） |
| Docker Desktop K8s | 最省事 | 只支持单节点 |

### 安装 kind

```bash
# 用 Go 安装（你已有 Go 环境）
go install sigs.k8s.io/kind@latest

# 验证
kind version
```

### 安装 kubectl

```bash
# Windows（用 scoop）
scoop install kubectl

# 或直接下载
# https://dl.k8s.io/release/v1.31.0/bin/windows/amd64/kubectl.exe
# 放到 PATH 目录下

# 验证
kubectl version --client
```

### 创建 3 节点集群

我们用一个配置文件创建集群，1 个控制平面 + 2 个工作节点：

```bash
kind create cluster --config kind-config.yaml --name k8s-lab
```

`kind-config.yaml` 内容：

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 30080
    hostPort: 30080
    protocol: TCP
- role: worker
- role: worker
```

> **extraPortMappings 是什么？** kind 的节点是容器，外部默认访问不到。这个配置把节点的 30080 端口映射到宿主机，后面用 NodePort Service 时会用到。

### 验证集群

```bash
# 查看集群
kind get clusters

# 查看节点状态
kubectl get nodes
# 输出类似：
# NAME                    STATUS   ROLES           AGE   VERSION
# k8s-lab-control-plane   Ready    control-plane   2m    v1.35.0
# k8s-lab-worker          Ready    <none>          1m    v1.35.0
# k8s-lab-worker2         Ready    <none>          1m    v1.35.0

# 查看集群信息
kubectl cluster-info

# 查看所有系统 Pod（K8s 自身的组件）
kubectl get pods -n kube-system
```

### 常见问题

**Q: `kind create cluster` 报错 "docker not found"？**

kind 默认用 docker，用 podman 需要设置环境变量：

```bash
# Linux/Mac
export KIND_EXPERIMENTAL_PROVIDER=podman

# Windows PowerShell
$env:KIND_EXPERIMENTAL_PROVIDER="podman"

# Windows CMD
set KIND_EXPERIMENTAL_PROVIDER=podman
```

**Q: 节点状态一直是 NotReady？**

等一会儿，系统组件需要时间启动。可以用以下命令观察：

```bash
kubectl get nodes -w   # -w 是 watch，实时刷新
```

## 四、理解 Pod

### Pod 是什么？

Pod 是 K8s 的**最小调度单位**，不是容器。

| 对比 | Docker | K8s |
|------|--------|-----|
| 最小单位 | 容器 | Pod |
| 关系 | — | 一个 Pod 可以包含多个容器 |
| 网络 | 每个容器独立网络 | 同一 Pod 内的容器共享网络（localhost 互通） |
| 存储 | 各自独立 | 同一 Pod 内可共享 Volume |

> 类比：Pod 就像一个"房间"，里面可以放多张"桌子"（容器），它们共用同一个门牌号（IP 地址）。

### Pod 基本操作

```bash
# 运行一个 Pod（最简方式）
kubectl run nginx --image=nginx:alpine

# 查看 Pod 列表
kubectl get pods
kubectl get pods -o wide    # 多显示 IP 和 Node 信息

# 查看 Pod 详细信息（排障必用）
kubectl describe pod nginx

# 查看 Pod 日志
kubectl logs nginx
kubectl logs nginx -f       # 实时跟踪日志

# 进入 Pod 执行命令
kubectl exec -it nginx -- sh
# 进去后可以 ls、curl 等，exit 退出

# 删除 Pod
kubectl delete pod nginx
```

### kubectl describe 输出解读

```bash
kubectl describe pod nginx
```

重点看这几个部分：

| 字段 | 含义 |
|------|------|
| Status | Running / Pending / Failed / CrashLoopBackOff |
| IP | Pod 的集群内 IP |
| Node | Pod 运行在哪个节点 |
| Containers | 容器状态、镜像、端口 |
| Events | 最重要！调度、拉镜像、启动的全过程，排障首先看这里 |

## 五、Deployment —— 生产环境用这个

### 为什么不直接用 Pod？

直接创建的 Pod（叫"裸 Pod"）有个致命问题：**挂了不会自动恢复**。

Deployment 的作用：
- 管理多个 Pod 副本
- Pod 挂了自动重建
- 支持滚动更新和回滚
- 支持扩缩容

### 关系链

```
Deployment → 创建 ReplicaSet → 创建 Pod
   你定义       自动管理         实际运行
```

你只需要告诉 Deployment"我要 3 个副本"，剩下的它全搞定。

### 编写 Deployment YAML

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: go-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: go-app
  template:
    metadata:
      labels:
        app: go-app
    spec:
      containers:
      - name: go-app
        image: localhost/k8s-day1:latest
        imagePullPolicy: Never
        ports:
        - containerPort: 8080
```

**YAML 字段解读：**

| 字段 | 含义 |
|------|------|
| apiVersion: apps/v1 | API 版本，Deployment 用 apps/v1 |
| kind: Deployment | 资源类型 |
| metadata.name | Deployment 的名字 |
| spec.replicas | 副本数（要运行几个 Pod） |
| spec.selector.matchLabels | 用标签选择器匹配 Pod（必须和 template.labels 一致） |
| spec.template | Pod 的模板（每个副本都按这个模板创建） |
| spec.template.metadata.labels | Pod 的标签（被 selector 匹配） |
| spec.template.spec.containers | 容器列表（名字、镜像、端口） |

> **selector 和 labels 为什么要一致？** Deployment 通过标签找到属于自己的 Pod。不一致会导致 Deployment 找不到 Pod，创建无限多副本。

### 部署 Day 1 的 Go 服务到 kind

```bash
# 1. 加载本地镜像到 kind 集群
# kind 的节点是容器，看不到你宿主机的镜像，需要手动加载
# 用 podman 构建的镜像，先导出再导入：
podman save k8s-day1:latest -o myapp.tar
kind load image-archive myapp.tar --name k8s-lab

# 注意：加载后镜像在节点上的名字带 localhost/ 前缀
# 所以 YAML 里要写 image: localhost/k8s-day1:latest
# 同时必须设置 imagePullPolicy: Never，否则 K8s 会尝试从远端拉取

# 2. 部署
kubectl apply -f deployment.yaml

# 3. 查看结果
kubectl get deploy,rs,pods
# deploy = Deployment, rs = ReplicaSet, pods = Pod
```

### Deployment 常用操作

```bash
# 扩容到 5 个副本
kubectl scale deploy go-app --replicas=5

# 查看扩容进度
kubectl rollout status deploy go-app

# 更新镜像（触发滚动更新）
# 先修改 Day1 代码，重新构建一个 v2 版本：
#   podman build -t k8s-day1:v2 .
#   kind load docker-image k8s-day1:v2 --name k8s-lab
kubectl set image deploy/go-app go-app=k8s-day1:v2

# 查看更新历史
kubectl rollout history deploy go-app

# 回滚到上一个版本
kubectl rollout undo deploy go-app

# 回滚到指定版本
kubectl rollout undo deploy go-app --to-revision=1
```

### 端口转发 —— 快速访问 Pod

```bash
# 将本机 8080 转发到 Pod 的 8080
kubectl port-forward deploy/go-app 8080:8080

# 浏览器访问 http://localhost:8080
# Ctrl+C 停止转发
```

> port-forward 是**开发调试用的**，生产环境用 Service（Day 3 会学）。

## 六、kubectl 命令速查

### 高频命令（必须肌肉记忆）

| 命令 | 作用 | 示例 |
|------|------|------|
| get | 列出资源 | `kubectl get pods -o wide` |
| describe | 资源详情（含 Events） | `kubectl describe pod nginx` |
| logs | 查看日志 | `kubectl logs nginx -f` |
| exec | 进入容器 | `kubectl exec -it nginx -- sh` |
| apply | 创建/更新资源 | `kubectl apply -f deployment.yaml` |
| delete | 删除资源 | `kubectl delete -f deployment.yaml` |
| port-forward | 端口转发 | `kubectl port-forward deploy/go-app 8080:8080` |
| scale | 扩缩容 | `kubectl scale deploy go-app --replicas=5` |

### 输出格式

```bash
kubectl get pods               # 默认表格
kubectl get pods -o wide       # 多显示 IP、Node
kubectl get pods -o yaml       # 完整 YAML
kubectl get pods -o json       # 完整 JSON
kubectl get pods -o name       # 只显示名字
```

### 有用的技巧

```bash
# 查看某个资源的 YAML 结构说明
kubectl explain deployment.spec.strategy
kubectl explain pod.spec.containers

# 用标签筛选
kubectl get pods -l app=go-app

# 查看所有命名空间的资源
kubectl get pods -A

# 实时监控变化
kubectl get pods -w
```

## 七、常见问题和踩坑记录

**Q: `kubectl apply` 后 Pod 状态是 ImagePullBackOff 或 ErrImageNeverPull？**

两个常见原因：
1. 镜像没加载到 kind 节点：先 `podman save` 再 `kind load image-archive`
2. 镜像名不匹配：Podman 导出后在 kind 节点上镜像名带 `localhost/` 前缀，YAML 里必须写 `image: localhost/k8s-day1:latest`
3. 忘了设置 `imagePullPolicy: Never`：本地镜像必须加这个，否则 K8s 会去 Docker Hub 拉

**Q: Pod 状态是 Pending？**

`kubectl describe pod <name>` 看 Events。常见原因：
- 资源不足（Node 的 CPU/内存不够）
- 镜像没加载到集群

**Q: Pod 状态是 CrashLoopBackOff？**

应用启动就崩了。`kubectl logs <pod-name> --previous` 看上一次崩溃日志。

**Q: `kubectl get nodes` 显示连接被拒？**

kubeconfig 没设置好。kind 创建集群时会自动配置，确认 `~/.kube/config` 存在：

```bash
kubectl config current-context
# 应该显示 kind-k8s-lab
```

**Q: Deployment 的 3 个 Pod 都在同一个 Node 上？**

正常，Scheduler 默认按资源平衡调度，小集群可能分配不均。可以用 `kubectl get pods -o wide` 查看分布。

## 八、概念总结

```
裸 Pod          → 手动管理，挂了就没了
Deployment      → 自动管理 Pod 副本，支持更新回滚
  └── ReplicaSet → Deployment 自动创建，维护指定数量的 Pod
       └── Pod   → 最小调度单位，包含一或多个容器
            └── Container → 实际跑你代码的地方
```

| 概念 | 类比 |
|------|------|
| Cluster | 整个工厂 |
| Node | 工厂里的车间 |
| Pod | 车间里的工位 |
| Container | 工位上的工人 |
| Deployment | 排班表（"保持 3 个工人在岗"） |
| kubectl | 厂长的对讲机 |
| kind | 一个迷你模拟工厂（用于练习） |

## 九、今日产出验收

```bash
# 1. 集群状态正常
kubectl get nodes
# 3 个节点都是 Ready

# 2. Deployment 正常运行
kubectl get deploy go-app
# READY 显示 3/3

# 3. Pod 分布查看
kubectl get pods -o wide
# 3 个 Pod 都是 Running

# 4. 能访问服务
kubectl port-forward deploy/go-app 8080:8080
# 浏览器打开 http://localhost:8080 看到响应
```

- [ ] kind 3 节点集群创建成功，`kubectl get nodes` 全部 Ready
- [ ] Deployment 部署 go-app，3 个 Pod 全部 Running
- [ ] 能用 `kubectl port-forward` 访问到 Day 1 的 Go 服务
- [ ] 掌握 kubectl get/describe/logs/exec/apply/delete

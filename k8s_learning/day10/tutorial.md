# Day 10：容器底层原理 + 排障综合

> 最后一天，两个目标：动手验证容器隔离原理（不再只是八股），以及建立完整的 K8s 排障能力。

## 一、容器 = Namespace + Cgroup + rootfs

### 容器不是虚拟机

| | 虚拟机 | 容器 |
|---|---|---|
| 隔离方式 | 硬件虚拟化（hypervisor） | 内核特性（namespace + cgroup） |
| 开销 | 大（需要独立内核） | 小（共享宿主机内核） |
| 启动时间 | 分钟级 | 秒级 |

容器本质上就是一个**受限的 Linux 进程**：
- **Namespace**：让进程看到隔离的"世界"（PID、网络、文件系统等）
- **Cgroup**：限制进程能用多少资源（CPU、内存）
- **rootfs**：给进程一个独立的文件系统（镜像）

### Linux Namespace 种类

| Namespace | 隔离什么 | 效果 |
|-----------|----------|------|
| PID | 进程 ID | 容器内 PID 1 = 你的应用 |
| Network | 网络栈 | 容器有自己的 IP、端口 |
| Mount | 文件系统挂载点 | 容器看到自己的 / |
| UTS | hostname | 容器有自己的主机名 |
| IPC | 进程间通信 | 信号量、共享内存隔离 |
| User | 用户 ID | 容器内 root ≠ 宿主机 root |

## 二、动手验证：Namespace 隔离

> 以下命令需要 Linux 环境（WSL2 或 kind 节点内执行）

### 进入 kind 节点

```bash
# 用 podman/docker 进入 kind 的 worker 节点
podman exec -it k8s-lab-worker bash
```

### PID Namespace

```bash
# 在 kind 节点内
unshare --pid --fork --mount-proc bash

# 现在你在一个新的 PID namespace 里
ps aux
# 只能看到 bash 自己（PID 1）和 ps

# 退出
exit
```

### Network Namespace

```bash
# 创建隔离的网络空间
unshare --net bash

ip a
# 只有 lo（loopback），没有 eth0
# 这就是容器刚创建时的网络状态

exit
```

### 组合使用

```bash
# 同时隔离 PID + Network + Mount
unshare --pid --net --mount --fork --mount-proc bash

# 这就是一个"手工容器"：
# - 看不到其他进程
# - 没有网络
# - 文件系统是隔离的

exit
```

## 三、动手验证：nsenter 进入容器

```bash
# 在 kind 节点上，找到某个容器的 PID
# 方法 1：用 crictl
crictl ps | grep go-app
CONTAINER_ID=$(crictl ps | grep go-app | head -1 | awk '{print $1}')
PID=$(crictl inspect $CONTAINER_ID | grep -m1 '"pid"' | grep -o '[0-9]*')

# 进入容器的网络 namespace
nsenter -t $PID -n ip a       # 看容器的网络接口
nsenter -t $PID -n ss -tlnp   # 看容器监听的端口

# 进入所有 namespace（等同于 docker exec）
nsenter -t $PID -a bash
```

## 四、动手验证：Cgroup 限制

```bash
# 在 kind 节点上查看某个容器的 cgroup 限制
# cgroup v2 路径
cat /sys/fs/cgroup/system.slice/containerd-<id>.scope/memory.max
cat /sys/fs/cgroup/system.slice/containerd-<id>.scope/cpu.max

# 或者用 kubectl 直接看
kubectl describe pod <pod-name> | grep -A2 "Limits"
```

### 验证内存限制

```bash
# 创建一个限制 64Mi 内存的 Pod
kubectl run mem-test --image=busybox --limits=memory=64Mi -- sh -c "sleep 3600"

# 查看 cgroup 设置
kubectl exec mem-test -- cat /sys/fs/cgroup/memory.max
# 输出 67108864（64 * 1024 * 1024 字节）

# 清理
kubectl delete pod mem-test
```

## 五、K8s 排障综合演练

### 场景 1：CrashLoopBackOff

```bash
# 制造故障
kubectl apply -f scenarios/crash-loop.yaml

# 排障
kubectl get pods crash-demo              # STATUS: CrashLoopBackOff
kubectl describe pod crash-demo          # Events 有 Error
kubectl logs crash-demo --previous       # 看上次崩溃日志

# 原因：容器启动命令不存在
# 修复：改正 command 或 image
kubectl delete pod crash-demo
```

### 场景 2：OOMKilled

```bash
# 制造故障
kubectl apply -f scenarios/oom-killed.yaml

# 排障
kubectl get pods oom-demo                # STATUS: OOMKilled 或 CrashLoopBackOff
kubectl describe pod oom-demo            # 看 Last State: OOMKilled

# 原因：应用使用内存超过 limits
# 修复：增加 memory limits 或优化应用内存使用
kubectl delete pod oom-demo
```

### 场景 3：ImagePullBackOff

```bash
# 制造故障
kubectl apply -f scenarios/image-pull-error.yaml

# 排障
kubectl get pods pull-demo               # STATUS: ImagePullBackOff
kubectl describe pod pull-demo           # Events 里有具体错误
# "Failed to pull image: rpc error... not found"

# 原因：镜像不存在或无权限
# 修复：修正镜像名，或配置 imagePullSecrets
kubectl delete pod pull-demo
```

### 场景 4：Service 不通

```bash
# 制造故障
kubectl apply -f scenarios/wrong-selector-svc.yaml

# 排障
kubectl get endpoints svc-demo           # 空！
kubectl get svc svc-demo -o yaml         # selector: app=wrong
kubectl get pods --show-labels           # 真实 label 是什么

# 修复
kubectl patch svc svc-demo -p '{"spec":{"selector":{"app":"go-app-chart"}}}'
kubectl get endpoints svc-demo           # 有 IP 了

# 清理
kubectl delete svc svc-demo
```

### 场景 5：Pod Pending（资源不足）

```bash
# 制造故障
kubectl apply -f scenarios/pending-pod.yaml

# 排障
kubectl get pods pending-demo            # STATUS: Pending
kubectl describe pod pending-demo        # Events: "Insufficient cpu/memory"

# 原因：requests 超过集群可用资源
# 修复：降低 requests 或增加节点
kubectl delete pod pending-demo
```

## 六、排障 Checklist 速查表

### Pod 级别

```
Pod 状态异常？
├── Pending
│   └── describe pod → Events
│       ├── "Insufficient cpu" → 降低 requests / 加节点
│       ├── "no nodes available" → 检查 node taint/tolerations
│       └── "Unschedulable" → 检查 PVC / affinity
├── CrashLoopBackOff
│   └── logs --previous → 看崩溃日志
│       ├── 命令不存在 → 修正 command/image
│       ├── 配置错误 → 检查 env/configmap
│       └── 端口冲突 → 修改 containerPort
├── OOMKilled
│   └── describe pod → Last State
│       └── 增加 memory limits 或优化应用
├── ImagePullBackOff
│   └── describe pod → Events
│       ├── "not found" → 修正镜像名
│       ├── "unauthorized" → 配置 imagePullSecrets
│       └── "timeout" → 检查网络 / registry 可达性
└── Running 但不正常
    └── logs → 应用级别错误
```

### 网络级别

```
Service 不通？
├── get endpoints → 空？
│   └── selector 不匹配 Pod labels
├── get endpoints → 有 IP？
│   ├── 端口不对 → 检查 targetPort
│   ├── Pod 没监听 → exec ss -tlnp
│   └── NetworkPolicy → get networkpolicy
└── DNS 不通？
    ├── CoreDNS Running? → get pods -n kube-system
    └── resolv.conf 对? → exec cat /etc/resolv.conf
```

## 七、常见问题

**Q: 在 Windows/Mac 上能做 namespace 实验吗？**

不能，namespace 是 Linux 内核特性。但你可以：
- 进入 kind 节点（它是 Linux 容器）：`podman exec -it k8s-lab-worker bash`
- 使用 WSL2

**Q: cgroup v1 和 v2 的区别？**

- v1：每种资源独立目录（`/sys/fs/cgroup/cpu/`, `/sys/fs/cgroup/memory/`）
- v2：统一层级（`/sys/fs/cgroup/`），kind 默认用 v2

**Q: OOMKilled 但应用没泄漏？**

可能是 limits 设太低。用 `kubectl top pods` 看实际内存使用，设 limits 为实际用量的 1.5-2 倍。

**Q: CrashLoopBackOff 的重试间隔？**

K8s 用指数退避：10s → 20s → 40s → ... 最长 5 分钟。修复后不想等？直接删 Pod 让 Deployment 重建。

## 八、10 天学习总结

```
Day 1:  Docker 实操 → 理解容器化
Day 2:  K8s 集群 + Pod → 理解编排
Day 3:  Service + Ingress → 理解网络
Day 4:  存储 + 工作负载 → 理解有状态
Day 5:  client-go → 用代码操作 K8s
Day 6:  Operator → 扩展 K8s
Day 7:  CI/CD + Helm → 自动化
Day 8:  网络排障 → 排障方法论
Day 9:  Prometheus → 可观测性
Day 10: 底层原理 + 综合排障 → 融会贯通
```

## ✅ 最终产出验收

- [ ] 能在 kind 节点内用 unshare 演示 PID/Network namespace 隔离
- [ ] 能用 nsenter 进入容器 namespace 查看网络和进程
- [ ] 能独立排障 5 种故障场景
- [ ] 有自己整理的排障 checklist
- [ ] 10 天课程全部完成！

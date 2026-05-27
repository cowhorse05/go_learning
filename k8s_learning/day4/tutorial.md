# Day 4：K8s 存储 + 工作负载 + 弹性

> 前三天你学会了无状态应用（Deployment），今天学有状态应用（数据库）、各种工作负载类型，以及自动扩缩容。

## 一、存储：为什么需要持久化？

### 问题

容器的文件系统是**临时的**：Pod 重启后数据全丢。跑数据库这种东西，数据不能丢。

### K8s 存储三件套

```
StorageClass（存储模板）
    ↓ 根据 PVC 的请求自动创建
PV (PersistentVolume)（实际存储资源）
    ↓ 绑定
PVC (PersistentVolumeClaim)（Pod 的存储申请）
    ↓ 挂载
Pod 里的容器
```

| 概念 | 类比 | 说明 |
|------|------|------|
| PV | 仓库里的货架 | 集群级别的存储资源（一块磁盘、一个 NFS 目录） |
| PVC | 申请单 | "我需要 1Gi 的存储"，K8s 自动匹配或创建 PV |
| StorageClass | 货架型号目录 | 定义用什么方式创建 PV（SSD、HDD、NFS 等） |

### 静态 vs 动态

- **静态**：管理员手动创建 PV → 用户创建 PVC → K8s 匹配绑定
- **动态**：配置 StorageClass → 用户创建 PVC → K8s 自动创建 PV（推荐）

> kind 内置了一个 StorageClass（`standard`），默认使用 hostPath，适合本地测试。

## 二、工作负载类型

### 全家福

| 类型 | 适用场景 | 是否有状态 | 典型示例 |
|------|----------|-----------|---------|
| Deployment | 无状态应用 | 否 | Web 服务、API |
| StatefulSet | 有状态应用 | 是 | 数据库、消息队列 |
| DaemonSet | 每个 Node 跑一个 | 否 | 日志采集、监控 agent |
| Job | 一次性任务 | 否 | 数据迁移、批处理 |
| CronJob | 定时任务 | 否 | 定时备份、清理 |

### StatefulSet vs Deployment

| 特性 | Deployment | StatefulSet |
|------|-----------|-------------|
| Pod 名称 | 随机（go-app-abc123） | 有序（mysql-0, mysql-1） |
| 网络标识 | 不稳定 | 稳定（通过 Headless Service） |
| 存储 | Pod 删了就没了 | 每个 Pod 绑定独立 PVC，删 Pod 数据还在 |
| 扩缩容 | 并行 | 按序（0→1→2 创建，2→1→0 删除） |
| 用途 | 无状态服务 | 数据库、有状态中间件 |

## 三、HPA —— 自动扩缩容

### 原理

```
metrics-server 采集 Pod 的 CPU/内存指标
    ↓
HPA Controller 定期检查（默认 15s）
    ↓
当前 CPU 使用率 > 目标值？ → 扩容（增加 Pod）
当前 CPU 使用率 < 目标值？ → 缩容（减少 Pod）
```

### 公式

```
期望副本数 = ceil(当前副本数 × (当前指标值 / 目标指标值))
```

例如：当前 3 个 Pod，平均 CPU 80%，目标 50%：
```
期望副本数 = ceil(3 × (80 / 50)) = ceil(4.8) = 5
```

## 四、实战：StatefulSet 部署 MySQL

### Headless Service（StatefulSet 必须）

```yaml
# mysql-headless-svc.yaml
apiVersion: v1
kind: Service
metadata:
  name: mysql
spec:
  clusterIP: None      # Headless Service：不分配 ClusterIP
  selector:
    app: mysql
  ports:
  - port: 3306
```

> **Headless Service 是什么？** `clusterIP: None` 表示不创建虚拟 IP，DNS 直接返回 Pod IP。StatefulSet 需要它来给每个 Pod 提供稳定的 DNS 名称：`mysql-0.mysql.default.svc.cluster.local`。

### StatefulSet

```yaml
# mysql-statefulset.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  serviceName: mysql       # 关联 Headless Service
  replicas: 1
  selector:
    matchLabels:
      app: mysql
  template:
    metadata:
      labels:
        app: mysql
    spec:
      containers:
      - name: mysql
        image: mysql:8.0
        env:
        - name: MYSQL_ROOT_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret       # 复用 Day 3 创建的 Secret
              key: DB_PASSWORD
        ports:
        - containerPort: 3306
        volumeMounts:
        - name: mysql-data
          mountPath: /var/lib/mysql
  volumeClaimTemplates:
  - metadata:
      name: mysql-data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 1Gi
```

**关键字段：**

| 字段 | 含义 |
|------|------|
| serviceName | 关联的 Headless Service 名称 |
| volumeClaimTemplates | 每个 Pod 自动创建独立的 PVC |
| accessModes: ReadWriteOnce | 一次只能被一个 Node 挂载读写 |

### 部署和验证

```bash
# 确保 Day 3 的 db-secret 还在
kubectl get secret db-secret

# 部署
kubectl apply -f mysql-headless-svc.yaml
kubectl apply -f mysql-statefulset.yaml

# 查看状态（StatefulSet 启动慢，MySQL 镜像大）
kubectl get statefulset mysql
kubectl get pods -l app=mysql -w

# 查看自动创建的 PVC
kubectl get pvc
# NAME                STATUS   VOLUME     CAPACITY   ACCESS MODES   AGE
# mysql-data-mysql-0  Bound    pvc-xxx    1Gi        RWO            1m

# 进入 MySQL 验证
kubectl exec -it mysql-0 -- mysql -uroot -pmypassword123 -e "CREATE DATABASE testdb; SHOW DATABASES;"
```

### 验证持久化：删 Pod 后数据还在

```bash
# 删除 Pod（StatefulSet 会自动重建 mysql-0）
kubectl delete pod mysql-0

# 等重建完成
kubectl get pods -l app=mysql -w

# 再次查看数据库 —— testdb 还在！
kubectl exec -it mysql-0 -- mysql -uroot -pmypassword123 -e "SHOW DATABASES;"
```

## 五、实战：DaemonSet

```yaml
# daemonset.yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: log-agent
spec:
  selector:
    matchLabels:
      app: log-agent
  template:
    metadata:
      labels:
        app: log-agent
    spec:
      tolerations:
      - key: node-role.kubernetes.io/control-plane
        effect: NoSchedule
      containers:
      - name: agent
        image: busybox
        command: ["sh", "-c", "while true; do echo $(date) collecting logs from $(hostname); sleep 60; done"]
```

> 加了 `tolerations` 让 DaemonSet 也能跑在 control-plane 节点上。

```bash
# 部署
kubectl apply -f daemonset.yaml

# 查看 —— 每个 Node 上都有一个 Pod
kubectl get pods -l app=log-agent -o wide
# NAME              READY   STATUS    NODE
# log-agent-abc     1/1     Running   k8s-lab-control-plane
# log-agent-def     1/1     Running   k8s-lab-worker
# log-agent-ghi     1/1     Running   k8s-lab-worker2

# 查看日志
kubectl logs -l app=log-agent --tail=1
```

## 六、实战：Job 和 CronJob

### 一次性 Job

```yaml
# job.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: hello-job
spec:
  template:
    spec:
      containers:
      - name: hello
        image: busybox
        command: ["echo", "Hello from K8s Job!"]
      restartPolicy: Never
  backoffLimit: 3
```

### CronJob（定时任务）

```yaml
# cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: cleanup
spec:
  schedule: "*/5 * * * *"      # 每 5 分钟执行
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: cleanup
            image: busybox
            command: ["sh", "-c", "echo cleanup done at $(date)"]
          restartPolicy: OnFailure
```

```bash
# 部署
kubectl apply -f job.yaml
kubectl apply -f cronjob.yaml

# 查看 Job 执行结果
kubectl get jobs
kubectl logs job/hello-job

# 查看 CronJob
kubectl get cronjob
kubectl get jobs -w   # 等几分钟看定时触发
```

## 七、实战：HPA 自动扩缩容

### 安装 metrics-server

kind 集群需要额外配置 metrics-server（默认证书验证会失败）：

```yaml
# metrics-server-patch.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: metrics-server
  namespace: kube-system
spec:
  template:
    spec:
      containers:
      - name: metrics-server
        args:
        - --cert-dir=/tmp
        - --secure-port=10250
        - --kubelet-preferred-address-types=InternalIP
        - --kubelet-insecure-tls    # kind 必须加这个，跳过证书验证
```

```bash
# 安装
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# 打补丁（kind 必须）
kubectl patch deploy metrics-server -n kube-system --type=json -p='[
  {"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}
]'

# 等待就绪
kubectl -n kube-system rollout status deploy metrics-server

# 验证（等 1-2 分钟让它采集数据）
kubectl top nodes
kubectl top pods
```

### 配置 HPA

```bash
# 给 go-app 设置资源请求（HPA 需要 requests 才能计算百分比）
kubectl set resources deploy go-app --requests=cpu=50m,memory=64Mi --limits=cpu=200m,memory=128Mi

# 创建 HPA
kubectl autoscale deploy go-app --min=2 --max=10 --cpu-percent=50

# 查看 HPA
kubectl get hpa
```

### 压测触发扩容

```bash
# 启动压测（持续请求 go-app-svc）
kubectl run load-gen --image=busybox --restart=Never -- sh -c "while true; do wget -q -O- http://go-app-svc; done"

# 另一个终端监控
kubectl get hpa -w
kubectl get pods -l app=go-app -w

# 观察到 Pod 数从 2 增加到更多
# 停止压测
kubectl delete pod load-gen

# 等几分钟，Pod 数会缩回到 2
```

## 八、常见问题和踩坑记录

**Q: StatefulSet 的 Pod 一直 Pending？**

`kubectl describe pod mysql-0` 看 Events。常见原因：
- PVC 绑定失败：`kubectl get pvc`，如果是 Pending 状态说明没有可用的 PV/StorageClass
- kind 默认有 `standard` StorageClass，正常应该没问题

**Q: `kubectl top` 报 "metrics not available"？**

metrics-server 需要 1-2 分钟启动和采集。确认：
```bash
kubectl get pods -n kube-system | grep metrics
# metrics-server 必须是 Running 且 READY 1/1
```

**Q: HPA 显示 `<unknown>/50%`？**

两个原因：
1. metrics-server 还没采集到数据，等 1-2 分钟
2. Deployment 没设置 `resources.requests`，HPA 无法计算百分比

**Q: DaemonSet 的 Pod 在 control-plane 上没跑？**

control-plane 节点有 taint `node-role.kubernetes.io/control-plane:NoSchedule`，需要加 tolerations。

**Q: CronJob 没触发？**

`kubectl describe cronjob cleanup` 看 Events 和 Last Schedule Time。schedule 的 cron 表达式格式是 `分 时 日 月 周`。

## 九、概念总结

```
工作负载全家福
├── Deployment         ← 无状态（Web/API）
├── StatefulSet        ← 有状态（DB/MQ），稳定网络+持久存储
├── DaemonSet          ← 每个 Node 一个（日志/监控）
├── Job                ← 跑一次就完
└── CronJob            ← 定时跑

存储链路
StorageClass → (动态创建) PV ← (绑定) PVC ← (挂载) Pod

弹性伸缩
metrics-server → HPA Controller → 调整 Deployment.replicas
```

| 概念 | 类比 |
|------|------|
| PV | 仓库货架（存储空间） |
| PVC | 领料单（"给我 1G 空间"） |
| StorageClass | 货架规格（SSD 还是 HDD） |
| StatefulSet | 有固定工号和专属储物柜的员工 |
| DaemonSet | 每层楼都要放一个灭火器 |
| Job | 一次性搬家任务 |
| CronJob | 每天下班后的自动打扫 |
| HPA | 根据排队人数自动加开窗口 |

## 十、今日产出验收

```bash
# 1. MySQL StatefulSet 运行
kubectl get statefulset mysql
kubectl get pvc

# 2. 数据持久化验证
kubectl exec -it mysql-0 -- mysql -uroot -pmypassword123 -e "SHOW DATABASES;"
# testdb 应该还在

# 3. DaemonSet 每个 Node 一个 Pod
kubectl get pods -l app=log-agent -o wide

# 4. HPA 状态
kubectl get hpa
```

- [ ] MySQL StatefulSet 运行，PVC 绑定成功
- [ ] 删 Pod 后数据还在（testdb 仍存在）
- [ ] DaemonSet 在每个 Node 上都有 Pod
- [ ] HPA 创建成功，能响应负载变化
- [ ] 理解 StatefulSet vs Deployment 的核心区别

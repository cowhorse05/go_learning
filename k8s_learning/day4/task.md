Day 4：K8s 存储 + 工作负载 + 弹性
上午：概念（1.5h）
存储
- PV (PersistentVolume)：集群级别的存储资源
- PVC (PersistentVolumeClaim)：Pod 对存储的申请
- StorageClass：动态创建 PV 的模板

工作负载类型
| 类型 | 用途 |
|------|------|
| Deployment | 无状态应用 |
| StatefulSet | 有状态应用（稳定网络标识、持久存储） |
| DaemonSet | 每个 Node 跑一个 Pod（日志/监控 agent） |
| Job | 一次性任务 |
| CronJob | 定时任务 |

HPA (Horizontal Pod Autoscaler)
- 基于 CPU/内存/自定义指标自动扩缩 Pod 数量
- 需要 metrics-server

下午：实战（4h）
1. StatefulSet 部署 MySQL
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  serviceName: mysql
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
              name: db-secret
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

2. DaemonSet 示例
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
      containers:
      - name: agent
        image: busybox
        command: ["sh", "-c", "while true; do echo collecting logs; sleep 60; done"]

3. HPA 配置 + 压测
# 安装 metrics-server（kind 需要额外配置）
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# 给 Deployment 设置资源限制
kubectl set resources deploy go-app --requests=cpu=50m,memory=64Mi --limits=cpu=200m,memory=128Mi

# 创建 HPA
kubectl autoscale deploy go-app --min=2 --max=10 --cpu-percent=50

# 压测触发扩容
kubectl run load-gen --image=busybox -- sh -c "while true; do wget -q -O- http://go-app-svc; done"

# 观察
kubectl get hpa -w
kubectl get pods -w

4. Job 和 CronJob
apiVersion: batch/v1
kind: CronJob
metadata:
  name: cleanup
spec:
  schedule: "*/5 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: cleanup
            image: busybox
            command: ["echo", "cleanup done"]
          restartPolicy: OnFailure

晚上：今日产出验收
- [ ] MySQL StatefulSet 运行，数据持久化（删 Pod 后数据还在）
- [ ] HPA 压测时自动扩容，压测停止后缩容
- [ ] 理解 PV/PVC 绑定关系
- [ ] 能解释 StatefulSet vs Deployment 的区别

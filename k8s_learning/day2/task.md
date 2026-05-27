# Day 2：K8s 集群搭建 + Pod 基础

## 上午：架构理解 + 环境搭建（2h）

### K8s 架构核心组件

- **API Server**：所有操作的入口，RESTful API
- **etcd**：集群状态存储（key-value）
- **Scheduler**：决定 Pod 调度到哪个 Node
- **Controller Manager**：维护期望状态（Deployment/ReplicaSet controller）
- **Kubelet**：每个 Node 上的 agent，管理 Pod 生命周期
- **kube-proxy**：Service 的网络规则（iptables/ipvs）

### 搭建 kind 集群

```bash
# 安装 kind
go install sigs.k8s.io/kind@latest

# 创建 3 节点集群
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
EOF

# 验证
kubectl get nodes
kubectl cluster-info
```

## 下午：Pod 与 Deployment 实操（4h）

### 1. 理解 Pod

```bash
# 运行一个 Pod
kubectl run nginx --image=nginx:alpine
kubectl get pods -o wide
kubectl describe pod nginx
kubectl logs nginx
kubectl exec -it nginx -- sh
kubectl delete pod nginx
```

### 2. Deployment 管理

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
        image: <你Day1构建的镜像>
        ports:
        - containerPort: 8080
```

### 3. 实战操作

```bash
kubectl apply -f deployment.yaml
kubectl get deploy,rs,pods
kubectl scale deploy go-app --replicas=5
kubectl rollout status deploy go-app
kubectl set image deploy/go-app go-app=<新镜像>
kubectl rollout history deploy go-app
kubectl rollout undo deploy go-app
```

### 4. 将 Day 1 的 Go 服务部署到 kind

```bash
# 加载本地镜像到 kind
kind load docker-image myapp:v1
# 部署
kubectl apply -f deployment.yaml
kubectl port-forward deploy/go-app 8080:8080
# 浏览器访问 http://localhost:8080
```

## 晚上：kubectl 命令复盘（30min）

高频命令必须肌肉记忆：

- `kubectl get/describe/logs/exec/apply/delete`
- `kubectl port-forward`
- `kubectl get pods -o wide/yaml/json`
- `kubectl explain deployment.spec.strategy`

# Day 3：K8s 网络与配置

> 昨天你学会了把 Pod 跑起来，但 Pod 之间怎么互相找到对方？外部怎么访问？配置怎么管理？今天解决这些问题。

## 一、为什么需要 Service？

### Pod IP 的问题

每个 Pod 有自己的 IP，但这个 IP 有两个致命问题：

1. **不稳定**：Pod 重启后 IP 会变（Deployment 滚动更新、扩缩容都会产生新 Pod）
2. **不能负载均衡**：你有 3 个 go-app Pod，客户端不知道该访问哪个

Service 就是解决这两个问题的：它提供一个**稳定的虚拟 IP（ClusterIP）**，自动将流量分发到后端的多个 Pod。

### 流量链路

```
客户端请求
    ↓
Service（稳定 IP + 端口）
    ↓ kube-proxy 通过 iptables/ipvs 转发
Endpoints（Pod IP 列表，自动维护）
    ↓
Pod 1 / Pod 2 / Pod 3（轮询负载均衡）
```

## 二、Service 类型

| 类型 | 访问范围 | 用途 | 类比 |
|------|----------|------|------|
| ClusterIP | 仅集群内部 | 微服务间调用 | 公司内网分机号 |
| NodePort | 集群外通过 `NodeIP:端口` | 开发测试 | 公司前台电话 |
| LoadBalancer | 外部负载均衡器 | 云环境生产 | 400 客服热线 |
| ExternalName | DNS 别名 | 引用外部服务 | 通讯录里存的外号 |

### ClusterIP（默认）

最常用。创建后得到一个虚拟 IP（如 `10.96.100.50`），只在集群内可用。其他 Pod 可以通过这个 IP 或 **Service 名称**（DNS）访问。

### NodePort

在 ClusterIP 基础上，额外在每个 Node 的某个端口（默认 30000-32767）暴露服务。外部可通过 `<任意NodeIP>:<NodePort>` 访问。

### 什么时候用什么？

- 服务只给集群内其他服务调用 → ClusterIP
- 开发调试，需要从本机访问 → NodePort 或 `kubectl port-forward`
- 生产环境对外暴露 → LoadBalancer（云）或 Ingress

## 三、Ingress —— L7 层路由

### Service vs Ingress

| | Service (NodePort) | Ingress |
|---|---|---|
| 层级 | L4（TCP/UDP） | L7（HTTP/HTTPS） |
| 路由能力 | 只能按端口 | 按域名、路径 |
| 证书终结 | 不支持 | 支持 HTTPS |
| 一个端口多服务 | 不行 | 可以 |

### 工作原理

```
外部请求 → Ingress Controller（Nginx Pod）
              ↓ 根据 Host/Path 规则匹配
           Service A（api.example.com）
           Service B（web.example.com）
           Service C（example.com/admin）
```

Ingress 本身只是一个**规则定义**（YAML），真正干活的是 **Ingress Controller**（如 nginx-ingress-controller）。没装 Controller，Ingress 规则不生效。

## 四、ConfigMap 和 Secret

### 为什么不把配置写死在镜像里？

同一个镜像可能要跑在开发、测试、生产不同环境，配置不同（数据库地址、日志级别等）。ConfigMap/Secret 让你把配置和镜像解耦。

| | ConfigMap | Secret |
|---|---|---|
| 用途 | 非敏感配置 | 敏感数据（密码、密钥、证书） |
| 存储 | 明文 | base64 编码（注意：不是加密！） |
| 注入方式 | 环境变量 / 挂载文件 | 环境变量 / 挂载文件 |
| 示例 | APP_ENV=production | DB_PASSWORD=xxx |

> **Secret 安全吗？** base64 不是加密，任何人 `kubectl get secret -o yaml` 都能解码。生产环境要配合 RBAC 权限控制 + etcd 加密。

## 五、实战：创建 Service

### ClusterIP Service

```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: go-app-svc
spec:
  selector:
    app: go-app        # 匹配 Pod 标签
  ports:
  - port: 80           # Service 对外暴露的端口
    targetPort: 8080   # Pod 实际监听的端口
  type: ClusterIP
```

**关键字段解读：**

| 字段 | 含义 |
|------|------|
| selector | 通过标签匹配后端 Pod。必须和 Deployment 的 Pod labels 一致 |
| port | Service 自己监听的端口（集群内访问用这个） |
| targetPort | 转发到 Pod 的端口（你 Go 服务监听 8080） |
| type | Service 类型，默认 ClusterIP |

### NodePort Service

```yaml
# service-nodeport.yaml
apiVersion: v1
kind: Service
metadata:
  name: go-app-nodeport
spec:
  selector:
    app: go-app
  ports:
  - port: 80
    targetPort: 8080
    nodePort: 30080    # 指定 Node 上的端口（30000-32767）
  type: NodePort
```

### 部署和验证

```bash
# 确保 Day 2 的 go-app Deployment 还在运行
kubectl get deploy go-app

# 创建 Service
kubectl apply -f service.yaml
kubectl apply -f service-nodeport.yaml

# 查看 Service
kubectl get svc
# NAME              TYPE        CLUSTER-IP      PORT(S)        AGE
# go-app-svc        ClusterIP   10.96.x.x       80/TCP         5s
# go-app-nodeport   NodePort    10.96.x.x       80:30080/TCP   5s

# 查看 Endpoints（Service 自动发现的后端 Pod）
kubectl get endpoints go-app-svc
# NAME         ENDPOINTS                                      AGE
# go-app-svc   10.244.1.5:8080,10.244.1.6:8080,10.244.2.3:8080

# 从集群内部测试 ClusterIP（用一个临时 Pod）
kubectl run curl-test --image=curlimages/curl --rm -it -- curl http://go-app-svc:80
# 看到 Go 服务返回的内容就对了

# 测试 NodePort
# 如果 kind 集群创建时配了 extraPortMappings 30080，可以直接：
curl http://localhost:30080
# 如果没配（现有集群），用 port-forward 替代验证：
kubectl port-forward svc/go-app-nodeport 30080:80
curl http://localhost:30080
```

### Service DNS

集群内任何 Pod 都可以通过 DNS 名称访问 Service：

```
<service-name>.<namespace>.svc.cluster.local
```

简写（同 namespace 内）：
```
go-app-svc         → 完整是 go-app-svc.default.svc.cluster.local
```

## 六、实战：Ingress

### 安装 Ingress Controller（kind 专用）

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

# 等待 Controller 就绪
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=120s

# 验证
kubectl get pods -n ingress-nginx
```

### 创建 Ingress 规则

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: go-app-ingress
spec:
  rules:
  - host: myapp.local
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: go-app-svc
            port:
              number: 80
```

### 配置本机 hosts

Ingress 需要通过域名访问。在本机 hosts 文件添加：

```
# Windows: C:\Windows\System32\drivers\etc\hosts
# Linux/Mac: /etc/hosts
127.0.0.1 myapp.local
```

### 验证

```bash
# 需要 kind 集群的 extraPortMappings 或 port-forward
kubectl port-forward -n ingress-nginx svc/ingress-nginx-controller 8080:80

# 然后访问
curl -H "Host: myapp.local" http://localhost:8080
# 或浏览器打开 http://myapp.local:8080（如果 hosts 已配置）
```

## 七、实战：ConfigMap 和 Secret

### 创建 ConfigMap

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  APP_ENV: "production"
  LOG_LEVEL: "info"
```

### 创建 Secret

```yaml
# secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: db-secret
type: Opaque
stringData:          # stringData 自动 base64 编码，比手动写 data 方便
  DB_PASSWORD: "mypassword123"
```

### 在 Deployment 中引用

```yaml
# deployment-with-config.yaml
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
        envFrom:
        - configMapRef:
            name: app-config       # 把 ConfigMap 所有键注入为环境变量
        env:
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: DB_PASSWORD     # 单独注入某个 Secret 键
```

### 部署和验证

```bash
# 创建 ConfigMap 和 Secret
kubectl apply -f configmap.yaml
kubectl apply -f secret.yaml

# 更新 Deployment
kubectl apply -f deployment-with-config.yaml

# 验证环境变量注入
kubectl exec -it deploy/go-app -- env | grep -E "APP_ENV|LOG_LEVEL|DB_PASSWORD"
# APP_ENV=production
# LOG_LEVEL=info
# DB_PASSWORD=mypassword123

# 查看 Secret 的 base64 编码
kubectl get secret db-secret -o yaml
# data:
#   DB_PASSWORD: bXlwYXNzd29yZDEyMw==

# 解码验证
echo "bXlwYXNzd29yZDEyMw==" | base64 -d
# mypassword123
```

## 八、深入理解：Service → Endpoints → Pod

### 排障关键：Endpoints

当 Service 不通时，第一步看 Endpoints：

```bash
kubectl get endpoints go-app-svc
```

- **Endpoints 有 IP**：Service 到 Pod 的路由正常，问题在别的地方
- **Endpoints 为空**：Service 的 selector 没匹配到任何 Pod（标签写错了）

### 手动模拟故障

```bash
# 修改 Pod 标签，让 Service 找不到它
kubectl label pod <pod-name> app=wrong --overwrite

# 观察 Endpoints 变化
kubectl get endpoints go-app-svc
# 少了一个 IP！

# 恢复
kubectl label pod <pod-name> app=go-app --overwrite
```

## 九、常见问题和踩坑记录

**Q: Service 创建了但 curl 不通？**

排查顺序：
1. `kubectl get endpoints <svc-name>` — Endpoints 为空说明 selector 不匹配
2. `kubectl get pods -l app=go-app` — Pod 是否 Running
3. Pod 内部端口对不对 — `kubectl exec <pod> -- ss -tlnp`

**Q: NodePort 从宿主机访问不到？**

kind 特殊：节点是容器，NodePort 不直接暴露到宿主机。需要在 `kind-config.yaml` 中配置 `extraPortMappings`（Day 2 已配置了 30080）。

**Q: Ingress 规则创建了但不生效？**

1. Ingress Controller 装了吗？`kubectl get pods -n ingress-nginx`
2. 域名解析到 127.0.0.1 了吗？检查 hosts 文件
3. Controller 日志：`kubectl logs -n ingress-nginx deploy/ingress-nginx-controller`

**Q: ConfigMap 改了但 Pod 里的环境变量没变？**

环境变量注入是 Pod 启动时读取的，**修改 ConfigMap 后需要重启 Pod**：

```bash
kubectl rollout restart deploy go-app
```

如果用 Volume 挂载方式（而非 env），大约 1 分钟后自动更新。

**Q: Secret 的 stringData 和 data 有什么区别？**

- `stringData`：写明文，K8s 自动 base64 编码存储（推荐，方便阅读）
- `data`：需要你自己先 base64 编码后再写入

## 十、概念总结

```
外部流量 → Ingress（L7 路由：按域名/路径）
              ↓
           Service（L4 负载均衡：稳定 IP + DNS）
              ↓
           Endpoints（Pod IP 列表，自动维护）
              ↓
           Pod 1 / Pod 2 / Pod 3
              ↑
        配置注入：ConfigMap（明文）/ Secret（敏感）
```

| 概念 | 类比 |
|------|------|
| ClusterIP Service | 内网 DNS（只有公司内部能用） |
| NodePort Service | 在门口贴了个端口号，外面能打进来 |
| Ingress | 公司前台，根据你找谁转接到不同分机 |
| Ingress Controller | 前台坐着的真人（没人就转不了） |
| ConfigMap | 办公室白板上的公共信息 |
| Secret | 锁在抽屉里的密码本 |
| Endpoints | Service 的通讯录（自动更新） |

## 十一、今日产出验收

```bash
# 1. Service 工作
kubectl get svc go-app-svc go-app-nodeport
kubectl get endpoints go-app-svc

# 2. 集群内访问
kubectl run curl-test --image=curlimages/curl --rm -it -- curl http://go-app-svc:80

# 3. NodePort 外部访问
curl http://localhost:30080

# 4. ConfigMap/Secret 注入
kubectl exec -it deploy/go-app -- env | grep APP_ENV
```

- [ ] ClusterIP Service 使集群内 Pod 通过 `go-app-svc:80` 互通
- [ ] NodePort Service 从宿主机通过 `localhost:30080` 可访问
- [ ] ConfigMap/Secret 正确注入为环境变量
- [ ] 理解 Endpoints 为空时的排障思路

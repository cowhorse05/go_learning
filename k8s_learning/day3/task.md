# Day 3：K8s 网络与配置

## 上午：Service 与 Ingress 概念（1.5h）

### Service 类型

| 类型 | 访问范围 | 用途 |
|------|----------|------|
| ClusterIP | 集群内部 | 默认，服务间调用 |
| NodePort | 集群外通过 Node IP:Port | 开发测试 |
| LoadBalancer | 外部负载均衡 | 云环境生产 |

### Ingress

- L7 层路由（HTTP/HTTPS），基于域名/路径分发
- 需要 Ingress Controller（如 nginx-ingress）

### ConfigMap & Secret

- ConfigMap：非敏感配置（环境变量、配置文件）
- Secret：敏感数据（密码、证书），base64 编码存储

## 下午：实战（4h）

### 1. 创建 Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: go-app-svc
spec:
  selector:
    app: go-app
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP
---
# NodePort 版本供外部访问
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
    nodePort: 30080
  type: NodePort
```

### 2. 安装 Nginx Ingress Controller + 配置 Ingress

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl wait --namespace ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=90s
```

```yaml
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

### 3. ConfigMap 和 Secret 实践

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  APP_ENV: "production"
  LOG_LEVEL: "info"
---
apiVersion: v1
kind: Secret
metadata:
  name: db-secret
type: Opaque
stringData:
  DB_PASSWORD: "mypassword123"
```

在 Deployment 中引用：

```yaml
envFrom:
- configMapRef:
    name: app-config
env:
- name: DB_PASSWORD
  valueFrom:
    secretKeyRef:
      name: db-secret
      key: DB_PASSWORD
```

### 4. 验证

```bash
# 测试 Service
kubectl exec -it <pod> -- curl go-app-svc:80
# 测试 Ingress（需在 /etc/hosts 加 myapp.local）
curl http://myapp.local
# 验证 ConfigMap 注入
kubectl exec -it <pod> -- env | grep APP_ENV
```

## ✅ 今日产出验收

- [ ] ClusterIP Service 使集群内服务互通
- [ ] Ingress 通过域名路由到服务
- [ ] ConfigMap/Secret 正确注入到 Pod
- [ ] 理解 Service → Endpoints → Pod 的链路

# Day 8：网络排障

> 你说网络排障是弱项，今天重点补。

## 上午：K8s 网络模型（2h）

### 三层网络

1. **Pod 网络**：每个 Pod 有独立 IP，Pod 间可直接通信（CNI 实现）
2. **Service 网络**：虚拟 IP（ClusterIP），kube-proxy 通过 iptables/ipvs 转发
3. **外部网络**：NodePort / LoadBalancer / Ingress 暴露到集群外

### CNI (Container Network Interface)

- 负责给 Pod 分配 IP、配置路由
- 常见：Calico、Cilium、Flannel
- kind 默认用 kindnet

### kube-proxy 工作模式

- **iptables 模式**：规则多时性能下降
- **ipvs 模式**：内核级负载均衡，大规模集群推荐

### DNS

- CoreDNS 为集群内提供 DNS 解析
- Service DNS：`<svc-name>.<namespace>.svc.cluster.local`
- Pod DNS：`<pod-ip-dashed>.<namespace>.pod.cluster.local`

## 下午：排障实战（4h）

### 1. 基础工具

```bash
# 在 Pod 内排障
kubectl run debug --image=nicolaka/netshoot -it --rm -- bash

# DNS 排查
nslookup go-app-svc.default.svc.cluster.local
dig @10.96.0.10 go-app-svc.default.svc.cluster.local

# 连通性测试
curl http://go-app-svc:80
wget -qO- http://go-app-svc:80
```

### 2. tcpdump 抓包

```bash
# 用 ephemeral container
kubectl debug <pod> -it --image=nicolaka/netshoot -- tcpdump -i eth0 port 8080
```

### 3. Service 不通排障 Checklist

```bash
# 1. 检查 Pod 是否运行
kubectl get pods -l app=go-app

# 2. 检查 Service selector 是否匹配
kubectl get svc go-app-svc -o yaml | grep selector
kubectl get pods -l app=go-app

# 3. 检查 Endpoints
kubectl get endpoints go-app-svc

# 4. 检查 Pod 端口是否正确
kubectl exec <pod> -- ss -tlnp

# 5. 检查 NetworkPolicy
kubectl get networkpolicy

# 6. 检查 iptables 规则
iptables -t nat -L KUBE-SERVICES | grep go-app
```

### 4. 常见网络问题模拟与修复

- **DNS 不解析**：检查 CoreDNS Pod、resolv.conf
- **Pod 间不通**：检查 CNI、Node 路由
- **Service 不通**：Endpoints 为空 / selector 错误
- **Ingress 不通**：Ingress Controller 日志、后端 Service

## ✅ 今日产出验收

- [ ] 能用 tcpdump 抓到 Pod 间的 HTTP 流量
- [ ] 能排障 Service 不通的问题（走完 checklist）
- [ ] 理解 ClusterIP → iptables → Pod 的流量路径
- [ ] 能用 nslookup/dig 排查 DNS 问题

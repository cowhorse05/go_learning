# Day 8：网络排障

> 网络问题是 K8s 排障中最常见也最头疼的。今天建立排障方法论：不靠猜，靠 checklist 一步步定位。

## 一、K8s 网络模型

### 三层网络

```
┌─────────────────────────────────────────┐
│ 外部网络                                │
│ （NodePort / LoadBalancer / Ingress）    │
└───────────────────┬─────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│ Service 网络（虚拟 IP，kube-proxy 实现）│
│ ClusterIP: 10.96.0.0/12                 │
└───────────────────┬─────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│ Pod 网络（CNI 实现，每个 Pod 一个 IP）  │
│ Pod CIDR: 10.244.0.0/16                 │
└─────────────────────────────────────────┘
```

### 核心规则

1. **Pod 到 Pod**：任何两个 Pod 都能直接通信（不需要 NAT）
2. **Pod 到 Service**：通过 ClusterIP / DNS 名称访问
3. **外部到 Pod**：必须经过 NodePort / Ingress / LoadBalancer

### CNI 是什么？

CNI（Container Network Interface）负责：
- 给 Pod 分配 IP
- 配置 Pod 之间的路由
- 不同 Node 上的 Pod 如何互通

| CNI | 特点 |
|-----|------|
| kindnet | kind 默认，简单 |
| Flannel | 轻量，用 VXLAN 隧道 |
| Calico | 功能全，支持 NetworkPolicy |
| Cilium | eBPF 驱动，性能最好 |

### DNS

CoreDNS 是集群的 DNS 服务器：

```
Service DNS:  <service>.<namespace>.svc.cluster.local
简写（同NS）: <service>

例如: go-app-chart.default.svc.cluster.local
简写: go-app-chart
```

## 二、排障方法论

### 黄金法则：从近到远

```
1. Pod 本身有问题吗？（Running? Ready?）
2. Pod 端口在监听吗？（ss -tlnp）
3. Service 能找到 Pod 吗？（Endpoints 有内容吗？）
4. DNS 能解析 Service 吗？（nslookup）
5. 网络策略阻止了吗？（NetworkPolicy）
6. 外部入口配置对吗？（Ingress / NodePort）
```

### Service 不通完整 Checklist

```bash
# Step 1: Pod 在运行吗？
kubectl get pods -l <selector>
# 不是 Running → 先解决 Pod 问题

# Step 2: Pod 端口在监听吗？
kubectl exec <pod> -- ss -tlnp
# 确认你期望的端口有 LISTEN

# Step 3: Service 的 Endpoints 有内容吗？
kubectl get endpoints <service-name>
# 空 → selector 不匹配或 Pod 不 Ready

# Step 4: 从集群内能访问 Service 吗？
kubectl run debug --image=curlimages/curl --rm -it -- curl http://<svc>:<port>
# 不通 → kube-proxy / iptables 问题

# Step 5: DNS 能解析吗？
kubectl run debug --image=nicolaka/netshoot --rm -it -- nslookup <svc>
# 不能 → CoreDNS 问题

# Step 6: NetworkPolicy 阻止了吗？
kubectl get networkpolicy -A
# 有 policy → 检查是否允许目标流量
```

## 三、排障工具箱

### netshoot —— 万能调试镜像

```bash
# 启动一个临时调试 Pod
kubectl run debug --image=nicolaka/netshoot -it --rm -- bash

# 里面有所有网络工具
nslookup go-app-chart
curl http://go-app-chart:80
ping 10.244.1.5
traceroute 10.244.2.3
tcpdump -i eth0 port 80
ss -tlnp
ip route
iptables -t nat -L
```

### tcpdump 抓包

```bash
# 方法 1: 用 debug 容器（K8s 1.25+）
kubectl debug <pod-name> -it --image=nicolaka/netshoot -- tcpdump -i eth0 -nn port 8080

# 方法 2: 启动独立调试 Pod
kubectl run tcpdump --image=nicolaka/netshoot --rm -it -- tcpdump -i any host <pod-ip> -nn
```

### 常用排障命令速查

| 目的 | 命令 |
|------|------|
| 查 Pod IP | `kubectl get pods -o wide` |
| 查 Endpoints | `kubectl get endpoints <svc>` |
| 查 DNS | `nslookup <svc>.default.svc.cluster.local` |
| 查端口监听 | `kubectl exec <pod> -- ss -tlnp` |
| 查 iptables | `iptables -t nat -L KUBE-SERVICES` |
| 抓包 | `tcpdump -i eth0 -nn port <port>` |
| 查路由 | `ip route` |
| 查 CoreDNS | `kubectl logs -n kube-system -l k8s-app=kube-dns` |

## 四、故障场景实战

### 场景 1：Service selector 不匹配

```bash
# 制造故障：创建一个 selector 错误的 Service
kubectl apply -f scenarios/wrong-selector.yaml

# 排障
kubectl get endpoints wrong-svc     # 空！
kubectl get svc wrong-svc -o yaml   # selector: app=wrong
kubectl get pods --show-labels       # 实际 label 是 app=go-app-chart

# 修复
kubectl patch svc wrong-svc -p '{"spec":{"selector":{"app":"go-app-chart"}}}'
kubectl get endpoints wrong-svc     # 有 IP 了
```

### 场景 2：端口不匹配

```bash
# 制造故障：targetPort 写错
kubectl apply -f scenarios/wrong-port.yaml

# 排障
kubectl get endpoints wrong-port-svc  # 有 IP（selector 是对的）
kubectl run debug --image=curlimages/curl --rm -it -- curl http://wrong-port-svc:80
# connection refused！

# 看 Service 配置
kubectl get svc wrong-port-svc -o yaml  # targetPort: 9999（但 Pod 监听 8080）

# 修复
kubectl patch svc wrong-port-svc --type=json -p='[{"op":"replace","path":"/spec/ports/0/targetPort","value":8080}]'
```

### 场景 3：NetworkPolicy 阻止流量

```bash
# 制造故障：创建一个拒绝所有入站流量的 Policy
kubectl apply -f scenarios/deny-all-policy.yaml

# 排障
kubectl run debug --image=curlimages/curl --rm -it -- curl --max-time 5 http://go-app-chart:80
# timeout！

kubectl get networkpolicy
kubectl describe networkpolicy deny-all

# 修复
kubectl delete networkpolicy deny-all
```

## 五、深入理解 Service 流量路径

### ClusterIP 流量路径

```
Pod A curl go-app-chart:80
    ↓ DNS 解析
CoreDNS 返回 ClusterIP（如 10.96.20.81）
    ↓
iptables/ipvs 规则匹配 10.96.20.81:80
    ↓ DNAT（目标地址转换）
转发到后端 Pod IP（如 10.244.1.5:8080）
    ↓
Pod B 收到请求
```

### 查看 iptables 规则

```bash
# 进入 kind 节点
docker exec -it k8s-lab-worker bash
# 或 podman
podman exec -it k8s-lab-worker bash

# 查看 Service 相关规则
iptables -t nat -L KUBE-SERVICES | grep go-app
iptables -t nat -L KUBE-SVC-<hash>
```

## 六、常见问题和踩坑记录

**Q: `nslookup` 返回 NXDOMAIN？**

1. Service 不存在：`kubectl get svc`
2. 命名空间错了：完整 DNS 是 `svc.namespace.svc.cluster.local`
3. CoreDNS 挂了：`kubectl get pods -n kube-system -l k8s-app=kube-dns`

**Q: Endpoints 有 IP 但 curl 还是不通？**

1. Pod 端口没在监听（`ss -tlnp` 确认）
2. Pod 里的应用绑定了 127.0.0.1 而不是 0.0.0.0
3. NetworkPolicy 阻止了

**Q: Pod 间 ping 不通？**

1. CNI 没正常工作：`kubectl get pods -n kube-system`
2. Node 间路由问题（多 Node 集群）
3. 防火墙/安全组阻止了（云环境）

**Q: 间歇性不通？**

可能是负载均衡问题：某个后端 Pod 不健康但没被剔除。检查 readinessProbe。

## 七、概念总结

```
排障路线图
Pod 不通？ → 先确认 Pod Running
    ↓
Service 不通？ → 看 Endpoints
    ↓ 空？→ selector 不匹配
    ↓ 有？→ 端口对不对 / NetworkPolicy
DNS 不通？ → 看 CoreDNS
    ↓
外部不通？ → 看 Ingress / NodePort 配置
```

| 概念 | 类比 |
|------|------|
| Pod 网络 | 每个人有手机号（Pod IP） |
| Service | 总机号（ClusterIP），转接到工位（Pod） |
| Endpoints | 总机的转接名单 |
| kube-proxy | 电话交换机 |
| CoreDNS | 电话簿（名字 → 号码） |
| NetworkPolicy | 来电过滤（黑名单/白名单） |
| tcpdump | 电话录音（抓包分析） |

## ✅ 今日产出验收

- [ ] 能用 netshoot Pod 从集群内 curl Service
- [ ] 能排障 Endpoints 为空的问题（selector 不匹配）
- [ ] 能排障端口不匹配的问题
- [ ] 理解 ClusterIP → iptables → Pod 流量路径
- [ ] 能用 nslookup 排查 DNS 问题

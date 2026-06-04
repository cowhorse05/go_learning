# K8s 学习环境安装指南

## 你需要装的（按顺序）

### 1. k3s — 轻量 K8s 集群

```bash
# 用 sudo 安装（安装脚本会放到 /usr/local/bin）
curl -sfL https://get.k3s.io | sudo sh -

# 验证安装成功
sudo kubectl get nodes
# 应该看到: NAME   STATUS   ROLES   AGE   VERSION
```

k3s 装完后会自动启动，`kubectl` 命令也会配好。

### 2. 给你的用户配 kubectl 权限

因为 k3s 用 sudo 装的，配置文件属于 root。执行以下操作让自己的用户也能用 kubectl：

```bash
# 拷贝 kubeconfig 到自己的目录
mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown $USER:$USER ~/.kube/config
```

之后就可以不用 sudo 直接 `kubectl get nodes` 了。

### 3. 验证环境

```bash
kubectl get nodes          # 看集群节点
kubectl get pods -A        # 看所有 pod
kubectl version --short    # 看版本
```

---

## 不需要装的

- **Docker**：k3s 自带 containerd，不需要额外装 Docker
- **minikube / kind**：k3s 更简单，一条命令搞定

---

## 装完之后告诉我

装完验证成功后告诉我，我帮你把 `day1/main.go` 的 HTTP 服务打包成镜像并部署到 K8s 上。

# Day 5：Go + client-go

> 前 4 天你用 kubectl 命令操作 K8s，今天学用 Go 代码操作。kubectl 底层就是调 K8s API，client-go 是官方 Go 客户端库。

## 一、K8s API 基础

### 一切皆资源

K8s 的设计哲学：所有东西都是**资源**（Resource），通过 RESTful API 操作。

```
kubectl get pods
   → GET /api/v1/namespaces/default/pods

kubectl scale deploy go-app --replicas=5
   → PUT /apis/apps/v1/namespaces/default/deployments/go-app/scale
```

### GVR —— 资源的"坐标"

每个资源都有唯一的坐标，叫 GVR（Group / Version / Resource）：

| 资源 | Group | Version | Resource |
|------|-------|---------|----------|
| Pod | "" (core) | v1 | pods |
| Deployment | apps | v1 | deployments |
| Service | "" (core) | v1 | services |
| Ingress | networking.k8s.io | v1 | ingresses |

### client-go 的核心模块

| 模块 | 作用 | 类比 |
|------|------|------|
| `kubernetes.Clientset` | 强类型客户端，每种资源有专门方法 | 按菜单点菜 |
| `dynamic.Interface` | 通用客户端，用 map 操作任何资源 | 随意搭配 |
| `Informer` | List + Watch + 本地缓存 | 订阅消息 + 本地副本 |
| `Workqueue` | 事件队列，控制处理速率 | 排队系统 |

## 二、环境准备

```bash
# 在 day5 目录初始化
cd k8s_learning/day5
go mod init k8s-tool
go get k8s.io/client-go@latest
go get k8s.io/apimachinery@latest
```

## 三、连接集群

```go
import (
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/util/homedir"
)

// 读取 ~/.kube/config 创建客户端
kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
clientset, err := kubernetes.NewForConfig(config)
```

**关键点：**
- `kubeconfig` 文件是 kubectl 用的同一个配置文件
- `Clientset` 创建后可复用，不需要每次操作都重新创建
- 在 Pod 内运行时用 `rest.InClusterConfig()` 自动获取 ServiceAccount 凭证

## 四、实战：CLI 工具

我们写一个 `k8s-tool`，支持以下命令：

```
k8s-tool pods                     列出所有 Pod
k8s-tool list                     列出所有 Deployment
k8s-tool scale <name> <replicas>  扩缩容
k8s-tool restart <name>           滚动重启
```

### 核心函数

**列出 Pod：**
```go
func listPods(clientset kubernetes.Interface, namespace string) error {
    pods, err := clientset.CoreV1().Pods(namespace).List(
        context.TODO(), metav1.ListOptions{})
    for _, pod := range pods.Items {
        fmt.Printf("Pod: %s | Status: %s | Node: %s\n",
            pod.Name, pod.Status.Phase, pod.Spec.NodeName)
    }
    return nil
}
```

**扩缩容：**
```go
func scaleDeployment(clientset kubernetes.Interface, namespace, name string, replicas int32) error {
    scale, _ := clientset.AppsV1().Deployments(namespace).GetScale(
        context.TODO(), name, metav1.GetOptions{})
    scale.Spec.Replicas = replicas
    _, err := clientset.AppsV1().Deployments(namespace).UpdateScale(
        context.TODO(), name, scale, metav1.UpdateOptions{})
    return err
}
```

**滚动重启：**
```go
func restartDeployment(clientset kubernetes.Interface, namespace, name string) error {
    // 和 kubectl rollout restart 一样的原理：
    // 给 Pod template 加一个 annotation 触发 rollout
    patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{
        "kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
    _, err := clientset.AppsV1().Deployments(namespace).Patch(
        context.TODO(), name, types.StrategicMergePatchType,
        []byte(patch), metav1.PatchOptions{})
    return err
}
```

> **注意：** 函数参数用 `kubernetes.Interface` 而不是 `*kubernetes.Clientset`，这样单元测试可以传入 fake client。

## 五、单元测试（fake clientset）

client-go 提供了 `k8s.io/client-go/kubernetes/fake` 包，无需真实集群就能测试：

```go
import "k8s.io/client-go/kubernetes/fake"

func TestScaleDeployment(t *testing.T) {
    // 创建一个假客户端，预置一个 Deployment
    clientset := fake.NewSimpleClientset(
        &appsv1.Deployment{
            ObjectMeta: metav1.ObjectMeta{Name: "go-app", Namespace: "default"},
            Spec:       appsv1.DeploymentSpec{Replicas: int32Ptr(3)},
        },
    )

    // 调用被测函数
    err := scaleDeployment(clientset, "default", "go-app", 5)
    if err != nil {
        t.Fatal(err)
    }

    // 验证结果
    deploy, _ := clientset.AppsV1().Deployments("default").Get(
        context.TODO(), "go-app", metav1.GetOptions{})
    if *deploy.Spec.Replicas != 5 {
        t.Errorf("expected 5, got %d", *deploy.Spec.Replicas)
    }
}
```

### 为什么用 fake client？

| | 真实集群 | fake clientset |
|---|---|---|
| 速度 | 慢（网络请求） | 毫秒级 |
| 依赖 | 需要运行的集群 | 无依赖 |
| CI 友好 | 需要额外配置 | 开箱即用 |
| 适合 | 集成测试 | 单元测试 |

## 六、Watch 机制 —— 实时监控

```go
watcher, _ := clientset.CoreV1().Pods("default").Watch(
    context.TODO(), metav1.ListOptions{})

for event := range watcher.ResultChan() {
    pod := event.Object.(*corev1.Pod)
    fmt.Printf("[%s] Pod %s: %s\n",
        event.Type, pod.Name, pod.Status.Phase)
}
```

Watch 会建立一个**长连接**，实时推送变化事件（ADDED/MODIFIED/DELETED）。

### Watch 的问题

- 连接断了怎么办？需要重连 + 断点续传（resourceVersion）
- 每次重连都要 List 全量？浪费
- 多个地方都 Watch 同一资源？重复连接

→ 所以有了 Informer。

## 七、Informer —— 生产级别的 Watch

```go
import (
    "k8s.io/client-go/informers"
    "k8s.io/client-go/tools/cache"
)

factory := informers.NewSharedInformerFactory(clientset, 30*time.Second)
podInformer := factory.Core().V1().Pods().Informer()

podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc:    func(obj interface{}) { fmt.Println("Pod Added") },
    UpdateFunc: func(old, new interface{}) { fmt.Println("Pod Updated") },
    DeleteFunc: func(obj interface{}) { fmt.Println("Pod Deleted") },
})

// 启动
stopCh := make(chan struct{})
factory.Start(stopCh)
factory.WaitForCacheSync(stopCh)
// 阻塞等待
<-stopCh
```

### Informer 工作原理

```
API Server
    ↓ List（第一次全量获取）
Local Cache（本地存一份所有 Pod 的数据）
    ↓ Watch（后续只接收增量变化）
Event Handler（你的回调函数）
```

### 为什么 Informer 比裸 Watch 好？

| 特性 | 裸 Watch | Informer |
|------|---------|----------|
| 断线重连 | 需要自己实现 | 内置 |
| 本地缓存 | 没有 | 有，读取不走 API Server |
| 多个使用者 | 各自建连接 | 共享一个连接（Shared Informer） |
| 生产就绪 | 不是 | 是 |

## 八、运行验证

```bash
# 编译
go build -o k8s-tool .

# 列出 Pod
./k8s-tool pods

# 列出 Deployment
./k8s-tool list

# 扩容
./k8s-tool scale go-app 5
kubectl get pods -l app=go-app   # 验证变成 5 个

# 缩容
./k8s-tool scale go-app 3

# 滚动重启
./k8s-tool restart go-app
kubectl rollout status deploy go-app

# 运行单元测试
go test -v ./...
```

## 九、常见问题和踩坑记录

**Q: `go get k8s.io/client-go` 报错？**

client-go 版本要和集群 K8s 版本兼容。通常用 `@latest` 即可。

**Q: 函数参数用 `kubernetes.Interface` 还是 `*kubernetes.Clientset`？**

用 `kubernetes.Interface`（接口）。这样测试时可以传入 `fake.NewSimpleClientset()`。

**Q: Informer 的 resyncPeriod 设多少？**

`30*time.Second` 表示每 30 秒强制全量同步一次（即使没有变化）。生产环境通常设 0（禁用定期 resync）或更长时间。

**Q: 在 Pod 内运行时找不到 kubeconfig？**

在 K8s Pod 内不用 kubeconfig，用 ServiceAccount：
```go
config, err := rest.InClusterConfig()
```

## 十、概念总结

```
kubectl get pods
    ↓ 其实是调用
GET /api/v1/namespaces/default/pods
    ↓ client-go 封装为
clientset.CoreV1().Pods("default").List(...)
    ↓ 生产级别
Informer = List + Watch + Cache + EventHandler
```

| 概念 | 类比 |
|------|------|
| Clientset | 按类型整理好的 API 工具箱 |
| GVR | 资源的邮编地址 |
| Watch | 实时直播（连接断了就没了） |
| Informer | 订阅 + 录像（断了自动续，还有本地缓存） |
| fake.Clientset | 模拟器（不需要真集群就能测试） |

## ✅ 今日产出验收

- [ ] `go build` 编译通过
- [ ] `go test -v ./...` 4 个测试全部 PASS
- [ ] `./k8s-tool pods` 列出集群中的 Pod
- [ ] `./k8s-tool scale go-app 5` 能扩容 Deployment
- [ ] 理解 Informer 的 List-Watch-Cache 机制

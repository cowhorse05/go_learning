Day 5：Go + client-go
> 你有 Go 基础（写过聊天室），今天学习用 Go 操作 K8s API。

上午：client-go 基础（2h）
K8s API 核心概念
- GVR：Group/Version/Resource（如 apps/v1/deployments）
- client-go 模块：
  - kubernetes.Clientset：typed client（强类型）
  - dynamic.Interface：unstructured client（通用）
  - Informer：List-Watch 缓存机制
  - Workqueue：事件队列

环境准备
mkdir k8s-tool && cd k8s-tool
go mod init k8s-tool
go get k8s.io/client-go@latest
go get k8s.io/apimachinery@latest

下午：实战编码（4h）
任务 1：列出所有 Pod 并监控状态
package main

import (
    "context"
    "fmt"
    "path/filepath"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/util/homedir"
)

func main() {
    kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
    config, _ := clientcmd.BuildConfigFromFlags("", kubeconfig)
    clientset, _ := kubernetes.NewForConfig(config)

    pods, _ := clientset.CoreV1().Pods("default").List(
        context.TODO(), metav1.ListOptions{})

    for _, pod := range pods.Items {
        fmt.Printf("Pod: %s | Status: %s | Node: %s\n",
            pod.Name, pod.Status.Phase, pod.Spec.NodeName)
    }
}

任务 2：Watch Pod 事件（实时监控）
// 使用 Watch 接口监听 Pod 变化
watcher, _ := clientset.CoreV1().Pods("default").Watch(
    context.TODO(), metav1.ListOptions{})

for event := range watcher.ResultChan() {
    pod := event.Object.(*v1.Pod)
    fmt.Printf("[%s] Pod %s: %s\n",
        event.Type, pod.Name, pod.Status.Phase)
}

任务 3：自动化 Deployment 管理
写一个 CLI 工具：
- ./k8s-tool list — 列出所有 Deployment 及副本数
- ./k8s-tool scale <name> <replicas> — 扩缩容
- ./k8s-tool restart <name> — 滚动重启

任务 4：Informer 机制体验
// Informer = List + Watch + 本地缓存
// 避免每次都请求 API Server
factory := informers.NewSharedInformerFactory(clientset, 30*time.Second)
podInformer := factory.Core().V1().Pods().Informer()

podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc:    func(obj interface{}) { fmt.Println("Pod Added") },
    UpdateFunc: func(old, new interface{}) { fmt.Println("Pod Updated") },
    DeleteFunc: func(obj interface{}) { fmt.Println("Pod Deleted") },
})

晚上：复盘（30min）
- client-go 的 Informer 为什么比直接 Watch 更好？
- 什么场景用 typed client vs dynamic client？

今日产出验收
- [ ] 能用 Go 连接 K8s 集群并 List Pod
- [ ] Watch 机制能实时输出 Pod 状态变化
- [ ] CLI 工具能 scale Deployment
- [ ] 理解 Informer 的 List-Watch-Cache 模式

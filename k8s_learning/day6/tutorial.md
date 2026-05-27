# Day 6：K8s Operator 开发

> 前面你一直在用别人定义好的资源（Deployment、Service），今天你自己定义一种新资源，并写代码让 K8s 自动管理它。

## 一、什么是 Operator？

### 一句话

Operator = **自定义资源（CRD）** + **自定义控制器（Controller）**

你告诉 K8s："我发明了一种新东西叫 SimpleApp"，然后写一段代码告诉 K8s "当有人创建 SimpleApp 时，帮他自动创建 Deployment + Service"。

### 为什么需要 Operator？

| 方式 | 适合 | 问题 |
|------|------|------|
| 手动 kubectl | 一次性操作 | 不可重复，人会犯错 |
| Shell 脚本 | 简单自动化 | 不能感知变化，不能自愈 |
| Helm | 模板化部署 | 只管安装不管运行期 |
| **Operator** | 全生命周期 | 安装、更新、自愈、清理全自动 |

### Reconcile Loop —— 核心思想

```
          ┌─────────────────────────────┐
          │                             │
          ▼                             │
  观察当前状态 → 对比期望状态 → 有差异？ → 修正 → 回到观察
                                │
                                └─ 无差异 → 等待下次触发
```

这就是**声明式 API** 的精髓：你不说"执行 A、B、C 步骤"，你只说"我要最终状态是 X"，Controller 自己想办法达到。

## 二、CRD —— 自定义资源

### 什么是 CRD？

CRD（Custom Resource Definition）让你扩展 K8s API，定义新资源类型。

定义后你就能：
```bash
kubectl get simpleapps
kubectl describe simpleapp my-web
kubectl delete simpleapp my-web
```

### CR 示例

```yaml
# simpleapp-sample.yaml
apiVersion: apps.example.com/v1
kind: SimpleApp
metadata:
  name: my-web
spec:
  image: nginx:alpine
  replicas: 3
  port: 80
```

## 三、实战项目结构

我们**不用** Kubebuilder 脚手架（那个生成太多文件），直接用 controller-runtime 写一个精简版，理解核心原理：

```
day6/
├── main.go                    # 入口：启动 Manager
├── controller.go              # Reconcile 逻辑
├── controller_test.go         # 单元测试
├── api/
│   └── v1/
│       └── types.go           # CRD 类型定义
├── config/
│   ├── crd.yaml               # CRD 定义 YAML
│   └── sample.yaml            # CR 示例
└── go.mod
```

## 四、代码实现

### 1. CRD 类型定义 (api/v1/types.go)

```go
package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// SimpleApp 是我们自定义的资源
type SimpleApp struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec              SimpleAppSpec   `json:"spec,omitempty"`
    Status            SimpleAppStatus `json:"status,omitempty"`
}

type SimpleAppSpec struct {
    Image    string `json:"image"`
    Replicas int32  `json:"replicas"`
    Port     int32  `json:"port"`
}

type SimpleAppStatus struct {
    ReadyReplicas int32  `json:"readyReplicas"`
    Phase         string `json:"phase"`
}

type SimpleAppList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []SimpleApp `json:"items"`
}
```

### 2. Controller（Reconcile 逻辑）

核心步骤：
1. 收到事件 → 获取 SimpleApp CR
2. 构造期望的 Deployment（根据 CR 的 spec）
3. 看集群里有没有这个 Deployment
   - 没有 → 创建
   - 有但 spec 不同 → 更新
4. 更新 CR 的 Status

```go
func (r *SimpleAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. 获取 CR
    var app v1.SimpleApp
    if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 2. 构造期望的 Deployment
    desired := r.desiredDeployment(&app)

    // 3. CreateOrUpdate
    // ...

    // 4. 更新 Status
    app.Status.ReadyReplicas = found.Status.ReadyReplicas
    r.Status().Update(ctx, &app)

    return ctrl.Result{}, nil
}
```

### 3. OwnerReference —— 级联删除

```go
ctrl.SetControllerReference(&app, deploy, r.Scheme)
```

设置 OwnerReference 后，删除 SimpleApp CR 时，它创建的 Deployment 和 Service 会**自动被删除**。

## 五、关键概念深入

### Reconcile 为什么要幂等？

同一个事件可能被触发多次（网络重试、Controller 重启等）。如果你的 Reconcile 不幂等：

```
第一次：创建 Deployment ✓
第二次：又创建一个 Deployment ✗ （名字冲突报错）
```

幂等的写法：先 Get，存在就 Update，不存在才 Create。

### Level-triggered vs Edge-triggered

| | Edge-triggered | Level-triggered |
|---|---|---|
| 含义 | "发生了变化"才触发 | "当前状态不对"就触发 |
| K8s 方式 | ✗ | ✓ |
| 好处 | — | 不怕遗漏事件，Controller 重启后自愈 |

K8s Controller 是 **level-triggered**：不关心"发生了什么"，只关心"当前状态 vs 期望状态"。

### Manager 和 Controller 的关系

```
Manager（进程级别）
├── Controller A（Watch SimpleApp）
│   └── Reconcile()
├── Controller B（Watch 其他资源）
│   └── Reconcile()
├── Cache（共享的 Informer 缓存）
└── Client（操作 K8s API）
```

## 六、运行和测试

### 安装 CRD

```bash
kubectl apply -f config/crd.yaml
kubectl get crd simpleapps.apps.example.com
```

### 运行 Controller（本地）

```bash
go run . 
# 或
go build -o operator . && ./operator
```

### 创建 CR，观察效果

```bash
# 创建 SimpleApp
kubectl apply -f config/sample.yaml

# Controller 应该自动创建 Deployment
kubectl get deploy my-web
kubectl get pods -l simpleapp=my-web

# 修改副本数
kubectl patch simpleapp my-web --type=merge -p '{"spec":{"replicas":5}}'
kubectl get pods -l simpleapp=my-web   # 变成 5 个

# 删除 CR → 级联删除 Deployment
kubectl delete simpleapp my-web
kubectl get deploy my-web   # 应该 NotFound
```

### 单元测试

```bash
go test -v ./...
```

## 七、对比 Kubebuilder

| | 我们的精简版 | Kubebuilder |
|---|---|---|
| 文件数 | ~5 个 | ~50 个 |
| 理解难度 | 低（核心逻辑清晰） | 高（大量脚手架） |
| 生产就绪 | 否（缺少 webhook、RBAC 等） | 是 |
| 学习价值 | 理解原理 | 工程实践 |

**建议路径**：先看懂我们的精简版 → 再用 Kubebuilder 生成完整项目 → 对照理解每个文件的作用。

### Kubebuilder 快速上手（可选）

```bash
go install sigs.k8s.io/kubebuilder/v4/cmd/kubebuilder@latest
mkdir app-operator && cd app-operator
kubebuilder init --domain example.com --repo app-operator
kubebuilder create api --group apps --version v1 --kind SimpleApp --resource --controller
# 生成完整项目结构，在 internal/controller/ 下编写 Reconcile 逻辑
make manifests && make install && make run
```

## 八、常见问题和踩坑记录

**Q: CRD apply 后 kubectl get simpleapps 报错？**

确认 CRD 的 `group` + `plural` 拼写正确。`kubectl get crd` 看是否安装成功。

**Q: Reconcile 没被触发？**

1. Controller 是否 Watch 了正确的资源？
2. RBAC 权限够吗？本地运行用 kubeconfig 权限足够
3. `kubectl logs` 看 Controller 输出

**Q: 删 CR 后 Deployment 没被删？**

OwnerReference 没设置。确保在创建 Deployment 时调用了 `ctrl.SetControllerReference`。

**Q: 更新 CR 后 Deployment 没变？**

Reconcile 里 Update 逻辑是否正确？需要 Get → 修改 → Update，不能直接 Create。

## 九、概念总结

```
用户创建 SimpleApp CR
        ↓
Controller 感知到事件（via Informer/Watch）
        ↓
Reconcile() 被调用
        ↓
对比期望状态（CR.Spec）vs 实际状态（Deployment）
        ↓
创建/更新/删除子资源
        ↓
更新 CR.Status
```

| 概念 | 类比 |
|------|------|
| CRD | 发明一种新表格（自定义资源类型） |
| CR | 填好的一张表（资源实例） |
| Controller | 审批员（看到表就干活） |
| Reconcile | 审批流程（反复检查直到完成） |
| OwnerReference | 文件归属（删主文件，附件跟着删） |
| Manager | 办公室主任（管所有审批员） |

## ✅ 今日产出验收

- [ ] `go build` 编译通过
- [ ] `go test -v` 测试通过
- [ ] CRD 安装到集群，`kubectl get simpleapps` 可用
- [ ] 创建 CR 后 Deployment 自动出现
- [ ] 修改 CR replicas 后 Pod 数跟随变化
- [ ] 删除 CR 后 Deployment 被级联删除

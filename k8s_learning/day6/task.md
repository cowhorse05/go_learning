Day 6：K8s Operator 开发
> Operator = CRD + Controller，是 K8s 生态的核心扩展模式，对混合云部门尤其重要。

上午：Operator 概念（1.5h）
核心思想
- CRD (Custom Resource Definition)：自定义 K8s 资源类型
- Controller：Watch CR 变化 → Reconcile 到期望状态
- Reconcile Loop：不断对比 desired state vs actual state，修正差异

开发框架选择
- controller-runtime：底层库（Kubebuilder/Operator-SDK 的基础）
- Kubebuilder：脚手架工具，推荐入门用

下午：实战 — 用 Kubebuilder 写一个简单 Operator（4h）
1. 初始化项目
# 安装 kubebuilder
go install sigs.k8s.io/kubebuilder/v4/cmd/kubebuilder@latest

# 创建项目
mkdir app-operator && cd app-operator
kubebuilder init --domain example.com --repo app-operator
kubebuilder create api --group apps --version v1 --kind SimpleApp --resource --controller

2. 定义 CRD（api/v1/simpleapp_types.go）
type SimpleAppSpec struct {
    // 应用镜像
    Image string `json:"image"`
    // 副本数
    Replicas int32 `json:"replicas"`
    // 端口
    Port int32 `json:"port"`
}

type SimpleAppStatus struct {
    // 当前就绪副本数
    ReadyReplicas int32 `json:"readyReplicas"`
    // 状态
    Phase string `json:"phase"`
}

3. 实现 Reconcile 逻辑
Controller 要做的事：
1. 获取 SimpleApp CR
2. 检查对应的 Deployment 是否存在
3. 不存在 → 创建 Deployment + Service
4. 存在 → 对比 spec，有变化则更新
5. 更新 Status

func (r *SimpleAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    var app appsv1.SimpleApp
    if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 确保 Deployment 存在且 spec 一致
    deploy := &appsv1.Deployment{}
    // ... 创建或更新逻辑

    // 更新 status
    app.Status.ReadyReplicas = deploy.Status.ReadyReplicas
    r.Status().Update(ctx, &app)

    return ctrl.Result{}, nil
}

4. 测试运行
make manifests      # 生成 CRD YAML
make install        # 安装 CRD 到集群
make run            # 本地运行 controller

# 创建 CR
kubectl apply -f config/samples/apps_v1_simpleapp.yaml
# 观察 controller 自动创建 Deployment
kubectl get deploy,svc

晚上：复盘
- Reconcile 为什么要幂等？
- OwnerReference 的作用是什么？（级联删除）
- 与直接写脚本管理应用的区别？

今日产出验收
- [ ] CRD 成功安装到集群
- [ ] 创建 SimpleApp CR 后，Controller 自动创建 Deployment + Service
- [ ] 修改 CR 的 replicas 后，Deployment 副本数跟随变化
- [ ] 删除 CR 后，关联资源被级联清理

Day 10：容器底层原理 + 排障综合
> 你八股已知道原理，今天动手验证 + 综合排障演练。

上午：动手验证容器隔离（2h）
1. 用 unshare 手动创建隔离环境
# PID namespace 隔离
sudo unshare --pid --fork --mount-proc bash
ps aux  # 只能看到当前 bash

# Network namespace 隔离
sudo unshare --net bash
ip a  # 只有 loopback

# 组合：模拟一个 "容器"
sudo unshare --pid --net --mount --fork --mount-proc bash

2. nsenter 进入容器 namespace
# 找到容器进程 PID
PID=$(docker inspect --format {{.State.Pid}} <container>)

# 进入其 network namespace
sudo nsenter -t $PID -n ip a
sudo nsenter -t $PID -n ss -tlnp

# 进入所有 namespace
sudo nsenter -t $PID -a bash

3. Cgroup 限制验证
# 创建一个限制 CPU 的容器
docker run -d --cpus=0.5 --memory=128m --name limited stress --cpu 2

# 查看 cgroup 配置
cat /sys/fs/cgroup/cpu/docker/<container-id>/cpu.cfs_quota_us
cat /sys/fs/cgroup/memory/docker/<container-id>/memory.limit_in_bytes

# 观察限制效果
docker stats limited

下午：K8s 排障综合演练（4h）
场景 1：Pod CrashLoopBackOff
# 制造问题：错误的启动命令
kubectl run crash --image=busybox -- /bin/nonexistent

# 排障步骤
kubectl describe pod crash          # 看 Events
kubectl logs crash --previous       # 看上次崩溃的日志
kubectl get pod crash -o yaml       # 看 containerStatuses.lastState

场景 2：OOMKilled
# 制造问题：内存不够
kubectl run oom --image=polinux/stress --limits=memory=64Mi -- stress --vm 1 --vm-bytes 128M

# 排障
kubectl describe pod oom  # 看 OOMKilled
kubectl get pod oom -o jsonpath="{.status.containerStatuses[0].lastState}"
# 修复：增加 memory limits

场景 3：ImagePullBackOff
# 制造问题：错误的镜像名
kubectl run badimage --image=nonexistent/image:v99

# 排障
kubectl describe pod badimage  # Events 里有 pull 失败原因
# 修复：检查镜像名、仓库认证（imagePullSecrets）

场景 4：Service 不通
# 制造问题：selector 不匹配
kubectl create deploy web --image=nginx
kubectl expose deploy web --port=80 --selector=app=wrong

# 排障
kubectl get endpoints web          # 空！
kubectl get pods --show-labels     # 找到正确 label
# 修复：kubectl patch svc web -p '{"spec":{"selector":{"app":"web"}}}'

场景 5：Node NotReady
# 排障
kubectl describe node <node>       # 看 Conditions
kubectl get pods -n kube-system    # 检查 kubelet 相关组件
journalctl -u kubelet              # Node 上看 kubelet 日志

排障 Checklist 总结
| 症状 | 第一步 | 关键命令 |
|------|--------|----------|
| Pending | 资源不足/调度失败 | describe pod 看 Events |
| CrashLoopBackOff | 应用崩溃 | logs --previous |
| OOMKilled | 内存超限 | describe pod + 增加 limits |
| ImagePullBackOff | 镜像拉取失败 | describe pod 看 pull 错误 |
| Service 不通 | Endpoints 为空 | get endpoints + 检查 selector |
| Node NotReady | kubelet 异常 | describe node + journalctl |

今日产出验收
- [ ] 能用 unshare/nsenter 演示容器隔离
- [ ] 能独立排障以上 5 种故障场景
- [ ] 整理了自己的排障 checklist
- [ ] 10 天学习完成，能力自测通过！

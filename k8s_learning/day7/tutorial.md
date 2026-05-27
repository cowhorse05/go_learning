# Day 7：CI/CD 流水线

> 前面你手动 build 镜像、手动 kubectl apply。今天学自动化：代码一提交，自动测试、打包、部署。

## 一、CI/CD 是什么？

### 一句话

- **CI (Continuous Integration)**：代码合并后自动跑测试、lint，确保不 break
- **CD (Continuous Delivery/Deployment)**：测试通过后自动打包部署到环境

### 流水线全貌

```
开发者 push 代码
    ↓
GitHub Actions 触发
    ↓
┌─────────────────────────────────────┐
│ 1. Lint（代码风格检查）              │
│ 2. Test（单元测试）                  │
│ 3. Build（编译 + 构建 Docker 镜像） │
│ 4. Push（推送镜像到 Registry）       │
│ 5. Deploy（更新 K8s Deployment）     │
└─────────────────────────────────────┘
    ↓
用户访问新版本
```

### 为什么需要 CI/CD？

| 手动部署 | CI/CD |
|----------|-------|
| 每次手动操作容易出错 | 全自动，步骤可重复 |
| "在我机器上能跑" | 统一环境，结果一致 |
| 部署慢，一天部署一次 | 分钟级部署，一天多次 |
| 出问题难回滚 | 一键回滚 |

## 二、Helm —— K8s 的包管理器

### 为什么需要 Helm？

你的 Go 服务需要：Deployment + Service + Ingress + ConfigMap + Secret。不同环境（dev/staging/prod）配置不同（副本数、域名、资源限制）。

手动维护方式：复制 N 份 YAML，改差异部分 → 容易遗漏、难维护

Helm 方式：一套模板 + 不同的 values.yaml → 一条命令部署

### Helm 核心概念

| 概念 | 类比 | 说明 |
|------|------|------|
| Chart | 安装包 | 一组模板化的 K8s YAML |
| Release | 安装实例 | Chart 部署后的一次实例 |
| Values | 配置文件 | 控制模板渲染的变量 |
| Repository | 应用商店 | 存放 Chart 的仓库 |

### Chart 目录结构

```
go-app-chart/
├── Chart.yaml            # Chart 元信息（名称、版本）
├── values.yaml           # 默认配置
├── templates/
│   ├── _helpers.tpl      # 模板辅助函数
│   ├── deployment.yaml   # Deployment 模板
│   ├── service.yaml      # Service 模板
│   └── ingress.yaml      # Ingress 模板
└── .helmignore           # 忽略文件
```

## 三、实战：GitHub Actions CI/CD

### 创建工作流文件

```yaml
# .github/workflows/ci.yml
name: CI/CD
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Setup Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.22'

    - name: Lint
      uses: golangci/golangci-lint-action@v4

    - name: Test
      run: go test -v ./...

  build-and-push:
    needs: test
    runs-on: ubuntu-latest
    if: github.event_name == 'push'
    steps:
    - uses: actions/checkout@v4

    - name: Build Docker Image
      run: docker build -t ghcr.io/${{ github.repository }}:${{ github.sha }} .

    - name: Login to GHCR
      run: echo ${{ secrets.GITHUB_TOKEN }} | docker login ghcr.io -u ${{ github.actor }} --password-stdin

    - name: Push Image
      run: docker push ghcr.io/${{ github.repository }}:${{ github.sha }}
```

### 关键字段解读

| 字段 | 含义 |
|------|------|
| `on.push.branches` | 哪些分支 push 触发 |
| `jobs.test` | 测试 job |
| `jobs.build-and-push` | 构建 job |
| `needs: test` | 依赖关系：test 通过才执行 build |
| `if: github.event_name == 'push'` | PR 只跑测试不构建 |
| `${{ github.sha }}` | 用 commit hash 作镜像 tag（每次唯一） |
| `${{ secrets.GITHUB_TOKEN }}` | GitHub 自动提供的认证 token |

## 四、实战：创建 Helm Chart

### 初始化

我们手写而不是用 `helm create`（生成的模板太复杂）：

### Chart.yaml

```yaml
apiVersion: v2
name: go-app-chart
description: A Helm chart for the Go learning app
type: application
version: 0.1.0
appVersion: "1.0.0"
```

### values.yaml

```yaml
replicaCount: 3

image:
  repository: localhost/k8s-day1
  tag: latest
  pullPolicy: Never

service:
  type: ClusterIP
  port: 80
  targetPort: 8080

ingress:
  enabled: false
  host: myapp.local

resources:
  requests:
    cpu: 50m
    memory: 64Mi
  limits:
    cpu: 200m
    memory: 128Mi

env:
  APP_ENV: production
  LOG_LEVEL: info
```

### templates/_helpers.tpl

```
{{- define "go-app.name" -}}
{{- .Chart.Name }}
{{- end }}

{{- define "go-app.labels" -}}
app: {{ include "go-app.name" . }}
chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end }}

{{- define "go-app.selectorLabels" -}}
app: {{ include "go-app.name" . }}
{{- end }}
```

### templates/deployment.yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "go-app.name" . }}
  labels:
    {{- include "go-app.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "go-app.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "go-app.selectorLabels" . | nindent 8 }}
    spec:
      containers:
      - name: {{ .Chart.Name }}
        image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
        imagePullPolicy: {{ .Values.image.pullPolicy }}
        ports:
        - containerPort: {{ .Values.service.targetPort }}
        resources:
          {{- toYaml .Values.resources | nindent 10 }}
        env:
        {{- range $key, $value := .Values.env }}
        - name: {{ $key }}
          value: {{ $value | quote }}
        {{- end }}
```

### templates/service.yaml

```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "go-app.name" . }}
  labels:
    {{- include "go-app.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
  - port: {{ .Values.service.port }}
    targetPort: {{ .Values.service.targetPort }}
  selector:
    {{- include "go-app.selectorLabels" . | nindent 4 }}
```

### templates/ingress.yaml

```yaml
{{- if .Values.ingress.enabled }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include "go-app.name" . }}
spec:
  rules:
  - host: {{ .Values.ingress.host }}
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: {{ include "go-app.name" . }}
            port:
              number: {{ .Values.service.port }}
{{- end }}
```

## 五、Helm 操作

```bash
# 渲染模板（不部署，只看生成的 YAML）
helm template go-app ./go-app-chart

# 安装（创建 Release）
helm install go-app ./go-app-chart

# 查看 Release 列表
helm list

# 修改配置并升级
helm upgrade go-app ./go-app-chart --set replicaCount=5

# 查看历史
helm history go-app

# 回滚到上一版本
helm rollback go-app 1

# 卸载
helm uninstall go-app
```

### Helm 模板语法速查

| 语法 | 含义 | 示例 |
|------|------|------|
| `{{ .Values.xxx }}` | 读取 values.yaml 中的值 | `{{ .Values.replicaCount }}` |
| `{{ include "name" . }}` | 调用 _helpers.tpl 中的模板 | `{{ include "go-app.name" . }}` |
| `{{- ... -}}` | 去除前后空白 | 格式控制 |
| `{{ toYaml ... \| nindent N }}` | 转 YAML 并缩进 N 格 | 嵌套对象 |
| `{{- range }}...{{- end }}` | 循环 | 遍历 env map |
| `{{- if }}...{{- end }}` | 条件 | ingress.enabled |

## 六、常见问题和踩坑记录

**Q: `helm template` 报错 "parse error"？**

Helm 模板语法很严格。常见错误：
- `{{` 和变量之间少了空格
- `nindent` 的缩进数不对导致 YAML 语法错误
- 忘了 `{{- end }}` 闭合

**Q: GitHub Actions 怎么访问 K8s 集群？**

需要配置 kubeconfig 作为 Secret，或用云厂商的 CLI 认证。本地 kind 集群不需要这步。

**Q: 镜像 tag 用 latest 还是 commit hash？**

生产用 commit hash（`${{ github.sha }}`）—— 不可变、可追溯。`latest` 只适合本地开发。

**Q: `helm upgrade` 没效果？**

检查 values 是否真的变了。`helm get values go-app` 看当前配置。

## 七、概念总结

```
CI/CD Pipeline
├── CI: 代码质量保证
│   ├── Lint（风格检查）
│   └── Test（单元/集成测试）
└── CD: 自动化交付
    ├── Build（编译 + 打镜像）
    ├── Push（上传到 Registry）
    └── Deploy（更新 K8s 资源）

Helm
├── Chart（模板包）
│   ├── templates/（YAML 模板）
│   └── values.yaml（默认配置）
├── Release（部署实例）
└── Repository（Chart 仓库）
```

| 概念 | 类比 |
|------|------|
| CI Pipeline | 代码的体检报告 |
| CD Pipeline | 自动快递到用户手里 |
| Helm Chart | 菜谱（可重复使用） |
| values.yaml | 口味偏好（辣/不辣） |
| Release | 做好的一道菜 |
| `helm rollback` | 退菜换上一道 |

## ✅ 今日产出验收

```bash
# 1. Helm 模板渲染
helm template go-app ./go-app-chart

# 2. 部署
helm install go-app ./go-app-chart
kubectl get deploy,svc,pods

# 3. 升级
helm upgrade go-app ./go-app-chart --set replicaCount=5
kubectl get pods  # 5 个

# 4. 回滚
helm rollback go-app 1
kubectl get pods  # 回到 3 个
```

- [ ] `helm template` 渲染出正确的 YAML
- [ ] `helm install` 部署成功
- [ ] `helm upgrade --set` 能动态修改配置
- [ ] `helm rollback` 能回滚
- [ ] 理解 CI/CD 各环节作用

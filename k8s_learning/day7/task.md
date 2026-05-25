Day 7：CI/CD 流水线
上午：概念（1.5h）
CI/CD 流程
代码提交 → Lint/Test → Build 镜像 → Push Registry → Deploy to K8s

Git 工作流
- Trunk-based（携程常用）：main 分支为主，短生命周期 feature branch
- 每次 merge 触发 CI，通过则自动部署

Helm
- K8s 的包管理器
- Chart = 模板化的 YAML 集合
- Values.yaml 控制不同环境的差异

下午：实战（4h）
1. GitHub Actions CI/CD
创建 .github/workflows/ci.yml：
name: CI/CD
on:
  push:
    branches: [main]

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Setup Go
      uses: actions/setup-go@v5
      with:
        go-version: 1.22

    - name: Lint
      uses: golangci/golangci-lint-action@v4

    - name: Test
      run: go test ./...

    - name: Build & Push Docker Image
      run: |
        docker build -t ghcr.io/${{ github.repository }}:${{ github.sha }} .
        echo ${{ secrets.GITHUB_TOKEN }} | docker login ghcr.io -u ${{ github.actor }} --password-stdin
        docker push ghcr.io/${{ github.repository }}:${{ github.sha }}

    - name: Deploy to K8s
      run: |
        kubectl set image deploy/go-app go-app=ghcr.io/${{ github.repository }}:${{ github.sha }}

2. 创建 Helm Chart
helm create go-app-chart

关键文件：
- values.yaml：配置参数
- templates/deployment.yaml：Deployment 模板
- templates/service.yaml：Service 模板
- templates/ingress.yaml：Ingress 模板

自定义 values.yaml：
replicaCount: 3
image:
  repository: myapp
  tag: latest
service:
  type: ClusterIP
  port: 80
ingress:
  enabled: true
  host: myapp.local
resources:
  requests:
    cpu: 50m
    memory: 64Mi
  limits:
    cpu: 200m
    memory: 128Mi

3. Helm 部署
# 本地渲染查看
helm template go-app ./go-app-chart

# 安装
helm install go-app ./go-app-chart -n default

# 升级
helm upgrade go-app ./go-app-chart --set image.tag=v2

# 回滚
helm rollback go-app 1

今日产出验收
- [ ] GitHub Actions 流水线能自动构建推送镜像
- [ ] Helm Chart 能模板化部署应用
- [ ] 能用 helm upgrade/rollback 管理版本
- [ ] 理解 CI/CD 各环节的作用

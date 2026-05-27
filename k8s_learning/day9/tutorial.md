# Day 9：Prometheus + Grafana 监控

> 服务跑起来了不等于没问题。你需要知道 QPS 多少、延迟多少、错误率多少。今天学怎么给 Go 服务加监控。

## 一、Prometheus 工作原理

### Pull 模型

Prometheus **主动拉取**指标（不像 StatsD 推送）：

```
你的 Go 服务暴露 /metrics 端点
    ↓
Prometheus 每 15s 来拉一次
    ↓
存入时序数据库
    ↓
Grafana 从 Prometheus 查询展示
```

### 四种指标类型

| 类型 | 特点 | 示例 | 用途 |
|------|------|------|------|
| Counter | 只增不减 | `http_requests_total` | 请求计数、错误计数 |
| Gauge | 可增可减 | `memory_usage_bytes` | 当前连接数、队列长度 |
| Histogram | 分桶统计 | `http_request_duration_seconds` | 延迟分布（P50/P95/P99） |
| Summary | 客户端分位数 | — | 类似 Histogram，较少使用 |

### 指标格式

```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",path="/",status="200"} 42
http_requests_total{method="GET",path="/health",status="200"} 100
```

## 二、给 Go 服务添加指标

### 核心代码

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// 定义指标
var httpRequestsTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "http_requests_total",
        Help: "Total number of HTTP requests",
    },
    []string{"method", "path", "status"},  // 标签维度
)

var httpRequestDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "http_request_duration_seconds",
        Buckets: prometheus.DefBuckets,  // 默认桶：.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
    },
    []string{"method", "path"},
)

func init() {
    prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
}

// 包装 handler，自动记录指标
func instrumentHandler(path string, handler http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        handler(w, r)
        duration := time.Since(start).Seconds()
        httpRequestsTotal.WithLabelValues(r.Method, path, "200").Inc()
        httpRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
    }
}

// 暴露 /metrics 端点
http.Handle("/metrics", promhttp.Handler())
```

### 验证

```bash
# 本地运行
go run main.go

# 访问几次
curl http://localhost:8080/
curl http://localhost:8080/health

# 看指标
curl http://localhost:8080/metrics | grep http_requests
```

## 三、部署到 K8s

### 构建镜像并部署

```bash
# 构建
podman build -t metrics-app:latest .
podman save metrics-app:latest -o metrics-app.tar
kind load image-archive metrics-app.tar --name k8s-lab

# 部署
kubectl apply -f deployment.yaml
kubectl get pods -l app=metrics-app

# 验证 /metrics
kubectl port-forward svc/metrics-app 8080:80
curl http://localhost:8080/metrics
```

## 四、Prometheus + Grafana 部署

### 安装 kube-prometheus-stack

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install monitoring prometheus-community/kube-prometheus-stack \
  -n monitoring --create-namespace \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false

# 等待就绪
kubectl get pods -n monitoring -w
```

### 访问 Grafana

```bash
kubectl port-forward svc/monitoring-grafana 3000:80 -n monitoring
# 浏览器打开 http://localhost:3000
# 账号: admin  密码: prom-operator
```

### ServiceMonitor —— 自动发现

```yaml
# servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: go-app-monitor
  labels:
    release: monitoring     # 必须匹配 Prometheus 的 serviceMonitorSelector
spec:
  selector:
    matchLabels:
      app: metrics-app      # 匹配 Service 的 label
  endpoints:
  - port: http              # Service 中定义的端口名
    interval: 15s
    path: /metrics
```

```bash
kubectl apply -f servicemonitor.yaml
# Prometheus 会在下个 scrape 周期自动发现并采集
```

## 五、PromQL 实战

### 基础查询

```promql
# 总请求数
http_requests_total

# 按 path 分组的总请求数
sum by(path) (http_requests_total)

# 最近 5 分钟的 QPS（每秒请求数）
rate(http_requests_total[5m])

# 按 path 分组的 QPS
sum by(path) (rate(http_requests_total[5m]))
```

### 延迟分析

```promql
# P50 延迟
histogram_quantile(0.5, rate(http_request_duration_seconds_bucket[5m]))

# P95 延迟
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# P99 延迟
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))

# 平均延迟
rate(http_request_duration_seconds_sum[5m]) / rate(http_request_duration_seconds_count[5m])
```

### 错误率

```promql
# 5xx 错误率
sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))

# 非 200 的比例
1 - sum(rate(http_requests_total{status="200"}[5m])) / sum(rate(http_requests_total[5m]))
```

### 系统指标

```promql
# 节点 CPU 使用率
1 - avg(rate(node_cpu_seconds_total{mode="idle"}[5m])) by (instance)

# Pod 内存使用
container_memory_working_set_bytes{container!=""} / 1024 / 1024
```

## 六、单元测试策略

使用 `prometheus/testutil` 验证指标：

```go
import "github.com/prometheus/client_golang/prometheus/testutil"

func TestInstrumentHandler(t *testing.T) {
    httpRequestsTotal.Reset()
    handler := instrumentHandler("/test", myHandler)
    handler(httptest.NewRecorder(), httptest.NewRequest("GET", "/test", nil))

    count := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "/test", "200"))
    if count != 1 {
        t.Errorf("expected 1, got %f", count)
    }
}
```

## 七、常见问题和踩坑记录

**Q: Prometheus 没采集到我的指标？**

1. ServiceMonitor 的 `selector` 是否匹配 Service 的 label？
2. ServiceMonitor 的 `release: monitoring` label 是否正确？
3. Service 的端口名（`name: http`）是否和 ServiceMonitor 的 `port: http` 一致？
4. `kubectl port-forward svc/metrics-app 8080:80` 后 `curl localhost:8080/metrics` 有输出吗？

**Q: Histogram 的 bucket 怎么选？**

默认 `DefBuckets` 适合大多数 HTTP 服务（5ms~10s）。如果你的服务：
- 很快（<1ms）：用 `[]float64{.0001, .0005, .001, .005, .01}`
- 很慢（>10s）：加大桶 `[]float64{1, 5, 10, 30, 60}`

**Q: rate() 和 increase() 的区别？**

- `rate()` = 每秒速率（适合 QPS）
- `increase()` = 时间段内的增量（适合"最近 5 分钟有多少请求"）

## 八、概念总结

```
你的 Go 服务
├── /metrics 端点（暴露 Counter/Histogram/Gauge）
└── instrumentHandler（自动记录每个请求）

Prometheus
├── ServiceMonitor → 自动发现 targets
├── Scrape → 定期拉取 /metrics
└── TSDB → 存储时序数据

Grafana
├── 数据源 → Prometheus
├── Dashboard → 面板
└── PromQL → 查询语言
```

| 概念 | 类比 |
|------|------|
| /metrics | 体检报告（实时更新） |
| Prometheus | 定期上门体检的医生 |
| PromQL | 医生的诊断工具 |
| Grafana | 健康监测大屏 |
| ServiceMonitor | 预约体检的挂号单 |
| Counter | 计步器（只增不减） |
| Histogram | 身高体重分布图 |

## ✅ 今日产出验收

```bash
# 1. Go 服务编译和测试
go build -o metrics-app .
go test -v ./...   # 4 个 PASS

# 2. /metrics 端点正常
./metrics-app &
curl localhost:8080/metrics | grep http_requests_total

# 3. 部署到 K8s（可选，需要 Prometheus stack）
kubectl apply -f deployment.yaml
kubectl apply -f servicemonitor.yaml
```

- [ ] `go build` 编译通过
- [ ] `go test` 4 个测试全部 PASS
- [ ] `/metrics` 端点返回 Prometheus 格式指标
- [ ] 理解 Counter/Histogram 的使用场景
- [ ] 能写基础 PromQL 查询（rate、histogram_quantile）

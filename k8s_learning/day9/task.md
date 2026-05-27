# Day 9：Prometheus + Grafana 监控

## 上午：监控架构（1.5h）

### Prometheus 核心

- **Pull 模型**：主动从 targets 拉取 /metrics
- **时序数据库**：存储 metric_name{labels} value timestamp
- **PromQL**：查询语言
- **Alertmanager**：告警路由

### 指标类型

| 类型 | 用途 | 示例 |
|------|------|------|
| Counter | 只增不减的计数 | http_requests_total |
| Gauge | 可增可减的瞬时值 | memory_usage_bytes |
| Histogram | 分布统计（分桶） | http_request_duration_seconds |
| Summary | 类似 Histogram，客户端计算分位数 | — |

## 下午：实战（4h）

### 1. 部署 kube-prometheus-stack

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install monitoring prometheus-community/kube-prometheus-stack -n monitoring --create-namespace

# 访问 Grafana
kubectl port-forward svc/monitoring-grafana 3000:80 -n monitoring
# 默认账号 admin / prom-operator
```

### 2. 给 Go 服务添加 /metrics

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    httpRequests = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total HTTP requests",
        },
        []string{"method", "path", "status"},
    )
    httpDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path"},
    )
)

func init() {
    prometheus.MustRegister(httpRequests, httpDuration)
}
```

### 3. 创建 ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: go-app-monitor
  labels:
    release: monitoring
spec:
  selector:
    matchLabels:
      app: go-app
  endpoints:
  - port: http
    interval: 15s
```

### 4. PromQL 实战查询

```
# QPS
rate(http_requests_total[5m])

# P99 延迟
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))

# 错误率
sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))
```

## ✅ 今日产出验收

- [ ] Prometheus 能采集到 Go 服务的自定义指标
- [ ] Grafana 看板展示 QPS、延迟、错误率
- [ ] 能写基础 PromQL 查询
- [ ] 理解 ServiceMonitor 自动发现机制

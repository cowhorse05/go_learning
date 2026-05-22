# 上午
1. 写一个简单的 Go HTTP 服务（返回 hostname + 当前时间）
2. 手写 Dockerfile（multi-stage）：
  - Stage 1: golang:1.22-alpine，编译
  - Stage 2: alpine:3.19，只拷贝二进制
3. 构建并运行，验证 http://localhost:8080

---
# 下午
docker-compose 多容器编排
编排以下服务：
- app: 你的 Go 服务（依赖 redis）
- redis: redis:7-alpine
- nginx: nginx:alpine，反代 app 的 8080 端口
文件结构：
project/
├── app/
│   ├── main.go
│   └── Dockerfile
├── nginx/
│   └── nginx.conf
└── docker-compose.yml

任务 3：排障练习
- 故意写错 Dockerfile，观察报错信息
- docker logs 查看容器日志
- docker exec 进入容器检查文件系统
- docker network inspect 查看容器间通信
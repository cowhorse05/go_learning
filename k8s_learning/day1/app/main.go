package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client

func main() {
	portFlag := flag.String("port", "", "服务监听端口，默认 8080")
	flag.Parse()

	port := *portFlag
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/info", infoHandler)

	fmt.Printf("Server starting on port %s...\n", port)
	fmt.Printf("Redis: %s\n", redisAddr)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
		os.Exit(1)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	count, _ := rdb.Incr(ctx, "visit_count").Result()
	fmt.Fprintf(w, "Welcome to K8s Learning HTTP Server!\n")
	fmt.Fprintf(w, "访问次数: %d\n", count)
	fmt.Fprintf(w, "当前时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	err := rdb.Ping(ctx).Err()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status": "unhealthy", "error": "%s"}`, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "healthy", "service": "k8s-learning", "redis": "connected"}`)
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	fmt.Fprintf(w, "=== Service Info ===\n")
	fmt.Fprintf(w, "Hostname: %s\n", hostname)
	fmt.Fprintf(w, "Time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Redis: %s\n", os.Getenv("REDIS_ADDR"))
}

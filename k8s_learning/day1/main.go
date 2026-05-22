/*
a simple htpp service for learning k8s
*/
package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/info", infoHandler)

	fmt.Printf("Server staring on port%s...\n", port)
	fmt.Printf("访问http://localhost:%s\n", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
		os.Exit(1)
	}
}

// 首页处理函数
func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to K8s Learning HTTP Server!\n")
	fmt.Fprintf(w, "请求路径: %s\n", r.URL.Path)
	fmt.Fprintf(w, "请求方法: %s\n", r.Method)
}

// 健康检查处理函数（K8s 常用）
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "healthy", "service": "k8s-learning"}`)
}

// 信息展示处理函数
func infoHandler(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	fmt.Fprintf(w, "=== Service Info ===\n")
	fmt.Fprintf(w, "Hostname: %s\n", hostname)
	fmt.Fprintf(w, "Pod IP: %s\n", r.Host)
	fmt.Fprintf(w, "User-Agent: %s\n", r.Header.Get("User-Agent"))
}

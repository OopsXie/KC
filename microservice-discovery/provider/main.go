package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// 服务信息结构体
type ServiceInfo struct {
	ServiceName string `json:"service_name"`
	Host        string `json:"host"`
	Port        string `json:"port"`
	Version     string `json:"version"`
}

// 健康检查响应
type HealthResponse struct {
	Status string `json:"status"`
}

func main() {
	// 获取环境变量中的端口，默认8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 获取主机名(在K8s中会是Pod名称)
	host, err := os.Hostname()
	if err != nil {
		log.Fatalf("无法获取主机名: %v", err)
	}

	// 注册处理函数
	http.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		info := ServiceInfo{
			ServiceName: "data-provider",
			Host:        host,
			Port:        port,
			Version:     "v1.0.0",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	})

	// 健康检查接口(供K8s liveness probe使用)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		response := HealthResponse{Status: "healthy"}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// 数据接口
	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]string{
			"message": "Hello from provider service",
			"source":  host,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	})

	log.Printf("服务提供者启动在端口 %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

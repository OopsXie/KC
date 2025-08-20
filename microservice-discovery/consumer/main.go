package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

// 从提供者获取的数据结构
type ProviderData struct {
	Message string `json:"message"`
	Source  string `json:"source"`
}

// 消费者响应结构
type ConsumerResponse struct {
	Status  string       `json:"status"`
	Data    ProviderData `json:"data"`
	Time    string       `json:"time"`
	Version string       `json:"version"`
}

func main() {
	// 获取环境变量中的端口，默认8081
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	// 服务提供者的服务名(在K8s中是Service名称)
	providerService := os.Getenv("PROVIDER_SERVICE")
	if providerService == "" {
		providerService = "provider-service" // 默认服务名
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// 调用提供者服务的接口
	http.HandleFunc("/fetch", func(w http.ResponseWriter, r *http.Request) {
		// 通过服务名调用(由K8s DNS解析)
		url := fmt.Sprintf("http://%s/data", providerService)

		resp, err := client.Get(url)
		if err != nil {
			http.Error(w, fmt.Sprintf("调用服务提供者失败: %v", err), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("读取响应失败: %v", err), http.StatusInternalServerError)
			return
		}

		var providerData ProviderData
		if err := json.Unmarshal(body, &providerData); err != nil {
			http.Error(w, fmt.Sprintf("解析响应失败: %v", err), http.StatusInternalServerError)
			return
		}

		// 构建消费者响应
		response := ConsumerResponse{
			Status:  "success",
			Data:    providerData,
			Time:    time.Now().Format(time.RFC3339),
			Version: "v1.0.0",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// 健康检查接口
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	log.Printf("服务消费者启动在端口 %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

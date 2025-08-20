package logging_service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"msa/client/config"
	"net/http"
	"time"
)

// LogRequest 日志请求结构体
type LogRequest struct {
	ServiceName string `json:"serviceName"`
	ServiceID   string `json:"serviceId"`
	DateTime    string `json:"datetime"`
	Level       string `json:"level"`
	Message     string `json:"message"`
}

// StartLoggingService 启动日志收集服务，根据配置的时间间隔发送日志
func StartLoggingService(cfg *config.Config) {
	interval := time.Duration(cfg.Client.LoggingInterval) * time.Second

	log.Printf("[logging] 启动日志收集服务，目标地址: %s/api/loggingservice, 发送间隔: %d秒",
		cfg.Logging.BaseURL, cfg.Client.LoggingInterval)

	// 启动时立即发送一次日志
	sendLog(cfg)

	// 然后按照配置的间隔定时发送
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		sendLog(cfg)
	}
}

// sendLog 发送日志到日志收集服务
func sendLog(cfg *config.Config) {
	// 创建日志请求
	logReq := LogRequest{
		ServiceName: cfg.Client.ServiceName,
		ServiceID:   cfg.Client.ServiceID,
		DateTime:    getCurrentGMTTimeWithMillis(),
		Level:       "info",
		Message:     "Client status is OK.",
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(logReq)
	if err != nil {
		log.Printf("[logging] JSON序列化失败: %v", err)
		return
	}

	// 发送HTTP请求
	url := fmt.Sprintf("%s/api/loggingservice", cfg.Logging.BaseURL)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[logging] 发送日志失败: %v", err)
		return
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		log.Printf("[logging] 日志服务返回错误状态: %d", resp.StatusCode)
	} else {
		log.Printf("[logging] 日志发送成功: %s - %s", logReq.DateTime, logReq.Message)
	}
}

// getCurrentGMTTimeWithMillis 获取当前GMT时间，带毫秒
func getCurrentGMTTimeWithMillis() string {
	now := time.Now().UTC()
	return now.Format("2006-01-02 15:04:05.000")
}

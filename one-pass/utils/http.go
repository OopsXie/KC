package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"one-pass/config"

	"github.com/google/uuid"
)

var (
	// 全局HTTP客户端，复用连接池
	httpClient *http.Client
	clientOnce sync.Once
)

type PayRequest struct {
	TransactionId string  `json:"transactionId"`
	UID           int64   `json:"uid"`
	Amount        float64 `json:"amount"`
}

type PayResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// getHTTPClient 获取全局HTTP客户端，优化连接管理
func getHTTPClient() *http.Client {
	clientOnce.Do(func() {
		// 自定义Dialer，优化TCP连接
		dialer := &net.Dialer{
			Timeout:   10 * time.Second, // 连接超时
			KeepAlive: 30 * time.Second, // TCP Keep-Alive
		}

		// 优化的Transport配置
		transport := &http.Transport{
			Dial:                dialer.Dial,
			MaxIdleConns:        300,              // 大幅增加总连接池
			MaxIdleConnsPerHost: 150,              // 大幅增加单主机连接数
			IdleConnTimeout:     45 * time.Second, // 延长空闲超时
			TLSHandshakeTimeout: 10 * time.Second,
			DisableKeepAlives:   false, // 启用Keep-Alive
			DisableCompression:  false,
			// 关键：强制复用连接
			MaxConnsPerHost: 0, // 不限制单主机连接数
		}

		httpClient = &http.Client{
			Transport: transport,
			Timeout:   15 * time.Second, // 适中的请求超时
		}
	})
	return httpClient
}

func CallPay(cfg *config.Config, transactionId string, uid int64, amount float64) (bool, error) {
	reqBody := PayRequest{
		TransactionId: transactionId,
		UID:           uid,
		Amount:        amount,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", cfg.API.PayURL, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-KSY-KINGSTAR-ID", cfg.Kingstar.ID)
	req.Header.Set("X-KSY-TOKEN", cfg.Kingstar.Token)
	req.Header.Set("X-KSY-REQUEST-ID", uuid.NewString())

	// 重要：强制启用连接复用
	req.Header.Set("Connection", "keep-alive")

	// 使用全局客户端，复用连接池
	client := getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[ERROR] 支付请求失败: %v\n", err)
		return false, err
	}
	defer resp.Body.Close()

	var result PayResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("[ERROR] 解析响应失败: %v\n", err)
		return false, err
	}

	//fmt.Printf("[DEBUG] 支付响应: Code=%d, Msg=%s\n", result.Code, result.Msg)
	return result.Code == 200, nil
}

// BatchPayBeginRequest 批量支付开始请求
type BatchPayBeginRequest struct {
	BatchPayId string  `json:"batchPayId"`
	UIDs       []int64 `json:"uids"`
}

// BatchPayFinishRequest 批量支付完成请求
type BatchPayFinishRequest struct {
	BatchPayId string  `json:"batchPayId"`
	UIDs       []int64 `json:"uids"`
}

// BatchResponse 批量支付响应
type BatchResponse struct {
	Code      int    `json:"code"`
	Node      string `json:"node"`
	Msg       string `json:"msg"`
	RequestId string `json:"requestId"`
	Data      string `json:"data"`
	Ok        bool   `json:"ok"`
}

// CallBatchPayBegin 调用批量支付开始接口
func CallBatchPayBegin(cfg *config.Config, batchPayId string, uids []int64) (bool, error) {
	reqBody := BatchPayBeginRequest{
		BatchPayId: batchPayId,
		UIDs:       uids,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	//fmt.Printf("[DEBUG] 批量支付开始请求体: %s\n", string(bodyBytes))

	req, _ := http.NewRequest("POST", cfg.API.BatchPayBeginURL, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-KSY-KINGSTAR-ID", cfg.Kingstar.ID)
	req.Header.Set("X-KSY-TOKEN", cfg.Kingstar.Token)
	req.Header.Set("X-KSY-REQUEST-ID", uuid.NewString())
	req.Header.Set("Connection", "keep-alive")

	// 使用全局客户端
	client := getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[ERROR] 批量支付开始请求失败: %v\n", err)
		return false, err
	}
	defer resp.Body.Close()

	// 读取原始响应内容
	responseBody := make([]byte, 1024)
	n, _ := resp.Body.Read(responseBody)
	responseStr := string(responseBody[:n])
	fmt.Printf("[DEBUG] 批量支付开始原始响应: %s\n", responseStr)

	var result BatchResponse
	if err := json.Unmarshal(responseBody[:n], &result); err != nil {
		fmt.Printf("[ERROR] 解析批量支付开始响应失败: %v\n", err)
		return false, err
	}

	fmt.Printf("[DEBUG] 批量支付开始响应: Code=%d, Msg=%s\n", result.Code, result.Msg)

	// 检查响应是否成功，如果失败则返回具体错误信息
	if result.Code != 200 {
		// 优先使用 data 字段的错误信息，如果没有则使用 msg
		errorMsg := result.Data
		if errorMsg == "" {
			errorMsg = result.Msg
		}
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("批量支付开始失败，状态码: %d", result.Code)
		}
		return false, fmt.Errorf("%s", errorMsg)
	}

	return true, nil
}

// CallBatchPayFinish 调用批量支付完成接口
func CallBatchPayFinish(cfg *config.Config, batchPayId string) (bool, error) {
	// 使用 URL 参数而不是请求体
	url := fmt.Sprintf("%s?batchPayId=%s", cfg.API.BatchPayFinishURL, batchPayId)

	req, _ := http.NewRequest("POST", url, nil) // 没有请求体
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-KSY-KINGSTAR-ID", cfg.Kingstar.ID)
	req.Header.Set("X-KSY-TOKEN", cfg.Kingstar.Token)
	req.Header.Set("X-KSY-REQUEST-ID", uuid.NewString())
	req.Header.Set("Connection", "keep-alive")

	fmt.Printf("[DEBUG] 调用批量支付完成接口: URL=%s\n", url)
	fmt.Printf("[DEBUG] 请求头: KingstarID=%s, Token=%s\n", cfg.Kingstar.ID, cfg.Kingstar.Token)

	// 使用全局客户端
	client := getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[ERROR] 批量支付完成请求失败: %v\n", err)
		return false, err
	}
	defer resp.Body.Close()

	// 读取原始响应内容
	responseBody := make([]byte, 1024)
	n, _ := resp.Body.Read(responseBody)
	responseStr := string(responseBody[:n])
	fmt.Printf("[DEBUG] 批量支付完成原始响应: %s\n", responseStr)

	var result BatchResponse
	if err := json.Unmarshal(responseBody[:n], &result); err != nil {
		fmt.Printf("[ERROR] 解析批量支付完成响应失败: %v\n", err)
		return false, err
	}

	fmt.Printf("[DEBUG] 批量支付完成响应: Code=%d, Msg=%s\n", result.Code, result.Msg)

	// 检查响应是否成功，如果失败则返回具体错误信息
	if result.Code != 200 {
		// 优先使用 data 字段的错误信息，如果没有则使用 msg
		errorMsg := result.Data
		if errorMsg == "" {
			errorMsg = result.Msg
		}
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("批量支付完成失败，状态码: %d", result.Code)
		}
		return false, fmt.Errorf("%s", errorMsg)
	}

	return true, nil
}

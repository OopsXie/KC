package main

import (
	"fmt"
	"log"
	"msa/client/config"
	"msa/client/heartbeat"
	"msa/client/logging_service"
	"msa/client/register"
	"msa/client/router"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	configPath := config.GetConfigPathFromArgs()
	config.LoadConfig(configPath)

	// 注册到注册中心
	if err := register.Register(&config.Cfg); err != nil {
		log.Fatalf("[error] 服务注册失败: %v", err)
	}

	// 启动心跳服务
	go heartbeat.StartHeartbeat(&config.Cfg)

	// 启动日志收集服务
	go logging_service.StartLoggingService(&config.Cfg)

	// 启动HTTP服务器
	r := router.SetupRouter()
	addr := fmt.Sprintf(":%d", config.Cfg.Client.Port)
	go func() {
		log.Printf("[success] time-service 启动成功，监听地址: %s", addr)
		if err := r.Run(addr); err != nil {
			log.Fatalf("[error] 启动服务失败: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // 阻塞主线程，直到收到退出信号

	log.Println("收到退出信号，准备注销服务")
	register.Unregister(&config.Cfg)
}

package main

import (
	"fmt"
	"log"
	"msa/time_service/config"
	"msa/time_service/heartbeat"
	"msa/time_service/register"
	"msa/time_service/router"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 获取配置文件路径（带默认值）
	configPath := config.GetConfigPathFromArgs()
	config.LoadConfig(configPath)

	// 获取 IP

	// 注册服务
	if err := register.Register(&config.Cfg); err != nil {
		log.Fatalf("[error] 服务注册失败: %v", err)
	}

	go heartbeat.StartHeartbeat(&config.Cfg)

	r := router.SetupRouter()
	addr := fmt.Sprintf(":%d", config.Cfg.Service.Port)
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

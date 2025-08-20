package main

import (
	"fmt"
	"log"
	"msa/registry/cluster"
	"msa/registry/config"
	"msa/registry/router"
	"msa/registry/storage"
	"time"
)

func main() {
	configPath := config.GetConfigPathFromArgs()
	config.LoadConfig(configPath)

	// 启动集群管理器
	cluster.Manager.Start()
	defer cluster.Manager.Stop()

	// 设置集群管理器到storage层
	storage.SetClusterManager(cluster.Manager)

	// 监听主节点状态变化并管理清理任务
	go func() {
		var cleanupRunning bool

		// 初始检查是否为主节点
		time.Sleep(2 * time.Second) // 等待2秒让集群状态稳定
		if cluster.Manager.IsMaster() {
			log.Printf("[registry] 启动过期实例清理任务")
			// 立即检查一次过期实例，然后启动定时任务
			go func() {
				currentCount := storage.GetTotalServiceCount()
				log.Printf("[registry] 启动时本地服务数量: %d", currentCount)

				// 打印调试信息
				storage.DebugPrintAllInstances()

				// 立即进行一次过期检测
				storage.CheckAndCleanExpiredInstances()

				// 然后启动定时清理任务
				cleanupInterval := time.Duration(config.GetCleanupIntervalSeconds()) * time.Second
				storage.StartExpiredInstanceCleanup(cleanupInterval)
			}()
			cleanupRunning = true
		} // 监听主节点状态变化
		for isMaster := range cluster.Manager.GetMasterChangeNotification() {
			if isMaster && !cleanupRunning {
				log.Printf("[registry] 成为主节点，启动过期实例清理任务")
				// 立即检查过期实例，然后启动定时任务
				go func() {
					currentCount := storage.GetTotalServiceCount()
					log.Printf("[registry] 当前本地服务数量: %d", currentCount)

					// 打印调试信息
					storage.DebugPrintAllInstances()

					// 立即进行一次过期检测
					storage.CheckAndCleanExpiredInstances()

					// 启动定时清理任务
					cleanupInterval := time.Duration(config.GetCleanupIntervalSeconds()) * time.Second
					storage.StartExpiredInstanceCleanup(cleanupInterval)
				}()
				cleanupRunning = true
			} else if !isMaster && cleanupRunning {
				log.Printf("[registry] 不再是主节点，清理任务将由新主节点处理")
				cleanupRunning = false
			}
		}
	}()

	r := router.SetupRegistryRouter()
	addr := fmt.Sprintf(":%d", config.Cfg.Registry.Port)

	role := "slave"
	if cluster.Manager.IsMaster() {
		role = "master"
	}

	log.Printf("Registry [%s] listening on %s... (Role: %s)\n",
		config.Cfg.Registry.InstanceID, addr, role)
	log.Fatal(r.Run(addr))
}

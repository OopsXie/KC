package main

import (
	"net/http"
	"one-pass/config"
	"one-pass/handler"
	"one-pass/middleware"
	"one-pass/service"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 加载配置
	cfg := config.Load()

	// 2. 初始化数据库
	db := config.InitDB(cfg)

	// 3. 初始化 Redis
	rdb := config.InitRedis(cfg)

	// 4. 初始化业务服务对象
	svc := service.NewService(cfg, db, rdb)

	// 5. 设置 Gin 路由
	r := gin.Default()

	// 添加健康检查接口
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "one-pass",
			"timestamp": time.Now().Unix(),
		})
	})

	// 添加中间件
	// 全局并发限制（最大同时处理500个请求）
	r.Use(middleware.ConcurrencyLimit(500))

	// 速率限制（每秒最多100个请求）
	rateLimiter := middleware.NewRateLimiter(100, time.Second)
	r.Use(middleware.RateLimit(rateLimiter))

	// 注册路由
	handler.RegisterRoutes(r, svc)

	// 6. 启动服务
	r.Run(":40008")
}

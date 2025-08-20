package handler

import (
	"net/http"
	"one-pass/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BatchPayInput struct {
	BatchPayId string  `json:"batchPayId"`
	UIDs       []int64 `json:"uids"`
}

func RegisterRoutes(r *gin.Engine, svc *service.Service) {
	// 注册批量支付路由
	RegisterBatchPayRoutes(r, svc)
	// 注册查询路由
	RegisterQueryRoutes(r, svc)
	// 注册交易路由
	RegisterTradeRoutes(r, svc)
}

// RegisterBatchPayRoutes 注册批量支付相关路由
func RegisterBatchPayRoutes(r *gin.Engine, svc *service.Service) {
	r.POST("/onePass/batchPay", func(c *gin.Context) {
		var input BatchPayInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
			return
		}
		if err := svc.HandleBatchPay(input.BatchPayId, input.UIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// c.JSON(http.StatusOK, gin.H{"msg": "batch pay success"})
		c.JSON(http.StatusOK, gin.H{
			"msg":       "ok",
			"code":      200,
			"requestId": uuid.New().String(),
			"data":      nil,
		})

	})

	// 添加Redis缓存初始化接口
	r.POST("/onePass/initRedisCache", func(c *gin.Context) {
		if err := svc.InitializeRedisCache(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":  500,
				"msg":   "初始化Redis缓存失败",
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":      200,
			"msg":       "Redis缓存初始化成功",
			"requestId": uuid.New().String(),
			"data":      nil,
		})
	})

	// 添加用户余额验证
	r.POST("/onePass/validateBalance", func(c *gin.Context) {
		var input struct {
			UID            int64   `json:"uid"`
			ExpectedAmount float64 `json:"expectedAmount"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":  400,
				"msg":   "参数错误",
				"error": err.Error(),
			})
			return
		}

		if err := svc.ValidateAndFixUserBalance(input.UID, input.ExpectedAmount); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":      400,
				"msg":       "余额验证失败",
				"error":     err.Error(),
				"requestId": uuid.New().String(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":      200,
			"msg":       "余额验证通过",
			"requestId": uuid.New().String(),
			"data":      nil,
		})
	})
}

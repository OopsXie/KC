package handler

import (
	"context"
	"net/http"
	"one-pass/model"
	"one-pass/service"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RegisterTradeRoutes 注册交易相关路由
func RegisterTradeRoutes(r *gin.Engine, svc *service.Service) {
	// 用户交易接口
	r.POST("/onePass/userTrade", UserTradeHandler(svc))
}

// UserTradeHandler 用户交易处理器 - 高并发优化版本，使用字符串接收金额
func UserTradeHandler(svc *service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置请求超时
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		// 获取请求ID
		requestID := c.GetHeader("X-KSY-REQUEST-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 使用字符串接收金额
		var req model.UserTradeRequestString
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, model.UserTradeResponse{
				Code:      400,
				Msg:       "参数格式错误",
				RequestID: requestID,
				Data:      nil,
			})
			return
		}

		// 验证金额精度
		if err := req.ValidateAmountPrecision(); err != nil {
			c.JSON(http.StatusBadRequest, model.UserTradeResponse{
				Code:      400,
				Msg:       err.Error(),
				RequestID: requestID,
				Data:      nil,
			})
			return
		}

		// 获取 float64 格式的金额用于后续处理
		amount, err := req.GetAmountFloat64()
		if err != nil {
			c.JSON(http.StatusBadRequest, model.UserTradeResponse{
				Code:      400,
				Msg:       "金额格式错误",
				RequestID: requestID,
				Data:      nil,
			})
			return
		}

		// 在 goroutine 中执行交易，监控超时
		resultChan := make(chan error, 1)
		go func() {
			err := svc.UserTrade(req.SourceUID, req.TargetUID, amount)
			resultChan <- err
		}()

		select {
		case err := <-resultChan:
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.UserTradeResponse{
					Code:      500,
					Msg:       err.Error(),
					RequestID: requestID,
					Data:      nil,
				})
				return
			}

			// 返回成功
			c.JSON(http.StatusOK, model.UserTradeResponse{
				Code:      200,
				Msg:       "ok",
				RequestID: requestID,
				Data:      nil,
			})

		case <-ctx.Done():
			// 请求超时
			c.JSON(http.StatusRequestTimeout, model.UserTradeResponse{
				Code:      408,
				Msg:       "交易处理超时，请稍后重试",
				RequestID: requestID,
				Data:      nil,
			})
		}
	}
}

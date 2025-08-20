package handler

import (
	"net/http"
	"one-pass/model"
	"one-pass/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RegisterQueryRoutes 注册查询相关路由
func RegisterQueryRoutes(r *gin.Engine, svc *service.Service) {
	// 查询用户余额接口
	r.POST("/onePass/queryUserAmount", QueryUserAmountHandler(svc))
}

// QueryUserAmountHandler 查询用户余额
func QueryUserAmountHandler(svc *service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取请求ID
		requestID := c.GetHeader("X-KSY-REQUEST-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		var uids model.QueryUserAmountRequest
		if err := c.ShouldBindJSON(&uids); err != nil {
			c.JSON(http.StatusBadRequest, model.QueryUserAmountResponse{
				Code:      400,
				Msg:       "invalid input",
				RequestID: requestID,
				Data:      []model.UserAmountData{},
			})
			return
		}

		// 调用service查询用户余额
		userAmounts, err := svc.QueryUserAmounts(uids)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.QueryUserAmountResponse{
				Code:      500,
				Msg:       err.Error(),
				RequestID: requestID,
				Data:      []model.UserAmountData{},
			})
			return
		}

		// 返回成功响应
		c.JSON(http.StatusOK, model.QueryUserAmountResponse{
			Code:      200,
			Msg:       "ok",
			RequestID: requestID,
			Data:      userAmounts,
		})
	}
}

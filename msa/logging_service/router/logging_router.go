package router

import (
	"msa/logging_service/handler"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	r.POST("/api/loggingservice", handler.HandleLog)
}

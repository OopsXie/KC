package router

import (
	"msa/time_service/handler"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/api/getDateTime", handler.HandlerGetDateTime)
	return r
}

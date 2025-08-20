package main

import (
	"msa/logging_service/router"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	router.RegisterRoutes(r)
	r.Run(":28400")
}

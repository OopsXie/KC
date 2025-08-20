package handler

import (
	"fmt"
	"log"
	"msa/logging_service/models"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	logMutex sync.Mutex
	logs     []models.LogEntry
)

func respondWithFieldError(c *gin.Context, fieldName string, entry models.LogEntry) {
	c.JSON(400, models.LogResponse{
		Code: 400,
		Msg:  fmt.Sprintf("字段 %s 不能为空", fieldName),
		Data: entry,
	})
}

func HandleLog(c *gin.Context) {
	var entry models.LogEntry

	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(400, models.LogResponse{
			Code: 400,
			Msg:  "参数格式错误",
			Data: models.LogEntry{},
		})
		return
	}

	if entry.ServiceName == "" {
		respondWithFieldError(c, "serviceName", entry)
		return
	}
	if entry.ServiceId == "" {
		respondWithFieldError(c, "serviceId", entry)
		return
	}
	if entry.Datetime == "" {
		respondWithFieldError(c, "datetime", entry)
		return
	}
	if entry.Level == "" {
		respondWithFieldError(c, "level", entry)
		return
	}
	if entry.Message == "" {
		respondWithFieldError(c, "message", entry)
		return
	}

	logMutex.Lock()
	defer logMutex.Unlock()

	logs = append(logs, entry)
	writeSortedLogsToFile()

	c.JSON(200, models.LogResponse{
		Code: 200,
		Msg:  "日志接收成功",
		Data: entry,
	})
}

func writeSortedLogsToFile() {
	_ = os.MkdirAll("log", os.ModePerm)
	filePath := filepath.Join("log", "log.txt")

	sort.Slice(logs, func(i, j int) bool {
		ti, _ := time.Parse("2006-01-02 15:04:05.000", logs[i].Datetime)
		tj, _ := time.Parse("2006-01-02 15:04:05.000", logs[j].Datetime)
		return ti.Before(tj)
	})

	file, err := os.Create(filePath)
	if err != nil {
		log.Printf("[error] 无法创建日志文件: %v\n", err)
		return
	}
	defer file.Close()

	for _, logEntry := range logs {
		line := fmt.Sprintf(
			"[%s] ServiceName: %s ServiceId: %s Level: %s Message: %s\n",
			logEntry.Datetime,
			logEntry.ServiceName,
			logEntry.ServiceId,
			logEntry.Level,
			logEntry.Message,
		)
		file.WriteString(line)
	}
}

package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.POST("/webhook", func(c *gin.Context) {
		var payload map[string]interface{}
		if err := c.BindJSON(&payload); err != nil {
			log.Println("無法解析 JSON:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}

		if repo, ok := payload["repository"].(map[string]interface{}); ok {
			fmt.Println("收到 Push 事件 - Repo 名稱：", repo["full_name"])
		}
		if ref, ok := payload["ref"].(string); ok {
			fmt.Println("分支：", ref)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Webhook received"})
	})

	log.Println("伺服器啟動於 http://localhost:8080")
	router.Run(":8080")
}

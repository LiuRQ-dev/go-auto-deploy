package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const secret = "qwedsa8898"

func verifySignature(secret string, body []byte, signatureHeader string) bool {
	if !strings.HasPrefix(signatureHeader, "sha256=") {
		return false
	}
	signature := signatureHeader[len("sha256="):]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)
	expectedSignature := hex.EncodeToString(expectedMAC)

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func main() {
	router := gin.Default()

	router.POST("/webhook", func(c *gin.Context) {

		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			log.Println("讀取 body 失敗:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
			return
		}

		signature := c.GetHeader("X-Hub-Signature-256")
		if !verifySignature(secret, body, signature) {
			log.Println("Webhook 驗證失敗")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "signature mismatch"})
			return
		}

		c.Request.Body = ioutil.NopCloser(strings.NewReader(string(body)))
		var payload map[string]interface{}
		if err := c.BindJSON(&payload); err != nil {
			log.Println("解析 JSON 失敗:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}

		if repo, ok := payload["repository"].(map[string]interface{}); ok {
			fmt.Println("收到 Push 事件 - Repo 名稱：", repo["full_name"])
		}
		if ref, ok := payload["ref"].(string); ok {
			fmt.Println("分支：", ref)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Webhook received and verified"})
	})

	log.Println("伺服器啟動於 http://localhost:3020")
	router.Run(":3020")
}

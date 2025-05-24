package handler

import (
	"backendPt/config"
	models "backendPt/model"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	config   *config.Config
	db       *models.Database
	executor *Executor
	notifier *Notifier
}

type GitHubPayload struct {
	Repository struct {
		FullName string `json:"full_name"`
		Name     string `json:"name"`
	} `json:"repository"`
	Ref        string `json:"ref"`
	HeadCommit struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Author  struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"head_commit"`
	Pusher struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"pusher"`
}

func NewHandler(cfg *config.Config, db *models.Database) *Handler {
	return &Handler{
		config:   cfg,
		db:       db,
		executor: NewExecutor(db),
		notifier: NewNotifier(cfg),
	}
}

func (h *Handler) verifySignature(secret string, body []byte, signatureHeader string) bool {
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

func (h *Handler) HandleWebhook(c *gin.Context) {

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("讀取 body 失敗: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	signature := c.GetHeader("X-Hub-Signature-256")
	if !h.verifySignature(h.config.Webhook.Secret, body, signature) {
		log.Println("Webhook 驗證失敗")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "signature mismatch"})
		return
	}

	var payload GitHubPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("解析 JSON 失敗: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	deployConfig := h.config.GetDeploymentConfig(payload.Repository.FullName)
	if deployConfig == nil {
		log.Printf("未找到倉庫 %s 的部署配置", payload.Repository.FullName)
		c.JSON(http.StatusOK, gin.H{"message": "repository not configured for deployment"})
		return
	}

	expectedRef := fmt.Sprintf("refs/heads/%s", deployConfig.Branch)
	if payload.Ref != expectedRef {
		log.Printf("分支不匹配: 期望 %s, 收到 %s", expectedRef, payload.Ref)
		c.JSON(http.StatusOK, gin.H{"message": "branch not configured for deployment"})
		return
	}

	record := &models.DeployRecord{
		Repository: payload.Repository.FullName,
		Branch:     deployConfig.Branch,
		Commit:     payload.HeadCommit.ID,
		Status:     "pending",
		StartTime:  time.Now(),
		Output:     fmt.Sprintf("Triggered by push from %s", payload.Pusher.Name),
	}

	deployID, err := h.db.InsertDeploy(record)
	if err != nil {
		log.Printf("插入部署記錄失敗: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create deploy record"})
		return
	}

	log.Printf("開始部署: Repo=%s, Branch=%s, Commit=%s, DeployID=%d",
		payload.Repository.FullName,
		deployConfig.Branch,
		payload.HeadCommit.ID[:8],
		deployID)

	go func() {
		if err := h.executor.Execute(deployID, deployConfig); err != nil {
			log.Printf("部署失敗 (ID: %d): %v", deployID, err)
			h.notifier.NotifyFailure(deployConfig, payload.HeadCommit.ID, err.Error())
		} else {
			log.Printf("部署成功 (ID: %d)", deployID)
			h.notifier.NotifySuccess(deployConfig, payload.HeadCommit.ID)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message":    "deployment started",
		"deploy_id":  deployID,
		"repository": payload.Repository.FullName,
		"commit":     payload.HeadCommit.ID[:8],
	})
}

func (h *Handler) GetDeploys(c *gin.Context) {
	deploys, err := h.db.GetDeploys(50)
	if err != nil {
		log.Printf("獲取部署記錄失敗: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get deploys"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deploys": deploys,
		"count":   len(deploys),
	})
}

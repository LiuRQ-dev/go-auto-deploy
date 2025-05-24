package handler

import (
	"backendPt/config"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"time"
)

type Notifier struct {
	config *config.Config
}

type SlackMessage struct {
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Attachment struct {
	Color  string  `json:"color"`
	Title  string  `json:"title"`
	Text   string  `json:"text"`
	Fields []Field `json:"fields,omitempty"`
	TS     int64   `json:"ts"`
}

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

func NewNotifier(cfg *config.Config) *Notifier {
	return &Notifier{config: cfg}
}

func (n *Notifier) NotifySuccess(deployConfig *config.DeploymentConfig, commit string) {
	message := fmt.Sprintf("部署成功: %s", deployConfig.Name)

	if n.config.Notification.Webhook.Enabled {
		n.sendSlackNotification(message, "good", deployConfig, commit)
	}

	if n.config.Notification.Email.Enabled {
		subject := fmt.Sprintf("部署成功: %s", deployConfig.Name)
		body := fmt.Sprintf(`
部署成功！

項目: %s
倉庫: %s
分支: %s
提交: %s
時間: %s

部署已成功完成。
`, deployConfig.Name, deployConfig.Repository, deployConfig.Branch, commit, time.Now().Format("2006-01-02 15:04:05"))

		n.sendEmail(subject, body)
	}
}

func (n *Notifier) NotifyFailure(deployConfig *config.DeploymentConfig, commit string, errorMsg string) {
	message := fmt.Sprintf("部署失敗: %s", deployConfig.Name)

	if n.config.Notification.Webhook.Enabled {
		n.sendSlackNotification(message, "danger", deployConfig, commit)
	}

	if n.config.Notification.Email.Enabled {
		subject := fmt.Sprintf("部署失敗: %s", deployConfig.Name)
		body := fmt.Sprintf(`
部署失敗！

項目: %s
倉庫: %s
分支: %s
提交: %s
時間: %s

錯誤信息:
%s

請檢查日誌並修復問題。
`, deployConfig.Name, deployConfig.Repository, deployConfig.Branch, commit, time.Now().Format("2006-01-02 15:04:05"), errorMsg)

		n.sendEmail(subject, body)
	}
}

func (n *Notifier) sendSlackNotification(message, color string, deployConfig *config.DeploymentConfig, commit string) {
	if n.config.Notification.Webhook.URL == "" {
		return
	}

	slackMsg := SlackMessage{
		Text: message,
		Attachments: []Attachment{
			{
				Color: color,
				Title: fmt.Sprintf("部署詳情: %s", deployConfig.Name),
				Fields: []Field{
					{Title: "項目", Value: deployConfig.Name, Short: true},
					{Title: "倉庫", Value: deployConfig.Repository, Short: true},
					{Title: "分支", Value: deployConfig.Branch, Short: true},
					{Title: "提交", Value: commit[:8], Short: true},
				},
				TS: time.Now().Unix(),
			},
		},
	}

	jsonData, err := json.Marshal(slackMsg)
	if err != nil {
		log.Printf("序列化 Slack 消息失敗: %v", err)
		return
	}

	resp, err := http.Post(n.config.Notification.Webhook.URL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("發送 Slack 通知失敗: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Slack 通知返回錯誤狀態: %d", resp.StatusCode)
	}
}

func (n *Notifier) sendEmail(subject, body string) {
	emailConfig := n.config.Notification.Email

	// 設置認證
	auth := smtp.PlainAuth("", emailConfig.Username, emailConfig.Password, emailConfig.SMTPHost)

	// 構建郵件
	to := emailConfig.To
	msg := []byte(fmt.Sprintf(`To: %s
Subject: %s
Content-Type: text/plain; charset=UTF-8

%s`,
		string(bytes.Join([][]byte{[]byte(to[0])}, []byte(", "))),
		subject,
		body))

	// 發送郵件
	addr := fmt.Sprintf("%s:%d", emailConfig.SMTPHost, emailConfig.SMTPPort)
	err := smtp.SendMail(addr, auth, emailConfig.Username, to, msg)
	if err != nil {
		log.Printf("發送郵件失敗: %v", err)
		return
	}

	log.Printf("郵件通知已發送: %s", subject)
}

func (n *Notifier) SendTestNotification() error {
	if n.config.Notification.Webhook.Enabled {
		testMsg := SlackMessage{
			Text: "🧪 測試通知",
			Attachments: []Attachment{
				{
					Color: "warning",
					Title: "系統測試",
					Text:  "這是一條測試通知，確認部署系統正常工作。",
					TS:    time.Now().Unix(),
				},
			},
		}

		jsonData, err := json.Marshal(testMsg)
		if err != nil {
			return err
		}

		resp, err := http.Post(n.config.Notification.Webhook.URL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("slack returned status %d", resp.StatusCode)
		}
	}

	if n.config.Notification.Email.Enabled {
		n.sendEmail("測試通知", "這是一條測試郵件，確認部署系統正常工作。")
	}

	return nil
}

package handler

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"backendPt/config"
	models "backendPt/model"
)

type Executor struct {
	db *models.Database
}

func NewExecutor(db *models.Database) *Executor {
	return &Executor{db: db}
}

func (e *Executor) Execute(deployID int64, config *config.DeploymentConfig) error {

	if err := e.db.UpdateDeploy(deployID, "running", nil, "", ""); err != nil {
		return fmt.Errorf("更新部署狀態失敗: %w", err)
	}

	var output bytes.Buffer
	var errorOutput bytes.Buffer

	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("獲取當前目錄失敗: %w", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(config.WorkDir); err != nil {
		errorMsg := fmt.Sprintf("切換到工作目錄失敗: %v", err)
		endTime := time.Now()
		e.db.UpdateDeploy(deployID, "failed", &endTime, output.String(), errorMsg)
		return fmt.Errorf(errorMsg)
	}

	output.WriteString(fmt.Sprintf("=== 開始部署 %s ===\n", config.Name))
	output.WriteString(fmt.Sprintf("工作目錄: %s\n", config.WorkDir))

	for i, command := range config.Commands {
		output.WriteString(fmt.Sprintf(">>> 步驟 %d: %s\n", i+1, command))

		if err := e.executeCommand(command, &output, &errorOutput); err != nil {
			errorMsg := fmt.Sprintf("命令執行失敗: %s\n錯誤: %v\n%s", command, err, errorOutput.String())
			output.WriteString(fmt.Sprintf("失敗: %s\n", command))

			endTime := time.Now()
			e.db.UpdateDeploy(deployID, "failed", &endTime, output.String(), errorMsg)
			return fmt.Errorf(errorMsg)
		}

		output.WriteString(fmt.Sprintf("成功: %s\n\n", command))
	}

	output.WriteString("=== 部署完成 ===\n")

	endTime := time.Now()
	if err := e.db.UpdateDeploy(deployID, "success", &endTime, output.String(), ""); err != nil {
		return fmt.Errorf("更新部署狀態失敗: %w", err)
	}

	return nil
}

func (e *Executor) executeCommand(command string, output, errorOutput *bytes.Buffer) error {

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("空命令")
	}

	cmd := exec.Command(parts[0], parts[1:]...)

	cmd.Stdout = output
	cmd.Stderr = errorOutput

	cmd.Env = append(os.Environ(),
		"DEPLOY_TIME="+time.Now().Format("2006-01-02T15:04:05Z"),
	)

	return cmd.Run()
}

func (e *Executor) GetCommandOutput(command string, workDir string) (string, error) {
	originalDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	defer os.Chdir(originalDir)

	if workDir != "" {
		if err := os.Chdir(workDir); err != nil {
			return "", err
		}
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("空命令")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

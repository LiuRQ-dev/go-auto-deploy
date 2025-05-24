

.PHONY: build run clean test install docker help


BINARY_NAME=go-auto-deploy
VERSION=1.0.0
BUILD_TIME=$(shell date +%Y-%m-%d\ %H:%M:%S)
BUILD_DIR=build
CONFIG_FILE=config.yaml


GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod


LDFLAGS=-ldflags "-X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)'"

help: 
	@echo "Go 自動部署系統 - 可用命令:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: 
	@echo "建構 $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "建構完成: $(BUILD_DIR)/$(BINARY_NAME)"


run: build 
	@echo "啟動 $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME) -config $(CONFIG_FILE)

dev: 
	@echo "開發模式啟動..."
	@$(GOCMD) run . -config $(CONFIG_FILE)

test: 
	@echo "運行測試..."
	@$(GOTEST) -v ./...

test-notify: build
	@echo "測試通知功能..."
	@./$(BUILD_DIR)/$(BINARY_NAME) -test-notify -config $(CONFIG_FILE)

clean: 
	@echo "清理建構文件..."
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "清理完成"

deps: 
	@echo "下載依賴..."
	@$(GOMOD) download
	@$(GOMOD) tidy
	@echo "依賴更新完成"

install: build 
	@echo "安裝 $(BINARY_NAME) 到 /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "安裝完成"

uninstall: 
	@echo "卸載 $(BINARY_NAME)..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "卸載完成"


service-install: install
	@echo "安裝 systemd 服務..."
	@sudo cp scripts/go-auto-deploy.service /etc/systemd/system/
	@sudo systemctl daemon-reload
	@sudo systemctl enable go-auto-deploy
	@echo "服務安裝完成"
	@echo "使用 'sudo systemctl start go-auto-deploy' 啟動服務"

service-uninstall: 
	@echo "卸載 systemd 服務..."
	@sudo systemctl stop go-auto-deploy || true
	@sudo systemctl disable go-auto-deploy || true
	@sudo rm -f /etc/systemd/system/go-auto-deploy.service
	@sudo systemctl daemon-reload
	@echo "服務卸載完成"


fmt: 
	@echo "格式化代碼..."
	@$(GOCMD) fmt ./...
	@echo "代碼格式化完成"

lint: 
	@echo "代碼檢查..."
	@golangci-lint run
	@echo "代碼檢查完成"


init: 
	@echo "初始化項目結構..."
	@mkdir -p logs data scripts web/static
	@touch logs/.gitkeep data/.gitkeep
	@if [ ! -f $(CONFIG_FILE) ]; then \
		echo "創建配置文件..."; \
		cp config.yaml.example $(CONFIG_FILE) 2>/dev/null || echo "請手動創建 $(CONFIG_FILE)"; \
	fi
	@echo "項目初始化完成"


backup: 
	@echo "備份數據..."
	@mkdir -p backups
	@tar -czf backups/backup_$(shell date +%Y%m%d_%H%M%S).tar.gz data/ logs/ $(CONFIG_FILE)
	@echo "備份完成"


version: 
	@echo "版本: $(VERSION)"
	@echo "建構時間: $(BUILD_TIME)"

logs: 
	@tail -f logs/deploy.log

status: 
	@if command -v systemctl >/dev/null 2>&1; then \
		systemctl status go-auto-deploy; \
	else \
		echo "systemd 不可用"; \
		ps aux | grep $(BINARY_NAME) | grep -v grep || echo "服務未運行"; \
	fi
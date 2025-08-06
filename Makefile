# AI Prompt Proxy Makefile
# 支持多平台编译：Windows、Linux、macOS 的 amd64 和 arm64 架构

# 项目信息
APP_NAME := ai-prompt-proxy
VERSION ?= 1.0.0
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 构建标志
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)

# 目录
BUILD_DIR := build
DIST_DIR := dist
WEB_DIR := web
CONFIG_DIR := configs

# 支持的平台
PLATFORMS := \
	windows/amd64 \
	windows/arm64 \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64

# 默认目标
.PHONY: all
all: clean deps build

# 清理构建文件
.PHONY: clean
clean:
	@echo "🧹 清理构建文件..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)

# 下载依赖
.PHONY: deps
deps:
	@echo "📦 下载依赖..."
	@go mod download
	@go mod tidy

# 构建所有平台
.PHONY: build
build: $(PLATFORMS)

# 构建单个平台的规则
.PHONY: $(PLATFORMS)
$(PLATFORMS): deps
	$(eval GOOS := $(word 1,$(subst /, ,$@)))
	$(eval GOARCH := $(word 2,$(subst /, ,$@)))
	$(eval OUTPUT_NAME := $(APP_NAME)$(if $(filter windows,$(GOOS)),.exe,))
	$(eval OUTPUT_DIR := $(BUILD_DIR)/$(APP_NAME)-$(GOOS)-$(GOARCH))
	@echo "🔨 编译 $(GOOS)/$(GOARCH)..."
	@mkdir -p $(OUTPUT_DIR)
	
	@GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build \
		-ldflags="$(LDFLAGS)" \
		-o $(OUTPUT_DIR)/$(OUTPUT_NAME) \
		.
	
	@if [ $$? -eq 0 ]; then \
		echo "✅ $(GOOS)/$(GOARCH) 编译成功"; \
		mkdir -p $(OUTPUT_DIR)/$(CONFIG_DIR); \
		$(MAKE) create-startup-scripts GOOS=$(GOOS) OUTPUT_DIR=$(OUTPUT_DIR) OUTPUT_NAME=$(OUTPUT_NAME); \
		$(MAKE) create-archive GOOS=$(GOOS) GOARCH=$(GOARCH) OUTPUT_DIR=$(OUTPUT_DIR); \
	else \
		echo "❌ $(GOOS)/$(GOARCH) 编译失败"; \
		exit 1; \
	fi

# 创建启动脚本
.PHONY: create-startup-scripts
create-startup-scripts:
	@if [ "$(GOOS)" = "windows" ]; then \
		echo '@echo off' > $(OUTPUT_DIR)/start.bat; \
		echo 'echo Starting AI Prompt Proxy...' >> $(OUTPUT_DIR)/start.bat; \
		echo 'echo Web interface will be available at: http://localhost:8081' >> $(OUTPUT_DIR)/start.bat; \
		echo 'echo Press Ctrl+C to stop the server' >> $(OUTPUT_DIR)/start.bat; \
		echo '$(OUTPUT_NAME)' >> $(OUTPUT_DIR)/start.bat; \
		echo 'pause' >> $(OUTPUT_DIR)/start.bat; \
		\
		echo 'Write-Host "Starting AI Prompt Proxy..." -ForegroundColor Green' > $(OUTPUT_DIR)/start.ps1; \
		echo 'Write-Host "Web interface will be available at: http://localhost:8081" -ForegroundColor Yellow' >> $(OUTPUT_DIR)/start.ps1; \
		echo 'Write-Host "Press Ctrl+C to stop the server" -ForegroundColor Yellow' >> $(OUTPUT_DIR)/start.ps1; \
		echo '& ".\$(OUTPUT_NAME)"' >> $(OUTPUT_DIR)/start.ps1; \
	else \
		echo '#!/bin/bash' > $(OUTPUT_DIR)/start.sh; \
		echo 'echo "Starting AI Prompt Proxy..."' >> $(OUTPUT_DIR)/start.sh; \
		echo 'echo "Web interface will be available at: http://localhost:8081"' >> $(OUTPUT_DIR)/start.sh; \
		echo 'echo "Press Ctrl+C to stop the server"' >> $(OUTPUT_DIR)/start.sh; \
		echo './$(OUTPUT_NAME)' >> $(OUTPUT_DIR)/start.sh; \
		chmod +x $(OUTPUT_DIR)/start.sh; \
	fi

# 创建压缩包
.PHONY: create-archive
create-archive:
	@mkdir -p $(DIST_DIR)
	$(eval ARCHIVE_NAME := $(APP_NAME)-$(VERSION)-$(GOOS)-$(GOARCH))
	@echo "📦 创建压缩包 $(ARCHIVE_NAME)..."
	
	@cd $(BUILD_DIR) && \
	if [ "$(GOOS)" = "windows" ]; then \
		if command -v zip >/dev/null 2>&1; then \
			zip -r ../$(DIST_DIR)/$(ARCHIVE_NAME).zip $(APP_NAME)-$(GOOS)-$(GOARCH)/; \
		else \
			echo "⚠️  zip 命令不可用，跳过 Windows 压缩包创建"; \
		fi; \
	else \
		tar -czf ../$(DIST_DIR)/$(ARCHIVE_NAME).tar.gz $(APP_NAME)-$(GOOS)-$(GOARCH)/; \
	fi

# 快速构建（仅当前平台）
.PHONY: build-local
build-local: deps
	@echo "🔨 编译本地平台..."
	@go build -ldflags="$(LDFLAGS)" -o $(APP_NAME) .
	@echo "✅ 本地编译完成: ./$(APP_NAME)"

# 运行程序
.PHONY: run
run: build-local
	@echo "🚀 启动 AI Prompt Proxy..."
	@./$(APP_NAME)

# 开发模式运行
.PHONY: dev
dev:
	@echo "🔧 开发模式启动..."
	@go run -ldflags="$(LDFLAGS)" . 

# 测试
.PHONY: test
test:
	@echo "🧪 运行测试..."
	@go test -v ./...

# 代码检查
.PHONY: lint
lint:
	@echo "🔍 代码检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint 未安装，跳过代码检查"; \
		echo "安装命令: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# 格式化代码
.PHONY: fmt
fmt:
	@echo "🎨 格式化代码..."
	@go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

# 创建发布说明
.PHONY: create-readme
create-readme:
	@mkdir -p $(DIST_DIR)
	@echo "📝 创建发布说明..."
	@echo "# AI Prompt Proxy v$(VERSION)" > $(DIST_DIR)/README.md
	@echo "" >> $(DIST_DIR)/README.md
	@echo "## 快速开始" >> $(DIST_DIR)/README.md
	@echo "" >> $(DIST_DIR)/README.md
	@echo "### Windows" >> $(DIST_DIR)/README.md
	@echo "1. 解压 \`ai-prompt-proxy-$(VERSION)-windows-amd64.zip\` 或 \`ai-prompt-proxy-$(VERSION)-windows-arm64.zip\`" >> $(DIST_DIR)/README.md
	@echo "2. 双击 \`start.bat\` 或运行 \`start.ps1\`" >> $(DIST_DIR)/README.md
	@echo "3. 打开浏览器访问 http://localhost:8081" >> $(DIST_DIR)/README.md
	@echo "" >> $(DIST_DIR)/README.md
	@echo "### Linux" >> $(DIST_DIR)/README.md
	@echo "1. 解压 \`ai-prompt-proxy-$(VERSION)-linux-amd64.tar.gz\` 或 \`ai-prompt-proxy-$(VERSION)-linux-arm64.tar.gz\`" >> $(DIST_DIR)/README.md
	@echo "2. 运行 \`./start.sh\` 或直接运行 \`./ai-prompt-proxy\`" >> $(DIST_DIR)/README.md
	@echo "3. 打开浏览器访问 http://localhost:8081" >> $(DIST_DIR)/README.md
	@echo "" >> $(DIST_DIR)/README.md
	@echo "### macOS" >> $(DIST_DIR)/README.md
	@echo "1. 解压 \`ai-prompt-proxy-$(VERSION)-darwin-amd64.tar.gz\` 或 \`ai-prompt-proxy-$(VERSION)-darwin-arm64.tar.gz\`" >> $(DIST_DIR)/README.md
	@echo "2. 运行 \`./start.sh\` 或直接运行 \`./ai-prompt-proxy\`" >> $(DIST_DIR)/README.md
	@echo "3. 打开浏览器访问 http://localhost:8081" >> $(DIST_DIR)/README.md
	@echo "" >> $(DIST_DIR)/README.md
	@echo "## 架构说明" >> $(DIST_DIR)/README.md
	@echo "- \`amd64\`: 适用于 Intel/AMD 64位处理器" >> $(DIST_DIR)/README.md
	@echo "- \`arm64\`: 适用于 ARM 64位处理器（如 Apple M1/M2、ARM服务器等）" >> $(DIST_DIR)/README.md
	@echo "" >> $(DIST_DIR)/README.md
	@echo "## 特性" >> $(DIST_DIR)/README.md
	@echo "- **单文件部署**: 前端界面已嵌入到可执行文件中，无需额外的web目录" >> $(DIST_DIR)/README.md
	@echo "- **即开即用**: 双击可执行文件即可启动，无需安装依赖" >> $(DIST_DIR)/README.md
	@echo "- **跨平台**: 支持 Windows、Linux、macOS 多平台" >> $(DIST_DIR)/README.md
	@echo "- **多架构**: 支持 amd64 和 arm64 架构" >> $(DIST_DIR)/README.md
	@echo "" >> $(DIST_DIR)/README.md
	@echo "## 配置" >> $(DIST_DIR)/README.md
	@echo "- 配置文件位于 \`configs/\` 目录" >> $(DIST_DIR)/README.md
	@echo "" >> $(DIST_DIR)/README.md
	@echo "## 构建信息" >> $(DIST_DIR)/README.md
	@echo "- 版本: $(VERSION)" >> $(DIST_DIR)/README.md
	@echo "- 构建时间: $(BUILD_TIME)" >> $(DIST_DIR)/README.md
	@echo "- Git提交: $(GIT_COMMIT)" >> $(DIST_DIR)/README.md
	@echo "" >> $(DIST_DIR)/README.md
	@echo "## 支持" >> $(DIST_DIR)/README.md
	@echo "如有问题，请访问项目仓库获取帮助。" >> $(DIST_DIR)/README.md

# 完整发布流程
.PHONY: release
release: clean deps test lint build create-readme
	@echo "🎉 发布构建完成！"
	@echo "📊 构建统计:"
	@echo "📁 构建文件: $(BUILD_DIR)/"
	@echo "📦 发布包: $(DIST_DIR)/"
	@if [ -d "$(DIST_DIR)" ]; then \
		echo "📋 发布包列表:"; \
		ls -la $(DIST_DIR)/; \
	fi

# 显示帮助信息
.PHONY: help
help:
	@echo "AI Prompt Proxy 构建系统"
	@echo ""
	@echo "可用命令:"
	@echo "  make all          - 清理、下载依赖并构建所有平台"
	@echo "  make build        - 构建所有平台"
	@echo "  make build-local  - 仅构建当前平台"
	@echo "  make clean        - 清理构建文件"
	@echo "  make deps         - 下载依赖"
	@echo "  make run          - 构建并运行程序"
	@echo "  make dev          - 开发模式运行"
	@echo "  make test         - 运行测试"
	@echo "  make lint         - 代码检查"
	@echo "  make fmt          - 格式化代码"
	@echo "  make release      - 完整发布流程"
	@echo "  make help         - 显示此帮助信息"
	@echo ""
	@echo "支持的平台:"
	@echo "  $(PLATFORMS)"
	@echo ""
	@echo "环境变量:"
	@echo "  VERSION           - 设置版本号 (默认: $(VERSION))"
	@echo ""
	@echo "示例:"
	@echo "  make VERSION=2.0.0 release"
	@echo "  make windows/amd64"
	@echo "  make linux/arm64"
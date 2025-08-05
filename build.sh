#!/bin/bash

# AI Prompt Proxy 多平台编译脚本
# 支持 Windows、Linux、macOS 平台的 amd64 和 arm64 架构

set -e

# 项目信息
APP_NAME="ai-prompt-proxy"
VERSION=${VERSION:-"1.0.0"}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 构建信息
LDFLAGS="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

# 输出目录
BUILD_DIR="build"
DIST_DIR="dist"

# 清理旧的构建文件
echo "🧹 清理旧的构建文件..."
rm -rf ${BUILD_DIR}
rm -rf ${DIST_DIR}
mkdir -p ${BUILD_DIR}
mkdir -p ${DIST_DIR}

# 支持的平台和架构
declare -a PLATFORMS=(
    "windows/amd64"
    "windows/arm64"
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

# 编译函数
build_binary() {
    local platform=$1
    local os=$(echo $platform | cut -d'/' -f1)
    local arch=$(echo $platform | cut -d'/' -f2)
    
    local output_name="${APP_NAME}"
    if [ "$os" = "windows" ]; then
        output_name="${APP_NAME}.exe"
    fi
    
    local output_path="${BUILD_DIR}/${APP_NAME}-${os}-${arch}"
    mkdir -p $output_path
    
    echo "🔨 编译 ${os}/${arch}..."
    
    # 设置环境变量并编译
    GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build \
        -ldflags="${LDFLAGS}" \
        -o "${output_path}/${output_name}" \
        .
    
    if [ $? -eq 0 ]; then
        echo "✅ ${os}/${arch} 编译成功"
        
        # 创建配置目录
        mkdir -p "${output_path}/configs"
        
        # 创建启动脚本
        create_startup_scripts "$output_path" "$os" "$output_name"
        
        # 创建压缩包
        create_archive "$output_path" "${os}" "${arch}"
    else
        echo "❌ ${os}/${arch} 编译失败"
        return 1
    fi
}

# 创建启动脚本
create_startup_scripts() {
    local output_path=$1
    local os=$2
    local binary_name=$3
    
    if [ "$os" = "windows" ]; then
        # Windows 批处理文件
        cat > "${output_path}/start.bat" << 'EOF'
@echo off
echo Starting AI Prompt Proxy...
echo Web interface will be available at: http://localhost:8081
echo Press Ctrl+C to stop the server
ai-prompt-proxy.exe
pause
EOF
        
        # Windows PowerShell 脚本
        cat > "${output_path}/start.ps1" << 'EOF'
Write-Host "Starting AI Prompt Proxy..." -ForegroundColor Green
Write-Host "Web interface will be available at: http://localhost:8081" -ForegroundColor Yellow
Write-Host "Press Ctrl+C to stop the server" -ForegroundColor Yellow
& ".\ai-prompt-proxy.exe"
EOF
    else
        # Unix shell 脚本
        cat > "${output_path}/start.sh" << EOF
#!/bin/bash
echo "Starting AI Prompt Proxy..."
echo "Web interface will be available at: http://localhost:8081"
echo "Press Ctrl+C to stop the server"
./${binary_name}
EOF
        chmod +x "${output_path}/start.sh"
    fi
}

# 创建压缩包
create_archive() {
    local output_path=$1
    local os=$2
    local arch=$3
    
    local archive_name="${APP_NAME}-${VERSION}-${os}-${arch}"
    
    echo "📦 创建压缩包 ${archive_name}..."
    
    cd ${BUILD_DIR}
    
    if [ "$os" = "windows" ]; then
        # Windows 使用 zip
        if command -v zip >/dev/null 2>&1; then
            zip -r "../${DIST_DIR}/${archive_name}.zip" "${APP_NAME}-${os}-${arch}/"
        else
            echo "⚠️  zip 命令不可用，跳过 Windows 压缩包创建"
        fi
    else
        # Unix 使用 tar.gz
        tar -czf "../${DIST_DIR}/${archive_name}.tar.gz" "${APP_NAME}-${os}-${arch}/"
    fi
    
    cd ..
}

# 创建 README 文件
create_readme() {
    cat > "${DIST_DIR}/README.md" << EOF
# AI Prompt Proxy v${VERSION}

## 快速开始

### Windows
1. 解压 \`ai-prompt-proxy-${VERSION}-windows-amd64.zip\` 或 \`ai-prompt-proxy-${VERSION}-windows-arm64.zip\`
2. 双击 \`start.bat\` 或运行 \`start.ps1\`
3. 打开浏览器访问 http://localhost:8081

### Linux
1. 解压 \`ai-prompt-proxy-${VERSION}-linux-amd64.tar.gz\` 或 \`ai-prompt-proxy-${VERSION}-linux-arm64.tar.gz\`
2. 运行 \`./start.sh\` 或直接运行 \`./ai-prompt-proxy\`
3. 打开浏览器访问 http://localhost:8081

### macOS
1. 解压 \`ai-prompt-proxy-${VERSION}-darwin-amd64.tar.gz\` 或 \`ai-prompt-proxy-${VERSION}-darwin-arm64.tar.gz\`
2. 运行 \`./start.sh\` 或直接运行 \`./ai-prompt-proxy\`
3. 打开浏览器访问 http://localhost:8081

## 架构说明
- \`amd64\`: 适用于 Intel/AMD 64位处理器
- \`arm64\`: 适用于 ARM 64位处理器（如 Apple M1/M2、ARM服务器等）

## 配置
- 配置文件位于 \`configs/\` 目录
- Web界面文件位于 \`web/\` 目录

## 构建信息
- 版本: ${VERSION}
- 构建时间: ${BUILD_TIME}
- Git提交: ${GIT_COMMIT}

## 支持
如有问题，请访问项目仓库获取帮助。
EOF
}

# 主编译流程
main() {
    echo "🚀 开始编译 AI Prompt Proxy v${VERSION}"
    echo "📅 构建时间: ${BUILD_TIME}"
    echo "🔗 Git提交: ${GIT_COMMIT}"
    echo ""
    
    # 检查 Go 环境
    if ! command -v go >/dev/null 2>&1; then
        echo "❌ Go 未安装或不在 PATH 中"
        exit 1
    fi
    
    echo "🔍 Go 版本: $(go version)"
    echo ""
    
    # 下载依赖
    echo "📦 下载依赖..."
    go mod download
    go mod tidy
    echo ""
    
    # 编译所有平台
    local success_count=0
    local total_count=${#PLATFORMS[@]}
    
    for platform in "${PLATFORMS[@]}"; do
        if build_binary "$platform"; then
            ((success_count++))
        fi
        echo ""
    done
    
    # 创建 README
    create_readme
    
    # 显示结果
    echo "🎉 编译完成！"
    echo "📊 成功: ${success_count}/${total_count}"
    echo ""
    echo "📁 构建文件位于: ${BUILD_DIR}/"
    echo "📦 发布包位于: ${DIST_DIR}/"
    echo ""
    
    if [ -d "${DIST_DIR}" ]; then
        echo "📋 发布包列表:"
        ls -la "${DIST_DIR}/"
    fi
}

# 运行主函数
main "$@"
#!/bin/bash

# SSH Key Manager v0.4.0 - 验证和测试脚本
# 用于验证所有新功能是否正常工作

set -e

echo "=========================================="
echo "  SSH Key Manager v0.4.0 验证脚本"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

info() {
    echo -e "ℹ️  $1"
}

# 1. 编译测试
echo "1️⃣  编译测试"
echo "-------------------"

info "清理旧文件..."
rm -f bin/skm bin/skm-server

info "编译客户端..."
if go build -o bin/skm ./cmd/skm/main.go 2>&1; then
    success "客户端编译成功"
else
    error "客户端编译失败"
    exit 1
fi

info "编译服务器..."
if go build -o bin/skm-server ./cmd/skm-server/main.go 2>&1; then
    success "服务器编译成功"
else
    error "服务器编译失败"
    exit 1
fi

echo ""

# 2. 代码质量检查
echo "2️⃣  代码质量检查"
echo "-------------------"

info "运行 go vet..."
if go vet ./... 2>&1; then
    success "代码检查通过"
else
    warning "代码检查有警告"
fi

info "检查 go fmt..."
if [ -z "$(gofmt -l . 2>&1 | grep -v vendor)" ]; then
    success "代码格式正确"
else
    warning "部分代码需要格式化"
fi

echo ""

# 3. 功能测试
echo "3️⃣  功能测试"
echo "-------------------"

info "测试基本命令..."
if ./bin/skm --help > /dev/null 2>&1; then
    success "基本命令正常"
else
    error "基本命令异常"
fi

info "测试 Shell 补全..."
if ./bin/skm completion bash > /dev/null 2>&1; then
    success "Bash 补全正常"
else
    error "Bash 补全异常"
fi

if ./bin/skm completion zsh > /dev/null 2>&1; then
    success "Zsh 补全正常"
else
    error "Zsh 补全异常"
fi

if ./bin/skm completion fish > /dev/null 2>&1; then
    success "Fish 补全正常"
else
    error "Fish 补全异常"
fi

info "测试 sync 命令..."
if ./bin/skm sync --help > /dev/null 2>&1; then
    success "Sync 命令正常"
else
    warning "Sync 命令可能需要配置"
fi

info "测试服务器帮助..."
if ./bin/skm-server --help > /dev/null 2>&1; then
    success "服务器帮助正常"
else
    error "服务器帮助异常"
fi

echo ""

# 4. 文件检查
echo "4️⃣  文件完整性检查"
echo "-------------------"

check_file() {
    if [ -f "$1" ]; then
        success "$1 存在"
        return 0
    else
        error "$1 不存在"
        return 1
    fi
}

# 检查新增的源代码文件
info "检查源代码文件..."

check_file "internal/server/webui.go"
check_file "internal/sync/incremental.go"
check_file "internal/sync/history.go"

# 检查文档文件


echo ""

# 5. 统计信息
echo "5️⃣  统计信息"
echo "-------------------"

info "Go 文件数量: $(find . -name "*.go" -not -path "./vendor/*" | wc -l)"
info "新增包数量: $(ls -d internal/*/ | wc -l)"

info "二进制文件大小:"
if [ -f bin/skm ]; then
    info "  skm: $(du -h bin/skm | cut -f1)"
fi
if [ -f bin/skm-server ]; then
    info "  skm-server: $(du -h bin/skm-server | cut -f1)"
fi

echo ""

# 6. 总结
echo "=========================================="
echo "  验证完成"
echo "=========================================="
echo ""
success "所有检查已完成！"
echo ""
echo "📖 下一步："

echo "  1. 设置补全: skm completion zsh > \"\${fpath[1]}/_skm\""
echo "  2. 初始化: skm init --device-name \"My Computer\""
echo "  3. 启动服务器: skm-server --addr :8080 --jwt-secret \"secret\""
echo ""
echo "🚀 享受 SSH Key Manager v0.4.0!"
echo ""


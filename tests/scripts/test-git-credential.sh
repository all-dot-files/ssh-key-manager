#!/bin/bash

# Git Credential Helper 测试脚本
# 用于测试 SKM 的 Git credential helper 实现

set -e

echo "==================================="
echo "Git Credential Helper 测试"
echo "==================================="
echo

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查 skm 是否存在
if [ ! -f "./bin/skm" ]; then
    echo -e "${RED}错误: ./bin/skm 不存在${NC}"
    echo "请先运行: make build"
    exit 1
fi

SKM="./bin/skm"

# 测试函数
test_get() {
    local protocol=$1
    local host=$2
    local username=$3
    
    echo -e "${YELLOW}测试 GET 操作:${NC}"
    echo "  Protocol: $protocol"
    echo "  Host: $host"
    echo "  Username: $username"
    echo
    
    # 构建输入
    input=""
    if [ -n "$protocol" ]; then
        input="${input}protocol=${protocol}\n"
    fi
    if [ -n "$host" ]; then
        input="${input}host=${host}\n"
    fi
    if [ -n "$username" ]; then
        input="${input}username=${username}\n"
    fi
    input="${input}\n"
    
    echo "输入:"
    echo -e "$input" | sed 's/^/  /'
    
    echo "输出:"
    result=$(echo -e "$input" | $SKM git helper get 2>&1)
    
    if [ -n "$result" ]; then
        echo "$result" | sed 's/^/  /'
        echo -e "${GREEN}✓ 返回了凭证${NC}"
    else
        echo "  (无输出)"
        echo -e "${YELLOW}! 未返回凭证 (可能是预期行为)${NC}"
    fi
    echo
}

test_store() {
    echo -e "${YELLOW}测试 STORE 操作:${NC}"
    
    input="protocol=ssh\nhost=github.com\nusername=git\npassword=dummy\n\n"
    
    echo "输入:"
    echo -e "$input" | sed 's/^/  /'
    
    echo "执行 store 操作..."
    if echo -e "$input" | $SKM git helper store 2>&1; then
        echo -e "${GREEN}✓ Store 操作成功${NC}"
    else
        echo -e "${RED}✗ Store 操作失败${NC}"
    fi
    echo
}

test_erase() {
    echo -e "${YELLOW}测试 ERASE 操作:${NC}"
    
    input="protocol=ssh\nhost=github.com\n\n"
    
    echo "输入:"
    echo -e "$input" | sed 's/^/  /'
    
    echo "执行 erase 操作..."
    if echo -e "$input" | $SKM git helper erase 2>&1; then
        echo -e "${GREEN}✓ Erase 操作成功${NC}"
    else
        echo -e "${RED}✗ Erase 操作失败${NC}"
    fi
    echo
}

# 运行测试
echo "==================================="
echo "1. GET 操作测试"
echo "==================================="
echo

# 测试 SSH 协议 (应该返回凭证，如果配置了 github.com)
test_get "ssh" "github.com" "git"

# 测试 HTTPS 协议 (应该被跳过)
test_get "https" "github.com" ""

# 测试无协议 (SSH 格式: git@github.com)
test_get "" "github.com" "git"

# 测试未配置的主机 (应该被跳过)
test_get "ssh" "example.com" ""

echo "==================================="
echo "2. STORE 操作测试"
echo "==================================="
echo

test_store

echo "==================================="
echo "3. ERASE 操作测试"
echo "==================================="
echo

test_erase

echo "==================================="
echo "测试完成"
echo "==================================="
echo
echo -e "${GREEN}所有基本操作测试完成${NC}"
echo
echo "注意事项:"
echo "1. GET 操作只有在配置了对应的 host 时才会返回凭证"
echo "2. HTTPS 协议会被自动跳过 (SKM 主要处理 SSH)"
echo "3. STORE 和 ERASE 操作会静默成功 (不实际存储/删除)"
echo
echo "配置 Git credential helper:"
echo "  全局: git config --global credential.helper '!$SKM git helper'"
echo "  本地: cd <repo> && git config credential.helper '!$SKM git helper'"

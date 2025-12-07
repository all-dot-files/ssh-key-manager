#!/bin/bash

# SKM Server 启动脚本

# Ensure we are in the project root
cd "$(git rev-parse --show-toplevel)"


# 配置
ADDR="${SKM_ADDR:-:8080}"
DATA_DIR="${SKM_DATA_DIR:-bin/data}"
JWT_SECRET="${SKM_JWT_SECRET:-mysecretkey123}"

# 颜色输出
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}   SKM Server Launcher${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""

# 检查服务器二进制文件
if [ ! -f "bin/skm-server" ]; then
    echo -e "${YELLOW}服务器二进制文件不存在，正在编译...${NC}"
    go build -o bin/skm-server cmd/skm-server/main.go
    if [ $? -ne 0 ]; then
        echo -e "${RED}编译失败！${NC}"
        exit 1
    fi
    echo -e "${GREEN}编译成功！${NC}"
    echo ""
fi

# 检查数据目录
if [ ! -d "$DATA_DIR" ]; then
    echo -e "${YELLOW}数据目录不存在，正在创建...${NC}"
    mkdir -p "$DATA_DIR/users"
    echo -e "${GREEN}数据目录已创建${NC}"
    echo ""
fi

# 检查是否存在用户数据
if [ ! -f "$DATA_DIR/users/admin.json" ]; then
    echo -e "${YELLOW}未找到用户数据，正在创建默认用户...${NC}"
    go run tools/reset-user.go "$DATA_DIR"
    echo ""
fi

echo -e "${GREEN}启动 SKM Server...${NC}"
echo -e "${BLUE}地址:${NC} http://localhost${ADDR#:}"
echo -e "${BLUE}数据目录:${NC} $DATA_DIR"
echo ""
echo -e "${GREEN}默认登录凭据:${NC}"
echo -e "  用户名: ${YELLOW}admin${NC}  密码: ${YELLOW}admin${NC}"
echo -e "  用户名: ${YELLOW}test${NC}   密码: ${YELLOW}test${NC}"
echo ""
echo -e "${BLUE}Web UI:${NC} http://localhost${ADDR#:}/"
echo -e "${BLUE}API:${NC}    http://localhost${ADDR#:}/api/v1/"
echo ""
echo -e "${YELLOW}按 Ctrl+C 停止服务器${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""

# 启动服务器
./bin/skm-server --addr "$ADDR" --data "$DATA_DIR" --jwt-secret "$JWT_SECRET"

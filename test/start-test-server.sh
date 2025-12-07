#!/bin/bash

echo "🔨 编译 SKM 服务器..."
go build -o bin/skm-server ./cmd/skm-server

if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi

echo "✅ 编译成功"

echo "🛑 停止旧的服务器进程..."
pkill -f "skm-server" 2>/dev/null

echo "🚀 启动服务器..."
./bin/skm-server -jwt-secret "test-secret-key" &
SERVER_PID=$!

echo "⏳ 等待服务器启动..."
sleep 2

# 检查服务器是否运行
if ps -p $SERVER_PID > /dev/null; then
    echo "✅ 服务器运行中 (PID: $SERVER_PID)"
    echo ""
    echo "📝 服务器信息:"
    echo "   Web UI: http://localhost:8080"
    echo "   API: http://localhost:8080/api/v1"
    echo ""
    echo "📖 快速测试:"
    echo "   1. 打开浏览器访问: http://localhost:8080/register"
    echo "   2. 注册一个新用户"
    echo "   3. 使用新用户登录"
    echo ""
    echo "💡 停止服务器: kill $SERVER_PID"
else
    echo "❌ 服务器启动失败"
    exit 1
fi

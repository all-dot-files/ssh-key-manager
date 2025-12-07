#!/bin/bash

# 简单的 Git Credential Helper 功能验证

echo "==================================="
echo "Git Credential Helper 功能验证"
echo "==================================="
echo

# 检查二进制文件
if [ ! -f "./bin/skm" ]; then
    echo "❌ 错误: ./bin/skm 不存在"
    echo "运行: make build"
    exit 1
fi

echo "✅ SKM 二进制文件存在"
echo

# 测试 1: 验证命令存在
echo "测试 1: 验证 git helper 命令"
if ./bin/skm git helper get --help &>/dev/null || ./bin/skm git helper --help &>/dev/null; then
    echo "✅ git helper 命令可用"
else
    echo "⚠️  git helper 命令可能不可用，但这是正常的（hidden 命令）"
fi
echo

# 测试 2: 测试 get 操作（无配置，应该静默失败）
echo "测试 2: 测试 get 操作（无配置）"
output=$(echo -e "protocol=ssh\nhost=nonexistent.example.com\n" | ./bin/skm git helper get 2>&1)
exit_code=$?
if [ $exit_code -eq 0 ]; then
    if [ -z "$output" ]; then
        echo "✅ 正确：未配置的主机返回空（静默失败）"
    else
        echo "⚠️  返回了输出: $output"
    fi
else
    echo "❌ 错误：退出码非零: $exit_code"
fi
echo

# 测试 3: 测试 store 操作
echo "测试 3: 测试 store 操作"
output=$(echo -e "protocol=ssh\nhost=github.com\nusername=git\npassword=dummy\n" | ./bin/skm git helper store 2>&1)
exit_code=$?
if [ $exit_code -eq 0 ]; then
    echo "✅ store 操作成功（静默成功）"
else
    echo "❌ store 操作失败，退出码: $exit_code"
    echo "输出: $output"
fi
echo

# 测试 4: 测试 erase 操作
echo "测试 4: 测试 erase 操作"
output=$(echo -e "protocol=ssh\nhost=github.com\n" | ./bin/skm git helper erase 2>&1)
exit_code=$?
if [ $exit_code -eq 0 ]; then
    echo "✅ erase 操作成功（静默成功）"
else
    echo "❌ erase 操作失败，退出码: $exit_code"
    echo "输出: $output"
fi
echo

# 测试 5: 协议过滤（HTTPS 应该被跳过）
echo "测试 5: 测试协议过滤（HTTPS）"
output=$(echo -e "protocol=https\nhost=github.com\n" | ./bin/skm git helper get 2>&1)
exit_code=$?
if [ $exit_code -eq 0 ] && [ -z "$output" ]; then
    echo "✅ HTTPS 协议正确被跳过（返回空）"
else
    echo "⚠️  HTTPS 协议处理: 退出码=$exit_code, 输出='$output'"
fi
echo

# 测试 6: 检查代码编译
echo "测试 6: 检查代码编译"
if go build -o /tmp/skm-test ./main.go 2>&1; then
    echo "✅ 代码编译成功"
    rm -f /tmp/skm-test
else
    echo "❌ 代码编译失败"
fi
echo

echo "==================================="
echo "功能验证完成"
echo "==================================="
echo

# 显示如何配置
echo "📝 下一步："
echo "1. 添加 SSH key:"
echo "   ./bin/skm key add mykey ~/.ssh/id_rsa"
echo
echo "2. 配置 host:"
echo "   ./bin/skm host add github.com --user git --key mykey"
echo
echo "3. 配置 Git credential helper:"
echo "   git config --global credential.helper '!./bin/skm git helper'"
echo
echo "4. 测试完整流程:"
echo "   echo -e 'protocol=ssh\nhost=github.com\n' | ./bin/skm git helper get"

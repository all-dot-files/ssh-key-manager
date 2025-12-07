# SSH Key Manager (SKM)

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)


**SKM** 是一个功能强大的 SSH 密钥管理工具，专为需要管理多个 SSH 密钥、跨设备同步和 Git 集成的开发者设计。

## 🆕 v0.4.0 新功能

- 🎯 **Shell 自动补全** - 支持 Bash/Zsh/Fish/PowerShell
- 🌐 **Web 管理界面** - 现代化的可视化管理
- 🔄 **增量同步** - 60% 减少数据传输
- 📊 **审计日志** - 完整的操作追踪
- ⚡ **性能优化** - 更快的文件 I/O 和并发处理





## ✨ 特性

### 本地密钥管理
- 🔐 **多种密钥类型**：支持 ED25519、RSA、ECDSA
- 🏷️ **组织管理**：使用名称、标签和注释管理密钥
- 🔒 **安全存储**：使用 AES-256-GCM + Argon2 加密私钥
- 📁 **灵活安装**：可选安装到 `~/.ssh` 目录
- 🔄 **密钥轮换**：自动化密钥轮换和过期检查

### SSH 配置自动化
- ⚙️ **自动配置**：自动更新 `~/.ssh/config`
- 🖥️ **主机管理**：配置主机与密钥的关联
- 🔄 **无缝集成**：与现有 SSH 工作流集成

### Git 集成
- 🔗 **仓库绑定**：为每个 Git 仓库配置特定的 SSH 密钥
- 🚀 **自动切换**：自动使用正确的密钥进行 push/pull/fetch
- 🎯 **精确控制**：支持 remote 和 host 级别的配置
- 🎣 **全局 Hook**：自动拦截所有 Git 操作，智能配置新仓库
- ✨ **智能创建**：自动创建缺失的主机和密钥配置

### 跨设备同步 🆕
- ☁️ **中央服务器**：可选的 SKM 服务器用于密钥同步
- 🔐 **安全传输**：HTTPS + JWT 认证
- 🔑 **选择性同步**：公钥默认同步，私钥可选加密同步
- 📱 **设备管理**：注册、撤销和审计设备
- **🆕 增量同步**：只同步变更的密钥（基于校验和）
- **🆕 冲突解决**：多种策略（本地优先、远程优先、最新优先）
- **🆕 同步历史**：完整的同步操作历史记录

### Web 管理界面 🆕
- 🌐 **Web UI**：现代化的 Web 管理界面
- 👤 **用户管理**：注册、登录、会话管理
- 🔑 **密钥管理**：通过浏览器管理 SSH 密钥
- 💻 **设备管理**：查看和撤销已注册设备
- 📊 **审计日志**：完整的操作审计追踪
- 📈 **统计面板**：密钥和设备统计信息

### 命令行增强 🆕
- 🎯 **Shell 补全**：支持 Bash、Zsh、Fish、PowerShell
- 💬 **智能提示**：上下文感知的命令补全
- 🎨 **美化输出**：彩色输出和 Emoji 图标
- ⚠️ **更好的错误**：详细错误信息和修复建议

## 🏗️ 架构

### 客户端 (skm)
```
~/.config/skm/
├── config.yaml          # 主配置文件
└── keys/               # 密钥存储（加密）
    ├── work            # 私钥（加密）
    ├── work.pub        # 公钥
    ├── personal
    └── personal.pub
```

### 服务端 (skm-server)
- REST API (JWT 认证)
- 文件存储后端（可扩展为数据库）
- 审计日志
- 设备管理

## 📦 安装

### 从源码构建

```bash
# 克隆仓库
git clone https://github.com/all-dot-files/ssh-key-manager.git
cd ssh-key-manager

# 构建客户端
go build -o skm ./cmd/skm/main.go

# 构建服务端
go build -o skm-server ./cmd/skm-server/main.go

# 安装到系统
sudo mv skm /usr/local/bin/
sudo mv skm-server /usr/local/bin/
```

### 使用 Go Install

```bash
go install github.com/all-dot-files/ssh-key-manager/cmd/skm@latest
go install github.com/all-dot-files/ssh-key-manager/cmd/skm-server@latest
```

## 🚀 快速开始

### 1. 初始化 SKM

```bash
# 初始化配置
skm init --device-name "MacBook Pro"
```

### 2. 生成 SSH 密钥

```bash
# 生成 ED25519 密钥（推荐）
skm key gen --name work --type ed25519

# 生成带密码保护的 RSA 密钥
skm key gen --name personal --type rsa --rsa-bits 4096 --passphrase

# 查看所有密钥
skm key list

# 查看密钥详情
skm key show work
```

### 3. 配置 SSH 主机

```bash
# 添加主机配置
skm host add github.com --user git --key work

# 自动创建密钥（如果不存在）
skm host add gitlab.com --user git --auto-create-key

# 添加自定义主机
skm host add myserver --user ubuntu --key personal --hostname 192.168.1.100 --port 2222

# 列出所有主机
skm host list
```

### 4. Git 仓库集成

```bash
# 绑定当前仓库
cd /path/to/your/repo
skm git bind . --host github.com

# 自动创建主机和密钥（如果不存在）
skm git bind . --host gitlab.com --auto-create

# 安装全局 Git Hook（自动配置所有新仓库）
skm git hook install

# 卸载全局 Hook
skm git hook uninstall

# 使用 SKM 执行 Git 命令
skm git exec . -- pull
skm git exec . -- push origin main

# 列出所有绑定的仓库
skm git list
```

### 5. 跨设备同步（可选）

```bash
# 登录到 SKM 服务器
skm server-login --server https://skm.example.com --user alice

# 注册设备
skm device-register --name "MacBook Pro"

# 推送公钥到服务器
skm sync push

# 在另一台设备上拉取密钥
skm sync pull
```

## 📚 命令参考

### 密钥管理

```bash
# 生成密钥
skm key gen --name <name> --type <ed25519|rsa|ecdsa> [--passphrase] [--rsa-bits N]

# 列出密钥
skm key list

# 显示密钥详情
skm key show <name> [--show-public]

# 安装密钥到 ~/.ssh
skm key install <name>

# 导出公钥
skm key export <name> --output <file>

# 删除密钥
skm key delete <name>
```

### 主机管理

```bash
# 添加主机
skm host add <hostname> --user <user> --key <keyname> [--port N] [--hostname <actual-host>]

# 自动创建密钥（如果不存在）
skm host add <hostname> --user <user> --auto-create-key

# 列出主机
skm host list

# 删除主机
skm host remove <hostname>
```

### Git 集成

```bash
# 绑定仓库
skm git bind <repo-path> --host <hostname> [--remote origin] [--user <user>] [--key <keyname>]

# 自动创建主机和密钥（如果不存在）
skm git bind <repo-path> --host <hostname> --auto-create

# 安装全局 Git Hook（自动配置所有新仓库）
skm git hook install

# 卸载全局 Hook
skm git hook uninstall

# 列出仓库
skm git list

# 执行 Git 命令
skm git exec <repo-path> -- <git-command>
```

### 同步管理 🆕

```bash
# 查看同步状态
skm sync status

# 推送密钥到服务器
skm sync push [--include-private]

# 从服务器拉取密钥
skm sync pull

# 查看同步历史
skm sync history [--limit N]

# 解决同步冲突
skm sync resolve --strategy <local|remote|newer>

# 清除同步历史
skm sync clear-history
```

### Shell 补全 🆕

```bash
# 生成补全脚本
skm completion <bash|zsh|fish|powershell>

# Zsh 安装示例
skm completion zsh > "${fpath[1]}/_skm"

# Bash 安装示例 (macOS)
skm completion bash > $(brew --prefix)/etc/bash_completion.d/skm

# Fish 安装示例
skm completion fish > ~/.config/fish/completions/skm.fish
```

### 同步

```bash
# 服务器登录
skm server-login --server <url> --user <username> [--password <pass>]

# 注册设备
skm device-register [--name <name>]

# 推送密钥
skm sync push [--include-private]

# 拉取密钥
skm sync pull [--include-private]
```

### 备份和恢复 🆕

```bash
# 创建备份
skm backup create [--output <file>]

# 列出备份
skm backup list

# 恢复备份
skm backup restore <backup-file>
```

## 🔒 安全设计

### 本地安全
- **私钥加密**：使用 AES-256-GCM + Argon2 KDF
- **安全存储**：文件权限 0600，目录权限 0700
- **密码保护**：可选密码保护每个私钥

### 传输安全
- **HTTPS Only**：所有网络通信使用 HTTPS
- **JWT 认证**：基于令牌的身份验证
- **加密私钥**：私钥在上传前已加密，服务器只存储密文

### 同步策略
- **默认行为**：只同步公钥
- **私钥可选**：私钥同步必须显式启用
- **端到端加密**：私钥使用客户端密码加密后再上传
- **设备隔离**：可为不同设备使用不同的加密密钥

### 审计
- **操作日志**：记录所有密钥操作
- **设备追踪**：跟踪哪些设备访问了哪些密钥
- **撤销机制**：可撤销已注册的设备

## 🖥️ 服务器部署

### 运行服务器

```bash
# 生成 JWT 密钥
JWT_SECRET=$(openssl rand -base64 32)

# 启动服务器
skm-server --addr :8080 --data ./skm-data --jwt-secret "$JWT_SECRET"
```

### Docker 部署

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -o skm-server ./cmd/skm-server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /build/skm-server /usr/local/bin/
EXPOSE 8080
ENTRYPOINT ["skm-server"]
CMD ["--addr", ":8080", "--data", "/data"]
```

```bash
# 构建镜像
docker build -t skm-server .

# 运行容器
docker run -d \
  -p 8080:8080 \
  -v skm-data:/data \
  -e JWT_SECRET="your-secret-here" \
  skm-server --jwt-secret "$JWT_SECRET"
```

## 🏛️ 项目结构

```
ssh-key-manager/
├── cmd/
│   ├── skm/              # CLI 入口
│   │   └── main.go
│   └── skm-server/       # 服务器入口
│       └── main.go
├── internal/
│   ├── cli/              # CLI 逻辑实现
│   │   ├── root.go
│   │   ├── key.go
│   │   └── ...
│   ├── models/           # 数据模型
│   ├── config/           # 配置管理
│   ├── keystore/         # 密钥存储
│   ├── sshconfig/        # SSH 配置管理
│   ├── git/              # Git 集成
│   ├── api/              # API 客户端
│   ├── backup/           # 备份逻辑
│   ├── server/           # 服务器实现
│   └── storage/          # 存储层 (YAML/SQLite)
├── pkg/                  # 公共库 (可复用)
│   ├── crypto/           # 加密功能
│   ├── fileio/           # 文件 I/O
│   ├── logger/           # 日志
│   ├── concurrency/      # 并发工具
│   ├── platform/         # 平台检测
│   └── errors/           # 错误定义
├── configs/              # 配置文件示例
├── test/                 # 测试
└── README.md
```

## 🛠️ 开发

### 运行测试

```bash
go test ./...
```

### 代码检查

```bash
go vet ./...
golangci-lint run
```

### 贡献

欢迎贡献！请遵循以下步骤：

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启 Pull Request

## 📄 配置文件示例

### config.yaml

```yaml
device_id: "550e8400-e29b-41d4-a716-446655440000"
device_name: "MacBook Pro"
user: "alice@example.com"
server: "https://skm.example.com"
keystore_path: "/Users/alice/.config/skm/keys"
ssh_dir: "/Users/alice/.ssh"
default_key_policy: "ask"

sync_policy:
  sync_public_keys: true
  sync_private_keys: false
  require_encryption: true

keys:
  - name: work
    type: ed25519
    path: "/Users/alice/.config/skm/keys/work"
    pub_path: "/Users/alice/.config/skm/keys/work.pub"
    tags: ["work", "github"]
    created_at: "2025-10-20T12:00:00Z"
    updated_at: "2025-10-20T12:00:00Z"
    fingerprint: "SHA256:..."
    installed: false
    has_passphrase: true

hosts:
  - host: github.com
    user: git
    key: work
    port: 0

repos:
  - path: "/Users/alice/projects/myapp"
    remote: origin
    host: github.com
```

## 🤝 致谢

- [Cobra](https://github.com/spf13/cobra) - CLI 框架
- [Viper](https://github.com/spf13/viper) - 配置管理
- [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) - 加密库
- [Age](https://age-encryption.org/) - 现代加密工具

## 📝 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## ⚠️ 注意事项

1. **私钥安全**：默认情况下，私钥不会上传到服务器。只有在明确要求且加密的情况下才会同步。
2. **备份**：请定期备份你的 `~/.config/skm` 目录。
3. **密码强度**：使用强密码保护你的私钥。
4. **设备撤销**：撤销设备不会自动删除已同步到其他设备的私钥。
5. **审计日志**：定期检查审计日志以监控异常活动。

## 🔮 未来计划

- [ ] 支持 macOS Keychain / Windows Credential Manager
- [ ] SSH Agent 集成
- [ ] 密钥轮换策略
- [ ] Web UI 管理界面
- [ ] 数据库后端支持（PostgreSQL, MySQL）
- [ ] 多因素认证 (MFA)
- [ ] 密钥使用统计和分析
- [ ] 团队密钥共享功能
- [ ] Kubernetes Secrets 集成
- [ ] Ansible Vault 集成

## 📧 联系方式

- 项目主页: https://github.com/all-dot-files/ssh-key-manager
- 问题反馈: https://github.com/all-dot-files/ssh-key-manager/issues

---

**Made with ❤️ for developers who manage multiple SSH keys**

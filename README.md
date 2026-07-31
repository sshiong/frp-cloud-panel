# FRP Cloud Panel

FRP 多用户云隧道管理平台（双面板架构增强版）

## 项目概述

这是一个基于 FRP (Fast Reverse Proxy) 的多用户云隧道管理平台，采用双面板架构：

- **Server Panel**: 部署在 FRPS 服务器，负责全局控制、安全、资源仲裁
- **Client Panel**: 用户侧本地服务，负责 Web 面板、本地 FRPC 管理和状态采集

## 技术栈

### 后端
- Go
- Gin (Web框架)
- GORM (ORM)

### 数据库
- SQLite (WAL模式)

### 前端
- Vue3
- TypeScript
- Element Plus

### 通信
- HTTPS REST API
- WebSocket

## 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/sshiong/frp-cloud-panel.git
cd frp-cloud-panel
```

### 2. 启动服务端

```bash
cd server
go mod tidy
go build -o bin/server ./cmd/server
./bin/server
```

服务端将在 `http://localhost:8080` 启动。

### 3. 启动 Server Panel（管理员面板）

```bash
cd server-panel
npm install
npm run dev
```

Server Panel 将在 `http://localhost:3000` 启动。

### 4. 启动 Client Panel（用户面板）

```bash
cd client-panel
npm install
npm run dev
```

Client Panel 将在 `http://localhost:3001` 启动。

### 5. 访问系统

- **Server Panel（管理员）**: http://localhost:3000
- **Client Panel（用户）**: http://localhost:3001

默认管理员账号：
- 用户名: `admin`
- 密码: `password`

## 功能特性

### Server Panel
- 管理员 Web 面板
- 用户权限管理
- FRP 用户凭证管理
- 全局端口管理
- 域名资源管理
- Cloudflare Token 加密保存
- DNS 管理
- ACME 证书申请和续期
- HTTP/HTTPS Router
- 审计日志

### Client Panel
- 用户 Web 面板
- FRPC 配置管理
- FRPC 进程管理
- 本地服务检测
- 状态采集
- 日志管理

## API 文档

### 认证相关

#### 用户登录
```bash
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password"
}
```

#### 用户注册
```bash
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "testuser",
  "password": "testpass123",
  "email": "test@example.com"
}
```

### 映射管理

#### 获取映射列表
```bash
GET /api/v1/mappings
Authorization: Bearer <token>
```

#### 创建映射
```bash
POST /api/v1/mappings
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "my-service",
  "type": "tcp",
  "local_ip": "127.0.0.1",
  "local_port": 8080,
  "remote_port": 0
}
```

### 客户端管理

#### 客户端注册
```bash
POST /api/v1/client/register
X-Client-ID: <client-id>
X-Device-Token: <device-token>
Content-Type: application/json

{
  "username": "admin",
  "password": "password",
  "device_name": "My Device"
}
```

#### 心跳
```bash
POST /api/v1/client/heartbeat
X-Client-ID: <client-id>
X-Device-Token: <device-token>
```

## 项目结构

```
.
├── server/          # Server Panel 服务端
│   ├── cmd/         # 入口文件
│   ├── internal/    # 内部包
│   ├── pkg/         # 公共包
│   └── web/         # 前端资源
├── client/          # Client Panel 客户端
│   ├── cmd/         # 入口文件
│   ├── internal/    # 内部包
│   └── web/         # 前端资源
├── frontend/        # 前端源码
│   ├── src/         # 源代码
│   └── public/      # 静态资源
└── shared/          # 共享类型和工具
    ├── types/       # 共享类型定义
    └── utils/       # 共享工具函数
```

## 开发进度

请查看 [PROGRESS.md](PROGRESS.md) 了解开发进度。

## 配置说明

### 服务端配置

服务端配置文件 `config.json`:

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080
  },
  "database": {
    "path": "./data/frp_panel.db"
  },
  "jwt": {
    "secret": "your-secret-key-change-in-production",
    "expiration": 24
  },
  "log": {
    "level": "info",
    "file": "./logs/server.log"
  }
}
```

### 客户端配置

客户端配置文件 `config.json`:

```json
{
  "server": {
    "address": "http://localhost:8080"
  },
  "frpc": {
    "path": "frpc",
    "config_path": "./data/frpc.toml",
    "log_path": "./logs/frpc.log"
  },
  "log": {
    "level": "info",
    "file": "./logs/client.log"
  },
  "device": {
    "client_id": "",
    "device_token": "",
    "device_name": ""
  }
}
```

## 部署说明

### 生产环境部署

1. 修改 JWT 密钥
2. 配置 HTTPS
3. 设置数据库路径
4. 配置日志级别
5. 设置防火墙规则

### Docker 部署

```bash
# 构建镜像
docker build -t frp-cloud-panel .

# 运行容器
docker run -d -p 8080:8080 frp-cloud-panel
```

## 常见问题

### 1. 数据库初始化失败
检查目录权限，确保程序有写入权限。

### 2. 前端无法访问后端 API
检查后端服务是否启动，端口是否正确。

### 3. 客户端无法连接服务端
检查网络连接，防火墙设置。

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

MIT License

## 联系方式

- GitHub: https://github.com/sshiong/frp-cloud-panel
- Issues: https://github.com/sshiong/frp-cloud-panel/issues

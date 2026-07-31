# FRP Cloud Panel

FRP 多用户云隧道管理平台（双面板架构增强版）

## 架构

系统采用两个独立部署组件：

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
└── shared/          # 共享类型和工具
    ├── types/       # 共享类型定义
    └── utils/       # 共享工具函数
```

## 快速开始

### Server Panel

```bash
cd server
go run cmd/server/main.go
```

### Client Panel

```bash
cd client
go run cmd/client/main.go
```

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

## 开发进度

请查看 [PROGRESS.md](PROGRESS.md) 了解开发进度。

## 许可证

MIT License

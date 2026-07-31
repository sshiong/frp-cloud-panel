# FRP Cloud Panel 部署指南

## 目录

1. [环境要求](#环境要求)
2. [快速部署](#快速部署)
3. [Docker 部署](#docker-部署)
4. [Kubernetes 部署](#kubernetes-部署)
5. [生产环境配置](#生产环境配置)
6. [安全配置](#安全配置)
7. [监控配置](#监控配置)
8. [备份策略](#备份策略)
9. [故障排查](#故障排查)
10. [性能优化](#性能优化)

---

## 环境要求

### 系统要求
- **操作系统**: Linux (推荐 Ubuntu 20.04+, CentOS 7+)
- **CPU**: 2核以上
- **内存**: 4GB以上
- **磁盘**: 50GB以上
- **网络**: 公网 IP，开放 80、443、7000、8080 端口

### 软件要求
- **Go**: 1.25+
- **Node.js**: 18+
- **npm**: 9+
- **Git**: 2.30+

---

## 快速部署

### 1. 克隆项目

```bash
git clone https://github.com/sshiong/frp-cloud-panel.git
cd frp-cloud-panel
```

### 2. 部署服务端

```bash
cd server

# 安装依赖
go mod tidy

# 编译
go build -o bin/server ./cmd/server

# 创建必要目录
mkdir -p data logs data/certs data/backups

# 启动服务
./bin/server
```

### 3. 部署前端

```bash
cd frontend

# 安装依赖
npm install

# 构建生产版本
npm run build

# 部署到 Nginx 或其他 Web 服务器
cp -r dist/* /var/www/html/
```

### 4. 配置 Nginx

```nginx
server {
    listen 80;
    server_name your-domain.com;

    # 前端静态文件
    location / {
        root /var/www/html;
        try_files $uri $uri/ /index.html;
    }

    # API 代理
    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket 代理
    location /api/v1/ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
    }
}
```

---

## Docker 部署

### 1. 创建 Dockerfile

```dockerfile
# 服务端 Dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY server/ .
RUN go mod tidy && go build -o bin/server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/bin/server .
COPY --from=builder /app/config.json .
RUN mkdir -p data logs data/certs data/backups

EXPOSE 8080
CMD ["./server"]
```

### 2. 创建 docker-compose.yml

```yaml
version: '3.8'

services:
  server:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
      - ./logs:/app/logs
    environment:
      - TZ=Asia/Shanghai
    restart: unless-stopped

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/conf.d/default.conf
      - ./frontend/dist:/usr/share/nginx/html
      - ./data/certs:/etc/nginx/certs
    depends_on:
      - server
    restart: unless-stopped
```

### 3. 启动服务

```bash
# 构建并启动
docker-compose up -d

# 查看日志
docker-compose logs -f server

# 停止服务
docker-compose down
```

---

## Kubernetes 部署

### 1. 创建 Namespace

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: frp-cloud-panel
```

### 2. 创建 ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: server-config
  namespace: frp-cloud-panel
data:
  config.json: |
    {
      "server": {
        "host": "0.0.0.0",
        "port": 8080
      },
      "database": {
        "path": "/app/data/frp_panel.db"
      },
      "jwt": {
        "secret": "your-secret-key",
        "expiration": 24
      },
      "log": {
        "level": "info",
        "file": "/app/logs/server.log"
      }
    }
```

### 3. 创建 Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: server
  namespace: frp-cloud-panel
spec:
  replicas: 2
  selector:
    matchLabels:
      app: server
  template:
    metadata:
      labels:
        app: server
    spec:
      containers:
      - name: server
        image: frp-cloud-panel/server:latest
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: config
          mountPath: /app/config.json
          subPath: config.json
        - name: data
          mountPath: /app/data
        - name: logs
          mountPath: /app/logs
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: server-config
      - name: data
        persistentVolumeClaim:
          claimName: server-data
      - name: logs
        persistentVolumeClaim:
          claimName: server-logs
```

### 4. 创建 Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: server
  namespace: frp-cloud-panel
spec:
  selector:
    app: server
  ports:
  - port: 8080
    targetPort: 8080
  type: ClusterIP
```

### 5. 创建 Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: server-ingress
  namespace: frp-cloud-panel
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - your-domain.com
    secretName: your-domain-tls
  rules:
  - host: your-domain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: server
            port:
              number: 8080
```

---

## 生产环境配置

### 1. 环境变量

```bash
# 服务端配置
export SERVER_HOST=0.0.0.0
export SERVER_PORT=8080
export DATABASE_PATH=/app/data/frp_panel.db
export JWT_SECRET=your-very-long-and-secure-secret-key
export JWT_EXPIRATION=24
export LOG_LEVEL=info
export LOG_FILE=/app/logs/server.log
```

### 2. 配置文件

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080
  },
  "database": {
    "path": "/app/data/frp_panel.db"
  },
  "jwt": {
    "secret": "your-very-long-and-secure-secret-key-at-least-32-chars",
    "expiration": 24
  },
  "log": {
    "level": "info",
    "file": "/app/logs/server.log"
  }
}
```

### 3. systemd 服务

```ini
[Unit]
Description=FRP Cloud Panel Server
After=network.target

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/frp-cloud-panel/server
ExecStart=/opt/frp-cloud-panel/server/bin/server
Restart=always
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

---

## 安全配置

### 1. HTTPS 配置

```bash
# 安装 Certbot
sudo apt install certbot python3-certbot-nginx

# 获取证书
sudo certbot --nginx -d your-domain.com

# 自动续期
sudo crontab -e
0 0 1 * * /usr/bin/certbot renew --quiet
```

### 2. 防火墙配置

```bash
# UFW 配置
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 8080/tcp
sudo ufw allow 7000/tcp
sudo ufw enable
```

### 3. SSL 配置

```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;
    ssl_prefer_server_ciphers off;

    # 其他配置...
}
```

---

## 监控配置

### 1. Prometheus 配置

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'frp-cloud-panel'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

### 2. Grafana 仪表盘

导入提供的 JSON 仪表盘文件，监控以下指标：
- 系统资源使用率
- API 请求统计
- 错误率统计
- 连接数统计

### 3. 告警规则

```yaml
groups:
  - name: frp-cloud-panel
    rules:
      - alert: HighCPUUsage
        expr: cpu_usage > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "CPU 使用率过高"
          description: "CPU 使用率超过 80% 已持续 5 分钟"

      - alert: HighMemoryUsage
        expr: memory_usage > 85
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "内存使用率过高"
          description: "内存使用率超过 85% 已持续 5 分钟"
```

---

## 备份策略

### 1. 自动备份脚本

```bash
#!/bin/bash
# backup.sh

BACKUP_DIR="/opt/backups/frp-cloud-panel"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/backup_$DATE.tar.gz"

# 创建备份目录
mkdir -p $BACKUP_DIR

# 停止服务
systemctl stop frp-cloud-panel

# 备份数据
tar -czf $BACKUP_FILE -C /opt/frp-cloud-panel data logs

# 启动服务
systemctl start frp-cloud-panel

# 删除7天前的备份
find $BACKUP_DIR -name "backup_*.tar.gz" -mtime +7 -delete

echo "Backup completed: $BACKUP_FILE"
```

### 2. 定时任务

```bash
# 每天凌晨2点备份
0 2 * * * /opt/scripts/backup.sh >> /var/log/backup.log 2>&1
```

### 3. 远程备份

```bash
# 同步到远程服务器
rsync -avz /opt/backups/frp-cloud-panel/ user@remote:/backups/frp-cloud-panel/

# 或上传到 S3
aws s3 sync /opt/backups/frp-cloud-panel/ s3://your-bucket/backups/
```

---

## 故障排查

### 1. 常见问题

#### 服务无法启动
```bash
# 检查日志
tail -f /opt/frp-cloud-panel/logs/server.log

# 检查端口占用
netstat -tlnp | grep 8080

# 检查配置文件
cat /opt/frp-cloud-panel/config.json
```

#### 数据库错误
```bash
# 检查数据库文件权限
ls -la /opt/frp-cloud-panel/data/

# 检查磁盘空间
df -h

# 检查数据库完整性
sqlite3 /opt/frp-cloud-panel/data/frp_panel.db "PRAGMA integrity_check;"
```

#### 性能问题
```bash
# 检查系统资源
top
htop
iostat

# 检查网络连接
netstat -an | grep :8080
ss -tlnp | grep :8080
```

### 2. 日志分析

```bash
# 查看错误日志
grep -i "error" /opt/frp-cloud-panel/logs/server.log

# 查看访问日志
tail -f /opt/frp-cloud-panel/logs/server.log | grep "200\|404\|500"

# 统计请求量
awk '{print $1}' /opt/frp-cloud-panel/logs/server.log | sort | uniq -c | sort -nr
```

### 3. 性能调试

```bash
# 启用 pprof
curl http://localhost:8080/debug/pprof/ > profile.out
go tool pprof profile.out

# 查看 goroutine
curl http://localhost:8080/debug/pprof/goroutine?debug=1

# 查看内存使用
curl http://localhost:8080/debug/pprof/heap?debug=1
```

---

## 性能优化

### 1. 数据库优化

```sql
-- 创建索引
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_clients_client_id ON clients(client_id);
CREATE INDEX idx_mappings_user_id ON proxy_mappings(user_id);
CREATE INDEX idx_mappings_status ON proxy_mappings(status);

-- 分析查询
EXPLAIN QUERY PLAN SELECT * FROM users WHERE username = 'admin';
```

### 2. 缓存配置

```go
// Redis 缓存
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
    DB:   0,
})

// 缓存用户信息
func GetUserByID(id uint) (*User, error) {
    key := fmt.Sprintf("user:%d", id)
    
    // 尝试从缓存获取
    if cached, err := redisClient.Get(key).Result(); err == nil {
        var user User
        json.Unmarshal([]byte(cached), &user)
        return &user, nil
    }
    
    // 从数据库获取
    var user User
    if err := db.First(&user, id).Error; err != nil {
        return nil, err
    }
    
    // 写入缓存
    data, _ := json.Marshal(user)
    redisClient.Set(key, data, 10*time.Minute)
    
    return &user, nil
}
```

### 3. 连接池配置

```go
// 数据库连接池
sqlDB, _ := db.DB()
sqlDB.SetMaxOpenConns(100)
sqlDB.SetMaxIdleConns(10)
sqlDB.SetConnMaxLifetime(time.Hour)

// HTTP 客户端连接池
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,
}
client := &http.Client{Transport: transport}
```

### 4. 负载均衡

```nginx
upstream backend {
    least_conn;
    server backend1.example.com:8080 weight=3;
    server backend2.example.com:8080 weight=2;
    server backend3.example.com:8080 weight=1;
}

server {
    location /api/ {
        proxy_pass http://backend;
    }
}
```

---

## 运维手册

### 1. 日常运维

```bash
# 检查服务状态
systemctl status frp-cloud-panel

# 重启服务
systemctl restart frp-cloud-panel

# 查看日志
journalctl -u frp-cloud-panel -f

# 检查磁盘空间
df -h

# 清理日志
find /opt/frp-cloud-panel/logs -name "*.log" -mtime +30 -delete
```

### 2. 版本升级

```bash
# 备份当前版本
cp -r /opt/frp-cloud-panel /opt/frp-cloud-panel.backup

# 拉取最新代码
cd /opt/frp-cloud-panel
git pull origin main

# 重新编译
cd server
go build -o bin/server ./cmd/server

# 重启服务
systemctl restart frp-cloud-panel

# 检查服务状态
systemctl status frp-cloud-panel
```

### 3. 数据迁移

```bash
# 导出数据
sqlite3 /opt/frp-cloud-panel/data/frp_panel.db ".dump" > dump.sql

# 导入数据
sqlite3 new_database.db < dump.sql

# 备份数据库
cp /opt/frp-cloud-panel/data/frp_panel.db /opt/backups/
```

---

## 总结

本部署指南涵盖了 FRP Cloud Panel 的各种部署方式，包括：

- 快速部署
- Docker 部署
- Kubernetes 部署
- 生产环境配置
- 安全配置
- 监控配置
- 备份策略
- 故障排查
- 性能优化

根据实际需求选择合适的部署方式，并按照最佳实践进行配置，可以确保系统的稳定性、安全性和高性能。

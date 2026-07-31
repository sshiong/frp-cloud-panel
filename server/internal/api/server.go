package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/frp-cloud-panel/server/internal/config"
	"github.com/frp-cloud-panel/server/internal/middleware"
	"github.com/frp-cloud-panel/server/internal/services"
	"github.com/frp-cloud-panel/server/internal/websocket"
	"github.com/gin-gonic/gin"
)

// Server API 服务器
type Server struct {
	cfg           *config.Config
	router        *gin.Engine
	http          *http.Server
	cfService     *services.CloudflareService
	wsHub         *websocket.Hub
	acmeService   *services.ACMEService
	configService *services.ConfigService
	backupService *services.BackupService
}

// NewServer 创建新的 API 服务器
func NewServer(cfg *config.Config) *Server {
	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// 添加中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	wsHub := websocket.NewHub()
	go wsHub.Run()

	acmeService := services.NewACMEService(cfg)
	configService := services.NewConfigService(wsHub)
	backupService := services.NewBackupService(cfg.JWT.Secret)

	server := &Server{
		cfg:           cfg,
		router:        router,
		cfService:     services.NewCloudflareService(cfg.JWT.Secret),
		wsHub:         wsHub,
		acmeService:   acmeService,
		configService: configService,
		backupService: backupService,
	}

	// 注册路由
	server.registerRoutes()

	return server
}

// registerRoutes 注册路由
func (s *Server) registerRoutes() {
	// 健康检查
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// API v1
	v1 := s.router.Group("/api/v1")
	{
		// 认证相关
		auth := v1.Group("/auth")
		{
			auth.POST("/login", s.handleLogin)
			auth.POST("/register", s.handleRegister)
		}

		// 需要认证的路由
		protected := v1.Group("/")
		protected.Use(middleware.AuthMiddleware(&s.cfg.JWT))
		{
			// 用户相关
			users := protected.Group("/users")
			{
				users.GET("/me", s.handleGetMe)
				users.PUT("/me", s.handleUpdateMe)
				users.PUT("/me/password", s.handleUpdatePassword)
			}

			// 客户端管理
			clients := protected.Group("/clients")
			{
				clients.GET("", s.handleListClients)
				clients.GET("/:id", s.handleGetClient)
				clients.PUT("/:id", s.handleUpdateClient)
				clients.DELETE("/:id", s.handleDeleteClient)
			}

			// 代理映射
			mappings := protected.Group("/mappings")
			{
				mappings.GET("", s.handleListMappings)
				mappings.POST("", s.handleCreateMapping)
				mappings.GET("/:id", s.handleGetMapping)
				mappings.PUT("/:id", s.handleUpdateMapping)
				mappings.DELETE("/:id", s.handleDeleteMapping)
			}

			// 域名管理
			domains := protected.Group("/domains")
			{
				domains.GET("", s.handleListDomains)
				domains.POST("", s.handleCreateDomain)
				domains.GET("/:id", s.handleGetDomain)
				domains.PUT("/:id", s.handleUpdateDomain)
				domains.DELETE("/:id", s.handleDeleteDomain)
			}

			// Cloudflare Token
			cf := protected.Group("/cloudflare")
			{
				cf.POST("/token", s.handleSetCFToken)
				cf.GET("/token/status", s.handleGetCFTokenStatus)
				cf.DELETE("/token", s.handleDeleteCFToken)
				cf.POST("/token/test", s.handleTestCFToken)
			}

			// DNS 管理
			dns := protected.Group("/dns")
			{
				dns.GET("/records", s.handleListDNSRecords)
				dns.POST("/records", s.handleCreateDNSRecord)
				dns.PUT("/records/:domain", s.handleUpdateDNSRecord)
				dns.DELETE("/records/:domain", s.handleDeleteDNSRecord)
			}

			// 日志
			logs := protected.Group("/logs")
			{
				logs.GET("", s.handleListLogs)
			}

			// WebSocket
			ws := protected.Group("/ws")
			{
				ws.GET("", s.handleWebSocket)
			}

			// 证书管理
			certs := protected.Group("/certs")
			{
				certs.GET("/:domain", s.handleGetCert)
				certs.POST("/:domain/renew", s.handleRenewCert)
				certs.GET("/check", s.handleCheckCerts)
			}

			// 配置管理
			config := protected.Group("/config")
			{
				config.GET("/version/:client_id", s.handleGetConfigVersion)
				config.GET("/desired/:client_id", s.handleGetDesiredConfig)
				config.POST("/apply/:client_id", s.handleApplyConfig)
				config.GET("/sync/:client_id", s.handleCheckConfigSync)
				config.POST("/sync/:client_id", s.handleSyncConfig)
				config.GET("/export/:client_id", s.handleExportConfig)
				config.POST("/import/:client_id", s.handleImportConfig)
				config.GET("/frpc/:client_id", s.handleGenerateFRPCConfig)
			}

			// 备份管理
			backup := protected.Group("/backup")
			{
				backup.POST("/create", s.handleCreateBackup)
				backup.POST("/restore", s.handleRestoreBackup)
				backup.GET("/list", s.handleListBackups)
				backup.DELETE("/:filename", s.handleDeleteBackup)
			}
		}

		// 客户端 API (使用 device_token 认证)
		client := v1.Group("/client")
		client.Use(middleware.DeviceAuthMiddleware())
		{
			client.POST("/register", s.handleClientRegister)
			client.GET("/config", s.handleGetClientConfig)
			client.POST("/config/apply", s.handleApplyConfig)
			client.POST("/status", s.handleUpdateStatus)
			client.POST("/heartbeat", s.handleHeartbeat)
		}
	}
}

// Start 启动服务器
func (s *Server) Start(addr string) error {
	s.http = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	return s.http.ListenAndServe()
}

// Shutdown 关闭服务器
func (s *Server) Shutdown() error {
	if s.http != nil {
		return s.http.Shutdown(context.Background())
	}
	return nil
}

// corsMiddleware CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Device-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// response 统一响应结构
type response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// success 成功响应
func success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// errorResp 错误响应
func errorResp(c *gin.Context, httpCode int, code int, message string) {
	c.JSON(httpCode, response{
		Code:    code,
		Message: message,
	})
}

// badRequest 400 错误
func badRequest(c *gin.Context, message string) {
	errorResp(c, http.StatusBadRequest, 400, message)
}

// unauthorized 401 错误
func unauthorized(c *gin.Context, message string) {
	errorResp(c, http.StatusUnauthorized, 401, message)
}

// forbidden 403 错误
func forbidden(c *gin.Context, message string) {
	errorResp(c, http.StatusForbidden, 403, message)
}

// notFound 404 错误
func notFound(c *gin.Context, message string) {
	errorResp(c, http.StatusNotFound, 404, message)
}

// serverError 500 错误
func serverError(c *gin.Context, err error) {
	errorResp(c, http.StatusInternalServerError, 500, fmt.Sprintf("Internal server error: %v", err))
}

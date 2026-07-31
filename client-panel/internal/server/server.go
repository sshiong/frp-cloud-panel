package server

import (
	"context"
	"net/http"

	"github.com/frp-cloud-panel/client-panel/internal/api"
	"github.com/frp-cloud-panel/client-panel/internal/config"
	"github.com/frp-cloud-panel/client-panel/internal/frpc"
	"github.com/gin-gonic/gin"
)

// Server 本地 API 服务器
type Server struct {
	cfg        *config.Config
	router     *gin.Engine
	http       *http.Server
	apiClient  *api.Client
	frpcMgr    *frpc.Manager
}

// NewServer 创建新的本地服务器
func NewServer(cfg *config.Config, apiClient *api.Client, frpcMgr *frpc.Manager) *Server {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	server := &Server{
		cfg:       cfg,
		router:    router,
		apiClient: apiClient,
		frpcMgr:   frpcMgr,
	}

	server.registerRoutes()
	return server
}

// registerRoutes 注册路由
func (s *Server) registerRoutes() {
	// 健康检查
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"type":   "client-panel",
		})
	})

	// 本地 API
	local := s.router.Group("/api/local")
	{
		// 设备注册
		local.POST("/register", s.handleRegister)

		// FRPC 管理
		local.GET("/frpc/status", s.handleFRPCStatus)
		local.POST("/frpc/start", s.handleFRPCStart)
		local.POST("/frpc/stop", s.handleFRPCStop)
		local.POST("/frpc/restart", s.handleFRPCRestart)

		// 配置管理
		local.GET("/config", s.handleGetConfig)
		local.POST("/config/sync", s.handleSyncConfig)
		local.GET("/config/frpc", s.handleGetFRPCConfig)

		// 状态查询
		local.GET("/status", s.handleGetStatus)
		local.POST("/heartbeat", s.handleHeartbeat)
	}

	// 静态文件服务（前端）
	s.router.StaticFS("/static", http.Dir("./web/static"))
	s.router.NoRoute(func(c *gin.Context) {
		// 对于非 API 请求，返回前端页面
		c.File("./web/static/index.html")
	})
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
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

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

// serverError 500 错误
func serverError(c *gin.Context, err error) {
	errorResp(c, http.StatusInternalServerError, 500, err.Error())
}

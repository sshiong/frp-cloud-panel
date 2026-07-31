package server

import (
	"log"
	"time"

	"github.com/frp-cloud-panel/client-panel/internal/config"
	"github.com/frp-cloud-panel/client-panel/internal/frpc"
	"github.com/gin-gonic/gin"
)

// RegisterRequest 注册请求
type RegisterRequest struct {
	ServerAddr string `json:"server_addr" binding:"required"`
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	DeviceName string `json:"device_name" binding:"required"`
}

// handleRegister 处理设备注册
func (s *Server) handleRegister(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 更新服务端地址
	s.cfg.Server.Address = req.ServerAddr

	// 注册到 Server Panel
	resp, err := s.apiClient.Register(req.Username, req.Password, req.DeviceName)
	if err != nil {
		serverError(c, err)
		return
	}

	// 保存设备信息
	s.cfg.Device.ClientID = resp.ClientID
	s.cfg.Device.DeviceToken = resp.DeviceToken
	s.cfg.Device.DeviceName = req.DeviceName

	// 保存配置
	if err := config.SaveConfig(s.cfg, "config.json"); err != nil {
		log.Printf("Failed to save config: %v", err)
	}

	success(c, gin.H{
		"client_id":    resp.ClientID,
		"device_token": resp.DeviceToken,
		"message":      "Device registered successfully",
	})
}

// handleFRPCStatus 获取 FRPC 状态
func (s *Server) handleFRPCStatus(c *gin.Context) {
	status := s.frpcMgr.GetStatus()
	success(c, gin.H{
		"running": status.Running,
		"error":   status.Error,
	})
}

// handleFRPCStart 启动 FRPC
func (s *Server) handleFRPCStart(c *gin.Context) {
	if err := s.frpcMgr.Start(); err != nil {
		serverError(c, err)
		return
	}

	success(c, gin.H{
		"message": "FRPC started successfully",
	})
}

// handleFRPCStop 停止 FRPC
func (s *Server) handleFRPCStop(c *gin.Context) {
	if err := s.frpcMgr.Stop(); err != nil {
		serverError(c, err)
		return
	}

	success(c, gin.H{
		"message": "FRPC stopped successfully",
	})
}

// handleFRPCRestart 重启 FRPC
func (s *Server) handleFRPCRestart(c *gin.Context) {
	if err := s.frpcMgr.Restart(); err != nil {
		serverError(c, err)
		return
	}

	success(c, gin.H{
		"message": "FRPC restarted successfully",
	})
}

// handleGetConfig 获取配置
func (s *Server) handleGetConfig(c *gin.Context) {
	// 从 Server Panel 获取配置
	configResp, err := s.apiClient.GetConfig()
	if err != nil {
		serverError(c, err)
		return
	}

	success(c, configResp)
}

// handleSyncConfig 同步配置
func (s *Server) handleSyncConfig(c *gin.Context) {
	// 从 Server Panel 获取最新配置
	configResp, err := s.apiClient.GetConfig()
	if err != nil {
		serverError(c, err)
		return
	}

	// 生成 FRPC 配置文件
	if err := frpc.GenerateConfig(s.cfg, configResp.Mappings, configResp.ServerAddr); err != nil {
		serverError(c, err)
		return
	}

	// 如果 FRPC 正在运行，重启它
	if s.frpcMgr.IsRunning() {
		if err := s.frpcMgr.Restart(); err != nil {
			// 上报失败
			s.apiClient.ApplyConfig(configResp.Version, "error", err.Error())
			serverError(c, err)
			return
		}
	}

	// 上报成功
	if err := s.apiClient.ApplyConfig(configResp.Version, "success", ""); err != nil {
		log.Printf("Failed to report config applied: %v", err)
	}

	success(c, gin.H{
		"message": "Config synced successfully",
		"version": configResp.Version,
	})
}

// handleGetFRPCConfig 获取 FRPC 配置内容
func (s *Server) handleGetFRPCConfig(c *gin.Context) {
	// 从 Server Panel 获取配置
	configResp, err := s.apiClient.GetConfig()
	if err != nil {
		serverError(c, err)
		return
	}

	// 生成配置内容
	content := frpc.GenerateConfigContent(configResp.Mappings, configResp.ServerAddr)

	c.Header("Content-Type", "text/plain")
	c.String(200, content)
}

// handleGetStatus 获取状态
func (s *Server) handleGetStatus(c *gin.Context) {
	frpcStatus := s.frpcMgr.GetStatus()

	success(c, gin.H{
		"client_id":   s.cfg.Device.ClientID,
		"device_name": s.cfg.Device.DeviceName,
		"server_addr": s.cfg.Server.Address,
		"frpc_running": frpcStatus.Running,
		"frpc_error":   frpcStatus.Error,
		"last_heartbeat": time.Now().Format(time.RFC3339),
	})
}

// handleHeartbeat 处理心跳
func (s *Server) handleHeartbeat(c *gin.Context) {
	// 向 Server Panel 发送心跳
	resp, err := s.apiClient.Heartbeat()
	if err != nil {
		serverError(c, err)
		return
	}

	// 检查是否需要同步配置
	if resp.NeedSync {
		log.Println("Config sync needed, syncing...")
		go s.syncConfig()
	}

	success(c, gin.H{
		"status":    resp.Status,
		"need_sync": resp.NeedSync,
		"version":   resp.Version,
	})
}

// syncConfig 同步配置（后台执行）
func (s *Server) syncConfig() {
	// 从 Server Panel 获取最新配置
	configResp, err := s.apiClient.GetConfig()
	if err != nil {
		log.Printf("Failed to get config: %v", err)
		return
	}

	// 生成 FRPC 配置文件
	if err := frpc.GenerateConfig(s.cfg, configResp.Mappings, configResp.ServerAddr); err != nil {
		log.Printf("Failed to generate config: %v", err)
		return
	}

	// 如果 FRPC 正在运行，重启它
	if s.frpcMgr.IsRunning() {
		if err := s.frpcMgr.Restart(); err != nil {
			log.Printf("Failed to restart FRPC: %v", err)
			s.apiClient.ApplyConfig(configResp.Version, "error", err.Error())
			return
		}
	}

	// 上报成功
	if err := s.apiClient.ApplyConfig(configResp.Version, "success", ""); err != nil {
		log.Printf("Failed to report config applied: %v", err)
	}
}

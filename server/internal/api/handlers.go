package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/frp-cloud-panel/server/internal/database"
	"github.com/frp-cloud-panel/server/internal/middleware"
	"github.com/frp-cloud-panel/server/internal/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
	Email    string `json:"email" binding:"required,email"`
}

// handleLogin 处理登录
func (s *Server) handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 查找用户
	var user models.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		unauthorized(c, "Invalid username or password")
		return
	}

	// 检查用户状态
	if user.Status != "active" {
		forbidden(c, "Account is disabled")
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		unauthorized(c, "Invalid username or password")
		return
	}

	// 生成 Token
	token, err := middleware.GenerateToken(&s.cfg.JWT, user.ID, user.Username, user.Role)
	if err != nil {
		serverError(c, err)
		return
	}

	// 记录审计日志
	database.DB.Create(&models.AuditLog{
		UserID:   user.ID,
		Action:   "login",
		Resource: "user",
		IP:       c.ClientIP(),
	})

	success(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

// handleRegister 处理注册
func (s *Server) handleRegister(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 检查用户名是否已存在
	var count int64
	database.DB.Model(&models.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		badRequest(c, "Username already exists")
		return
	}

	// 检查邮箱是否已存在
	database.DB.Model(&models.User{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		badRequest(c, "Email already exists")
		return
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		serverError(c, err)
		return
	}

	// 创建用户
	user := models.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Email:    req.Email,
		Role:     "user",
		Status:   "active",
	}

	if err := database.DB.Create(&user).Error; err != nil {
		serverError(c, err)
		return
	}

	// 记录审计日志
	database.DB.Create(&models.AuditLog{
		UserID:   user.ID,
		Action:   "register",
		Resource: "user",
		IP:       c.ClientIP(),
	})

	success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

// handleGetMe 获取当前用户信息
func (s *Server) handleGetMe(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		notFound(c, "User not found")
		return
	}

	success(c, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"role":       user.Role,
		"status":     user.Status,
		"created_at": user.CreatedAt,
	})
}

// handleUpdateMe 更新当前用户信息
func (s *Server) handleUpdateMe(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Email string `json:"email" binding:"omitempty,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		notFound(c, "User not found")
		return
	}

	if req.Email != "" {
		// 检查邮箱是否已被使用
		var count int64
		database.DB.Model(&models.User{}).Where("email = ? AND id != ?", req.Email, user.ID).Count(&count)
		if count > 0 {
			badRequest(c, "Email already exists")
			return
		}
		user.Email = req.Email
	}

	database.DB.Save(&user)
	success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

// handleUpdatePassword 更新密码
func (s *Server) handleUpdatePassword(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		notFound(c, "User not found")
		return
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		badRequest(c, "Invalid old password")
		return
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		serverError(c, err)
		return
	}

	user.Password = string(hashedPassword)
	database.DB.Save(&user)

	success(c, gin.H{
		"message": "Password updated successfully",
	})
}

// handleListClients 获取客户端列表
func (s *Server) handleListClients(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	var clients []models.Client
	query := database.DB

	// 非管理员只能看到自己的客户端
	if role.(string) != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	query.Model(&models.Client{}).Count(&total)

	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).
		Preload("User").
		Find(&clients).Error; err != nil {
		serverError(c, err)
		return
	}

	success(c, gin.H{
		"total": total,
		"page":  page,
		"size":  pageSize,
		"items": clients,
	})
}

// handleGetClient 获取客户端详情
func (s *Server) handleGetClient(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	var client models.Client
	query := database.DB

	// 非管理员只能看到自己的客户端
	if role.(string) != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&client, id).Error; err != nil {
		notFound(c, "Client not found")
		return
	}

	success(c, client)
}

// handleUpdateClient 更新客户端
func (s *Server) handleUpdateClient(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	var req struct {
		DeviceName string `json:"device_name"`
		Status     string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	var client models.Client
	query := database.DB

	// 非管理员只能更新自己的客户端
	if role.(string) != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&client, id).Error; err != nil {
		notFound(c, "Client not found")
		return
	}

	if req.DeviceName != "" {
		client.DeviceName = req.DeviceName
	}
	if req.Status != "" {
		client.Status = req.Status
	}

	database.DB.Save(&client)
	success(c, client)
}

// handleDeleteClient 删除客户端
func (s *Server) handleDeleteClient(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	var client models.Client
	query := database.DB

	// 非管理员只能删除自己的客户端
	if role.(string) != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&client, id).Error; err != nil {
		notFound(c, "Client not found")
		return
	}

	// 检查是否有关联的映射
	var mappingCount int64
	database.DB.Model(&models.ProxyMapping{}).Where("client_id = ?", client.ID).Count(&mappingCount)
	if mappingCount > 0 {
		badRequest(c, "Cannot delete client with active mappings")
		return
	}

	database.DB.Delete(&client)
	success(c, gin.H{
		"message": "Client deleted successfully",
	})
}

// CreateMappingRequest 创建映射请求
type CreateMappingRequest struct {
	Name       string `json:"name" binding:"required"`
	Type       string `json:"type" binding:"required,oneof=tcp udp http https"`
	LocalIP    string `json:"local_ip" binding:"required"`
	LocalPort  int    `json:"local_port" binding:"required,min=1,max=65535"`
	RemotePort int    `json:"remote_port" binding:"omitempty,min=10000,max=20000"`
	Domain     string `json:"domain" binding:"omitempty"`
}

// handleListMappings 获取映射列表
func (s *Server) handleListMappings(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	var mappings []models.ProxyMapping
	query := database.DB

	// 非管理员只能看到自己的映射
	if role.(string) != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	query.Model(&models.ProxyMapping{}).Count(&total)

	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).
		Preload("Client").
		Find(&mappings).Error; err != nil {
		serverError(c, err)
		return
	}

	success(c, gin.H{
		"total": total,
		"page":  page,
		"size":  pageSize,
		"items": mappings,
	})
}

// handleCreateMapping 创建映射
func (s *Server) handleCreateMapping(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req CreateMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 检查客户端是否存在
	var client models.Client
	if err := database.DB.Where("id = ? AND user_id = ?", req.LocalIP, userID).First(&client).Error; err != nil {
		badRequest(c, "Client not found")
		return
	}

	// 开始事务
	tx := database.DB.Begin()

	// 分配端口
	var port models.Port
	if req.RemotePort > 0 {
		// 使用指定端口
		if err := tx.Where("port = ? AND status = ?", req.RemotePort, "free").First(&port).Error; err != nil {
			tx.Rollback()
			badRequest(c, "Port is not available")
			return
		}
	} else {
		// 自动分配端口
		if err := tx.Where("status = ?", "free").First(&port).Error; err != nil {
			tx.Rollback()
			badRequest(c, "No available ports")
			return
		}
	}

	// 创建映射
	mapping := models.ProxyMapping{
		UserID:     userID.(uint),
		ClientID:   client.ID,
		Name:       req.Name,
		Type:       req.Type,
		LocalIP:    req.LocalIP,
		LocalPort:  req.LocalPort,
		RemotePort: port.Port,
		Domain:     req.Domain,
		Status:     "pending_apply",
	}

	if err := tx.Create(&mapping).Error; err != nil {
		tx.Rollback()
		serverError(c, err)
		return
	}

	// 更新端口状态
	port.Status = "occupied"
	port.MappingID = &mapping.ID
	if err := tx.Save(&port).Error; err != nil {
		tx.Rollback()
		serverError(c, err)
		return
	}

	// 更新配置版本
	var configVersion models.ConfigVersion
	if err := tx.Where("client_id = ?", client.ID).First(&configVersion).Error; err != nil {
		// 创建新的配置版本
		configVersion = models.ConfigVersion{
			ClientID: client.ID,
			Version:  1,
		}
		tx.Create(&configVersion)
	} else {
		configVersion.Version++
		tx.Save(&configVersion)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		serverError(c, err)
		return
	}

	// 记录审计日志
	database.DB.Create(&models.AuditLog{
		UserID:   userID.(uint),
		Action:   "create_mapping",
		Resource: "proxy_mapping",
		Detail:   mapping.Name,
		IP:       c.ClientIP(),
	})

	success(c, mapping)
}

// handleGetMapping 获取映射详情
func (s *Server) handleGetMapping(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	var mapping models.ProxyMapping
	query := database.DB

	// 非管理员只能看到自己的映射
	if role.(string) != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&mapping, id).Error; err != nil {
		notFound(c, "Mapping not found")
		return
	}

	success(c, mapping)
}

// handleUpdateMapping 更新映射
func (s *Server) handleUpdateMapping(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	var req struct {
		Name      string `json:"name"`
		LocalIP   string `json:"local_ip"`
		LocalPort int    `json:"local_port"`
		Domain    string `json:"domain"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	var mapping models.ProxyMapping
	query := database.DB

	// 非管理员只能更新自己的映射
	if role.(string) != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&mapping, id).Error; err != nil {
		notFound(c, "Mapping not found")
		return
	}

	if req.Name != "" {
		mapping.Name = req.Name
	}
	if req.LocalIP != "" {
		mapping.LocalIP = req.LocalIP
	}
	if req.LocalPort > 0 {
		mapping.LocalPort = req.LocalPort
	}
	if req.Domain != "" {
		mapping.Domain = req.Domain
	}

	mapping.Status = "pending_apply"
	mapping.DesiredConfigVersion++

	database.DB.Save(&mapping)

	// 更新配置版本
	var configVersion models.ConfigVersion
	if err := database.DB.Where("client_id = ?", mapping.ClientID).First(&configVersion).Error; err == nil {
		configVersion.Version = mapping.DesiredConfigVersion
		database.DB.Save(&configVersion)
	}

	success(c, mapping)
}

// handleDeleteMapping 删除映射
func (s *Server) handleDeleteMapping(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	var mapping models.ProxyMapping
	query := database.DB

	// 非管理员只能删除自己的映射
	if role.(string) != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&mapping, id).Error; err != nil {
		notFound(c, "Mapping not found")
		return
	}

	// 标记为删除中
	mapping.Status = "deleting"
	database.DB.Save(&mapping)

	// 记录审计日志
	database.DB.Create(&models.AuditLog{
		UserID:   userID.(uint),
		Action:   "delete_mapping",
		Resource: "proxy_mapping",
		Detail:   mapping.Name,
		IP:       c.ClientIP(),
	})

	success(c, gin.H{
		"message": "Mapping marked for deletion",
	})
}

// handleListDomains 获取域名列表
func (s *Server) handleListDomains(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	var domains []models.Domain
	query := database.DB

	// 非管理员只能看到自己的域名
	if role.(string) != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Find(&domains).Error; err != nil {
		serverError(c, err)
		return
	}

	success(c, domains)
}

// handleCreateDomain 创建域名
func (s *Server) handleCreateDomain(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Domain    string `json:"domain" binding:"required"`
		HTTPSMode string `json:"https_mode" binding:"omitempty,oneof=none auto cf_proxy"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 检查域名是否已存在
	var count int64
	database.DB.Model(&models.Domain{}).Where("domain = ?", req.Domain).Count(&count)
	if count > 0 {
		badRequest(c, "Domain already exists")
		return
	}

	domain := models.Domain{
		UserID:    userID.(uint),
		Domain:    req.Domain,
		HTTPSMode: req.HTTPSMode,
	}

	if err := database.DB.Create(&domain).Error; err != nil {
		serverError(c, err)
		return
	}

	success(c, domain)
}

// handleGetDomain 获取域名详情
func (s *Server) handleGetDomain(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	var domain models.Domain
	query := database.DB

	// 非管理员只能看到自己的域名
	if role.(string) != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&domain, id).Error; err != nil {
		notFound(c, "Domain not found")
		return
	}

	success(c, domain)
}

// handleUpdateDomain 更新域名
func (s *Server) handleUpdateDomain(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	var req struct {
		HTTPSMode string `json:"https_mode" binding:"omitempty,oneof=none auto cf_proxy"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	var domain models.Domain
	query := database.DB

	// 非管理员只能更新自己的域名
	if role.(string) != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&domain, id).Error; err != nil {
		notFound(c, "Domain not found")
		return
	}

	if req.HTTPSMode != "" {
		domain.HTTPSMode = req.HTTPSMode
	}

	database.DB.Save(&domain)
	success(c, domain)
}

// handleDeleteDomain 删除域名
func (s *Server) handleDeleteDomain(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	var domain models.Domain
	query := database.DB

	// 非管理员只能删除自己的域名
	if role.(string) != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&domain, id).Error; err != nil {
		notFound(c, "Domain not found")
		return
	}

	// 检查是否有映射使用该域名
	var mappingCount int64
	database.DB.Model(&models.ProxyMapping{}).Where("domain = ?", domain.Domain).Count(&mappingCount)
	if mappingCount > 0 {
		badRequest(c, "Cannot delete domain with active mappings")
		return
	}

	database.DB.Delete(&domain)
	success(c, gin.H{
		"message": "Domain deleted successfully",
	})
}

// CFTokenRequest Cloudflare Token 请求
type CFTokenRequest struct {
	Token string `json:"token" binding:"required"`
	Email string `json:"email" binding:"omitempty,email"`
}

// handleSetCFToken 设置 Cloudflare Token
func (s *Server) handleSetCFToken(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req CFTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	// TODO: 加密 Token
	// TODO: 验证 Token

	// 保存或更新 Token
	var cfToken models.CloudflareToken
	if err := database.DB.Where("user_id = ?", userID).First(&cfToken).Error; err != nil {
		// 创建新的
		cfToken = models.CloudflareToken{
			UserID: userID.(uint),
			Token:  req.Token, // TODO: 加密
			Nonce:  "",        // TODO: 生成 nonce
			Email:  req.Email,
			Status: "active",
		}
		database.DB.Create(&cfToken)
	} else {
		// 更新现有的
		cfToken.Token = req.Token // TODO: 加密
		cfToken.Email = req.Email
		cfToken.Status = "active"
		database.DB.Save(&cfToken)
	}

	// 记录审计日志
	database.DB.Create(&models.AuditLog{
		UserID:   userID.(uint),
		Action:   "set_cf_token",
		Resource: "cloudflare_token",
		IP:       c.ClientIP(),
	})

	success(c, gin.H{
		"message": "Cloudflare token saved successfully",
	})
}

// handleGetCFTokenStatus 获取 Cloudflare Token 状态
func (s *Server) handleGetCFTokenStatus(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var cfToken models.CloudflareToken
	if err := database.DB.Where("user_id = ?", userID).First(&cfToken).Error; err != nil {
		success(c, gin.H{
			"status": "not_set",
		})
		return
	}

	success(c, gin.H{
		"status":     cfToken.Status,
		"email":      cfToken.Email,
		"created_at": cfToken.CreatedAt,
	})
}

// handleDeleteCFToken 删除 Cloudflare Token
func (s *Server) handleDeleteCFToken(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var cfToken models.CloudflareToken
	if err := database.DB.Where("user_id = ?", userID).First(&cfToken).Error; err != nil {
		notFound(c, "Cloudflare token not found")
		return
	}

	database.DB.Delete(&cfToken)

	// 记录审计日志
	database.DB.Create(&models.AuditLog{
		UserID:   userID.(uint),
		Action:   "delete_cf_token",
		Resource: "cloudflare_token",
		IP:       c.ClientIP(),
	})

	success(c, gin.H{
		"message": "Cloudflare token deleted successfully",
	})
}

// handleTestCFToken 测试 Cloudflare Token
func (s *Server) handleTestCFToken(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var cfToken models.CloudflareToken
	if err := database.DB.Where("user_id = ?", userID).First(&cfToken).Error; err != nil {
		notFound(c, "Cloudflare token not found")
		return
	}

	// TODO: 实际测试 Token

	success(c, gin.H{
		"status":  "valid",
		"message": "Token is valid",
	})
}

// handleListLogs 获取日志列表
func (s *Server) handleListLogs(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	var logs []models.AuditLog
	query := database.DB

	// 非管理员只能看到自己的日志
	if role.(string) != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	query.Model(&models.AuditLog{}).Count(&total)

	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).
		Order("created_at DESC").
		Find(&logs).Error; err != nil {
		serverError(c, err)
		return
	}

	success(c, gin.H{
		"total": total,
		"page":  page,
		"size":  pageSize,
		"items": logs,
	})
}

// handleClientRegister 客户端注册
func (s *Server) handleClientRegister(c *gin.Context) {
	clientID, _ := c.Get("client_id")

	var req struct {
		Username   string `json:"username" binding:"required"`
		Password   string `json:"password" binding:"required"`
		DeviceName string `json:"device_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 验证用户名密码
	var user models.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		unauthorized(c, "Invalid username or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		unauthorized(c, "Invalid username or password")
		return
	}

	// 检查客户端是否已注册
	var client models.Client
	if err := database.DB.Where("client_id = ?", clientID).First(&client).Error; err == nil {
		// 已注册，更新信息
		client.DeviceName = req.DeviceName
		client.IP = c.ClientIP()
		client.LastSeenAt = &[]time.Time{time.Now()}[0]
		database.DB.Save(&client)

		success(c, gin.H{
			"client_id":    client.ClientID,
			"device_token": "existing", // 不返回实际 token
		})
		return
	}

	// 生成设备 Token
	deviceToken := generateDeviceToken()

	// 创建客户端
	client = models.Client{
		UserID:      user.ID,
		ClientID:    clientID.(string),
		DeviceToken: deviceToken, // TODO: hash
		DeviceName:  req.DeviceName,
		IP:          c.ClientIP(),
		Status:      "active",
		LastSeenAt:  &[]time.Time{time.Now()}[0],
	}

	if err := database.DB.Create(&client).Error; err != nil {
		serverError(c, err)
		return
	}

	// 创建配置版本
	database.DB.Create(&models.ConfigVersion{
		ClientID: client.ID,
		Version:  1,
	})

	success(c, gin.H{
		"client_id":    client.ClientID,
		"device_token": deviceToken,
	})
}

// handleGetClientConfig 获取客户端配置
func (s *Server) handleGetClientConfig(c *gin.Context) {
	clientID, _ := c.Get("client_id")

	var client models.Client
	if err := database.DB.Where("client_id = ?", clientID).First(&client).Error; err != nil {
		notFound(c, "Client not found")
		return
	}

	// 获取映射列表
	var mappings []models.ProxyMapping
	database.DB.Where("client_id = ? AND status != ?", client.ID, "deleting").Find(&mappings)

	// 获取配置版本
	var configVersion models.ConfigVersion
	database.DB.Where("client_id = ?", client.ID).First(&configVersion)

	success(c, gin.H{
		"client_id":   client.ClientID,
		"version":     configVersion.Version,
		"mappings":    mappings,
		"server_addr": s.cfg.Server.GetAddress(),
	})
}

// handleApplyConfig 应用配置
func (s *Server) handleApplyConfig(c *gin.Context) {
	clientID, _ := c.Get("client_id")

	var req struct {
		Version int    `json:"version" binding:"required"`
		Status  string `json:"status" binding:"required"`
		Error   string `json:"error"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	var client models.Client
	if err := database.DB.Where("client_id = ?", clientID).First(&client).Error; err != nil {
		notFound(c, "Client not found")
		return
	}

	// 更新映射状态
	var mappings []models.ProxyMapping
	database.DB.Where("client_id = ?", client.ID).Find(&mappings)

	for i := range mappings {
		mappings[i].AppliedConfigVersion = req.Version
		if req.Status == "success" {
			mappings[i].Status = "running"
		} else {
			mappings[i].Status = "config_error"
		}
		database.DB.Save(&mappings[i])
	}

	success(c, gin.H{
		"message": "Config applied successfully",
	})
}

// handleUpdateStatus 更新状态
func (s *Server) handleUpdateStatus(c *gin.Context) {
	clientID, _ := c.Get("client_id")

	var req struct {
		Mappings []struct {
			ID     uint   `json:"id"`
			Status string `json:"status"`
		} `json:"mappings"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	var client models.Client
	if err := database.DB.Where("client_id = ?", clientID).First(&client).Error; err != nil {
		notFound(c, "Client not found")
		return
	}

	// 更新映射状态
	for _, m := range req.Mappings {
		var mapping models.ProxyMapping
		if err := database.DB.Where("id = ? AND client_id = ?", m.ID, client.ID).First(&mapping).Error; err != nil {
			continue
		}
		mapping.Status = m.Status
		database.DB.Save(&mapping)
	}

	// 更新客户端最后在线时间
	now := time.Now()
	client.LastSeenAt = &now
	client.IP = c.ClientIP()
	database.DB.Save(&client)

	success(c, gin.H{
		"message": "Status updated successfully",
	})
}

// handleHeartbeat 心跳
func (s *Server) handleHeartbeat(c *gin.Context) {
	clientID, _ := c.Get("client_id")

	var client models.Client
	if err := database.DB.Where("client_id = ?", clientID).First(&client).Error; err != nil {
		notFound(c, "Client not found")
		return
	}

	// 更新最后在线时间
	now := time.Now()
	client.LastSeenAt = &now
	client.IP = c.ClientIP()
	database.DB.Save(&client)

	// 检查配置版本
	var configVersion models.ConfigVersion
	database.DB.Where("client_id = ?", client.ID).First(&configVersion)

	var appliedVersion int
	if client.ID > 0 {
		var mapping models.ProxyMapping
		database.DB.Where("client_id = ?", client.ID).First(&mapping)
		appliedVersion = mapping.AppliedConfigVersion
	}

	needSync := configVersion.Version > appliedVersion

	success(c, gin.H{
		"status":     "ok",
		"need_sync":  needSync,
		"version":    configVersion.Version,
	})
}

// generateDeviceToken 生成设备 Token
func generateDeviceToken() string {
	// TODO: 实现安全的 token 生成
	return "device_token_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

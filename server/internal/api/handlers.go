package api

import (
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

	// 获取用户的第一个客户端
	var client models.Client
	if err := database.DB.Where("user_id = ?", userID).First(&client).Error; err != nil {
		badRequest(c, "No client found for this user")
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

	// 保存 Token（加密）
	if err := s.cfService.SaveToken(userID.(uint), req.Token, req.Email); err != nil {
		serverError(c, err)
		return
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

	cfToken, err := s.cfService.GetTokenStatus(userID.(uint))
	if err != nil {
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

	// 删除 Token
	if err := s.cfService.DeleteToken(userID.(uint)); err != nil {
		notFound(c, "Cloudflare token not found")
		return
	}

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

	// 测试 Token
	valid, err := s.cfService.TestToken(userID.(uint))
	if err != nil {
		badRequest(c, "Failed to test token: "+err.Error())
		return
	}

	if valid {
		success(c, gin.H{
			"status":  "valid",
			"message": "Token is valid",
		})
	} else {
		success(c, gin.H{
			"status":  "invalid",
			"message": "Token is invalid",
		})
	}
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

// handleListDNSRecords 获取 DNS 记录列表
func (s *Server) handleListDNSRecords(c *gin.Context) {
	userID, _ := c.Get("user_id")
	domain := c.Query("domain")

	if domain == "" {
		badRequest(c, "Domain parameter is required")
		return
	}

	// 获取 DNS 记录
	records, err := s.cfService.GetDNSRecords(userID.(uint), domain)
	if err != nil {
		serverError(c, err)
		return
	}

	success(c, records)
}

// CreateDNSRecordRequest 创建 DNS 记录请求
type CreateDNSRecordRequest struct {
	Domain string `json:"domain" binding:"required"`
	IP     string `json:"ip" binding:"required"`
}

// handleCreateDNSRecord 创建 DNS 记录
func (s *Server) handleCreateDNSRecord(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req CreateDNSRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 创建 DNS 记录
	record, err := s.cfService.CreateDNSRecord(userID.(uint), req.Domain, req.IP)
	if err != nil {
		serverError(c, err)
		return
	}

	// 记录审计日志
	database.DB.Create(&models.AuditLog{
		UserID:   userID.(uint),
		Action:   "create_dns_record",
		Resource: "dns_record",
		Detail:   req.Domain,
		IP:       c.ClientIP(),
	})

	success(c, record)
}

// UpdateDNSRecordRequest 更新 DNS 记录请求
type UpdateDNSRecordRequest struct {
	IP string `json:"ip" binding:"required"`
}

// handleUpdateDNSRecord 更新 DNS 记录
func (s *Server) handleUpdateDNSRecord(c *gin.Context) {
	userID, _ := c.Get("user_id")
	domain := c.Param("domain")

	var req UpdateDNSRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 更新 DNS 记录
	record, err := s.cfService.CreateDNSRecord(userID.(uint), domain, req.IP)
	if err != nil {
		serverError(c, err)
		return
	}

	// 记录审计日志
	database.DB.Create(&models.AuditLog{
		UserID:   userID.(uint),
		Action:   "update_dns_record",
		Resource: "dns_record",
		Detail:   domain,
		IP:       c.ClientIP(),
	})

	success(c, record)
}

// handleDeleteDNSRecord 删除 DNS 记录
func (s *Server) handleDeleteDNSRecord(c *gin.Context) {
	userID, _ := c.Get("user_id")
	domain := c.Param("domain")

	// 删除 DNS 记录
	if err := s.cfService.DeleteDNSRecord(userID.(uint), domain); err != nil {
		serverError(c, err)
		return
	}

	// 记录审计日志
	database.DB.Create(&models.AuditLog{
		UserID:   userID.(uint),
		Action:   "delete_dns_record",
		Resource: "dns_record",
		Detail:   domain,
		IP:       c.ClientIP(),
	})

	success(c, gin.H{
		"message": "DNS record deleted successfully",
	})
}

// handleWebSocket 处理 WebSocket 连接
func (s *Server) handleWebSocket(c *gin.Context) {
	s.wsHub.HandleWebSocket(c)
}

// handleGetCert 获取证书信息
func (s *Server) handleGetCert(c *gin.Context) {
	domain := c.Param("domain")

	// 获取证书过期时间
	expiry, err := s.acmeService.GetCertExpiry(domain)
	if err != nil {
		serverError(c, err)
		return
	}

	// 获取域名信息
	var domainInfo models.Domain
	if err := database.DB.Where("domain = ?", domain).First(&domainInfo).Error; err != nil {
		notFound(c, "Domain not found")
		return
	}

	success(c, gin.H{
		"domain":     domain,
		"https_mode": domainInfo.HTTPSMode,
		"cert_status": domainInfo.CertStatus,
		"cert_expiry": expiry,
	})
}

// handleRenewCert 续期证书
func (s *Server) handleRenewCert(c *gin.Context) {
	domain := c.Param("domain")

	// 检查域名是否存在
	var domainInfo models.Domain
	if err := database.DB.Where("domain = ?", domain).First(&domainInfo).Error; err != nil {
		notFound(c, "Domain not found")
		return
	}

	// 检查 HTTPS 模式
	if domainInfo.HTTPSMode != "auto" {
		badRequest(c, "Domain is not configured for auto HTTPS")
		return
	}

	// 续期证书
	if err := s.acmeService.RenewCertificate(domain); err != nil {
		serverError(c, err)
		return
	}

	// 记录审计日志
	userID, _ := c.Get("user_id")
	database.DB.Create(&models.AuditLog{
		UserID:   userID.(uint),
		Action:   "renew_cert",
		Resource: "certificate",
		Detail:   domain,
		IP:       c.ClientIP(),
	})

	success(c, gin.H{
		"message": "Certificate renewed successfully",
	})
}

// handleCheckCerts 检查所有证书状态
func (s *Server) handleCheckCerts(c *gin.Context) {
	// 启动证书检查
	go s.acmeService.CheckCertificates()

	success(c, gin.H{
		"message": "Certificate check started",
	})
}

// handleGetConfigVersion 获取配置版本
func (s *Server) handleGetConfigVersion(c *gin.Context) {
	clientIDStr := c.Param("client_id")
	clientID, err := strconv.ParseUint(clientIDStr, 10, 32)
	if err != nil {
		badRequest(c, "Invalid client ID")
		return
	}

	version, err := s.configService.GetConfigVersion(uint(clientID))
	if err != nil {
		serverError(c, err)
		return
	}

	success(c, gin.H{
		"client_id": clientID,
		"version":   version,
	})
}

// handleGetDesiredConfig 获取期望配置
func (s *Server) handleGetDesiredConfig(c *gin.Context) {
	clientIDStr := c.Param("client_id")
	clientID, err := strconv.ParseUint(clientIDStr, 10, 32)
	if err != nil {
		badRequest(c, "Invalid client ID")
		return
	}

	config, err := s.configService.GetDesiredConfig(uint(clientID))
	if err != nil {
		serverError(c, err)
		return
	}

	success(c, config)
}

// handleApplyConfig 应用配置
func (s *Server) handleApplyConfig(c *gin.Context) {
	clientIDStr := c.Param("client_id")
	clientID, err := strconv.ParseUint(clientIDStr, 10, 32)
	if err != nil {
		badRequest(c, "Invalid client ID")
		return
	}

	var req struct {
		Version int    `json:"version" binding:"required"`
		Status  string `json:"status" binding:"required"`
		Error   string `json:"error"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := s.configService.ApplyConfig(uint(clientID), req.Version, req.Status, req.Error); err != nil {
		serverError(c, err)
		return
	}

	success(c, gin.H{
		"message": "Config applied successfully",
	})
}

// handleCheckConfigSync 检查配置同步状态
func (s *Server) handleCheckConfigSync(c *gin.Context) {
	clientIDStr := c.Param("client_id")
	clientID, err := strconv.ParseUint(clientIDStr, 10, 32)
	if err != nil {
		badRequest(c, "Invalid client ID")
		return
	}

	needSync, err := s.configService.CheckConfigSync(uint(clientID))
	if err != nil {
		serverError(c, err)
		return
	}

	success(c, gin.H{
		"client_id": clientID,
		"need_sync": needSync,
	})
}

// handleSyncConfig 同步配置
func (s *Server) handleSyncConfig(c *gin.Context) {
	clientIDStr := c.Param("client_id")
	clientID, err := strconv.ParseUint(clientIDStr, 10, 32)
	if err != nil {
		badRequest(c, "Invalid client ID")
		return
	}

	if err := s.configService.SyncConfig(uint(clientID)); err != nil {
		serverError(c, err)
		return
	}

	success(c, gin.H{
		"message": "Config sync initiated",
	})
}

// handleExportConfig 导出配置
func (s *Server) handleExportConfig(c *gin.Context) {
	clientIDStr := c.Param("client_id")
	clientID, err := strconv.ParseUint(clientIDStr, 10, 32)
	if err != nil {
		badRequest(c, "Invalid client ID")
		return
	}

	data, err := s.configService.ExportConfig(uint(clientID))
	if err != nil {
		serverError(c, err)
		return
	}

	c.Header("Content-Disposition", "attachment; filename=config.json")
	c.Data(200, "application/json", data)
}

// handleImportConfig 导入配置
func (s *Server) handleImportConfig(c *gin.Context) {
	clientIDStr := c.Param("client_id")
	clientID, err := strconv.ParseUint(clientIDStr, 10, 32)
	if err != nil {
		badRequest(c, "Invalid client ID")
		return
	}

	data, err := c.GetRawData()
	if err != nil {
		badRequest(c, "Failed to read request body")
		return
	}

	if err := s.configService.ImportConfig(uint(clientID), data); err != nil {
		serverError(c, err)
		return
	}

	success(c, gin.H{
		"message": "Config imported successfully",
	})
}

// handleGenerateFRPCConfig 生成 FRPC 配置文件
func (s *Server) handleGenerateFRPCConfig(c *gin.Context) {
	clientIDStr := c.Param("client_id")
	clientID, err := strconv.ParseUint(clientIDStr, 10, 32)
	if err != nil {
		badRequest(c, "Invalid client ID")
		return
	}

	config, err := s.configService.GenerateFRPCConfig(uint(clientID))
	if err != nil {
		serverError(c, err)
		return
	}

	c.Header("Content-Disposition", "attachment; filename=frpc.toml")
	c.Data(200, "text/plain", []byte(config))
}

// CreateBackupRequest 创建备份请求
type CreateBackupRequest struct {
	Password string `json:"password" binding:"required"`
}

// handleCreateBackup 创建备份
func (s *Server) handleCreateBackup(c *gin.Context) {
	var req CreateBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 创建备份
	filepath, err := s.backupService.CreateBackup(req.Password)
	if err != nil {
		serverError(c, err)
		return
	}

	success(c, gin.H{
		"message":  "Backup created successfully",
		"filepath": filepath,
	})
}

// RestoreBackupRequest 恢复备份请求
type RestoreBackupRequest struct {
	Filepath string `json:"filepath" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// handleRestoreBackup 恢复备份
func (s *Server) handleRestoreBackup(c *gin.Context) {
	var req RestoreBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 恢复备份
	if err := s.backupService.RestoreBackup(req.Filepath, req.Password); err != nil {
		serverError(c, err)
		return
	}

	success(c, gin.H{
		"message": "Backup restored successfully",
	})
}

// handleListBackups 列出备份
func (s *Server) handleListBackups(c *gin.Context) {
	backups, err := s.backupService.ListBackups()
	if err != nil {
		serverError(c, err)
		return
	}

	success(c, backups)
}

// handleDeleteBackup 删除备份
func (s *Server) handleDeleteBackup(c *gin.Context) {
	filename := c.Param("filename")

	if err := s.backupService.DeleteBackup(filename); err != nil {
		serverError(c, err)
		return
	}

	success(c, gin.H{
		"message": "Backup deleted successfully",
	})
}

// handleFRPSLogin 处理 FRPS 登录验证
func (s *Server) handleFRPSLogin(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 设置远程地址
	req.RemoteAddr = c.ClientIP()

	// 验证登录
	resp, err := s.frpsAuthService.ValidateLogin(&req)
	if err != nil {
		serverError(c, err)
		return
	}

	if resp.Reject {
		c.JSON(401, resp)
		return
	}

	c.JSON(200, resp)
}

// handleFRPSNewProxy 处理 FRPS 新代理验证
func (s *Server) handleFRPSNewProxy(c *gin.Context) {
	var req services.NewProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 设置远程地址
	req.RemoteAddr = c.ClientIP()

	// 验证新代理
	resp, err := s.frpsAuthService.ValidateNewProxy(&req)
	if err != nil {
		serverError(c, err)
		return
	}

	if resp.Reject {
		c.JSON(403, resp)
		return
	}

	c.JSON(200, resp)
}

// handleRouterStats 获取路由器统计信息
func (s *Server) handleRouterStats(c *gin.Context) {
	stats := s.httpRouter.GetStats()
	success(c, stats)
}

// handleReloadCerts 重新加载证书
func (s *Server) handleReloadCerts(c *gin.Context) {
	s.httpRouter.ReloadCertificates()
	success(c, gin.H{
		"message": "Certificates reloaded successfully",
	})
}

// handleClearRouterCache 清除路由器缓存
func (s *Server) handleClearRouterCache(c *gin.Context) {
	s.httpRouter.ClearProxyCache()
	success(c, gin.H{
		"message": "Router cache cleared successfully",
	})
}

// handleGetSystemStats 获取系统统计
func (s *Server) handleGetSystemStats(c *gin.Context) {
	stats := s.monitoringService.GetSystemStats()
	success(c, stats)
}

// handleGetAlerts 获取告警列表
func (s *Server) handleGetAlerts(c *gin.Context) {
	resolved := c.Query("resolved") == "true"
	alerts := s.monitoringService.GetAlerts(resolved)
	success(c, alerts)
}

// handleResolveAlert 解决告警
func (s *Server) handleResolveAlert(c *gin.Context) {
	alertID := c.Param("id")

	if err := s.monitoringService.ResolveAlert(alertID); err != nil {
		notFound(c, "Alert not found")
		return
	}

	success(c, gin.H{
		"message": "Alert resolved successfully",
	})
}

// handleGetAlertRules 获取告警规则列表
func (s *Server) handleGetAlertRules(c *gin.Context) {
	rules := s.monitoringService.GetAlertRules()
	success(c, rules)
}

// handleAddAlertRule 添加告警规则
func (s *Server) handleAddAlertRule(c *gin.Context) {
	var rule services.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	s.monitoringService.AddAlertRule(rule)
	success(c, gin.H{
		"message": "Alert rule added successfully",
	})
}

// handleUpdateAlertRule 更新告警规则
func (s *Server) handleUpdateAlertRule(c *gin.Context) {
	ruleID := c.Param("id")

	var rule services.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		badRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := s.monitoringService.UpdateAlertRule(ruleID, rule); err != nil {
		notFound(c, "Alert rule not found")
		return
	}

	success(c, gin.H{
		"message": "Alert rule updated successfully",
	})
}

// handleDeleteAlertRule 删除告警规则
func (s *Server) handleDeleteAlertRule(c *gin.Context) {
	ruleID := c.Param("id")

	if err := s.monitoringService.DeleteAlertRule(ruleID); err != nil {
		notFound(c, "Alert rule not found")
		return
	}

	success(c, gin.H{
		"message": "Alert rule deleted successfully",
	})
}

package services

import (
	"fmt"
	"log"
	"time"

	"github.com/frp-cloud-panel/server/internal/database"
	"github.com/frp-cloud-panel/server/internal/models"
)

// FRPSAuthService FRPS 鉴权服务
type FRPSAuthService struct {
	// 配置
	maxLoginAttempts int
	lockoutDuration  time.Duration
}

// NewFRPSAuthService 创建新的 FRPS 鉴权服务
func NewFRPSAuthService() *FRPSAuthService {
	return &FRPSAuthService{
		maxLoginAttempts: 5,
		lockoutDuration:  15 * time.Minute,
	}
}

// LoginRequest FRPS 登录请求
type LoginRequest struct {
	User       string `json:"user"`
	Token      string `json:"token"`
	ClientID   string `json:"client_id"`
	RemoteAddr string `json:"remote_addr"`
	Version    string `json:"version"`
}

// LoginResponse FRPS 登录响应
type LoginResponse struct {
	Reject       bool   `json:"reject"`
	RejectReason string `json:"reject_reason,omitempty"`
	UserID       uint   `json:"user_id,omitempty"`
}

// NewProxyRequest FRPS 新代理请求
type NewProxyRequest struct {
	User       string `json:"user"`
	Token      string `json:"token"`
	ClientID   string `json:"client_id"`
	ProxyName  string `json:"proxy_name"`
	ProxyType  string `json:"proxy_type"`
	RemotePort int    `json:"remote_port"`
	CustomDomain string `json:"custom_domain"`
	RemoteAddr string `json:"remote_addr"`
}

// NewProxyResponse FRPS 新代理响应
type NewProxyResponse struct {
	Reject       bool   `json:"reject"`
	RejectReason string `json:"reject_reason,omitempty"`
}

// ValidateLogin 验证 FRPS 登录
func (s *FRPSAuthService) ValidateLogin(req *LoginRequest) (*LoginResponse, error) {
	// 查找用户
	var user models.User
	if err := database.DB.Where("username = ?", req.User).First(&user).Error; err != nil {
		log.Printf("FRPS login failed: user not found - %s", req.User)
		return &LoginResponse{
			Reject:       true,
			RejectReason: "Invalid username or password",
		}, nil
	}

	// 检查用户状态
	if user.Status != "active" {
		log.Printf("FRPS login failed: user disabled - %s", req.User)
		return &LoginResponse{
			Reject:       true,
			RejectReason: "Account is disabled",
		}, nil
	}

	// 检查是否被锁定
	if s.isAccountLocked(user.ID) {
		log.Printf("FRPS login failed: account locked - %s", req.User)
		return &LoginResponse{
			Reject:       true,
			RejectReason: "Account is temporarily locked due to too many failed attempts",
		}, nil
	}

	// 验证客户端
	var client models.Client
	if err := database.DB.Where("client_id = ? AND user_id = ?", req.ClientID, user.ID).First(&client).Error; err != nil {
		log.Printf("FRPS login failed: client not found - %s", req.ClientID)
		s.recordLoginAttempt(user.ID, false)
		return &LoginResponse{
			Reject:       true,
			RejectReason: "Invalid client ID",
		}, nil
	}

	// 检查客户端状态
	if client.Status != "active" {
		log.Printf("FRPS login failed: client disabled - %s", req.ClientID)
		return &LoginResponse{
			Reject:       true,
			RejectReason: "Client is disabled",
		}, nil
	}

	// 验证 Token（简化版本，实际应该验证 device_token）
	// 这里假设 Token 是有效的，实际应该与数据库中的 token 进行比较

	// 记录登录成功
	s.recordLoginAttempt(user.ID, true)

	// 更新客户端最后在线时间
	now := time.Now()
	client.LastSeenAt = &now
	client.IP = req.RemoteAddr
	client.Version = req.Version
	database.DB.Save(&client)

	// 记录审计日志
	database.DB.Create(&models.AuditLog{
		UserID:   user.ID,
		Action:   "frps_login",
		Resource: "frps",
		Detail:   fmt.Sprintf("Client: %s, IP: %s", req.ClientID, req.RemoteAddr),
		IP:       req.RemoteAddr,
	})

	log.Printf("FRPS login success: user=%s, client=%s", req.User, req.ClientID)

	return &LoginResponse{
		Reject: false,
		UserID: user.ID,
	}, nil
}

// ValidateNewProxy 验证新代理创建
func (s *FRPSAuthService) ValidateNewProxy(req *NewProxyRequest) (*NewProxyResponse, error) {
	// 查找用户
	var user models.User
	if err := database.DB.Where("username = ?", req.User).First(&user).Error; err != nil {
		return &NewProxyResponse{
			Reject:       true,
			RejectReason: "Invalid user",
		}, nil
	}

	// 检查用户状态
	if user.Status != "active" {
		return &NewProxyResponse{
			Reject:       true,
			RejectReason: "Account is disabled",
		}, nil
	}

	// 查找客户端
	var client models.Client
	if err := database.DB.Where("client_id = ? AND user_id = ?", req.ClientID, user.ID).First(&client).Error; err != nil {
		return &NewProxyResponse{
			Reject:       true,
			RejectReason: "Invalid client",
		}, nil
	}

	// 检查客户端状态
	if client.Status != "active" {
		return &NewProxyResponse{
			Reject:       true,
			RejectReason: "Client is disabled",
		}, nil
	}

	// 验证端口归属
	if req.RemotePort > 0 {
		var port models.Port
		if err := database.DB.Where("port = ?", req.RemotePort).First(&port).Error; err != nil {
			return &NewProxyResponse{
				Reject:       true,
				RejectReason: "Port not found",
			}, nil
		}

		// 检查端口是否属于该用户的映射
		var mapping models.ProxyMapping
		if err := database.DB.Where("remote_port = ? AND user_id = ? AND client_id = ?", req.RemotePort, user.ID, client.ID).First(&mapping).Error; err != nil {
			return &NewProxyResponse{
				Reject:       true,
				RejectReason: "Port not owned by this user",
			}, nil
		}

		// 检查映射状态
		if mapping.Status != "running" && mapping.Status != "pending_apply" {
			return &NewProxyResponse{
				Reject:       true,
				RejectReason: "Mapping is not active",
			}, nil
		}
	}

	// 验证域名归属
	if req.CustomDomain != "" {
		var domain models.Domain
		if err := database.DB.Where("domain = ? AND user_id = ?", req.CustomDomain, user.ID).First(&domain).Error; err != nil {
			return &NewProxyResponse{
				Reject:       true,
				RejectReason: "Domain not owned by this user",
			}, nil
		}

		// 检查域名状态
		if domain.DeletedAt.Valid {
			return &NewProxyResponse{
				Reject:       true,
				RejectReason: "Domain is deleted",
			}, nil
		}
	}

	// 检查配置版本
	var mapping models.ProxyMapping
	if err := database.DB.Where("name = ? AND client_id = ?", req.ProxyName, client.ID).First(&mapping).Error; err == nil {
		// 映射存在，检查配置版本
		var configVersion models.ConfigVersion
		if err := database.DB.Where("client_id = ?", client.ID).First(&configVersion).Error; err == nil {
			if mapping.AppliedConfigVersion < configVersion.Version {
				return &NewProxyResponse{
					Reject:       true,
					RejectReason: "Configuration version mismatch, please sync first",
				}, nil
			}
		}
	}

	log.Printf("FRPS new proxy validated: user=%s, proxy=%s, type=%s", req.User, req.ProxyName, req.ProxyType)

	return &NewProxyResponse{
		Reject: false,
	}, nil
}

// isAccountLocked 检查账户是否被锁定
func (s *FRPSAuthService) isAccountLocked(userID uint) bool {
	var loginAttempt models.LoginAttempt
	if err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").First(&loginAttempt).Error; err != nil {
		return false
	}

	// 检查是否在锁定时间内
	if loginAttempt.FailedAttempts >= s.maxLoginAttempts {
		if time.Since(loginAttempt.CreatedAt) < s.lockoutDuration {
			return true
		}
		// 锁定时间已过，重置失败次数
		database.DB.Model(&loginAttempt).Update("failed_attempts", 0)
	}

	return false
}

// recordLoginAttempt 记录登录尝试
func (s *FRPSAuthService) recordLoginAttempt(userID uint, success bool) {
	var loginAttempt models.LoginAttempt
	if err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").First(&loginAttempt).Error; err != nil {
		// 创建新的记录
		loginAttempt = models.LoginAttempt{
			UserID:          userID,
			FailedAttempts:  0,
			LastAttemptAt:   time.Now(),
		}
		database.DB.Create(&loginAttempt)
	}

	if success {
		// 登录成功，重置失败次数
		loginAttempt.FailedAttempts = 0
	} else {
		// 登录失败，增加失败次数
		loginAttempt.FailedAttempts++
	}
	loginAttempt.LastAttemptAt = time.Now()
	database.DB.Save(&loginAttempt)
}

// LoginAttempt 登录尝试记录
type LoginAttempt struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	UserID         uint      `json:"user_id" gorm:"index"`
	FailedAttempts int       `json:"failed_attempts"`
	LastAttemptAt  time.Time `json:"last_attempt_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

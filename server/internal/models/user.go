package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Username  string         `json:"username" gorm:"uniqueIndex;size:50;not null"`
	Password  string         `json:"-" gorm:"size:100;not null"`
	Email     string         `json:"email" gorm:"uniqueIndex;size:100"`
	Role      string         `json:"role" gorm:"size:20;default:user"` // admin, user
	Status    string         `json:"status" gorm:"size:20;default:active"` // active, disabled
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// Client 客户端设备模型
type Client struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	UserID       uint           `json:"user_id" gorm:"index;not null"`
	ClientID     string         `json:"client_id" gorm:"uniqueIndex;size:50;not null"`
	DeviceToken  string         `json:"-" gorm:"size:100;not null"` // hash
	DeviceName   string         `json:"device_name" gorm:"size:100"`
	IP           string         `json:"ip" gorm:"size:50"`
	Version      string         `json:"version" gorm:"size:20"`
	Status       string         `json:"status" gorm:"size:20;default:active"` // active, disabled
	LastSeenAt   *time.Time     `json:"last_seen_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
	User         User           `json:"user" gorm:"foreignKey:UserID"`
}

// ProxyMapping 代理映射模型
type ProxyMapping struct {
	ID                  uint           `json:"id" gorm:"primaryKey"`
	UserID              uint           `json:"user_id" gorm:"index;not null"`
	ClientID            uint           `json:"client_id" gorm:"index;not null"`
	Name                string         `json:"name" gorm:"size:100;not null"`
	Type                string         `json:"type" gorm:"size:20;not null"` // tcp, udp, http, https
	LocalIP             string         `json:"local_ip" gorm:"size:50;not null"`
	LocalPort           int            `json:"local_port" gorm:"not null"`
	RemotePort          int            `json:"remote_port"`
	Domain              string         `json:"domain" gorm:"size:100"`
	Status              string         `json:"status" gorm:"size:20;default:pending_apply"` // reserved, pending_apply, running, offline, config_error, disabled, deleting
	DesiredConfigVersion int           `json:"desired_config_version" gorm:"default:1"`
	AppliedConfigVersion int           `json:"applied_config_version" gorm:"default:0"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"index"`
	User                User           `json:"user" gorm:"foreignKey:UserID"`
	Client              Client         `json:"client" gorm:"foreignKey:ClientID"`
}

// Port 端口资源模型
type Port struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	ServerID   uint      `json:"server_id" gorm:"index;default:1"`
	Port       int       `json:"port" gorm:"uniqueIndex:idx_server_port;not null"`
	Status     string    `json:"status" gorm:"size:20;default:free"` // free, occupied, reserved
	MappingID  *uint     `json:"mapping_id" gorm:"index"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Domain 域名资源模型
type Domain struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	UserID        uint      `json:"user_id" gorm:"index;not null"`
	Domain        string    `json:"domain" gorm:"uniqueIndex;size:100;not null"`
	HTTPSMode     string    `json:"https_mode" gorm:"size:20;default:none"` // none, auto, cf_proxy
	CertStatus    string    `json:"cert_status" gorm:"size:20;default:none"` // none, pending, active, expired
	CertExpiry    *time.Time `json:"cert_expiry"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
	User          User      `json:"user" gorm:"foreignKey:UserID"`
}

// CloudflareToken Cloudflare Token 模型
type CloudflareToken struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"uniqueIndex;not null"`
	Token     string    `json:"-" gorm:"size:500;not null"` // AES-256-GCM encrypted
	Nonce     string    `json:"-" gorm:"size:50;not null"`
	Email     string    `json:"email" gorm:"size:100"`
	Status    string    `json:"status" gorm:"size:20;default:active"` // active, invalid
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
}

// AuditLog 审计日志模型
type AuditLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index"`
	Action    string    `json:"action" gorm:"size:50;not null"` // login, create_mapping, delete_mapping, etc.
	Resource  string    `json:"resource" gorm:"size:50"`
	Detail    string    `json:"detail" gorm:"size:500"`
	IP        string    `json:"ip" gorm:"size:50"`
	CreatedAt time.Time `json:"created_at"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
}

// ConfigVersion 配置版本模型
type ConfigVersion struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ClientID  uint      `json:"client_id" gorm:"uniqueIndex;not null"`
	Version   int       `json:"version" gorm:"default:1"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Client    Client    `json:"client" gorm:"foreignKey:ClientID"`
}

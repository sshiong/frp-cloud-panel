package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/frp-cloud-panel/server/internal/config"
	"github.com/frp-cloud-panel/server/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init 初始化数据库
func Init(cfg *config.DatabaseConfig) error {
	// 确保数据库目录存在
	dir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(cfg.Path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	// 启用 WAL 模式
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	_, err = sqlDB.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// 自动迁移数据库表
	if err := autoMigrate(); err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	// 初始化默认数据
	if err := initDefaultData(); err != nil {
		return fmt.Errorf("failed to init default data: %w", err)
	}

	log.Println("Database initialized successfully")
	return nil
}

// autoMigrate 自动迁移数据库表
func autoMigrate() error {
	return DB.AutoMigrate(
		&models.User{},
		&models.Client{},
		&models.ProxyMapping{},
		&models.Port{},
		&models.Domain{},
		&models.CloudflareToken{},
		&models.AuditLog{},
		&models.ConfigVersion{},
	)
}

// initDefaultData 初始化默认数据
func initDefaultData() error {
	// 检查是否已有管理员用户
	var count int64
	DB.Model(&models.User{}).Where("role = ?", "admin").Count(&count)
	if count > 0 {
		return nil
	}

	// 创建默认管理员用户
	// 密码: admin123 (bcrypt hash)
	admin := models.User{
		Username: "admin",
		Password: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // password
		Email:    "admin@example.com",
		Role:     "admin",
		Status:   "active",
	}

	if err := DB.Create(&admin).Error; err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	// 初始化端口范围 (10000-20000)
	ports := make([]models.Port, 0)
	for i := 10000; i <= 20000; i++ {
		ports = append(ports, models.Port{
			Port:   i,
			Status: "free",
		})
	}

	// 批量插入端口
	batchSize := 1000
	for i := 0; i < len(ports); i += batchSize {
		end := i + batchSize
		if end > len(ports) {
			end = len(ports)
		}
		if err := DB.CreateInBatches(ports[i:end], batchSize).Error; err != nil {
			return fmt.Errorf("failed to create ports: %w", err)
		}
	}

	log.Println("Default data initialized successfully")
	return nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}

// Close 关闭数据库连接
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

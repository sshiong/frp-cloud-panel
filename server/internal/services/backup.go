package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/frp-cloud-panel/server/internal/database"
	"github.com/frp-cloud-panel/server/internal/models"
)

// BackupService 数据备份服务
type BackupService struct {
	backupDir     string
	encryptionKey []byte
}

// BackupData 备份数据结构
type BackupData struct {
	Version   string      `json:"version"`
	Timestamp time.Time   `json:"timestamp"`
	Users     []models.User `json:"users"`
	Clients   []models.Client `json:"clients"`
	Mappings  []models.ProxyMapping `json:"mappings"`
	Domains   []models.Domain `json:"domains"`
	Ports     []models.Port `json:"ports"`
	AuditLogs []models.AuditLog `json:"audit_logs"`
}

// NewBackupService 创建新的备份服务
func NewBackupService(encryptionKey string) *BackupService {
	backupDir := "./data/backups"
	os.MkdirAll(backupDir, 0755)

	// 确保密钥长度为 32 字节（AES-256）
	key := make([]byte, 32)
	copy(key, []byte(encryptionKey))

	return &BackupService{
		backupDir:     backupDir,
		encryptionKey: key,
	}
}

// CreateBackup 创建备份
func (s *BackupService) CreateBackup(password string) (string, error) {
	// 收集数据
	data, err := s.collectData()
	if err != nil {
		return "", fmt.Errorf("failed to collect data: %w", err)
	}

	// 序列化数据
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	// 加密数据
	encryptedData, err := s.encrypt(jsonData, password)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt data: %w", err)
	}

	// 生成备份文件名
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("backup_%s.enc", timestamp)
	filepath := filepath.Join(s.backupDir, filename)

	// 保存备份文件
	if err := os.WriteFile(filepath, encryptedData, 0600); err != nil {
		return "", fmt.Errorf("failed to save backup file: %w", err)
	}

	// 记录审计日志
	database.DB.Create(&models.AuditLog{
		Action:   "create_backup",
		Resource: "backup",
		Detail:   filename,
	})

	return filepath, nil
}

// RestoreBackup 恢复备份
func (s *BackupService) RestoreBackup(filepath, password string) error {
	// 读取备份文件
	encryptedData, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// 解密数据
	jsonData, err := s.decrypt(encryptedData, password)
	if err != nil {
		return fmt.Errorf("failed to decrypt data: %w", err)
	}

	// 解析数据
	var data BackupData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	// 恢复数据
	if err := s.restoreData(&data); err != nil {
		return fmt.Errorf("failed to restore data: %w", err)
	}

	// 记录审计日志
	database.DB.Create(&models.AuditLog{
		Action:   "restore_backup",
		Resource: "backup",
		Detail:   filepath,
	})

	return nil
}

// ListBackups 列出备份文件
func (s *BackupService) ListBackups() ([]BackupInfo, error) {
	files, err := os.ReadDir(s.backupDir)
	if err != nil {
		return nil, err
	}

	var backups []BackupInfo
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".enc" {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		backups = append(backups, BackupInfo{
			Filename:  file.Name(),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}

	return backups, nil
}

// DeleteBackup 删除备份文件
func (s *BackupService) DeleteBackup(filename string) error {
	filepath := filepath.Join(s.backupDir, filename)
	if err := os.Remove(filepath); err != nil {
		return err
	}

	// 记录审计日志
	database.DB.Create(&models.AuditLog{
		Action:   "delete_backup",
		Resource: "backup",
		Detail:   filename,
	})

	return nil
}

// collectData 收集数据
func (s *BackupService) collectData() (*BackupData, error) {
	var data BackupData

	// 收集用户数据
	if err := database.DB.Find(&data.Users).Error; err != nil {
		return nil, err
	}

	// 收集客户端数据
	if err := database.DB.Find(&data.Clients).Error; err != nil {
		return nil, err
	}

	// 收集映射数据
	if err := database.DB.Find(&data.Mappings).Error; err != nil {
		return nil, err
	}

	// 收集域名数据
	if err := database.DB.Find(&data.Domains).Error; err != nil {
		return nil, err
	}

	// 收集端口数据
	if err := database.DB.Find(&data.Ports).Error; err != nil {
		return nil, err
	}

	// 收集审计日志数据
	if err := database.DB.Find(&data.AuditLogs).Error; err != nil {
		return nil, err
	}

	data.Version = "1.0.0"
	data.Timestamp = time.Now()

	return &data, nil
}

// restoreData 恢复数据
func (s *BackupService) restoreData(data *BackupData) error {
	// 开始事务
	tx := database.DB.Begin()

	// 清空现有数据
	if err := tx.Exec("DELETE FROM audit_logs").Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Exec("DELETE FROM ports").Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Exec("DELETE FROM domains").Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Exec("DELETE FROM proxy_mappings").Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Exec("DELETE FROM clients").Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Exec("DELETE FROM users").Error; err != nil {
		tx.Rollback()
		return err
	}

	// 恢复用户数据
	for _, user := range data.Users {
		if err := tx.Create(&user).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// 恢复客户端数据
	for _, client := range data.Clients {
		if err := tx.Create(&client).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// 恢复映射数据
	for _, mapping := range data.Mappings {
		if err := tx.Create(&mapping).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// 恢复域名数据
	for _, domain := range data.Domains {
		if err := tx.Create(&domain).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// 恢复端口数据
	for _, port := range data.Ports {
		if err := tx.Create(&port).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// 恢复审计日志数据
	for _, log := range data.AuditLogs {
		if err := tx.Create(&log).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// 提交事务
	return tx.Commit().Error
}

// encrypt 加密数据
func (s *BackupService) encrypt(data []byte, password string) ([]byte, error) {
	// 使用密码派生密钥
	key := s.deriveKey(password)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 生成随机 IV
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// 使用 CBC 模式加密
	stream := cipher.NewCBCEncrypter(block, iv)
	stream.CryptBlocks(data, s.padData(data))

	// 返回 IV + 密文
	return append(iv, data...), nil
}

// decrypt 解密数据
func (s *BackupService) decrypt(data []byte, password string) ([]byte, error) {
	// 使用密码派生密钥
	key := s.deriveKey(password)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 提取 IV
	if len(data) < aes.BlockSize {
		return nil, fmt.Errorf("invalid encrypted data")
	}
	iv := data[:aes.BlockSize]
	data = data[aes.BlockSize:]

	// 使用 CBC 模式解密
	stream := cipher.NewCBCDecrypter(block, iv)
	stream.CryptBlocks(data, data)

	// 去除填充
	return s.unpadData(data), nil
}

// deriveKey 派生密钥
func (s *BackupService) deriveKey(password string) []byte {
	// 简单的密钥派生，实际应该使用 PBKDF2 或 similar
	key := make([]byte, 32)
	copy(key, []byte(password))
	return key
}

// padData 填充数据
func (s *BackupService) padData(data []byte) []byte {
	padding := aes.BlockSize - len(data)%aes.BlockSize
	padtext := make([]byte, padding)
	for i := range padtext {
		padtext[i] = byte(padding)
	}
	return append(data, padtext...)
}

// unpadData 去除填充
func (s *BackupService) unpadData(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	padding := int(data[len(data)-1])
	return data[:len(data)-padding]
}

// BackupInfo 备份信息
type BackupInfo struct {
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

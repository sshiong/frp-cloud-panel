package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/frp-cloud-panel/server/internal/database"
	"github.com/frp-cloud-panel/server/internal/models"
	"github.com/frp-cloud-panel/server/pkg/cloudflare"
)

// CloudflareService Cloudflare 服务
type CloudflareService struct {
	encryptionKey []byte
}

// NewCloudflareService 创建新的 Cloudflare 服务
func NewCloudflareService(encryptionKey string) *CloudflareService {
	// 确保密钥长度为 32 字节（AES-256）
	key := make([]byte, 32)
	copy(key, []byte(encryptionKey))
	return &CloudflareService{
		encryptionKey: key,
	}
}

// SaveToken 保存 Cloudflare Token
func (s *CloudflareService) SaveToken(userID uint, token, email string) error {
	// 加密 Token
	encryptedToken, nonce, err := s.encrypt(token)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	// 保存或更新 Token
	var cfToken models.CloudflareToken
	if err := database.DB.Where("user_id = ?", userID).First(&cfToken).Error; err != nil {
		// 创建新的
		cfToken = models.CloudflareToken{
			UserID: userID,
			Token:  encryptedToken,
			Nonce:  nonce,
			Email:  email,
			Status: "active",
		}
		return database.DB.Create(&cfToken).Error
	}

	// 更新现有的
	cfToken.Token = encryptedToken
	cfToken.Nonce = nonce
	cfToken.Email = email
	cfToken.Status = "active"
	return database.DB.Save(&cfToken).Error
}

// GetToken 获取 Cloudflare Token
func (s *CloudflareService) GetToken(userID uint) (string, error) {
	var cfToken models.CloudflareToken
	if err := database.DB.Where("user_id = ?", userID).First(&cfToken).Error; err != nil {
		return "", fmt.Errorf("token not found")
	}

	// 解密 Token
	token, err := s.decrypt(cfToken.Token, cfToken.Nonce)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt token: %w", err)
	}

	return token, nil
}

// GetTokenStatus 获取 Token 状态
func (s *CloudflareService) GetTokenStatus(userID uint) (*models.CloudflareToken, error) {
	var cfToken models.CloudflareToken
	if err := database.DB.Where("user_id = ?", userID).First(&cfToken).Error; err != nil {
		return nil, err
	}
	return &cfToken, nil
}

// DeleteToken 删除 Token
func (s *CloudflareService) DeleteToken(userID uint) error {
	return database.DB.Where("user_id = ?", userID).Delete(&models.CloudflareToken{}).Error
}

// TestToken 测试 Token 是否有效
func (s *CloudflareService) TestToken(userID uint) (bool, error) {
	token, err := s.GetToken(userID)
	if err != nil {
		return false, err
	}

	client := cloudflare.NewClient(token)
	return client.ValidateToken()
}

// GetClient 获取 Cloudflare 客户端
func (s *CloudflareService) GetClient(userID uint) (*cloudflare.Client, error) {
	token, err := s.GetToken(userID)
	if err != nil {
		return nil, err
	}

	return cloudflare.NewClient(token), nil
}

// CreateDNSRecord 创建 DNS 记录
func (s *CloudflareService) CreateDNSRecord(userID uint, domain, ip string) (*cloudflare.DNSRecord, error) {
	client, err := s.GetClient(userID)
	if err != nil {
		return nil, err
	}

	return client.CreateOrUpdateARecord(domain, ip)
}

// DeleteDNSRecord 删除 DNS 记录
func (s *CloudflareService) DeleteDNSRecord(userID uint, domain string) error {
	client, err := s.GetClient(userID)
	if err != nil {
		return err
	}

	return client.DeleteARecord(domain)
}

// GetDNSRecords 获取 DNS 记录列表
func (s *CloudflareService) GetDNSRecords(userID uint, domain string) ([]cloudflare.DNSRecord, error) {
	client, err := s.GetClient(userID)
	if err != nil {
		return nil, err
	}

	// 查找 Zone
	zone, err := client.FindZoneByName(domain)
	if err != nil {
		return nil, err
	}

	return client.ListDNSRecords(zone.ID)
}

// encrypt 加密数据
func (s *CloudflareService) encrypt(plaintext string) (string, string, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", "", err
	}

	// 生成随机 nonce
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", err
	}

	// 使用 AES-GCM 加密
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}

	ciphertext := aesGCM.Seal(nil, nonce, []byte(plaintext), nil)

	// 编码为 base64
	encodedCiphertext := base64.StdEncoding.EncodeToString(ciphertext)
	encodedNonce := base64.StdEncoding.EncodeToString(nonce)

	return encodedCiphertext, encodedNonce, nil
}

// decrypt 解密数据
func (s *CloudflareService) decrypt(ciphertext, nonce string) (string, error) {
	// 解码 base64
	decodedCiphertext, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	decodedNonce, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return "", err
	}

	// 创建 AES cipher
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	// 使用 AES-GCM 解密
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	plaintext, err := aesGCM.Open(nil, decodedNonce, decodedCiphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

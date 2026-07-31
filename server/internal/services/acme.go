package services

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/frp-cloud-panel/server/internal/config"
	"github.com/frp-cloud-panel/server/internal/database"
	"github.com/frp-cloud-panel/server/internal/models"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

// ACMEService ACME 证书服务
type ACMEService struct {
	cfg      *config.Config
	manager  *autocert.Manager
	certDir  string
}

// NewACMEService 创建新的 ACME 服务
func NewACMEService(cfg *config.Config) *ACMEService {
	certDir := "./data/certs"
	os.MkdirAll(certDir, 0755)

	manager := &autocert.Manager{
		Cache:      autocert.DirCache(certDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(),
		Email:      cfg.Server.Host, // 使用服务器主机名作为邮箱
	}

	return &ACMEService{
		cfg:     cfg,
		manager: manager,
		certDir: certDir,
	}
}

// GetCertificate 获取证书
func (s *ACMEService) GetCertificate(domain string) (*tls.Certificate, error) {
	// 检查是否已有证书
	certPath := filepath.Join(s.certDir, domain+".crt")
	keyPath := filepath.Join(s.certDir, domain+".key")

	if _, err := os.Stat(certPath); err == nil {
		// 加载现有证书
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate: %w", err)
		}
		return &cert, nil
	}

	// 申请新证书
	return s.requestCertificate(domain)
}

// requestCertificate 申请证书
func (s *ACMEService) requestCertificate(domain string) (*tls.Certificate, error) {
	// 创建 ACME 客户端
	client := &acme.Client{
		DirectoryURL: "https://acme-v02.api.letsencrypt.org/directory",
	}

	// 生成私钥
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// 注册账户
	account := &acme.Account{
		Contact: []string{"mailto:admin@" + domain},
	}
	_, err = client.Register(context.Background(), account, func(tosURL string) bool {
		return true // 接受 TOS
	})
	if err != nil {
		// 忽略已注册错误
		if acmeErr, ok := err.(*acme.Error); ok && acmeErr.StatusCode == 400 {
			// 账户已存在，继续
		} else {
			return nil, fmt.Errorf("failed to register account: %w", err)
		}
	}

	// 创建授权
	auth, err := client.Authorize(context.Background(), domain)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorization: %w", err)
	}

	// 处理挑战
	for _, challenge := range auth.Challenges {
		if challenge.Type == "http-01" {
			// HTTP-01 验证
			token := challenge.Token
			keyAuth, err := client.HTTP01ChallengeResponse(token)
			if err != nil {
				return nil, fmt.Errorf("failed to create challenge response: %w", err)
			}

			// 设置验证路径
			challengePath := filepath.Join(s.certDir, ".well-known", "acme-challenge", token)
			os.MkdirAll(filepath.Dir(challengePath), 0755)
			if err := os.WriteFile(challengePath, []byte(keyAuth), 0644); err != nil {
				return nil, fmt.Errorf("failed to write challenge file: %w", err)
			}

			// 接受挑战
			_, err = client.Accept(context.Background(), challenge)
			if err != nil {
				return nil, fmt.Errorf("failed to accept challenge: %w", err)
			}

			// 等待验证完成
			_, err = client.WaitAuthorization(context.Background(), auth.URI)
			if err != nil {
				return nil, fmt.Errorf("authorization failed: %w", err)
			}

			break
		}
	}

	// 创建证书请求
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		DNSNames: []string{domain},
	}, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSR: %w", err)
	}

	// 申请证书
	cert, _, err := client.CreateOrderCert(context.Background(), auth.URI, csr, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// 保存证书
	certPath := filepath.Join(s.certDir, domain+".crt")
	keyPath := filepath.Join(s.certDir, domain+".key")

	// 保存证书
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert[0],
	})
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return nil, fmt.Errorf("failed to save certificate: %w", err)
	}

	// 保存私钥
	keyPEM, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	keyBlock := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyPEM,
	})
	if err := os.WriteFile(keyPath, keyBlock, 0600); err != nil {
		return nil, fmt.Errorf("failed to save private key: %w", err)
	}

	// 更新数据库中的证书状态
	s.updateCertStatus(domain, "active")

	// 返回证书
	tlsCert, err := tls.X509KeyPair(certPEM, keyBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS certificate: %w", err)
	}

	return &tlsCert, nil
}

// RenewCertificate 续期证书
func (s *ACMEService) RenewCertificate(domain string) error {
	// 检查证书是否需要续期
	certPath := filepath.Join(s.certDir, domain+".crt")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return fmt.Errorf("certificate not found")
	}

	// 加载证书
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("failed to read certificate: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// 检查是否在30天内过期
	if time.Until(cert.NotAfter) > 30*24*time.Hour {
		return nil // 还不需要续期
	}

	// 删除旧证书
	os.Remove(certPath)
	os.Remove(filepath.Join(s.certDir, domain+".key"))

	// 申请新证书
	_, err = s.requestCertificate(domain)
	return err
}

// CheckCertificates 检查证书状态
func (s *ACMEService) CheckCertificates() {
	// 获取所有域名
	var domains []models.Domain
	database.DB.Find(&domains)

	for _, domain := range domains {
		if domain.HTTPSMode == "auto" {
			// 检查证书状态
			certPath := filepath.Join(s.certDir, domain.Domain+".crt")
			if _, err := os.Stat(certPath); os.IsNotExist(err) {
				// 证书不存在，申请新证书
				go func(d models.Domain) {
					if _, err := s.requestCertificate(d.Domain); err != nil {
						log.Printf("Failed to request certificate for %s: %v", d.Domain, err)
						s.updateCertStatus(d.Domain, "error")
					}
				}(domain)
			} else {
				// 检查是否需要续期
				go func(d models.Domain) {
					if err := s.RenewCertificate(d.Domain); err != nil {
						log.Printf("Failed to renew certificate for %s: %v", d.Domain, err)
					}
				}(domain)
			}
		}
	}
}

// updateCertStatus 更新证书状态
func (s *ACMEService) updateCertStatus(domain, status string) {
	database.DB.Model(&models.Domain{}).Where("domain = ?", domain).Update("cert_status", status)
}

// GetCertExpiry 获取证书过期时间
func (s *ACMEService) GetCertExpiry(domain string) (*time.Time, error) {
	certPath := filepath.Join(s.certDir, domain+".crt")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return nil, nil
	}

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	return &cert.NotAfter, nil
}

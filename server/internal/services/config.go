package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/frp-cloud-panel/server/internal/database"
	"github.com/frp-cloud-panel/server/internal/models"
)

// ConfigService 配置版本控制服务
type ConfigService struct {
	wsHub interface {
		NotifyConfigChange(userID uint, clientID string, version int)
	}
}

// NewConfigService 创建新的配置服务
func NewConfigService(wsHub interface {
	NotifyConfigChange(userID uint, clientID string, version int)
}) *ConfigService {
	return &ConfigService{
		wsHub: wsHub,
	}
}

// GetConfigVersion 获取配置版本
func (s *ConfigService) GetConfigVersion(clientID uint) (int, error) {
	var configVersion models.ConfigVersion
	if err := database.DB.Where("client_id = ?", clientID).First(&configVersion).Error; err != nil {
		return 0, err
	}
	return configVersion.Version, nil
}

// IncrementConfigVersion 增加配置版本
func (s *ConfigService) IncrementConfigVersion(clientID uint) (int, error) {
	var configVersion models.ConfigVersion
	if err := database.DB.Where("client_id = ?", clientID).First(&configVersion).Error; err != nil {
		// 创建新的配置版本
		configVersion = models.ConfigVersion{
			ClientID: clientID,
			Version:  1,
		}
		if err := database.DB.Create(&configVersion).Error; err != nil {
			return 0, err
		}
		return 1, nil
	}

	// 增加版本号
	configVersion.Version++
	if err := database.DB.Save(&configVersion).Error; err != nil {
		return 0, err
	}

	// 通知客户端配置变更
	s.notifyConfigChange(clientID, configVersion.Version)

	return configVersion.Version, nil
}

// GetDesiredConfig 获取期望配置
func (s *ConfigService) GetDesiredConfig(clientID uint) (*ClientConfig, error) {
	// 获取客户端信息
	var client models.Client
	if err := database.DB.First(&client, clientID).Error; err != nil {
		return nil, err
	}

	// 获取映射列表
	var mappings []models.ProxyMapping
	if err := database.DB.Where("client_id = ? AND status != ?", clientID, "deleting").Find(&mappings).Error; err != nil {
		return nil, err
	}

	// 获取配置版本
	version, err := s.GetConfigVersion(clientID)
	if err != nil {
		return nil, err
	}

	// 构建配置
	config := &ClientConfig{
		ClientID:   client.ClientID,
		Version:    version,
		Mappings:   make([]MappingConfig, 0),
		ServerAddr: "0.0.0.0:8080", // TODO: 从配置获取
	}

	for _, mapping := range mappings {
		config.Mappings = append(config.Mappings, MappingConfig{
			ID:         mapping.ID,
			Name:       mapping.Name,
			Type:       mapping.Type,
			LocalIP:    mapping.LocalIP,
			LocalPort:  mapping.LocalPort,
			RemotePort: mapping.RemotePort,
			Domain:     mapping.Domain,
			Status:     mapping.Status,
		})
	}

	return config, nil
}

// ApplyConfig 应用配置
func (s *ConfigService) ApplyConfig(clientID uint, version int, status string, errMsg string) error {
	// 获取客户端信息
	var client models.Client
	if err := database.DB.First(&client, clientID).Error; err != nil {
		return err
	}

	// 更新映射状态
	var mappings []models.ProxyMapping
	if err := database.DB.Where("client_id = ?", clientID).Find(&mappings).Error; err != nil {
		return err
	}

	for i := range mappings {
		mappings[i].AppliedConfigVersion = version
		if status == "success" {
			mappings[i].Status = "running"
		} else {
			mappings[i].Status = "config_error"
		}
		database.DB.Save(&mappings[i])
	}

	// 记录审计日志
	database.DB.Create(&models.AuditLog{
		UserID:   client.UserID,
		Action:   "apply_config",
		Resource: "config",
		Detail:   fmt.Sprintf("Version: %d, Status: %s", version, status),
	})

	return nil
}

// SyncConfig 同步配置
func (s *ConfigService) SyncConfig(clientID uint) error {
	// 获取期望配置版本
	desiredVersion, err := s.GetConfigVersion(clientID)
	if err != nil {
		return err
	}

	// 获取应用的配置版本
	var mapping models.ProxyMapping
	if err := database.DB.Where("client_id = ?", clientID).First(&mapping).Error; err != nil {
		return err
	}

	// 检查是否需要同步
	if desiredVersion > mapping.AppliedConfigVersion {
		// 通知客户端同步配置
		s.notifyConfigChange(clientID, desiredVersion)
	}

	return nil
}

// notifyConfigChange 通知配置变更
func (s *ConfigService) notifyConfigChange(clientID uint, version int) {
	// 获取客户端信息
	var client models.Client
	if err := database.DB.First(&client, clientID).Error; err != nil {
		log.Printf("Failed to get client info: %v", err)
		return
	}

	// 通过 WebSocket 通知
	if s.wsHub != nil {
		s.wsHub.NotifyConfigChange(client.UserID, client.ClientID, version)
	}
}

// CheckConfigSync 检查配置同步状态
func (s *ConfigService) CheckConfigSync(clientID uint) (bool, error) {
	// 获取期望配置版本
	desiredVersion, err := s.GetConfigVersion(clientID)
	if err != nil {
		return false, err
	}

	// 获取应用的配置版本
	var mapping models.ProxyMapping
	if err := database.DB.Where("client_id = ?", clientID).First(&mapping).Error; err != nil {
		return false, err
	}

	return desiredVersion > mapping.AppliedConfigVersion, nil
}

// ClientConfig 客户端配置
type ClientConfig struct {
	ClientID   string          `json:"client_id"`
	Version    int             `json:"version"`
	Mappings   []MappingConfig `json:"mappings"`
	ServerAddr string          `json:"server_addr"`
}

// MappingConfig 映射配置
type MappingConfig struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	LocalIP    string `json:"local_ip"`
	LocalPort  int    `json:"local_port"`
	RemotePort int    `json:"remote_port"`
	Domain     string `json:"domain"`
	Status     string `json:"status"`
}

// GenerateFRPCConfig 生成 FRPC 配置文件内容
func (s *ConfigService) GenerateFRPCConfig(clientID uint) (string, error) {
	config, err := s.GetDesiredConfig(clientID)
	if err != nil {
		return "", err
	}

	// 生成配置内容
	content := fmt.Sprintf(`# FRPC Configuration
# Generated by FRP Cloud Panel
# Version: %d
# Generated at: %s

serverAddr = "%s"
serverPort = 7000

[proxies]
`, config.Version, time.Now().Format(time.RFC3339), config.ServerAddr)

	for _, mapping := range config.Mappings {
		if mapping.Status == "deleting" || mapping.Status == "disabled" {
			continue
		}

		switch mapping.Type {
		case "tcp":
			content += fmt.Sprintf(`
[[proxies]]
name = "%s"
type = "tcp"
localIP = "%s"
localPort = %d
remotePort = %d
`, mapping.Name, mapping.LocalIP, mapping.LocalPort, mapping.RemotePort)

		case "udp":
			content += fmt.Sprintf(`
[[proxies]]
name = "%s"
type = "udp"
localIP = "%s"
localPort = %d
remotePort = %d
`, mapping.Name, mapping.LocalIP, mapping.LocalPort, mapping.RemotePort)

		case "http":
			content += fmt.Sprintf(`
[[proxies]]
name = "%s"
type = "http"
localIP = "%s"
localPort = %d
`, mapping.Name, mapping.LocalIP, mapping.LocalPort)

			if mapping.Domain != "" {
				content += fmt.Sprintf(`customDomains = ["%s"]
`, mapping.Domain)
			}

		case "https":
			content += fmt.Sprintf(`
[[proxies]]
name = "%s"
type = "https"
localIP = "%s"
localPort = %d
`, mapping.Name, mapping.LocalIP, mapping.LocalPort)

			if mapping.Domain != "" {
				content += fmt.Sprintf(`customDomains = ["%s"]
`, mapping.Domain)
			}
		}
	}

	return content, nil
}

// ExportConfig 导出配置
func (s *ConfigService) ExportConfig(clientID uint) ([]byte, error) {
	config, err := s.GetDesiredConfig(clientID)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(config, "", "  ")
}

// ImportConfig 导入配置
func (s *ConfigService) ImportConfig(clientID uint, data []byte) error {
	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// 更新映射
	for _, mappingConfig := range config.Mappings {
		var mapping models.ProxyMapping
		if err := database.DB.Where("id = ? AND client_id = ?", mappingConfig.ID, clientID).First(&mapping).Error; err != nil {
			// 创建新映射
			mapping = models.ProxyMapping{
				ClientID:   clientID,
				Name:       mappingConfig.Name,
				Type:       mappingConfig.Type,
				LocalIP:    mappingConfig.LocalIP,
				LocalPort:  mappingConfig.LocalPort,
				RemotePort: mappingConfig.RemotePort,
				Domain:     mappingConfig.Domain,
				Status:     "pending_apply",
			}
			database.DB.Create(&mapping)
		} else {
			// 更新现有映射
			mapping.Name = mappingConfig.Name
			mapping.Type = mappingConfig.Type
			mapping.LocalIP = mappingConfig.LocalIP
			mapping.LocalPort = mappingConfig.LocalPort
			mapping.RemotePort = mappingConfig.RemotePort
			mapping.Domain = mappingConfig.Domain
			mapping.Status = "pending_apply"
			database.DB.Save(&mapping)
		}
	}

	// 增加配置版本
	_, err := s.IncrementConfigVersion(clientID)
	return err
}

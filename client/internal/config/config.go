package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config 客户端配置
type Config struct {
	Server   ServerConfig   `json:"server"`
	FRPC     FRPCConfig     `json:"frpc"`
	Log      LogConfig      `json:"log"`
	Device   DeviceConfig   `json:"device"`
}

// ServerConfig 服务端配置
type ServerConfig struct {
	Address string `json:"address"`
}

// FRPCConfig FRPC 配置
type FRPCConfig struct {
	Path     string `json:"path"`
	ConfigPath string `json:"config_path"`
	LogPath  string `json:"log_path"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level string `json:"level"`
	File  string `json:"file"`
}

// DeviceConfig 设备配置
type DeviceConfig struct {
	ClientID    string `json:"client_id"`
	DeviceToken string `json:"device_token"`
	DeviceName  string `json:"device_name"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Address: "http://localhost:8080",
		},
		FRPC: FRPCConfig{
			Path:       "frpc",
			ConfigPath: "./data/frpc.toml",
			LogPath:    "./logs/frpc.log",
		},
		Log: LogConfig{
			Level: "info",
			File:  "./logs/client.log",
		},
		Device: DeviceConfig{
			ClientID:    "",
			DeviceToken: "",
			DeviceName:  "",
		},
	}
}

// LoadConfig 从文件加载配置
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(config *Config, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// IsRegistered 检查是否已注册
func (c *DeviceConfig) IsRegistered() bool {
	return c.ClientID != "" && c.DeviceToken != ""
}

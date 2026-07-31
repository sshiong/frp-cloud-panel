package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/frp-cloud-panel/client/internal/config"
)

// Client API 客户端
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewClient 创建新的 API 客户端
func NewClient(cfg *config.Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Register 注册设备
func (c *Client) Register(username, password, deviceName string) (*RegisterResponse, error) {
	req := RegisterRequest{
		Username:   username,
		Password:   password,
		DeviceName: deviceName,
	}

	var resp RegisterResponse
	if err := c.doRequest("POST", "/api/v1/client/register", req, &resp, true); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetConfig 获取配置
func (c *Client) GetConfig() (*ConfigResponse, error) {
	var resp ConfigResponse
	if err := c.doRequest("GET", "/api/v1/client/config", nil, &resp, false); err != nil {
		return nil, err
	}

	return &resp, nil
}

// ApplyConfig 应用配置结果
func (c *Client) ApplyConfig(version int, status string, errMsg string) error {
	req := ApplyConfigRequest{
		Version: version,
		Status:  status,
		Error:   errMsg,
	}

	var resp struct {
		Message string `json:"message"`
	}
	if err := c.doRequest("POST", "/api/v1/client/config/apply", req, &resp, false); err != nil {
		return err
	}

	return nil
}

// UpdateStatus 更新状态
func (c *Client) UpdateStatus(mappings []MappingStatus) error {
	req := UpdateStatusRequest{
		Mappings: mappings,
	}

	var resp struct {
		Message string `json:"message"`
	}
	if err := c.doRequest("POST", "/api/v1/client/status", req, &resp, false); err != nil {
		return err
	}

	return nil
}

// Heartbeat 心跳
func (c *Client) Heartbeat() (*HeartbeatResponse, error) {
	var resp HeartbeatResponse
	if err := c.doRequest("POST", "/api/v1/client/heartbeat", nil, &resp, false); err != nil {
		return nil, err
	}

	return &resp, nil
}

// doRequest 执行请求
func (c *Client) doRequest(method, path string, reqBody interface{}, respBody interface{}, ignoreAuth bool) error {
	url := c.cfg.Server.Address + path

	var body io.Reader
	if reqBody != nil {
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 设置设备认证
	if !ignoreAuth && c.cfg.Device.IsRegistered() {
		req.Header.Set("X-Client-ID", c.cfg.Device.ClientID)
		req.Header.Set("X-Device-Token", c.cfg.Device.DeviceToken)
	}

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(respData, &errResp); err != nil {
			return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respData))
		}
		return fmt.Errorf("request failed: %s", errResp.Message)
	}

	// 解析响应
	if respBody != nil {
		if err := json.Unmarshal(respData, respBody); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	DeviceName string `json:"device_name"`
}

// RegisterResponse 注册响应
type RegisterResponse struct {
	ClientID    string `json:"client_id"`
	DeviceToken string `json:"device_token"`
}

// ConfigResponse 配置响应
type ConfigResponse struct {
	ClientID   string    `json:"client_id"`
	Version    int       `json:"version"`
	Mappings   []Mapping `json:"mappings"`
	ServerAddr string    `json:"server_addr"`
}

// Mapping 映射配置
type Mapping struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	LocalIP    string `json:"local_ip"`
	LocalPort  int    `json:"local_port"`
	RemotePort int    `json:"remote_port"`
	Domain     string `json:"domain"`
	Status     string `json:"status"`
}

// ApplyConfigRequest 应用配置请求
type ApplyConfigRequest struct {
	Version int    `json:"version"`
	Status  string `json:"status"`
	Error   string `json:"error"`
}

// UpdateStatusRequest 更新状态请求
type UpdateStatusRequest struct {
	Mappings []MappingStatus `json:"mappings"`
}

// MappingStatus 映射状态
type MappingStatus struct {
	ID     uint   `json:"id"`
	Status string `json:"status"`
}

// HeartbeatResponse 心跳响应
type HeartbeatResponse struct {
	Status   string `json:"status"`
	NeedSync bool   `json:"need_sync"`
	Version  int    `json:"version"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

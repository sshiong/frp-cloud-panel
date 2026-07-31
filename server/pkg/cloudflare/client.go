package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	baseURL = "https://api.cloudflare.com/client/v4"
)

// Client Cloudflare API 客户端
type Client struct {
	token      string
	httpClient *http.Client
}

// NewClient 创建新的 Cloudflare 客户端
func NewClient(token string) *Client {
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Zone Cloudflare Zone
type Zone struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Paused  bool   `json:"paused"`
}

// DNSRecord DNS 记录
type DNSRecord struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

// APIResponse API 响应
type APIResponse struct {
	Success bool            `json:"success"`
	Errors  []APIError      `json:"errors"`
	Result  json.RawMessage `json:"result"`
}

// APIError API 错误
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ListZones 获取 Zone 列表
func (c *Client) ListZones() ([]Zone, error) {
	var zones []Zone
	page := 1

	for {
		url := fmt.Sprintf("%s/zones?page=%d&per_page=50", baseURL, page)
		resp, err := c.doRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		var result struct {
			Result []Zone `json:"result"`
		}
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return nil, err
		}

		zones = append(zones, result.Result...)

		// 检查是否还有更多页
		if len(result.Result) < 50 {
			break
		}
		page++
	}

	return zones, nil
}

// GetZone 获取 Zone 详情
func (c *Client) GetZone(zoneID string) (*Zone, error) {
	url := fmt.Sprintf("%s/zones/%s", baseURL, zoneID)
	resp, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var zone Zone
	if err := json.Unmarshal(resp.Result, &zone); err != nil {
		return nil, err
	}

	return &zone, nil
}

// FindZoneByName 根据域名查找 Zone
func (c *Client) FindZoneByName(domain string) (*Zone, error) {
	// 获取所有 Zone
	zones, err := c.ListZones()
	if err != nil {
		return nil, err
	}

	// 转换为小写
	domain = strings.ToLower(domain)

	// 查找匹配的 Zone
	var bestMatch *Zone
	bestMatchLen := 0

	for _, zone := range zones {
		zoneName := strings.ToLower(zone.Name)

		// 检查域名是否以 Zone 名称结尾
		if strings.HasSuffix(domain, "."+zoneName) || domain == zoneName {
			// 选择最长的匹配（最具体的）
			if len(zoneName) > bestMatchLen {
				bestMatch = &zone
				bestMatchLen = len(zoneName)
			}
		}
	}

	if bestMatch == nil {
		return nil, fmt.Errorf("no zone found for domain: %s", domain)
	}

	return bestMatch, nil
}

// ListDNSRecords 获取 DNS 记录列表
func (c *Client) ListDNSRecords(zoneID string) ([]DNSRecord, error) {
	var records []DNSRecord
	page := 1

	for {
		url := fmt.Sprintf("%s/zones/%s/dns_records?page=%d&per_page=100", baseURL, zoneID, page)
		resp, err := c.doRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		var result struct {
			Result []DNSRecord `json:"result"`
		}
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return nil, err
		}

		records = append(records, result.Result...)

		// 检查是否还有更多页
		if len(result.Result) < 100 {
			break
		}
		page++
	}

	return records, nil
}

// GetDNSRecord 获取 DNS 记录详情
func (c *Client) GetDNSRecord(zoneID, recordID string) (*DNSRecord, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", baseURL, zoneID, recordID)
	resp, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var record DNSRecord
	if err := json.Unmarshal(resp.Result, &record); err != nil {
		return nil, err
	}

	return &record, nil
}

// CreateDNSRecord 创建 DNS 记录
func (c *Client) CreateDNSRecord(zoneID string, record *DNSRecord) (*DNSRecord, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records", baseURL, zoneID)

	body, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	var createdRecord DNSRecord
	if err := json.Unmarshal(resp.Result, &createdRecord); err != nil {
		return nil, err
	}

	return &createdRecord, nil
}

// UpdateDNSRecord 更新 DNS 记录
func (c *Client) UpdateDNSRecord(zoneID, recordID string, record *DNSRecord) (*DNSRecord, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", baseURL, zoneID, recordID)

	body, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	var updatedRecord DNSRecord
	if err := json.Unmarshal(resp.Result, &updatedRecord); err != nil {
		return nil, err
	}

	return &updatedRecord, nil
}

// DeleteDNSRecord 删除 DNS 记录
func (c *Client) DeleteDNSRecord(zoneID, recordID string) error {
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", baseURL, zoneID, recordID)

	_, err := c.doRequest("DELETE", url, nil)
	return err
}

// FindDNSRecordByName 根据名称查找 DNS 记录
func (c *Client) FindDNSRecordByName(zoneID, name string) (*DNSRecord, error) {
	records, err := c.ListDNSRecords(zoneID)
	if err != nil {
		return nil, err
	}

	name = strings.ToLower(name)

	for _, record := range records {
		if strings.ToLower(record.Name) == name {
			return &record, nil
		}
	}

	return nil, nil
}

// CreateOrUpdateARecord 创建或更新 A 记录
func (c *Client) CreateOrUpdateARecord(domain, ip string) (*DNSRecord, error) {
	// 查找 Zone
	zone, err := c.FindZoneByName(domain)
	if err != nil {
		return nil, err
	}

	// 查找现有记录
	existingRecord, err := c.FindDNSRecordByName(zone.ID, domain)
	if err != nil {
		return nil, err
	}

	record := &DNSRecord{
		Type:    "A",
		Name:    domain,
		Content: ip,
		TTL:     1, // 自动
		Proxied: false,
	}

	if existingRecord != nil {
		// 更新现有记录
		return c.UpdateDNSRecord(zone.ID, existingRecord.ID, record)
	}

	// 创建新记录
	return c.CreateDNSRecord(zone.ID, record)
}

// DeleteARecord 删除 A 记录
func (c *Client) DeleteARecord(domain string) error {
	// 查找 Zone
	zone, err := c.FindZoneByName(domain)
	if err != nil {
		return err
	}

	// 查找现有记录
	record, err := c.FindDNSRecordByName(zone.ID, domain)
	if err != nil {
		return err
	}

	if record == nil {
		return fmt.Errorf("record not found: %s", domain)
	}

	return c.DeleteDNSRecord(zone.ID, record.ID)
}

// ValidateToken 验证 Token 是否有效
func (c *Client) ValidateToken() (bool, error) {
	url := fmt.Sprintf("%s/user/tokens/verify", baseURL)

	resp, err := c.doRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	return resp.Success, nil
}

// doRequest 执行请求
func (c *Client) doRequest(method, url string, body io.Reader) (*APIResponse, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 解析响应
	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 检查错误
	if !apiResp.Success {
		if len(apiResp.Errors) > 0 {
			return nil, fmt.Errorf("API error: %s", apiResp.Errors[0].Message)
		}
		return nil, fmt.Errorf("API request failed")
	}

	return &apiResp, nil
}

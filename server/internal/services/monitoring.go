package services

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/frp-cloud-panel/server/internal/database"
	"github.com/frp-cloud-panel/server/internal/models"
)

// MonitoringService 系统监控服务
type MonitoringService struct {
	alertRules []AlertRule
	alerts     []Alert
	mu         sync.RWMutex
}

// AlertRule 告警规则
type AlertRule struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Metric    string        `json:"metric"`
	Operator  string        `json:"operator"` // >, <, >=, <=, ==, !=
	Threshold float64       `json:"threshold"`
	Duration  time.Duration `json:"duration"`
	Enabled   bool          `json:"enabled"`
}

// Alert 告警
type Alert struct {
	ID        string    `json:"id"`
	RuleID    string    `json:"rule_id"`
	RuleName  string    `json:"rule_name"`
	Message   string    `json:"message"`
	Severity  string    `json:"severity"` // info, warning, error, critical
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	Resolved  bool      `json:"resolved"`
}

// SystemStats 系统统计
type SystemStats struct {
	CPUUsage       float64 `json:"cpu"`
	MemoryUsage    float64 `json:"memory"`
	DiskUsage      float64 `json:"disk"`
	OnlineClients  int     `json:"onlineClients"`
	ActiveMappings int     `json:"activeMappings"`
	WSConnections  int     `json:"wsConnections"`
	AvgResponseTime int    `json:"avgResponseTime"`
	TotalRequests  int     `json:"totalRequests"`
	ErrorRate      float64 `json:"errorRate"`
	DBSize         string  `json:"dbSize"`
	DBQueries      int     `json:"dbQueries"`
	DBConnections  int     `json:"dbConnections"`
}

// NewMonitoringService 创建新的监控服务
func NewMonitoringService() *MonitoringService {
	service := &MonitoringService{
		alertRules: getDefaultAlertRules(),
		alerts:     make([]Alert, 0),
	}

	// 启动监控协程
	go service.startMonitoring()

	return service
}

// getDefaultAlertRules 获取默认告警规则
func getDefaultAlertRules() []AlertRule {
	return []AlertRule{
		{
			ID:        "cpu_high",
			Name:      "CPU 使用率过高",
			Metric:    "cpu",
			Operator:  ">",
			Threshold: 80,
			Duration:  5 * time.Minute,
			Enabled:   true,
		},
		{
			ID:        "memory_high",
			Name:      "内存使用率过高",
			Metric:    "memory",
			Operator:  ">",
			Threshold: 85,
			Duration:  5 * time.Minute,
			Enabled:   true,
		},
		{
			ID:        "disk_high",
			Name:      "磁盘使用率过高",
			Metric:    "disk",
			Operator:  ">",
			Threshold: 90,
			Duration:  1 * time.Minute,
			Enabled:   true,
		},
		{
			ID:        "error_rate_high",
			Name:      "错误率过高",
			Metric:    "errorRate",
			Operator:  ">",
			Threshold: 5,
			Duration:  5 * time.Minute,
			Enabled:   true,
		},
	}
}

// startMonitoring 启动监控
func (s *MonitoringService) startMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.collectStats()
		s.checkAlerts()
	}
}

// collectStats 收集统计数据
func (s *MonitoringService) collectStats() {
	// 这里可以收集实际的系统统计数据
	// 目前返回模拟数据
}

// checkAlerts 检查告警
func (s *MonitoringService) checkAlerts() {
	stats := s.GetSystemStats()

	for _, rule := range s.alertRules {
		if !rule.Enabled {
			continue
		}

		var value float64
		switch rule.Metric {
		case "cpu":
			value = stats.CPUUsage
		case "memory":
			value = stats.MemoryUsage
		case "disk":
			value = stats.DiskUsage
		case "errorRate":
			value = stats.ErrorRate
		default:
			continue
		}

		if s.evaluateCondition(value, rule.Operator, rule.Threshold) {
			s.triggerAlert(rule, value)
		}
	}
}

// evaluateCondition 评估条件
func (s *MonitoringService) evaluateCondition(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return false
	}
}

// triggerAlert 触发告警
func (s *MonitoringService) triggerAlert(rule AlertRule, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已存在相同的未解决告警
	for _, alert := range s.alerts {
		if alert.RuleID == rule.ID && !alert.Resolved {
			return
		}
	}

	alert := Alert{
		ID:        generateAlertID(),
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Message:   rule.Name + ": 当前值 " + formatFloat(value) + " 超过阈值 " + formatFloat(rule.Threshold),
		Severity:  s.getSeverity(rule.ID),
		Value:     value,
		Timestamp: time.Now(),
		Resolved:  false,
	}

	s.alerts = append(s.alerts, alert)

	// 记录审计日志
	database.DB.Create(&models.AuditLog{
		Action:   "alert_triggered",
		Resource: "monitoring",
		Detail:   alert.Message,
	})

	log.Printf("Alert triggered: %s", alert.Message)

	// 发送通知（可扩展）
	s.sendNotification(alert)
}

// getSeverity 获取严重程度
func (s *MonitoringService) getSeverity(ruleID string) string {
	switch ruleID {
	case "cpu_high":
		return "warning"
	case "memory_high":
		return "warning"
	case "disk_high":
		return "error"
	case "error_rate_high":
		return "error"
	default:
		return "info"
	}
}

// sendNotification 发送通知
func (s *MonitoringService) sendNotification(alert Alert) {
	// 这里可以实现通知逻辑
	// 例如：发送邮件、Webhook、短信等
	log.Printf("Notification sent for alert: %s", alert.Message)
}

// GetSystemStats 获取系统统计
func (s *MonitoringService) GetSystemStats() *SystemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 获取在线客户端数
	var onlineClients int64
	database.DB.Model(&models.Client{}).Where("status = ?", "active").Count(&onlineClients)

	// 获取活跃映射数
	var activeMappings int64
	database.DB.Model(&models.ProxyMapping{}).Where("status = ?", "running").Count(&activeMappings)

	return &SystemStats{
		CPUUsage:       getCPUUsage(),
		MemoryUsage:    getMemoryUsage(m),
		DiskUsage:      getDiskUsage(),
		OnlineClients:  int(onlineClients),
		ActiveMappings: int(activeMappings),
		WSConnections:  0, // 需要从 WebSocket Hub 获取
		AvgResponseTime: 0, // 需要从性能监控获取
		TotalRequests:  0, // 需要从请求计数器获取
		ErrorRate:      0, // 需要从错误计数器获取
		DBSize:         getDBSize(),
		DBQueries:      0, // 需要从数据库监控获取
		DBConnections:  0, // 需要从连接池获取
	}
}

// GetAlerts 获取告警列表
func (s *MonitoringService) GetAlerts(resolved bool) []Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var alerts []Alert
	for _, alert := range s.alerts {
		if alert.Resolved == resolved {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

// ResolveAlert 解决告警
func (s *MonitoringService) ResolveAlert(alertID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, alert := range s.alerts {
		if alert.ID == alertID {
			s.alerts[i].Resolved = true

			// 记录审计日志
			database.DB.Create(&models.AuditLog{
				Action:   "alert_resolved",
				Resource: "monitoring",
				Detail:   alert.Message,
			})

			return nil
		}
	}

	return nil
}

// AddAlertRule 添加告警规则
func (s *MonitoringService) AddAlertRule(rule AlertRule) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.alertRules = append(s.alertRules, rule)
}

// UpdateAlertRule 更新告警规则
func (s *MonitoringService) UpdateAlertRule(ruleID string, rule AlertRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, r := range s.alertRules {
		if r.ID == ruleID {
			s.alertRules[i] = rule
			return nil
		}
	}

	return nil
}

// DeleteAlertRule 删除告警规则
func (s *MonitoringService) DeleteAlertRule(ruleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, rule := range s.alertRules {
		if rule.ID == ruleID {
			s.alertRules = append(s.alertRules[:i], s.alertRules[i+1:]...)
			return nil
		}
	}

	return nil
}

// GetAlertRules 获取告警规则列表
func (s *MonitoringService) GetAlertRules() []AlertRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.alertRules
}

// getCPUUsage 获取 CPU 使用率
func getCPUUsage() float64 {
	// 这里应该实现实际的 CPU 使用率获取
	// 目前返回模拟数据
	return 45.0
}

// getMemoryUsage 获取内存使用率
func getMemoryUsage(m runtime.MemStats) float64 {
	// 计算内存使用率
	totalMemory := float64(m.Sys)
	usedMemory := float64(m.Alloc)
	return (usedMemory / totalMemory) * 100
}

// getDiskUsage 获取磁盘使用率
func getDiskUsage() float64 {
	// 这里应该实现实际的磁盘使用率获取
	// 目前返回模拟数据
	return 38.0
}

// getDBSize 获取数据库大小
func getDBSize() string {
	// 这里应该实现实际的数据库大小获取
	// 目前返回模拟数据
	return "2.5 MB"
}

// generateAlertID 生成告警 ID
func generateAlertID() string {
	return "alert_" + time.Now().Format("20060102150405")
}

// formatFloat 格式化浮点数
func formatFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

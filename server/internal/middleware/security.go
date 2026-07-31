package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SecurityConfig 安全配置
type SecurityConfig struct {
	CSRFEnabled        bool
	XSSProtection      bool
	ContentTypeNosniff bool
	FrameOptions       bool
	HSTSEnabled        bool
	HSTSMaxAge         int
}

// DefaultSecurityConfig 默认安全配置
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		CSRFEnabled:        true,
		XSSProtection:      true,
		ContentTypeNosniff: true,
		FrameOptions:       true,
		HSTSEnabled:        true,
		HSTSMaxAge:         31536000, // 1年
	}
}

// SecurityMiddleware 安全中间件
func SecurityMiddleware(config *SecurityConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置安全头部
		if config.XSSProtection {
			c.Header("X-XSS-Protection", "1; mode=block")
		}

		if config.ContentTypeNosniff {
			c.Header("X-Content-Type-Options", "nosniff")
		}

		if config.FrameOptions {
			c.Header("X-Frame-Options", "DENY")
		}

		if config.HSTSEnabled {
			c.Header("Strict-Transport-Security", fmt.Sprintf("max-age=%d; includeSubDomains", config.HSTSMaxAge))
		}

		// CSP 头部
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline';")

		// CSRF 防护（排除登录和注册端点）
		path := c.Request.URL.Path
		isAuthEndpoint := path == "/api/v1/auth/login" || path == "/api/v1/auth/register"

		if config.CSRFEnabled && !isAuthEndpoint && c.Request.Method != "GET" && c.Request.Method != "HEAD" && c.Request.Method != "OPTIONS" {
			csrfToken := c.GetHeader("X-CSRF-Token")
			if csrfToken == "" {
				c.JSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "CSRF token required",
				})
				c.Abort()
				return
			}

			// 验证 CSRF Token
			if !validateCSRFToken(csrfToken) {
				c.JSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "Invalid CSRF token",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// CSRF Token 存储
var csrfTokens = make(map[string]time.Time)
var csrfMutex sync.RWMutex

// GenerateCSRFToken 生成 CSRF Token
func GenerateCSRFToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	token := hex.EncodeToString(bytes)

	csrfMutex.Lock()
	csrfTokens[token] = time.Now().Add(1 * time.Hour) // 1小时过期
	csrfMutex.Unlock()

	return token
}

// validateCSRFToken 验证 CSRF Token
func validateCSRFToken(token string) bool {
	csrfMutex.RLock()
	expiry, exists := csrfTokens[token]
	csrfMutex.RUnlock()

	if !exists {
		return false
	}

	if time.Now().After(expiry) {
		// Token 已过期，删除它
		csrfMutex.Lock()
		delete(csrfTokens, token)
		csrfMutex.Unlock()
		return false
	}

	return true
}

// CleanupCSRFTokens 清理过期的 CSRF Token
func CleanupCSRFTokens() {
	for {
		time.Sleep(1 * time.Hour)
		csrfMutex.Lock()
		for token, expiry := range csrfTokens {
			if time.Now().After(expiry) {
				delete(csrfTokens, token)
			}
		}
		csrfMutex.Unlock()
	}
}

// SQLInjectionMiddleware SQL 注入防护中间件
func SQLInjectionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查查询参数
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				if containsSQLInjection(value) {
					c.JSON(http.StatusBadRequest, gin.H{
						"code":    400,
						"message": "Invalid input detected",
					})
					c.Abort()
					return
				}
			}
			_ = key
		}

		// 检查路径参数
		for _, param := range c.Params {
			if containsSQLInjection(param.Value) {
				c.JSON(http.StatusBadRequest, gin.H{
					"code":    400,
					"message": "Invalid input detected",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// containsSQLInjection 检查是否包含 SQL 注入
func containsSQLInjection(input string) bool {
	// 转换为小写进行检查
	lower := strings.ToLower(input)

	// 常见的 SQL 注入模式
	sqlPatterns := []string{
		"'",
		"--",
		";",
		"/*",
		"*/",
		"xp_",
		"exec",
		"execute",
		"insert",
		"select",
		"delete",
		"update",
		"drop",
		"truncate",
		"union",
		"into",
		"load_file",
		"outfile",
	}

	for _, pattern := range sqlPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

// XSSMiddleware XSS 防护中间件
func XSSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置 XSS 防护头部
		c.Header("X-XSS-Protection", "1; mode=block")

		// 检查输入
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				if containsXSS(value) {
					c.JSON(http.StatusBadRequest, gin.H{
						"code":    400,
						"message": "Invalid input detected",
					})
					c.Abort()
					return
				}
			}
			_ = key
		}

		c.Next()
	}
}

// containsXSS 检查是否包含 XSS
func containsXSS(input string) bool {
	// 转换为小写进行检查
	lower := strings.ToLower(input)

	// 常见的 XSS 模式
	xssPatterns := []string{
		"<script",
		"javascript:",
		"onload=",
		"onerror=",
		"onclick=",
		"onmouseover=",
		"onfocus=",
		"onblur=",
		"onsubmit=",
		"alert(",
		"document.cookie",
		"document.write",
		"eval(",
	}

	for _, pattern := range xssPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

// RateLimiter 限流器
type RateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.RWMutex
	rate     int
	burst    int
}

// Visitor 访问者
type Visitor struct {
	count    int
	lastSeen time.Time
}

// NewRateLimiter 创建新的限流器
func NewRateLimiter(rate, burst int) *RateLimiter {
	limiter := &RateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate,
		burst:    burst,
	}

	// 启动清理协程
	go limiter.cleanup()

	return limiter
}

// Allow 检查是否允许请求
func (l *RateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	visitor, exists := l.visitors[ip]
	if !exists {
		l.visitors[ip] = &Visitor{
			count:    1,
			lastSeen: time.Now(),
		}
		return true
	}

	// 检查是否需要重置
	if time.Since(visitor.lastSeen) > 1*time.Minute {
		visitor.count = 1
		visitor.lastSeen = time.Now()
		return true
	}

	// 检查是否超过限制
	if visitor.count >= l.burst {
		return false
	}

	visitor.count++
	visitor.lastSeen = time.Now()
	return true
}

// cleanup 清理过期的访问者
func (l *RateLimiter) cleanup() {
	for {
		time.Sleep(1 * time.Minute)
		l.mu.Lock()
		for ip, visitor := range l.visitors {
			if time.Since(visitor.lastSeen) > 2*time.Minute {
				delete(l.visitors, ip)
			}
		}
		l.mu.Unlock()
	}
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !limiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "Too many requests",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// InputValidationMiddleware 输入验证中间件
func InputValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查 Content-Type
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			contentType := c.GetHeader("Content-Type")
			if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
				c.JSON(http.StatusBadRequest, gin.H{
					"code":    400,
					"message": "Invalid Content-Type",
				})
				c.Abort()
				return
			}
		}

		// 检查请求大小
		if c.Request.ContentLength > 10*1024*1024 { // 10MB
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"code":    413,
				"message": "Request too large",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

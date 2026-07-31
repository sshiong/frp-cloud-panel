package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/frp-cloud-panel/server/internal/config"
	"github.com/frp-cloud-panel/server/internal/database"
	"github.com/frp-cloud-panel/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func setupTestServer() *Server {
	// 设置测试数据库
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
		JWT: config.JWTConfig{
			Secret:     "test-secret",
			Expiration: 24,
		},
	}

	// 初始化数据库
	database.Init(&cfg.Database)

	// 创建测试用户
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
	database.DB.Create(&models.User{
		Username: "testuser",
		Password: string(hashedPassword),
		Email:    "test@example.com",
		Role:     "user",
		Status:   "active",
	})

	return NewServer(cfg)
}

func TestHealthCheck(t *testing.T) {
	server := setupTestServer()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "ok", response["status"])
}

func TestUserLogin(t *testing.T) {
	server := setupTestServer()

	// 测试成功登录
	loginData := map[string]string{
		"username": "testuser",
		"password": "testpassword",
	}
	jsonData, _ := json.Marshal(loginData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, 0, int(response["code"].(float64)))
	assert.NotNil(t, response["data"])

	// 测试错误密码
	loginData["password"] = "wrongpassword"
	jsonData, _ = json.Marshal(loginData)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserRegistration(t *testing.T) {
	server := setupTestServer()

	// 测试成功注册
	registerData := map[string]string{
		"username": "newuser",
		"password": "newpassword",
		"email":    "new@example.com",
	}
	jsonData, _ := json.Marshal(registerData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, 0, int(response["code"].(float64)))

	// 测试重复用户名
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMappingCRUD(t *testing.T) {
	server := setupTestServer()

	// 先登录获取 Token
	loginData := map[string]string{
		"username": "testuser",
		"password": "testpassword",
	}
	jsonData, _ := json.Marshal(loginData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	server.router.ServeHTTP(w, req)

	var loginResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResponse)
	token := loginResponse["data"].(map[string]interface{})["token"].(string)

	// 测试创建映射
	mappingData := map[string]interface{}{
		"name":       "test-mapping",
		"type":       "tcp",
		"local_ip":   "127.0.0.1",
		"local_port": 8080,
	}
	jsonData, _ = json.Marshal(mappingData)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/mappings", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	assert.Equal(t, 0, int(createResponse["code"].(float64)))

	// 测试获取映射列表
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/mappings", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var listResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &listResponse)
	assert.Equal(t, 0, int(listResponse["code"].(float64)))
	assert.NotNil(t, listResponse["data"])
}

func TestDomainCRUD(t *testing.T) {
	server := setupTestServer()

	// 先登录获取 Token
	loginData := map[string]string{
		"username": "testuser",
		"password": "testpassword",
	}
	jsonData, _ := json.Marshal(loginData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	server.router.ServeHTTP(w, req)

	var loginResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResponse)
	token := loginResponse["data"].(map[string]interface{})["token"].(string)

	// 测试创建域名
	domainData := map[string]interface{}{
		"domain":     "test.example.com",
		"https_mode": "none",
	}
	jsonData, _ = json.Marshal(domainData)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/domains", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	assert.Equal(t, 0, int(createResponse["code"].(float64)))

	// 测试获取域名列表
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/domains", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var listResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &listResponse)
	assert.Equal(t, 0, int(listResponse["code"].(float64)))
}

func TestCloudflareToken(t *testing.T) {
	server := setupTestServer()

	// 先登录获取 Token
	loginData := map[string]string{
		"username": "testuser",
		"password": "testpassword",
	}
	jsonData, _ := json.Marshal(loginData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	server.router.ServeHTTP(w, req)

	var loginResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResponse)
	token := loginResponse["data"].(map[string]interface{})["token"].(string)

	// 测试设置 Cloudflare Token
	cfData := map[string]interface{}{
		"token": "test-cf-token",
		"email": "test@example.com",
	}
	jsonData, _ = json.Marshal(cfData)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/cloudflare/token", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 测试获取 Token 状态
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/cloudflare/token/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var statusResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &statusResponse)
	assert.Equal(t, 0, int(statusResponse["code"].(float64)))
}

func TestBackupAndRestore(t *testing.T) {
	server := setupTestServer()

	// 先登录获取 Token
	loginData := map[string]string{
		"username": "testuser",
		"password": "testpassword",
	}
	jsonData, _ := json.Marshal(loginData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	server.router.ServeHTTP(w, req)

	var loginResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResponse)
	token := loginResponse["data"].(map[string]interface{})["token"].(string)

	// 测试创建备份
	backupData := map[string]interface{}{
		"password": "backuppassword",
	}
	jsonData, _ = json.Marshal(backupData)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/backup/create", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 测试获取备份列表
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/backup/list", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitoring(t *testing.T) {
	server := setupTestServer()

	// 先登录获取 Token
	loginData := map[string]string{
		"username": "testuser",
		"password": "testpassword",
	}
	jsonData, _ := json.Marshal(loginData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	server.router.ServeHTTP(w, req)

	var loginResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResponse)
	token := loginResponse["data"].(map[string]interface{})["token"].(string)

	// 测试获取系统统计
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/monitoring/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var statsResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &statsResponse)
	assert.Equal(t, 0, int(statsResponse["code"].(float64)))

	// 测试获取告警规则
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/monitoring/rules", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

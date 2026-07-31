#!/bin/bash

# 测试 API 脚本

SERVER_URL="http://localhost:8080"

echo "=== Testing FRP Cloud Panel API ==="
echo ""

# 1. 测试健康检查
echo "1. Testing health endpoint..."
curl -s "$SERVER_URL/health"
echo ""
echo ""

# 2. 测试用户注册
echo "2. Testing user registration..."
curl -s -X POST "$SERVER_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "testpass123",
    "email": "test@example.com"
  }'
echo ""
echo ""

# 3. 测试用户登录
echo "3. Testing user login..."
LOGIN_RESPONSE=$(curl -s -X POST "$SERVER_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "password"
  }')
echo "$LOGIN_RESPONSE"
echo ""

# 提取 token
TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
echo "Extracted token: $TOKEN"
echo ""

# 4. 测试获取用户信息
echo "4. Testing get user info..."
curl -s -X GET "$SERVER_URL/api/v1/users/me" \
  -H "Authorization: Bearer $TOKEN"
echo ""
echo ""

# 5. 测试客户端注册
echo "5. Testing client registration..."
CLIENT_ID="test-client-$(date +%s)"
REGISTER_RESPONSE=$(curl -s -X POST "$SERVER_URL/api/v1/client/register" \
  -H "Content-Type: application/json" \
  -H "X-Client-ID: $CLIENT_ID" \
  -H "X-Device-Token: temp-token" \
  -d '{
    "username": "admin",
    "password": "password",
    "device_name": "Test Device"
  }')
echo "$REGISTER_RESPONSE"
echo ""

# 提取设备 token
DEVICE_TOKEN=$(echo "$REGISTER_RESPONSE" | grep -o '"device_token":"[^"]*"' | cut -d'"' -f4)
echo "Client ID: $CLIENT_ID"
echo "Device Token: $DEVICE_TOKEN"
echo ""

# 6. 测试获取配置
echo "6. Testing get client config..."
curl -s -X GET "$SERVER_URL/api/v1/client/config" \
  -H "X-Client-ID: $CLIENT_ID" \
  -H "X-Device-Token: $DEVICE_TOKEN"
echo ""
echo ""

# 7. 测试创建映射
echo "7. Testing create mapping..."
curl -s -X POST "$SERVER_URL/api/v1/mappings" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "test-mapping",
    "type": "tcp",
    "local_ip": "127.0.0.1",
    "local_port": 8080,
    "remote_port": 0
  }'
echo ""
echo ""

# 8. 测试获取映射列表
echo "8. Testing list mappings..."
curl -s -X GET "$SERVER_URL/api/v1/mappings" \
  -H "Authorization: Bearer $TOKEN"
echo ""
echo ""

# 9. 测试心跳
echo "9. Testing heartbeat..."
curl -s -X POST "$SERVER_URL/api/v1/client/heartbeat" \
  -H "X-Client-ID: $CLIENT_ID" \
  -H "X-Device-Token: $DEVICE_TOKEN"
echo ""
echo ""

echo "=== API Testing Complete ==="

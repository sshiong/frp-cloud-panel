#!/bin/bash

# DNS 管理功能测试脚本

SERVER_URL="http://localhost:8080"

echo "=== DNS 管理功能测试 ==="
echo ""

# 1. 用户登录
echo "1. 用户登录..."
LOGIN_RESPONSE=$(curl -s -X POST "$SERVER_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"password"}')

TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "登录失败"
    exit 1
fi
echo "登录成功"
echo ""

# 2. 设置 Cloudflare Token（测试用）
echo "2. 设置 Cloudflare Token..."
curl -s -X POST "$SERVER_URL/api/v1/cloudflare/token" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"token":"test-token","email":"test@example.com"}'
echo ""
echo ""

# 3. 获取 Token 状态
echo "3. 获取 Token 状态..."
curl -s -X GET "$SERVER_URL/api/v1/cloudflare/token/status" \
    -H "Authorization: Bearer $TOKEN"
echo ""
echo ""

# 4. 测试 Token
echo "4. 测试 Token..."
curl -s -X POST "$SERVER_URL/api/v1/cloudflare/token/test" \
    -H "Authorization: Bearer $TOKEN"
echo ""
echo ""

# 5. 获取 DNS 记录（需要有效 Token）
echo "5. 获取 DNS 记录..."
curl -s -X GET "$SERVER_URL/api/v1/dns/records?domain=example.com" \
    -H "Authorization: Bearer $TOKEN"
echo ""
echo ""

# 6. 创建 DNS 记录（需要有效 Token）
echo "6. 创建 DNS 记录..."
curl -s -X POST "$SERVER_URL/api/v1/dns/records" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"domain":"test.example.com","ip":"1.2.3.4"}'
echo ""
echo ""

# 7. 更新 DNS 记录（需要有效 Token）
echo "7. 更新 DNS 记录..."
curl -s -X PUT "$SERVER_URL/api/v1/dns/records/test.example.com" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"ip":"5.6.7.8"}'
echo ""
echo ""

# 8. 删除 DNS 记录（需要有效 Token）
echo "8. 删除 DNS 记录..."
curl -s -X DELETE "$SERVER_URL/api/v1/dns/records/test.example.com" \
    -H "Authorization: Bearer $TOKEN"
echo ""
echo ""

echo "=== 测试完成 ==="
echo ""
echo "注意：DNS 管理功能需要有效的 Cloudflare API Token 才能正常工作"
echo "当前测试使用的是测试 Token，实际操作会失败"

#!/bin/bash

# FRP Cloud Panel 系统测试脚本

set -e

SERVER_URL="http://localhost:8080"
FRONTEND_URL="http://localhost:3000"

echo "=== FRP Cloud Panel 系统测试 ==="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试函数
test_endpoint() {
    local name=$1
    local method=$2
    local url=$3
    local data=$4
    local headers=$5

    echo -n "测试 $name ... "

    if [ -n "$data" ]; then
        response=$(curl -s -X $method "$url" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $TOKEN" \
            -d "$data")
    else
        response=$(curl -s -X $method "$url" \
            -H "Authorization: Bearer $TOKEN")
    fi

    if echo "$response" | grep -q '"status":"ok"\|"code":0'; then
        echo -e "${GREEN}✓ 成功${NC}"
        return 0
    else
        echo -e "${RED}✗ 失败${NC}"
        echo "响应: $response"
        return 1
    fi
}

# 1. 测试服务端健康检查
echo "1. 服务端健康检查"
test_endpoint "服务端健康检查" "GET" "$SERVER_URL/health"
echo ""

# 2. 测试前端访问
echo "2. 前端访问测试"
echo -n "测试前端页面访问 ... "
if curl -s "$FRONTEND_URL" | grep -q "FRP Cloud Panel"; then
    echo -e "${GREEN}✓ 成功${NC}"
else
    echo -e "${RED}✗ 失败${NC}"
fi
echo ""

# 3. 测试用户登录
echo "3. 用户登录测试"
LOGIN_RESPONSE=$(curl -s -X POST "$SERVER_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"password"}')

if echo "$LOGIN_RESPONSE" | grep -q '"code":0'; then
    echo -e "用户登录 ... ${GREEN}✓ 成功${NC}"
    TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
else
    echo -e "用户登录 ... ${RED}✗ 失败${NC}"
    exit 1
fi
echo ""

# 4. 测试获取用户信息
echo "4. 获取用户信息测试"
test_endpoint "获取用户信息" "GET" "$SERVER_URL/api/v1/users/me" "" "-H \"Authorization: Bearer $TOKEN\""
echo ""

# 5. 测试客户端注册
echo "5. 客户端注册测试"
CLIENT_ID="test-client-$(date +%s)"
REGISTER_RESPONSE=$(curl -s -X POST "$SERVER_URL/api/v1/client/register" \
    -H "Content-Type: application/json" \
    -H "X-Client-ID: $CLIENT_ID" \
    -H "X-Device-Token: temp-token" \
    -d '{"username":"admin","password":"password","device_name":"Test Device"}')

if echo "$REGISTER_RESPONSE" | grep -q '"code":0'; then
    echo -e "客户端注册 ... ${GREEN}✓ 成功${NC}"
    DEVICE_TOKEN=$(echo "$REGISTER_RESPONSE" | grep -o '"device_token":"[^"]*"' | cut -d'"' -f4)
else
    echo -e "客户端注册 ... ${RED}✗ 失败${NC}"
    echo "响应: $REGISTER_RESPONSE"
fi
echo ""

# 6. 测试创建映射
echo "6. 创建映射测试"
test_endpoint "创建映射" "POST" "$SERVER_URL/api/v1/mappings" \
    '{"name":"test-mapping","type":"tcp","local_ip":"127.0.0.1","local_port":8080,"remote_port":0}' \
    "-H \"Authorization: Bearer $TOKEN\""
echo ""

# 7. 测试获取映射列表
echo "7. 获取映射列表测试"
test_endpoint "获取映射列表" "GET" "$SERVER_URL/api/v1/mappings?page=1&page_size=10" "" "-H \"Authorization: Bearer $TOKEN\""
echo ""

# 8. 测试心跳
echo "8. 心跳测试"
echo -n "测试 心跳 ... "
HEARTBEAT_RESPONSE=$(curl -s -X POST "$SERVER_URL/api/v1/client/heartbeat" \
    -H "X-Client-ID: $CLIENT_ID" \
    -H "X-Device-Token: $DEVICE_TOKEN")

if echo "$HEARTBEAT_RESPONSE" | grep -q '"code":0'; then
    echo -e "${GREEN}✓ 成功${NC}"
else
    echo -e "${RED}✗ 失败${NC}"
    echo "响应: $HEARTBEAT_RESPONSE"
fi
echo ""

# 9. 测试前端代理
echo "9. 前端代理测试"
test_endpoint "前端代理登录" "POST" "$FRONTEND_URL/api/v1/auth/login" \
    '{"username":"admin","password":"password"}'
echo ""

echo "=== 测试完成 ==="
echo ""
echo "系统运行状态："
echo "- 服务端: $SERVER_URL"
echo "- 前端: $FRONTEND_URL"
echo ""
echo "默认管理员账号："
echo "- 用户名: admin"
echo "- 密码: password"
echo ""
echo "GitHub 仓库: https://github.com/sshiong/frp-cloud-panel"

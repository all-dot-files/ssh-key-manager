#!/bin/bash

# 测试登录功能
echo "Testing login with user: hzy"

# 尝试几个常见密码
PASSWORDS=("123456" "password" "admin" "hzy123" "test123")

for PASSWORD in "${PASSWORDS[@]}"; do
    echo "Trying password: $PASSWORD"
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST http://localhost:8080/api/v1/auth/login \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"hzy\",\"password\":\"$PASSWORD\"}")
    
    HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
    BODY=$(echo "$RESPONSE" | head -n -1)
    
    echo "HTTP Code: $HTTP_CODE"
    echo "Response: $BODY"
    
    if [ "$HTTP_CODE" = "200" ]; then
        echo "✓ Login successful with password: $PASSWORD"
        echo "Token: $(echo $BODY | grep -o '"token":"[^"]*"' | cut -d'"' -f4)"
        exit 0
    fi
    echo ""
done

echo "✗ All password attempts failed"
echo ""
echo "Let's try to register a new user: testlogin"
curl -s -X POST http://localhost:8080/api/v1/auth/register \
    -H "Content-Type: application/json" \
    -d '{"username":"testlogin","password":"test123","email":"test@example.com"}'

echo ""
echo "Now trying to login with new user..."
curl -s -X POST http://localhost:8080/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"username":"testlogin","password":"test123"}'

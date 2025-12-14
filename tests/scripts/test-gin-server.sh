#!/bin/bash

# Test script for gin-based SKM server with token authentication

BASE_URL="http://localhost:8080"
echo "Testing SKM Server at $BASE_URL"
echo "=================================="
echo ""

# Test 1: Access protected page without token (should redirect to login)
echo "Test 1: Access /dashboard without token..."
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/dashboard")
if [ "$RESPONSE" = "303" ] || [ "$RESPONSE" = "401" ]; then
    echo "✓ PASS: Got redirect/unauthorized ($RESPONSE) as expected"
else
    echo "✗ FAIL: Expected redirect, got $RESPONSE"
fi
echo ""

# Test 2: Register a new user
echo "Test 2: Register new user..."
REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d '{"username":"testuser","password":"testpass123","email":"test@example.com"}')
echo "Response: $REGISTER_RESPONSE"
echo ""

# Test 3: Login and get token
echo "Test 3: Login with credentials..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"testuser","password":"testpass123"}')
echo "Response: $LOGIN_RESPONSE"

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)
if [ -n "$TOKEN" ]; then
    echo "✓ PASS: Got token: ${TOKEN:0:20}..."
else
    echo "✗ FAIL: No token received"
    exit 1
fi
echo ""

# Test 4: Access protected page with valid token
echo "Test 4: Access /api/v1/devices with valid token..."
DEVICES_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" "$BASE_URL/api/v1/devices" \
    -H "Authorization: Bearer $TOKEN")
HTTP_CODE=$(echo "$DEVICES_RESPONSE" | grep "HTTP_CODE:" | cut -d':' -f2)
BODY=$(echo "$DEVICES_RESPONSE" | sed '/HTTP_CODE:/d')

if [ "$HTTP_CODE" = "200" ]; then
    echo "✓ PASS: Successfully accessed protected endpoint"
    echo "Response: $BODY"
else
    echo "✗ FAIL: Expected 200, got $HTTP_CODE"
fi
echo ""

# Test 5: Access protected page with invalid token
echo "Test 5: Access /api/v1/devices with invalid token..."
INVALID_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/devices" \
    -H "Authorization: Bearer invalid_token_12345")
if [ "$INVALID_RESPONSE" = "401" ]; then
    echo "✓ PASS: Got 401 Unauthorized as expected"
else
    echo "✗ FAIL: Expected 401, got $INVALID_RESPONSE"
fi
echo ""

# Test 6: Get public keys with token
echo "Test 6: Get public keys with token..."
KEYS_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" "$BASE_URL/api/v1/keys/public" \
    -H "Authorization: Bearer $TOKEN")
HTTP_CODE=$(echo "$KEYS_RESPONSE" | grep "HTTP_CODE:" | cut -d':' -f2)
BODY=$(echo "$KEYS_RESPONSE" | sed '/HTTP_CODE:/d')

if [ "$HTTP_CODE" = "200" ]; then
    echo "✓ PASS: Successfully retrieved keys"
    echo "Response: $BODY"
else
    echo "✗ FAIL: Expected 200, got $HTTP_CODE"
fi
echo ""

echo "=================================="
echo "Tests completed!"

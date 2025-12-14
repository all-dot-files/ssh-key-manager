#!/bin/bash

# Start the server in the background
echo "Starting server..."
go run cmd/skm-server/main.go --jwt-secret="test-secret" --addr=":8082" &
SERVER_PID=$!

# Wait for server to start
sleep 2

# Function to check URL
check_url() {
    url=$1
    expected_status=$2
    echo "Checking $url..."
    status=$(curl -s -o /dev/null -w "%{http_code}" "$url")
    if [ "$status" -eq "$expected_status" ]; then
        echo "✅ $url returned $status"
    else
        echo "❌ $url returned $status (expected $expected_status)"
        kill $SERVER_PID
        exit 1
    fi
}

# Check public pages
check_url "http://localhost:8082/" 200
check_url "http://localhost:8082/login" 200
check_url "http://localhost:8082/register" 200

# Check static files
check_url "http://localhost:8082/static/css/style.css" 200
check_url "http://localhost:8082/static/js/app.js" 200

# Register a user to get a token (using API)
echo "Registering test user..."
curl -s -X POST http://localhost:8082/api/v1/auth/register \
    -H "Content-Type: application/json" \
    -d '{"username":"testuser","password":"password123","email":"test@example.com"}'

# Login to get token
echo "Logging in..."
TOKEN=$(curl -s -X POST http://localhost:8082/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"username":"testuser","password":"password123"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "❌ Failed to get token"
    kill $SERVER_PID
    exit 1
fi

echo "Got token: ${TOKEN:0:10}..."

# Check protected pages with token in cookie
# Note: curl doesn't automatically handle cookies unless specified, so we'll simulate it with a header for simplicity
# or we can use the cookie jar. Let's use the Authorization header since the middleware checks that too.

check_protected_url() {
    url=$1
    echo "Checking protected $url..."
    status=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $TOKEN" "$url")
    if [ "$status" -eq "200" ]; then
        echo "✅ $url returned 200"
    else
        echo "❌ $url returned $status (expected 200)"
        # Don't exit here, just report
    fi
}

check_protected_url "http://localhost:8082/dashboard"
check_protected_url "http://localhost:8082/keys"
check_protected_url "http://localhost:8082/devices"
check_protected_url "http://localhost:8082/audit"
check_protected_url "http://localhost:8082/settings"

# Clean up
kill $SERVER_PID
echo "Verification complete!"

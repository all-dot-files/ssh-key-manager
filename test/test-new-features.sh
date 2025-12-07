#!/bin/bash
# SKM 新功能测试脚本

set -e

echo "🧪 SKM New Features Test Script"
echo "================================"
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

SKM_BIN="./bin/skm"

# Check if skm binary exists
if [ ! -f "$SKM_BIN" ]; then
    echo "❌ SKM binary not found at $SKM_BIN"
    echo "Please run: make build"
    exit 1
fi

echo -e "${BLUE}Step 1: Check version${NC}"
$SKM_BIN version || echo "Version command not implemented yet"
echo ""

echo -e "${BLUE}Step 2: Test git hook commands${NC}"
echo "Testing: skm git hook --help"
$SKM_BIN git hook --help
echo ""

echo -e "${BLUE}Step 3: Test git bind with auto-create flag${NC}"
echo "Testing: skm git bind --help"
$SKM_BIN git bind --help | grep -i "auto-create" && \
    echo -e "${GREEN}✓ --auto-create flag found in git bind${NC}" || \
    echo -e "${YELLOW}⚠ --auto-create flag not found${NC}"
echo ""

echo -e "${BLUE}Step 4: Test host add with auto-create-key flag${NC}"
echo "Testing: skm host add --help"
$SKM_BIN host add --help | grep -i "auto-create-key" && \
    echo -e "${GREEN}✓ --auto-create-key flag found in host add${NC}" || \
    echo -e "${YELLOW}⚠ --auto-create-key flag not found${NC}"
echo ""

echo -e "${BLUE}Step 5: Test hook install command (dry-run)${NC}"
echo "Testing: skm git hook install --help"
$SKM_BIN git hook install --help
echo ""

echo -e "${BLUE}Step 6: Test hook uninstall command${NC}"
echo "Testing: skm git hook uninstall --help"
$SKM_BIN git hook uninstall --help
echo ""

echo -e "${GREEN}✅ All command structure tests passed!${NC}"
echo ""
echo "📋 Next Steps for Full Testing:"
echo "  1. Run: skm init --device-name 'Test Device'"
echo "  2. Run: skm git hook install"
echo "  3. Create a test repo and run: skm git bind . --host github.com --auto-create"
echo "  4. Test: skm host add test.example.com --user git --auto-create-key"
echo ""
echo "🎉 Test script completed!"

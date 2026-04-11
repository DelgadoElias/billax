#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║        BOOTSTRAP FEATURE - AUTOMATED TEST SUITE               ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"

# TEST 1: Fresh start with bootstrap
echo -e "\n${BLUE}🧪 TEST 1: Fresh Start with Bootstrap${NC}"
echo "Cleaning up..."
docker compose down -v 2>/dev/null || true

echo "Configuring bootstrap vars..."
export BOOTSTRAP_TENANT_NAME="Test Company"
export BOOTSTRAP_TENANT_EMAIL="admin@testco.com"
export BOOTSTRAP_ADMIN_PASSWORD="Test@1234"
export BOOTSTRAP_TENANT_SLUG="test-company"

echo "Starting services..."
docker compose up -d payd postgres 2>&1 | grep -v "Warning" || true
sleep 5

# Check bootstrap completed
if docker compose logs payd 2>/dev/null | grep -q "bootstrap: completado"; then
  echo -e "${GREEN}✅ Bootstrap created tenant successfully${NC}"
else
  echo -e "${RED}❌ Bootstrap did not complete${NC}"
  echo "Logs:"
  docker compose logs payd | tail -30
  exit 1
fi

# Check tenant exists
echo "Verifying tenant via check-email endpoint..."
RESPONSE=$(curl -s -X POST http://localhost:8080/v1/backoffice/check-email \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@testco.com"}' 2>/dev/null)

if echo "$RESPONSE" | jq -e '.single_tenant' >/dev/null 2>&1; then
  echo -e "${GREEN}✅ Tenant exists and is accessible${NC}"
else
  echo -e "${RED}❌ Tenant not found${NC}"
  echo "Response: $RESPONSE"
  exit 1
fi

# Check login works
echo "Testing login without /v1/signup call..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/v1/backoffice/login \
  -H "Content-Type: application/json" \
  -d '{
    "email":"admin@testco.com",
    "password":"Test@1234",
    "tenant_slug":"test-company"
  }' 2>/dev/null)

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token // empty')
USER_ID=$(echo "$LOGIN_RESPONSE" | jq -r '.user.id // empty')

if [ -n "$TOKEN" ] && [ -n "$USER_ID" ]; then
  echo -e "${GREEN}✅ Login works - JWT token obtained${NC}"
  echo -e "${GREEN}   User: admin@testco.com (ID: ${USER_ID:0:8}...)${NC}"
else
  echo -e "${RED}❌ Login failed${NC}"
  echo "Response: $LOGIN_RESPONSE"
  exit 1
fi

# TEST 2: Idempotence on restart
echo -e "\n${BLUE}🧪 TEST 2: Idempotence on Restart${NC}"
echo "Restarting container..."
docker compose restart payd 2>&1 | grep -v "Warning" || true
sleep 3

# Check idempotence logs
if docker compose logs payd 2>/dev/null | grep -q "bootstrap: tenant ya existe"; then
  echo -e "${GREEN}✅ Detected existing tenant on restart${NC}"
else
  echo -e "${RED}❌ Did not detect existing tenant${NC}"
  exit 1
fi

if docker compose logs payd 2>/dev/null | grep -q "bootstrap: usuario admin ya existe"; then
  echo -e "${GREEN}✅ Skipped admin user creation (idempotent)${NC}"
else
  echo -e "${RED}❌ Did not skip admin user creation${NC}"
  exit 1
fi

# Check login still works with same user
echo "Verifying login after restart..."
LOGIN_RESPONSE2=$(curl -s -X POST http://localhost:8080/v1/backoffice/login \
  -H "Content-Type: application/json" \
  -d '{
    "email":"admin@testco.com",
    "password":"Test@1234",
    "tenant_slug":"test-company"
  }' 2>/dev/null)

USER_ID2=$(echo "$LOGIN_RESPONSE2" | jq -r '.user.id // empty')

if [ "$USER_ID" = "$USER_ID2" ]; then
  echo -e "${GREEN}✅ Same user returned (no duplicates)${NC}"
else
  echo -e "${RED}❌ Different user ID after restart (duplicates created?)${NC}"
  exit 1
fi

# TEST 3: Bootstrap disabled (default)
echo -e "\n${BLUE}🧪 TEST 3: Bootstrap Disabled (Default State)${NC}"
echo "Cleaning up..."
docker compose down -v 2>/dev/null || true

echo "Unsetting bootstrap vars..."
unset BOOTSTRAP_TENANT_NAME
unset BOOTSTRAP_TENANT_EMAIL
unset BOOTSTRAP_ADMIN_PASSWORD
unset BOOTSTRAP_TENANT_SLUG

echo "Starting services without bootstrap..."
docker compose up -d payd postgres 2>&1 | grep -v "Warning" || true
sleep 5

# Check no bootstrap logs
if ! docker compose logs payd 2>/dev/null | grep -q "bootstrap: creando"; then
  echo -e "${GREEN}✅ Bootstrap disabled (no creation logs)${NC}"
else
  echo -e "${RED}❌ Bootstrap ran when it should be disabled${NC}"
  exit 1
fi

# Check signup endpoint still works
echo "Testing /v1/signup endpoint..."
SIGNUP=$(curl -s -X POST http://localhost:8080/v1/signup \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Test Tenant",
    "email":"test@example.com",
    "slug":"test-tenant"
  }' 2>/dev/null)

if echo "$SIGNUP" | jq -e '.tenant.id' >/dev/null 2>&1; then
  echo -e "${GREEN}✅ /v1/signup works (manual onboarding works)${NC}"
else
  echo -e "${RED}❌ /v1/signup failed${NC}"
  echo "Response: $SIGNUP"
  exit 1
fi

# FINAL SUMMARY
echo -e "\n${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}✅ ALL TESTS PASSED!${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"

echo -e "\n${BLUE}Summary:${NC}"
echo -e "  ${GREEN}✓${NC} Bootstrap creates tenant + admin on first start"
echo -e "  ${GREEN}✓${NC} Login works immediately without /v1/signup"
echo -e "  ${GREEN}✓${NC} Restart is idempotent (no duplicates)"
echo -e "  ${GREEN}✓${NC} Bootstrap can be disabled (default)"
echo -e "  ${GREEN}✓${NC} Manual /v1/signup still works when bootstrap disabled"

echo -e "\n${YELLOW}Next Steps:${NC}"
echo "  • Read BOOTSTRAP_FEATURE.md for complete documentation"
echo "  • Test with your own environment variables"
echo "  • Deploy to Railway or Docker for production"

echo -e "\n${GREEN}Cleaning up...${NC}"
docker compose down 2>/dev/null || true

echo -e "${GREEN}Done!${NC}\n"

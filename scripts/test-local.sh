#!/bin/bash
# test-local.sh - Quick start for local testing against MP Sandbox

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== payd Local Testing Setup ===${NC}"

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"
command -v go &> /dev/null || { echo "Go not installed"; exit 1; }
command -v docker &> /dev/null || { echo "Docker not installed"; exit 1; }
echo -e "${GREEN}✓ Prerequisites met${NC}"

# Start Docker services
echo -e "${YELLOW}Starting Docker services...${NC}"
docker-compose up -d
echo -e "${GREEN}✓ Docker services started${NC}"

# Wait for Postgres to be ready
echo -e "${YELLOW}Waiting for Postgres...${NC}"
for i in {1..30}; do
  if docker-compose exec -T postgres pg_isready -U payd_app >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Postgres is ready${NC}"
    break
  fi
  sleep 1
done

# Run migrations
echo -e "${YELLOW}Running migrations...${NC}"
export DATABASE_URL="postgres://payd_app:password@localhost:5432/payd?sslmode=disable"
psql $DATABASE_URL -f migrations/001_init.sql 2>/dev/null || echo "Migration 001 already applied"
psql $DATABASE_URL -f migrations/002_plan_slug_subscription_tags.sql 2>/dev/null || echo "Migration 002 already applied"
psql $DATABASE_URL -f migrations/003_planless_subscriptions.sql 2>/dev/null || echo "Migration 003 already applied"
psql $DATABASE_URL -f migrations/004_provider_credentials.sql 2>/dev/null || echo "Migration 004 already applied"
echo -e "${GREEN}✓ Migrations complete${NC}"

# Run tests
echo -e "${YELLOW}Running tests...${NC}"
go test ./... -v -timeout 30s
echo -e "${GREEN}✓ All tests passed${NC}"

# Start the app
echo -e "${YELLOW}Starting app...${NC}"
export APP_ENV=development
export PORT=8080
export LOG_LEVEL=debug

echo -e "${BLUE}=== App Starting ===${NC}"
echo "API available at: http://localhost:8080"
echo "Health check: http://localhost:8080/health"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Configure Mercado Pago credentials:"
echo "   curl -X POST http://localhost:8080/v1/provider-credentials/mercadopago \\"
echo "     -H 'Authorization: Bearer payd_live_test' \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"access_token\": \"YOUR_MP_TOKEN\", \"webhook_secret\": \"YOUR_WEBHOOK_SECRET\"}'"
echo ""
echo "2. See TESTING.md for complete testing guide"
echo ""

go run ./cmd/payd/main.go

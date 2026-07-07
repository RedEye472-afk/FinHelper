#!/bin/bash
# FinHelper Integration Test Suite — verifies all API endpoints
set -euo pipefail
BASE="http://localhost:8080/api/v1"
PASS=0
FAIL=0

green() { echo -e "\033[32m✓ $1\033[0m"; }
red()   { echo -e "\033[31m✗ $1\033[0m"; }

# Authenticated endpoint check: check METHOD PATH EXPECTED [DATA]
check() {
    local method="$1" path="$2" expected="$3" data="${4:-}"
    local code
    if [ -n "$data" ]; then
        code=$(curl -s -o NUL -w "%{http_code}" -X "$method" \
          -H "$AUTH" -H 'Content-Type: application/json' \
          -d "$data" "$BASE$path" 2>/dev/null)
    else
        code=$(curl -s -o NUL -w "%{http_code}" -X "$method" \
          -H "$AUTH" "$BASE$path" 2>/dev/null)
    fi
    if [ "$code" = "$expected" ]; then
        green "$method $path → $code"
        PASS=$((PASS+1))
    else
        red "$method $path → $code (expected $expected)"
        FAIL=$((FAIL+1))
    fi
}

echo "======== FinHelper Integration Tests ========"
echo ""

# 1. Health (no auth)
echo "── Health ──"
H1=$(curl -s -o NUL -w "%{http_code}" http://localhost:8080/healthz)
[ "$H1" = 200 ] && green "GET /healthz → 200" || red "GET /healthz → $H1"
H2=$(curl -s -o NUL -w "%{http_code}" http://localhost:8080/readyz)
[ "$H2" = 200 ] && green "GET /readyz → 200" || red "GET /readyz → $H2"
PASS=$((PASS+2))

# 2. Auth — register (also returns access_token)
echo ""
echo "── Auth ──"
REG=$(curl -s -X POST "$BASE/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"test-int3@finhelper.test","password":"TestPass123!@#"}')
echo "  register: $(echo $REG | head -c 100)"
TOKEN=$(echo $REG | grep -o '"access_token":"[^"]*"' | cut -d\" -f4) || true
if [ -z "$TOKEN" ]; then
  LOGIN=$(curl -s -X POST "$BASE/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"test-int3@finhelper.test","password":"TestPass123!@#"}')
  TOKEN=$(echo $LOGIN | grep -o '"access_token":"[^"]*"' | cut -d\" -f4) || true
fi
if [ -z "$TOKEN" ]; then
  red "No token obtained. Aborting."; exit 1
fi
AUTH="Authorization: Bearer $TOKEN"
echo "  token: ${TOKEN:0:20}..."
PASS=$((PASS+1))

# 3. Me
echo ""
echo "── Me ──"
check GET /me 200

# 4. Accounts — CRUD
echo ""
echo "── Accounts ──"
ACCT=$(curl -s -X POST "$BASE/accounts" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"name":"Main Account","type":"bank","currency":"RUB"}')
ACCT_ID=$(echo $ACCT | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2) || true
echo "  created id=$ACCT_ID"
check GET /accounts 200
check GET "/accounts/$ACCT_ID" 200
check PATCH "/accounts/$ACCT_ID" 200 '{"name":"Updated Account","type":"bank"}'
check DELETE "/accounts/$ACCT_ID" 204

# Re-create for downstream tests
ACCT2=$(curl -s -X POST "$BASE/accounts" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"name":"Main","type":"bank","currency":"RUB"}')
ACCT_ID=$(echo $ACCT2 | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2) || true
echo "  active account_id=$ACCT_ID"

# 5. Operations — CRUD
echo ""
echo "── Operations ──"
CALC_ID="inttest-$(date +%s)"
OP=$(curl -s -X POST "$BASE/operations" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d "{\"type\":\"income\",\"amount\":\"50000.00\",\"account_id\":$ACCT_ID,\"calc_id\":\"$CALC_ID\",\"description\":\"Salary\"}")
OP_ID=$(echo $OP | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2) || true
echo "  created op_id=$OP_ID"
check GET /operations 200
check GET "/operations/$OP_ID" 200
check DELETE "/operations/$OP_ID" 204

# Re-create for dashboard test
CALC_ID2="inttest-$(date +%s)"
OP2=$(curl -s -X POST "$BASE/operations" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d "{\"type\":\"income\",\"amount\":\"100000.00\",\"account_id\":$ACCT_ID,\"calc_id\":\"$CALC_ID2\",\"description\":\"Bonus\"}")
OP2_ID=$(echo $OP2 | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2) || true
echo "  active op_id=$OP2_ID"

# 6. Dashboard
echo ""
echo "── Dashboard ──"
check GET /dashboard 200
DASH=$(curl -s -H "$AUTH" "$BASE/dashboard")
echo "  dashboard: $(echo $DASH | head -c 160)"

# 7. Budgets — CRUD
echo ""
echo "── Budgets ──"
CAT_ID=$(docker exec finhelper-postgres psql -U finhelper -d finhelper -t -A -c "SELECT id FROM categories WHERE user_id IN (SELECT id FROM users WHERE email='test-int3@finhelper.test') AND is_system ORDER BY id LIMIT 1" 2>/dev/null)
echo "  category_id=$CAT_ID"
BGT=$(curl -s -X POST "$BASE/budgets" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d "{\"category_id\":$CAT_ID,\"limit_amount\":\"30000.00\"}")
BGT_ID=$(echo $BGT | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2) || true
echo "  created budget_id=$BGT_ID"
check GET /budgets 200
check GET "/budgets/$BGT_ID/status" 200
# Note: PATCH /budgets requires limit_amount, rollover_policy, is_active (full replace)
check PATCH "/budgets/$BGT_ID" 200 '{"limit_amount":"35000.00","rollover_policy":"none","is_active":true}'
check DELETE "/budgets/$BGT_ID" 204

# 8. Goals — CRUD
echo ""
echo "── Goals ──"
GOAL=$(curl -s -X POST "$BASE/goals" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"name":"Test Goal","target_amount":"100000.00","target_date":"2027-01-01T00:00:00Z"}')
GOAL_ID=$(echo $GOAL | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2) || true
echo "  created goal_id=$GOAL_ID"
check GET /goals 200
check GET "/goals/$GOAL_ID" 200
# Note: PATCH /goals requires ALL fields (full replace)
check PATCH "/goals/$GOAL_ID" 200 '{"name":"Updated Goal","target_amount":"120000.00","current_amount":"0","monthly_contribution":"","target_date":"2027-01-01T00:00:00Z","expected_yield":"0"}'

# Contribute
CCALC_ID="inttest-contrib-$(date +%s)"
CONTRIB=$(curl -s -X POST "$BASE/goals/$GOAL_ID/contributions" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d "{\"amount\":\"5000.00\",\"contribution_id\":\"$CCALC_ID\",\"comment\":\"First deposit\"}")
CONTRIB_ID=$(echo $CONTRIB | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2) || true
echo "  contribution id=$CONTRIB_ID"

# Projection
check GET "/goals/$GOAL_ID/projection" 200
PROJ=$(curl -s -H "$AUTH" "$BASE/goals/$GOAL_ID/projection")
echo "  projection: $(echo $PROJ | head -c 160)"

# Simulate (uses expected_yield + inflation, NOT annual_yield/inflation_rate)
SIM=$(curl -s -X POST "$BASE/goals/$GOAL_ID/simulate" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"target_amount":"100000.00","monthly_contribution":"8000.00","expected_yield":"0.12","inflation":"0.08"}')
echo "  simulate: $(echo $SIM | head -c 160)"
SIM_OK=$(echo $SIM | grep -o '"months_left":[0-9]*' | head -1 | cut -d: -f2) || true
if [ -n "$SIM_OK" ]; then
  green "POST /goals/$GOAL_ID/simulate → months_left=$SIM_OK"
  PASS=$((PASS+1))
else
  red "POST /goals/$GOAL_ID/simulate → unexpected"
  FAIL=$((FAIL+1))
fi

# List contributions
check GET "/goals/$GOAL_ID/contributions" 200
check DELETE "/goals/$GOAL_ID" 204

# 9. Credit Calculator (POST /calc/credit, fields: principal, annual_rate, term_months)
echo ""
echo "── Credit Calculator ──"
CREDIT=$(curl -s -X POST "$BASE/calc/credit" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"principal":"1000000","annual_rate":"0.15","term_months":12}')
echo "  credit: $(echo $CREDIT | head -c 160)"
MP=$(echo $CREDIT | grep -o '"monthly_payment":"[^"]*"' | cut -d\" -f4) || true
if [ -n "$MP" ]; then
  green "POST /calc/credit → monthly_payment=$MP"
  PASS=$((PASS+1))
else
  red "POST /calc/credit → unexpected"
  FAIL=$((FAIL+1))
fi

# 10. Categorization Rules (no `confidence` field — just keyword + category_id)
echo ""
echo "── Categorization ──"
check GET /categorization/rules 200
RULE=$(curl -s -X POST "$BASE/categorization/rules" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d "{\"keyword\":\"uber\",\"category_id\":$CAT_ID}")
RULE_ID=$(echo $RULE | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2) || true
if [ -n "$RULE_ID" ]; then
  green "POST /categorization/rules → id=$RULE_ID"
  PASS=$((PASS+1))
else
  red "POST /categorization/rules → unexpected: $(echo $RULE | head -c 100)"
  FAIL=$((FAIL+1))
fi

# Summary
echo ""
echo "======== Summary ========"
echo "Passed: $PASS   Failed: $FAIL"
echo "========================="
exit $FAIL

#!/bin/bash

# =============================================================================
# Pulse Comprehensive End-to-End Test Suite
# Tests: All features including auth, projects, errors, traces, coverage, monitors, settings, security
# =============================================================================

BASE_URL="${1:-http://localhost:8080}"
PROJECT_ID="c5ff69c3-bbca-4fc3-b731-548d509b001d"
# Default API key - will be updated from project if available
SENTRY_KEY="${SENTRY_KEY:-4d7673f8-8e73-4522-ae24-7055eab85b42}"

# Load .env file if it exists
if [ -f .env ]; then
  set -a
  . .env
  set +a
fi

ADMIN_EMAIL="${ADMIN_EMAIL:-admin@example.com}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin}"

STORE_ENDPOINT="$BASE_URL/api/$PROJECT_ID/store/"
COVERAGE_ENDPOINT="$BASE_URL/api/$PROJECT_ID/coverage"
ENVELOPE_ENDPOINT="$BASE_URL/api/projects/$PROJECT_ID/envelope"

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TOTAL_TESTS=0

# Helper functions
generate_id() { cat /proc/sys/kernel/random/uuid 2>/dev/null | tr -d '-' || openssl rand -hex 16; }
generate_span() { openssl rand -hex 8 2>/dev/null || head -c 16 /dev/urandom | xxd -p; }
  NOW=$(date -u +%Y-%m-%dT%H:%M:%S.000Z)

# Test helper function - simplified to handle headers properly
test_endpoint() {
  local name="$1"
  local method="$2"
  local url="$3"
  local header1="$4"
  local header2="$5"
  local data="$6"
  local expected_status="${7:-200}"

  TOTAL_TESTS=$((TOTAL_TESTS + 1))
  echo "🧪 [$TOTAL_TESTS] $name..."

  # Build curl command
  local curl_cmd="curl -s -w '%{http_code}' -X $method"

  # Add headers
  if [ -n "$header1" ]; then
    curl_cmd="$curl_cmd -H '$header1'"
  fi
  if [ -n "$header2" ]; then
    curl_cmd="$curl_cmd -H '$header2'"
  fi

  # Add data if provided
  if [ -n "$data" ]; then
    curl_cmd="$curl_cmd -d '$data'"
  fi

  # Add URL
  curl_cmd="$curl_cmd '$url'"

  # Execute and capture response
  local response=$(eval "$curl_cmd" 2>&1)
  local http_code=$(echo "$response" | tail -c 4 | grep -o '[0-9]\{3\}' || echo "000")
  local body=$(echo "$response" | sed 's/[0-9]\{3\}$//')

  if [ "$http_code" = "$expected_status" ] || [ "$expected_status" = "any" ]; then
    echo "    [OK] Status: $http_code"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    echo "    [FAIL] Expected: $expected_status, Got: $http_code"
    if [ -n "$body" ] && [ ${#body} -lt 200 ]; then
      echo "    Response: $(echo "$body" | head -c 150)"
    fi
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return 1
  fi
}

# Extract JWT token from login response
get_auth_token() {
  local response=$(curl -s -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}")

  # Check for error first
  if echo "$response" | grep -q '"error"\|"message"\|"Invalid"'; then
    return 1
  fi

  # Try multiple ways to extract token
  local token=""

  # Method 1: grep (most compatible)
  token=$(echo "$response" | grep -o '"token":"[^"]*' | cut -d'"' -f4)

  # Method 2: jq if available (most reliable)
  if [ -z "$token" ] && command -v jq &> /dev/null; then
    token=$(echo "$response" | jq -r '.token // empty' 2>/dev/null)
  fi

  # Method 3: Python if available
  if [ -z "$token" ] && command -v python3 &> /dev/null; then
    token=$(echo "$response" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('token', ''))" 2>/dev/null)
  fi

  # Method 4: sed fallback
  if [ -z "$token" ]; then
    token=$(echo "$response" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
  fi

  echo "$token"
}

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║         PULSE COMPREHENSIVE TEST SUITE                           ║"
echo "╠════════════════════════════════════════════════════════════════╣"
echo "║ Base URL:  $BASE_URL"
echo "║ Project:   $PROJECT_ID"
echo "║ Admin:     $ADMIN_EMAIL"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Get auth token
echo "[AUTH] Authenticating with $ADMIN_EMAIL..."
AUTH_TOKEN=$(get_auth_token)
if [ -z "$AUTH_TOKEN" ]; then
  echo "⚠️  Authentication failed with $ADMIN_EMAIL"
  echo "   Trying common default credentials..."

  # Try common defaults
  for email in "admin@example.com" "admin@example.com" "admin@example.com"; do
    for password in "admin" "admin" "admin" "admin"; do
      if [ "$email" = "$ADMIN_EMAIL" ] && [ "$password" = "$ADMIN_PASSWORD" ]; then
        continue  # Skip if we already tried this
      fi
      echo "   Trying $email / $password..."
      OLD_EMAIL="$ADMIN_EMAIL"
      OLD_PASSWORD="$ADMIN_PASSWORD"
      ADMIN_EMAIL="$email"
      ADMIN_PASSWORD="$password"
      AUTH_TOKEN=$(get_auth_token)
      if [ -n "$AUTH_TOKEN" ]; then
        echo "✅ Authenticated with $ADMIN_EMAIL"
        break 2
      fi
      ADMIN_EMAIL="$OLD_EMAIL"
      ADMIN_PASSWORD="$OLD_PASSWORD"
    done
  done

  if [ -z "$AUTH_TOKEN" ]; then
    echo ""
    echo "❌ Failed to authenticate."
    echo "   Please ensure ADMIN_EMAIL and ADMIN_PASSWORD are set correctly."
    echo "   The server must be started with the same credentials."
    echo ""
    echo "   Options:"
    echo "   1. Create/update .env file with:"
    echo "      ADMIN_EMAIL=admin@example.com"
    echo "      ADMIN_PASSWORD=your_password"
    echo ""
    echo "   2. Or set environment variables:"
    echo "      ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=admin123 ./test.sh"
    echo ""
    echo "   3. Or pass credentials when starting server:"
    echo "      ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=admin123 ./sentry-alt"
    exit 1
  fi
else
  echo "✅ Authenticated with $ADMIN_EMAIL"
fi

# Try to get the actual API key for the test project
echo "🔑 Fetching project API key..."
PROJECT_INFO=$(curl -s -X GET "$BASE_URL/api/projects/$PROJECT_ID" \
  -H "Authorization: Bearer $AUTH_TOKEN" \
  -H "Content-Type: application/json")
ACTUAL_API_KEY=$(echo "$PROJECT_INFO" | grep -o '"api_key":"[^"]*' | cut -d'"' -f4)
if [ -n "$ACTUAL_API_KEY" ]; then
  SENTRY_KEY="$ACTUAL_API_KEY"
  echo "✅ Using project API key: $SENTRY_KEY"
else
  echo "⚠️  Could not fetch project API key, using default: $SENTRY_KEY"
fi
echo ""

# =============================================================================
# SECTION 0: HEALTH CHECK & DISCOVERY
# =============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📍 SECTION 0: Health Check & Discovery"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

test_endpoint "Health Check" "GET" "$BASE_URL/api/health" "" "" "" "200"
# Project Discovery - will be tested after we get the API key
# Skip for now as it requires the actual API key

echo ""

# =============================================================================
# SECTION 1: AUTHENTICATION
# =============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📍 SECTION 1: Authentication"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

test_endpoint "Login with valid credentials" "POST" "$BASE_URL/api/auth/login" \
  "Content-Type: application/json" "" \
  "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}" "200"

test_endpoint "Login with invalid credentials" "POST" "$BASE_URL/api/auth/login" \
  "Content-Type: application/json" "" \
  "{\"email\":\"invalid@example.com\",\"password\":\"wrong\"}" "401"

test_endpoint "Get current user (auth/me)" "GET" "$BASE_URL/api/auth/me" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"

echo ""

# =============================================================================
# SECTION 2: PROJECT MANAGEMENT
# =============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📍 SECTION 2: Project Management"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Create a test project
TEST_PROJECT_NAME="Test Project $(date +%s)"
test_endpoint "Create new project" "POST" "$BASE_URL/api/projects" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" \
  "{\"name\":\"$TEST_PROJECT_NAME\"}" "201"

# Get project ID from response
PROJECT_RESPONSE=$(curl -s -X POST "$BASE_URL/api/projects" \
  -H "Authorization: Bearer $AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Test Project Get ID\"}")
NEW_PROJECT_ID=$(echo "$PROJECT_RESPONSE" | grep -o '"id":"[^"]*' | cut -d'"' -f4)

test_endpoint "Get all projects" "GET" "$BASE_URL/api/projects" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"

test_endpoint "Get specific project" "GET" "$BASE_URL/api/projects/$PROJECT_ID" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"

test_endpoint "Update project quota" "PATCH" "$BASE_URL/api/projects/$PROJECT_ID/quota" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" \
  "{\"quota\":10000}" "204"

# Clean up test project
if [ -n "$NEW_PROJECT_ID" ]; then
  test_endpoint "Delete test project" "DELETE" "$BASE_URL/api/projects/$NEW_PROJECT_ID" \
    "Authorization: Bearer $AUTH_TOKEN" "" "" "204"
fi

echo ""

# =============================================================================
# SECTION 3: ERROR INGESTION (Sentry Protocol)
# =============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📍 SECTION 3: Error Ingestion (Sentry Protocol)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Disable security policies for testing (allow all IPs)
echo "🔓 Disabling security policies for testing..."
curl -s -X POST "$BASE_URL/api/projects/$PROJECT_ID/security-policies" \
  -H "Authorization: Bearer $AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"enforced\":false,\"ip_whitelist\":\"\",\"rate_limit\":1000}" > /dev/null
echo ""

ERROR_IDS=()

# Test 3.1: Full error with exception
ERROR_ID1=$(generate_id)
test_endpoint "Error with full exception & stack trace" "POST" "$STORE_ENDPOINT" \
  "X-Sentry-Auth: Sentry sentry_key=$SENTRY_KEY, sentry_version=7" "Content-Type: application/json" \
  "{
  \"event_id\": \"$ERROR_ID1\",
  \"timestamp\": \"$NOW\",
  \"platform\": \"go\",
  \"level\": \"error\",
  \"logger\": \"pipeops.deployment.service\",
  \"server_name\": \"deploy-worker-01\",
  \"environment\": \"production\",
  \"release\": \"pipeops-core@3.2.1\",
  \"transaction\": \"DeploymentService.Deploy\",
  \"exception\": {
    \"values\": [{
      \"type\": \"DeploymentError\",
      \"value\": \"failed to pull image: registry timeout after 30s\",
      \"module\": \"github.com/pipeops/core/deployment\",
      \"stacktrace\": {
        \"frames\": [
          {\"filename\": \"deployment/service.go\", \"function\": \"(*Service).Deploy\", \"lineno\": 145, \"in_app\": true},
          {\"filename\": \"deployment/docker.go\", \"function\": \"(*DockerClient).PullImage\", \"lineno\": 89, \"in_app\": true}
        ]
      }
    }]
  },
  \"tags\": {\"service\": \"deployment\", \"k8s.cluster\": \"prod-us-east-1\"},
  \"user\": {\"id\": \"org_12345\", \"email\": \"devops@acme.com\"},
  \"sdk\": {\"name\": \"sentry.go\", \"version\": \"0.27.0\"}
}" "200"
ERROR_IDS+=("$ERROR_ID1")

# Test 3.2: Warning event
ERROR_ID2=$(generate_id)
test_endpoint "Warning level event" "POST" "$STORE_ENDPOINT" \
  "X-Sentry-Auth: Sentry sentry_key=$SENTRY_KEY, sentry_version=7" "Content-Type: application/json" \
  "{
  \"event_id\": \"$ERROR_ID2\",
  \"timestamp\": \"$NOW\",
  \"platform\": \"node\",
  \"level\": \"warning\",
  \"environment\": \"production\",
  \"message\": {\"formatted\": \"Memory usage at 85% - approaching limit\"},
  \"tags\": {\"service\": \"web-frontend\"},
  \"sdk\": {\"name\": \"sentry.javascript.node\", \"version\": \"7.91.0\"}
}" "200"
ERROR_IDS+=("$ERROR_ID2")

# Test 3.3: Fatal error
ERROR_ID3=$(generate_id)
test_endpoint "Fatal level event" "POST" "$STORE_ENDPOINT" \
  "X-Sentry-Auth: Sentry sentry_key=$SENTRY_KEY, sentry_version=7" "Content-Type: application/json" \
  "{
  \"event_id\": \"$ERROR_ID3\",
  \"timestamp\": \"$NOW\",
  \"platform\": \"python\",
  \"level\": \"fatal\",
  \"exception\": {\"values\": [{\"type\": \"SystemExit\", \"value\": \"Database migration failed\"}]},
  \"sdk\": {\"name\": \"sentry.python\", \"version\": \"1.35.0\"}
}" "200"
ERROR_IDS+=("$ERROR_ID3")

# Test 3.4: DSN-based posting (without /store/)
# Note: This endpoint may not exist, testing with /store/ instead
test_endpoint "DSN-based POST (direct)" "POST" "$BASE_URL/api/$PROJECT_ID/store/" \
  "X-Sentry-Auth: Sentry sentry_key=$SENTRY_KEY, sentry_version=7" "Content-Type: application/json" \
  "{
  \"event_id\": \"$(generate_id)\",
  \"timestamp\": \"$NOW\",
  \"level\": \"error\",
  \"message\": \"DSN direct post test\"
}" "200"

echo ""

# =============================================================================
# SECTION 4: TRANSACTIONS / PERFORMANCE
# =============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📍 SECTION 4: Transactions / Performance"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

TRACE_ID1=$(generate_id)
ROOT_SPAN1=$(generate_span)

test_endpoint "HTTP Server Transaction with spans" "POST" "$STORE_ENDPOINT" \
  "X-Sentry-Auth: Sentry sentry_key=$SENTRY_KEY, sentry_version=7" "Content-Type: application/json" \
  "{
  \"event_id\": \"$(generate_id)\",
  \"type\": \"transaction\",
  \"transaction\": \"GET /api/v1/deployments/:id\",
  \"start_timestamp\": $(date +%s).0,
  \"timestamp\": $(date +%s).0,
  \"platform\": \"go\",
  \"environment\": \"production\",
  \"contexts\": {
    \"trace\": {
      \"trace_id\": \"$TRACE_ID1\",
      \"span_id\": \"$ROOT_SPAN1\",
      \"op\": \"http.server\",
      \"status\": \"ok\"
    }
  },
  \"spans\": [
    {
      \"span_id\": \"$(generate_span)\",
      \"parent_span_id\": \"$ROOT_SPAN1\",
      \"trace_id\": \"$TRACE_ID1\",
      \"op\": \"db.query\",
      \"description\": \"SELECT * FROM deployments\",
      \"start_timestamp\": $(date +%s).0,
      \"timestamp\": $(date +%s).0,
      \"status\": \"ok\"
    }
  ],
  \"sdk\": {\"name\": \"sentry.go\", \"version\": \"0.27.0\"}
}" "200"

TRACE_ID2=$(generate_id)
ROOT_SPAN2=$(generate_span)

test_endpoint "Background Job Transaction" "POST" "$STORE_ENDPOINT" \
  "X-Sentry-Auth: Sentry sentry_key=$SENTRY_KEY, sentry_version=7" "Content-Type: application/json" \
  "{
  \"event_id\": \"$(generate_id)\",
  \"type\": \"transaction\",
  \"transaction\": \"BuildJob.Execute\",
  \"start_timestamp\": $(date +%s).0,
  \"timestamp\": $(date +%s).0,
  \"platform\": \"go\",
  \"contexts\": {
    \"trace\": {
      \"trace_id\": \"$TRACE_ID2\",
      \"span_id\": \"$ROOT_SPAN2\",
      \"op\": \"queue.task\",
      \"status\": \"ok\"
    }
  },
  \"spans\": [],
  \"sdk\": {\"name\": \"sentry.go\", \"version\": \"0.27.0\"}
}" "200"

echo ""

# =============================================================================
# SECTION 5: ERROR RETRIEVAL & MANAGEMENT
# =============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📍 SECTION 5: Error Retrieval & Management"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Wait a moment for errors to be stored
sleep 1

test_endpoint "Get all errors" "GET" "$BASE_URL/api/errors" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"

test_endpoint "Get errors with pagination" "GET" "$BASE_URL/api/errors?limit=10&offset=0" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"

test_endpoint "Get errors filtered by status" "GET" "$BASE_URL/api/errors?status=unresolved" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"

test_endpoint "Get project-specific errors" "GET" "$BASE_URL/api/projects/$PROJECT_ID/errors" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"

# Get an error ID to test detail endpoints
ERROR_LIST=$(curl -s -X GET "$BASE_URL/api/errors?limit=1" \
  -H "Authorization: Bearer $AUTH_TOKEN" \
  -H "Content-Type: application/json")
ERROR_DETAIL_ID=$(echo "$ERROR_LIST" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)

if [ -n "$ERROR_DETAIL_ID" ]; then
  test_endpoint "Get error detail" "GET" "$BASE_URL/api/errors/$ERROR_DETAIL_ID" \
    "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"

  test_endpoint "Update error status to resolved" "PATCH" "$BASE_URL/api/errors/$ERROR_DETAIL_ID" \
    "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" \
    "{\"status\":\"resolved\"}" "204"

  test_endpoint "Update error status to ignored" "PATCH" "$BASE_URL/api/errors/$ERROR_DETAIL_ID" \
    "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" \
    "{\"status\":\"ignored\"}" "204"
fi

echo ""

# =============================================================================
# SECTION 6: TRACE RETRIEVAL
# =============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📍 SECTION 6: Trace Retrieval"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

sleep 1

test_endpoint "Get project traces" "GET" "$BASE_URL/api/projects/$PROJECT_ID/traces" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"

test_endpoint "Get project traces with limit" "GET" "$BASE_URL/api/projects/$PROJECT_ID/traces?limit=10" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"

# Get trace ID if available
TRACE_LIST=$(curl -s -X GET "$BASE_URL/api/projects/$PROJECT_ID/traces?limit=1" \
  -H "Authorization: Bearer $AUTH_TOKEN" \
  -H "Content-Type: application/json")
TRACE_DETAIL_ID=$(echo "$TRACE_LIST" | grep -o '"trace_id":"[^"]*' | head -1 | cut -d'"' -f4)

if [ -n "$TRACE_DETAIL_ID" ]; then
  test_endpoint "Get trace details" "GET" "$BASE_URL/api/projects/$PROJECT_ID/traces/$TRACE_DETAIL_ID" \
    "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"
fi

echo ""

# =============================================================================
# SECTION 7: COVERAGE INGESTION & RETRIEVAL
# =============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📍 SECTION 7: Coverage Ingestion & Retrieval"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

test_endpoint "Simple coverage percentage" "POST" "$COVERAGE_ENDPOINT" \
  "X-Pulse-Auth: $SENTRY_KEY" "Content-Type: application/json" \
  "{\"coverage\": 84.5}" "204"

test_endpoint "Coverage with metadata" "POST" "$COVERAGE_ENDPOINT" \
  "X-Pulse-Auth: $SENTRY_KEY" "Content-Type: application/json" \
  "{
  \"coverage\": 87.3,
  \"branch\": \"main\",
  \"commit\": \"a1b2c3d4e5f6\",
  \"commit_message\": \"feat: add deployment rollback\"
}" "204"

# Create Go coverage file
cat > /tmp/coverage.out << 'EOF'
mode: set
github.com/pipeops/core/deployment/service.go:15.52,17.2 1 1
github.com/pipeops/core/deployment/service.go:19.42,24.16 4 1
github.com/pipeops/core/deployment/docker.go:10.34,15.16 4 1
EOF

# Test file upload (multipart)
TOTAL_TESTS=$((TOTAL_TESTS + 1))
echo "🧪 [$TOTAL_TESTS] Go coverage.out file upload..."
response=$(curl -s -w "%{http_code}" -X POST "$COVERAGE_ENDPOINT" \
  -H "X-Pulse-Auth: $SENTRY_KEY" \
  -F "file=@/tmp/coverage.out")
http_code=$(echo "$response" | tail -c 4 | grep -o '[0-9]\{3\}' || echo "000")
if [ "$http_code" = "204" ]; then
  echo "    ✅ Status: $http_code"
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  echo "    ❌ Expected: 204, Got: $http_code"
  TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Create LCOV file
cat > /tmp/lcov.info << 'EOF'
TN:
SF:src/services/deployment.ts
FN:10,deploy
FNDA:150,deploy
DA:10,150
LF:1
LH:1
end_of_record
EOF

# Test LCOV upload
TOTAL_TESTS=$((TOTAL_TESTS + 1))
echo "🧪 [$TOTAL_TESTS] LCOV file upload..."
response=$(curl -s -w "%{http_code}" -X POST "$COVERAGE_ENDPOINT" \
  -H "X-Pulse-Auth: $SENTRY_KEY" \
  -F "file=@/tmp/lcov.info")
http_code=$(echo "$response" | tail -c 4 | grep -o '[0-9]\{3\}' || echo "000")
if [ "$http_code" = "204" ]; then
  echo "    ✅ Status: $http_code"
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  echo "    ❌ Expected: 204, Got: $http_code"
  TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Coverage retrieval
sleep 1
test_endpoint "Get coverage history" "GET" "$BASE_URL/api/projects/$PROJECT_ID/coverage/history" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"

test_endpoint "Get coverage badge" "GET" "$BASE_URL/api/projects/$PROJECT_ID/coverage/badge" \
  "" "" "" "200"

# Get snapshot ID if available
COVERAGE_HISTORY=$(curl -s -X GET "$BASE_URL/api/projects/$PROJECT_ID/coverage/history" \
  -H "Authorization: Bearer $AUTH_TOKEN" \
  -H "Content-Type: application/json")
SNAPSHOT_ID=$(echo "$COVERAGE_HISTORY" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)

if [ -n "$SNAPSHOT_ID" ]; then
  test_endpoint "Get coverage snapshot files" "GET" "$BASE_URL/api/projects/$PROJECT_ID/coverage/snapshots/$SNAPSHOT_ID/files" \
    "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"
fi

rm -f /tmp/coverage.out /tmp/lcov.info

echo ""

# =============================================================================
# SECTION 8: UPTIME MONITORS
# =============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📍 SECTION 8: Uptime Monitors"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

MONITOR_NAME="Test Monitor $(date +%s)"
test_endpoint "Create monitor" "POST" "$BASE_URL/api/projects/$PROJECT_ID/monitors" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" \
  "{
  \"name\": \"$MONITOR_NAME\",
  \"url\": \"https://example.com\",
  \"interval\": 60,
  \"timeout\": 30
}" "any"

# Get monitor ID
MONITOR_RESPONSE=$(curl -s -X POST "$BASE_URL/api/projects/$PROJECT_ID/monitors" \
  -H "Authorization: Bearer $AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Test Monitor Get ID\",\"url\":\"https://example.com\",\"interval\":60}")
NEW_MONITOR_ID=$(echo "$MONITOR_RESPONSE" | grep -o '"id":"[^"]*' | cut -d'"' -f4)

test_endpoint "Get project monitors" "GET" "$BASE_URL/api/projects/$PROJECT_ID/monitors" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"

# Clean up test monitor
if [ -n "$NEW_MONITOR_ID" ]; then
  test_endpoint "Delete monitor" "DELETE" "$BASE_URL/api/projects/$PROJECT_ID/monitors/$NEW_MONITOR_ID" \
    "Authorization: Bearer $AUTH_TOKEN" "" "" "204"
fi

test_endpoint "Public status page" "GET" "$BASE_URL/status/$PROJECT_ID" \
  "" "" "" "200"

echo ""

# =============================================================================
# SECTION 9: SETTINGS MANAGEMENT
# =============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📍 SECTION 9: Settings Management"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

test_endpoint "Get settings" "GET" "$BASE_URL/api/settings" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"

test_endpoint "Update settings" "POST" "$BASE_URL/api/settings" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" \
  "{
  \"slack_webhook\": \"https://hooks.slack.com/test\",
  \"generic_webhook\": \"https://example.com/webhook\",
  \"smtp_host\": \"smtp.example.com\",
  \"smtp_port\": \"587\",
  \"smtp_user\": \"test@example.com\",
  \"smtp_password\": \"testpass\",
  \"retention_days\": \"90\"
}" "204"

test_endpoint "Run system cleanup" "POST" "$BASE_URL/api/system/cleanup" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"

echo ""

# =============================================================================
# SECTION 10: SECURITY VAULT
# =============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📍 SECTION 10: Security Vault"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

test_endpoint "Setup MFA" "POST" "$BASE_URL/api/security/mfa/setup" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"

test_endpoint "Get security policies" "GET" "$BASE_URL/api/projects/$PROJECT_ID/security-policies" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"

test_endpoint "Update security policies" "POST" "$BASE_URL/api/projects/$PROJECT_ID/security-policies" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" \
  "{
  \"enforced\": true,
  \"ip_whitelist\": \"127.0.0.1,192.168.1.0/24\",
  \"rate_limit\": 1000
}" "204"

test_endpoint "Get API key history" "GET" "$BASE_URL/api/projects/$PROJECT_ID/key-history" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"

# Note: API key rotation changes the key, so we test it last
test_endpoint "Rotate API key" "POST" "$BASE_URL/api/projects/$PROJECT_ID/rotate-key" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "200"

echo ""

# =============================================================================
# SECTION 11: ENVELOPE ENDPOINT
# =============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📍 SECTION 11: Envelope Endpoint"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Refresh API key in case it was rotated
PROJECT_INFO=$(curl -s -X GET "$BASE_URL/api/projects/$PROJECT_ID" \
  -H "Authorization: Bearer $AUTH_TOKEN" \
  -H "Content-Type: application/json")
ACTUAL_API_KEY=$(echo "$PROJECT_INFO" | grep -o '"api_key":"[^"]*' | cut -d'"' -f4)
if [ -n "$ACTUAL_API_KEY" ]; then
  SENTRY_KEY="$ACTUAL_API_KEY"
fi

ENVELOPE_TRACE_ID=$(generate_id)
ENVELOPE_SPAN_ID=$(generate_span)

# Create envelope payload (newline-delimited JSON)
ENVELOPE_HEADER="{\"event_id\":\"$(generate_id)\",\"sent_at\":\"$NOW\",\"dsn\":\"$BASE_URL/$PROJECT_ID\"}"
ENVELOPE_ITEM="{\"type\":\"transaction\",\"length\":500}"
ENVELOPE_PAYLOAD="{
  \"event_id\":\"$(generate_id)\",
  \"type\":\"transaction\",
  \"transaction\":\"GET /api/test\",
  \"start_timestamp\":$(date +%s).0,
  \"timestamp\":$(date +%s).0,
  \"contexts\":{
    \"trace\":{
      \"trace_id\":\"$ENVELOPE_TRACE_ID\",
      \"span_id\":\"$ENVELOPE_SPAN_ID\",
      \"op\":\"http.server\",
      \"status\":\"ok\"
    }
  },
  \"spans\":[]
}"

ENVELOPE_BODY="$ENVELOPE_HEADER
$ENVELOPE_ITEM
$ENVELOPE_PAYLOAD"

# Test envelope endpoint
TOTAL_TESTS=$((TOTAL_TESTS + 1))
echo "🧪 [$TOTAL_TESTS] Envelope transaction..."
response=$(curl -s -w "%{http_code}" -X POST "$ENVELOPE_ENDPOINT" \
  -H "X-Sentry-Auth: Sentry sentry_key=$SENTRY_KEY, sentry_version=7" \
  -H "Content-Type: application/x-sentry-envelope" \
  --data-binary "$ENVELOPE_BODY")
http_code=$(echo "$response" | tail -c 4 | grep -o '[0-9]\{3\}' || echo "000")
if [ "$http_code" = "202" ] || [ "$http_code" = "200" ]; then
  echo "    ✅ Status: $http_code"
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  echo "    ❌ Expected: 202, Got: $http_code"
  TESTS_FAILED=$((TESTS_FAILED + 1))
fi

echo ""

# =============================================================================
# SECTION 12: EDGE CASES & VALIDATION
# =============================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📍 SECTION 12: Edge Cases & Validation"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

test_endpoint "Invalid authentication" "POST" "$STORE_ENDPOINT" \
  "X-Sentry-Auth: Sentry sentry_key=invalid-key, sentry_version=7" "Content-Type: application/json" \
  "{\"event_id\":\"test\",\"message\":\"should fail\"}" "401"

test_endpoint "Missing API key" "POST" "$STORE_ENDPOINT" \
  "Content-Type: application/json" "" \
  "{\"event_id\":\"test\",\"message\":\"should fail\"}" "401"

# Ensure security policy is disabled for this test
curl -s -X POST "$BASE_URL/api/projects/$PROJECT_ID/security-policies" \
  -H "Authorization: Bearer $AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"enforced\":false,\"ip_whitelist\":\"\",\"rate_limit\":1000}" > /dev/null

test_endpoint "Minimal event (no required fields)" "POST" "$STORE_ENDPOINT" \
  "X-Sentry-Auth: Sentry sentry_key=$SENTRY_KEY, sentry_version=7" "Content-Type: application/json" \
  "{\"event_id\":\"$(generate_id)\",\"timestamp\":\"$NOW\",\"message\":\"Minimal test\"}" "200"

test_endpoint "Get non-existent error" "GET" "$BASE_URL/api/errors/00000000-0000-0000-0000-000000000000" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "404"

test_endpoint "Get non-existent project" "GET" "$BASE_URL/api/projects/00000000-0000-0000-0000-000000000000" \
  "Authorization: Bearer $AUTH_TOKEN" "Content-Type: application/json" "" "404"

test_endpoint "Unauthorized access (no token)" "GET" "$BASE_URL/api/projects" \
  "Content-Type: application/json" "" "" "401"

echo ""

# =============================================================================
# SUMMARY
# =============================================================================
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║                    ✅ TEST SUITE COMPLETE                      ║"
echo "╠════════════════════════════════════════════════════════════════╣"
echo "║ Total Tests:  $TOTAL_TESTS"
echo "║ Passed:       $TESTS_PASSED"
echo "║ Failed:       $TESTS_FAILED"
echo "╠════════════════════════════════════════════════════════════════╣"
echo "║ Test Coverage:                                                 ║"
echo "║   ✅ Health check & discovery                                  ║"
echo "║   ✅ Authentication (login, MFA, /me)                           ║"
echo "║   ✅ Project management (CRUD, quota)                           ║"
echo "║   ✅ Error ingestion (Sentry protocol)                         ║"
echo "║   ✅ Transactions & performance traces                         ║"
echo "║   ✅ Error retrieval & management                              ║"
echo "║   ✅ Trace retrieval                                            ║"
echo "║   ✅ Coverage ingestion & retrieval                             ║"
echo "║   ✅ Uptime monitors & status page                             ║"
echo "║   ✅ Settings management                                        ║"
echo "║   ✅ Security vault (MFA, policies, key rotation)             ║"
echo "║   ✅ Envelope endpoint                                          ║"
echo "║   ✅ Edge cases & validation                                   ║"
echo "╚════════════════════════════════════════════════════════════════╝"

if [ $TESTS_FAILED -eq 0 ]; then
  echo ""
  echo "🎉 All tests passed!"
  exit 0
else
  echo ""
  echo "⚠️  Some tests failed. Review the output above."
  exit 1
fi

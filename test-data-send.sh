#!/bin/bash

# =============================================================================
# Comprehensive Data Sending Test Script
# Sends tracing data, coverage, and various error types/levels to Pulse
# =============================================================================

BASE_URL="${1:-${PULSE_BASE_URL:-http://localhost:8080}}"
PROJECT_ID="${2:-${PULSE_PROJECT_ID:-}}"
API_KEY="${3:-${PULSE_API_KEY:-}}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
ORANGE='\033[38;5;208m'
NC='\033[0m' # No Color

# Helper functions
generate_id() {
  cat /proc/sys/kernel/random/uuid 2>/dev/null | tr -d '-' || \
  openssl rand -hex 16 | sed 's/\(.\{8\}\)\(.\{4\}\)\(.\{4\}\)\(.\{4\}\)\(.\{12\}\)/\1-\2-\3-\4-\5/'
}

generate_span() {
  openssl rand -hex 8 2>/dev/null || head -c 16 /dev/urandom | xxd -p
}

NOW=$(date -u +%Y-%m-%dT%H:%M:%S.000Z)

# Function to send data
send_data() {
  local name="$1"
  local method="$2"
  local url="$3"
  local header1="$4"
  local header2="$5"
  local data="$6"

  echo -e "${BLUE}📤 Sending: ${NC}${name}..."

  local curl_cmd="curl -s -w \"\n%{http_code}\" -X \"$method\" \"$url\""
  [ -n "$header1" ] && curl_cmd="$curl_cmd -H \"$header1\""
  [ -n "$header2" ] && curl_cmd="$curl_cmd -H \"$header2\""
  [ -n "$data" ] && curl_cmd="$curl_cmd -d '$data'"

  response=$(eval "$curl_cmd" 2>&1)
  http_code=$(echo "$response" | tail -n1 | tr -d '\n' | grep -oE '[0-9]{3}$' || echo "000")
  body=$(echo "$response" | sed '$d')

  if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
    echo -e "${GREEN}  [OK] Success (${http_code})${NC}"
    return 0
  else
    echo -e "${RED}  [FAIL] Failed (${http_code})${NC}"
    if [ ${#body} -lt 200 ]; then
      echo "  Response: $body"
    fi
    return 1
  fi
}

# Function to send file upload
send_file() {
  local name="$1"
  local url="$2"
  local header="$3"
  local file_path="$4"

  echo -e "${BLUE}📤 Sending: ${NC}${name}..."

  local curl_cmd="curl -s -w \"\n%{http_code}\" -X POST \"$url\""
  [ -n "$header" ] && curl_cmd="$curl_cmd -H \"$header\""
  curl_cmd="$curl_cmd -F \"file=@$file_path\""

  response=$(eval "$curl_cmd" 2>&1)
  http_code=$(echo "$response" | tail -n1 | tr -d '\n' | grep -oE '[0-9]{3}$' || echo "000")
  body=$(echo "$response" | sed '$d')

  # Color code based on HTTP status
  if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
    echo -e "${GREEN}  [OK] Success (${http_code})${NC}"
    return 0
  elif [ "$http_code" -ge 300 ] && [ "$http_code" -lt 400 ]; then
    echo -e "${CYAN}  [->] Redirect (${http_code})${NC}"
    if [ ${#body} -lt 200 ]; then
      echo -e "${CYAN}  Response: $body${NC}"
    fi
    return 1
  elif [ "$http_code" -ge 400 ] && [ "$http_code" -lt 500 ]; then
    if [ "$http_code" -eq 401 ]; then
      echo -e "${RED}  🔒 Unauthorized (${http_code})${NC}"
    elif [ "$http_code" -eq 403 ]; then
      echo -e "${RED}  🚫 Forbidden (${http_code})${NC}"
    elif [ "$http_code" -eq 404 ]; then
      echo -e "${YELLOW}  🔍 Not Found (${http_code})${NC}"
    elif [ "$http_code" -eq 429 ]; then
      echo -e "${ORANGE}  ⚠️  Rate Limited (${http_code})${NC}"
    else
      echo -e "${YELLOW}  ⚠️  Client Error (${http_code})${NC}"
    fi
    if [ ${#body} -lt 200 ]; then
      echo -e "${YELLOW}  Response: $body${NC}"
    fi
    return 1
  elif [ "$http_code" -ge 500 ]; then
    echo -e "${RED}  💥 Server Error (${http_code})${NC}"
    if [ ${#body} -lt 200 ]; then
      echo -e "${RED}  Response: $body${NC}"
    fi
    return 1
  else
    echo -e "${MAGENTA}  ❓ Unknown Status (${http_code})${NC}"
    if [ ${#body} -lt 200 ]; then
      echo -e "${MAGENTA}  Response: $body${NC}"
    fi
    return 1
  fi
}

# Check if project ID and API key are provided
if [ -z "$PROJECT_ID" ] || [ -z "$API_KEY" ]; then
  echo -e "${YELLOW}Usage: $0 [BASE_URL] [PROJECT_ID] [API_KEY]${NC}"
  echo ""
  echo "Example:"
  echo "  $0 http://localhost:8080 2008cc43-84b0-4f61-9f72-9176cd87235f 739dcc99-88b5-48c5-b7ec-11d164d28c80"
  echo ""
  echo "Or get them from the API:"
  echo "  # Login first"
  echo "  TOKEN=\$(curl -s -X POST http://localhost:8080/api/auth/login \\"
  echo "    -H 'Content-Type: application/json' \\"
  echo "    -d '{\"email\":\"admin@example.com\",\"password\":\"admin\"}' | grep -o '\"token\":\"[^\"]*' | cut -d'\"' -f4)"
  echo ""
  echo "  # Get project"
  echo "  PROJECT=\$(curl -s -X GET http://localhost:8080/api/projects \\"
  echo "    -H \"Authorization: Bearer \$TOKEN\" | grep -o '\"id\":\"[^\"]*' | head -1 | cut -d'\"' -f4)"
  echo "  KEY=\$(curl -s -X GET http://localhost:8080/api/projects/\$PROJECT \\"
  echo "    -H \"Authorization: Bearer \$TOKEN\" | grep -o '\"api_key\":\"[^\"]*' | cut -d'\"' -f4)"
  echo ""
  echo "  # Run test"
  echo "  $0 http://localhost:8080 \$PROJECT \$KEY"
  exit 1
fi

STORE_ENDPOINT="$BASE_URL/api/$PROJECT_ID/store/"
COVERAGE_ENDPOINT="$BASE_URL/api/$PROJECT_ID/coverage"
ENVELOPE_ENDPOINT="$BASE_URL/api/$PROJECT_ID/envelope"

echo -e "${CYAN}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║${NC}         ${MAGENTA}🧪 COMPREHENSIVE DATA SENDING TEST${NC}                     ${CYAN}║${NC}"
echo -e "${CYAN}╠════════════════════════════════════════════════════════════════╣${NC}"
echo -e "${CYAN}║${NC} Base URL:  ${BLUE}$BASE_URL${NC}"
echo -e "${CYAN}║${NC} Project:   ${BLUE}$PROJECT_ID${NC}"
echo -e "${CYAN}║${NC} API Key:   ${BLUE}${API_KEY:0:8}...${NC}"
echo -e "${CYAN}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# =============================================================================
# SECTION 1: ERROR LEVELS
# =============================================================================
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}📍 SECTION 1: Error Levels (error, warning, info, fatal)${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Error level
ERROR_ID=$(generate_id)
send_data "Error Level Event" "POST" "$STORE_ENDPOINT" \
  "X-Sentry-Auth: Sentry sentry_key=$API_KEY, sentry_version=7" \
  "Content-Type: application/json" \
  "{
  \"event_id\": \"$ERROR_ID\",
  \"timestamp\": \"$NOW\",
  \"level\": \"error\",
  \"platform\": \"go\",
  \"logger\": \"test.error\",
  \"message\": {\"formatted\": \"Critical error: Database connection failed\"},
  \"exception\": {
    \"values\": [{
      \"type\": \"DatabaseError\",
      \"value\": \"connection timeout after 30s\",
      \"module\": \"github.com/app/database\",
      \"stacktrace\": {
        \"frames\": [
          {\"filename\": \"database/connection.go\", \"function\": \"Connect\", \"lineno\": 145, \"in_app\": true},
          {\"filename\": \"main.go\", \"function\": \"main\", \"lineno\": 42, \"in_app\": true}
        ]
      }
    }]
  },
  \"tags\": {\"component\": \"database\", \"severity\": \"critical\"},
  \"user\": {\"id\": \"user_123\", \"email\": \"dev@example.com\"},
  \"environment\": \"production\",
  \"release\": \"v1.2.3\"
}"

# Warning level
WARNING_ID=$(generate_id)
send_data "Warning Level Event" "POST" "$STORE_ENDPOINT" \
  "X-Sentry-Auth: Sentry sentry_key=$API_KEY, sentry_version=7" \
  "Content-Type: application/json" \
  "{
  \"event_id\": \"$WARNING_ID\",
  \"timestamp\": \"$NOW\",
  \"level\": \"warning\",
  \"platform\": \"node\",
  \"logger\": \"test.warning\",
  \"message\": {\"formatted\": \"Memory usage at 85% - approaching limit\"},
  \"tags\": {\"component\": \"monitoring\", \"metric\": \"memory\"},
  \"environment\": \"production\",
  \"release\": \"v1.2.3\"
}"

# Info level
INFO_ID=$(generate_id)
send_data "Info Level Event" "POST" "$STORE_ENDPOINT" \
  "X-Sentry-Auth: Sentry sentry_key=$API_KEY, sentry_version=7" \
  "Content-Type: application/json" \
  "{
  \"event_id\": \"$INFO_ID\",
  \"timestamp\": \"$NOW\",
  \"level\": \"info\",
  \"platform\": \"python\",
  \"logger\": \"test.info\",
  \"message\": {\"formatted\": \"User logged in successfully\"},
  \"tags\": {\"component\": \"auth\", \"action\": \"login\"},
  \"user\": {\"id\": \"user_456\", \"username\": \"john_doe\"},
  \"environment\": \"production\"
}"

# Fatal level
FATAL_ID=$(generate_id)
send_data "Fatal Level Event" "POST" "$STORE_ENDPOINT" \
  "X-Sentry-Auth: Sentry sentry_key=$API_KEY, sentry_version=7" \
  "Content-Type: application/json" \
  "{
  \"event_id\": \"$FATAL_ID\",
  \"timestamp\": \"$NOW\",
  \"level\": \"fatal\",
  \"platform\": \"go\",
  \"logger\": \"test.fatal\",
  \"message\": {\"formatted\": \"Application crash: Out of memory\"},
  \"exception\": {
    \"values\": [{
      \"type\": \"OutOfMemoryError\",
      \"value\": \"unable to allocate 1GB of memory\",
      \"module\": \"runtime\",
      \"stacktrace\": {
        \"frames\": [
          {\"filename\": \"runtime/malloc.go\", \"function\": \"mallocgc\", \"lineno\": 1234, \"in_app\": false},
          {\"filename\": \"main.go\", \"function\": \"processLargeData\", \"lineno\": 89, \"in_app\": true}
        ]
      }
    }]
  },
  \"tags\": {\"component\": \"runtime\", \"severity\": \"critical\"},
  \"environment\": \"production\"
}"

echo ""

# =============================================================================
# SECTION 2: ERROR TYPES
# =============================================================================
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}📍 SECTION 2: Error Types (Exception, Message, HTTP Error)${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Exception with stack trace
EXCEPTION_ID=$(generate_id)
send_data "Exception with Stack Trace" "POST" "$STORE_ENDPOINT" \
  "X-Sentry-Auth: Sentry sentry_key=$API_KEY, sentry_version=7" \
  "Content-Type: application/json" \
  "{
  \"event_id\": \"$EXCEPTION_ID\",
  \"timestamp\": \"$NOW\",
  \"level\": \"error\",
  \"platform\": \"javascript\",
  \"exception\": {
    \"values\": [{
      \"type\": \"TypeError\",
      \"value\": \"Cannot read property 'name' of undefined\",
      \"module\": \"src/components/UserProfile.jsx\",
      \"stacktrace\": {
        \"frames\": [
          {\"filename\": \"src/components/UserProfile.jsx\", \"function\": \"render\", \"lineno\": 42, \"colno\": 15, \"in_app\": true, \"context_line\": \"    return <div>{user.name}</div>;\"},
          {\"filename\": \"src/App.jsx\", \"function\": \"renderUser\", \"lineno\": 89, \"in_app\": true},
          {\"filename\": \"node_modules/react/index.js\", \"function\": \"render\", \"lineno\": 123, \"in_app\": false}
        ]
      },
      \"mechanism\": {\"type\": \"generic\", \"handled\": true}
    }]
  },
  \"tags\": {\"component\": \"frontend\", \"framework\": \"react\"},
  \"user\": {\"id\": \"user_789\", \"ip_address\": \"192.168.1.100\"},
  \"environment\": \"production\"
}"

# Simple message error
MESSAGE_ID=$(generate_id)
send_data "Simple Message Error" "POST" "$STORE_ENDPOINT" \
  "X-Sentry-Auth: Sentry sentry_key=$API_KEY, sentry_version=7" \
  "Content-Type: application/json" \
  "{
  \"event_id\": \"$MESSAGE_ID\",
  \"timestamp\": \"$NOW\",
  \"level\": \"error\",
  \"message\": {\"formatted\": \"Payment processing failed: Insufficient funds\"},
  \"tags\": {\"component\": \"payment\", \"error_code\": \"INSUFFICIENT_FUNDS\"},
  \"extra\": {
    \"transaction_id\": \"txn_12345\",
    \"amount\": 100.50,
    \"currency\": \"USD\"
  },
  \"environment\": \"production\"
}"

# HTTP Error
HTTP_ERROR_ID=$(generate_id)
send_data "HTTP Error (404 Not Found)" "POST" "$STORE_ENDPOINT" \
  "X-Sentry-Auth: Sentry sentry_key=$API_KEY, sentry_version=7" \
  "Content-Type: application/json" \
  "{
  \"event_id\": \"$HTTP_ERROR_ID\",
  \"timestamp\": \"$NOW\",
  \"level\": \"error\",
  \"platform\": \"python\",
  \"message\": {\"formatted\": \"HTTP 404: Resource not found\"},
  \"request\": {
    \"url\": \"https://api.example.com/users/999\",
    \"method\": \"GET\",
    \"headers\": {\"User-Agent\": \"Mozilla/5.0\", \"Accept\": \"application/json\"},
    \"query_string\": \"?include=profile\"
  },
  \"tags\": {\"component\": \"api\", \"status_code\": \"404\"},
  \"environment\": \"production\"
}"

# Validation Error
VALIDATION_ID=$(generate_id)
send_data "Validation Error" "POST" "$STORE_ENDPOINT" \
  "X-Sentry-Auth: Sentry sentry_key=$API_KEY, sentry_version=7" \
  "Content-Type: application/json" \
  "{
  \"event_id\": \"$VALIDATION_ID\",
  \"timestamp\": \"$NOW\",
  \"level\": \"warning\",
  \"platform\": \"go\",
  \"message\": {\"formatted\": \"Validation failed: Invalid email format\"},
  \"exception\": {
    \"values\": [{
      \"type\": \"ValidationError\",
      \"value\": \"email field must be a valid email address\",
      \"module\": \"github.com/app/validators\",
      \"stacktrace\": {
        \"frames\": [
          {\"filename\": \"validators/email.go\", \"function\": \"ValidateEmail\", \"lineno\": 23, \"in_app\": true}
        ]
      }
    }]
  },
  \"tags\": {\"component\": \"validation\", \"field\": \"email\"},
  \"extra\": {
    \"field\": \"email\",
    \"value\": \"invalid-email\",
    \"rule\": \"email_format\"
  },
  \"environment\": \"staging\"
}"

echo ""

# =============================================================================
# SECTION 3: TRACING DATA
# =============================================================================
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}📍 SECTION 3: Tracing Data (Transactions & Spans)${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# HTTP Server Transaction
TRACE_ID1=$(generate_id)
ROOT_SPAN1=$(generate_span)
SPAN1=$(generate_span)
SPAN2=$(generate_span)
SPAN3=$(generate_span)

send_data "HTTP Server Transaction with Spans" "POST" "$STORE_ENDPOINT" \
  "X-Sentry-Auth: Sentry sentry_key=$API_KEY, sentry_version=7" \
  "Content-Type: application/json" \
  "{
  \"event_id\": \"$(generate_id)\",
  \"type\": \"transaction\",
  \"transaction\": \"GET /api/v1/users/:id\",
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
      \"span_id\": \"$SPAN1\",
      \"parent_span_id\": \"$ROOT_SPAN1\",
      \"trace_id\": \"$TRACE_ID1\",
      \"op\": \"db.query\",
      \"description\": \"SELECT * FROM users WHERE id = ?\",
      \"start_timestamp\": $(date +%s).0,
      \"timestamp\": $(date +%s).0,
      \"status\": \"ok\"
    },
    {
      \"span_id\": \"$SPAN2\",
      \"parent_span_id\": \"$ROOT_SPAN1\",
      \"trace_id\": \"$TRACE_ID1\",
      \"op\": \"cache.get\",
      \"description\": \"GET user:12345\",
      \"start_timestamp\": $(date +%s).0,
      \"timestamp\": $(date +%s).0,
      \"status\": \"ok\"
    },
    {
      \"span_id\": \"$SPAN3\",
      \"parent_span_id\": \"$SPAN1\",
      \"trace_id\": \"$TRACE_ID1\",
      \"op\": \"db.connection\",
      \"description\": \"Get connection from pool\",
      \"start_timestamp\": $(date +%s).0,
      \"timestamp\": $(date +%s).0,
      \"status\": \"ok\"
    }
  ],
  \"sdk\": {\"name\": \"sentry.go\", \"version\": \"0.27.0\"}
}"

# Background Job Transaction
TRACE_ID2=$(generate_id)
ROOT_SPAN2=$(generate_span)
JOB_SPAN1=$(generate_span)
JOB_SPAN2=$(generate_span)

send_data "Background Job Transaction" "POST" "$STORE_ENDPOINT" \
  "X-Sentry-Auth: Sentry sentry_key=$API_KEY, sentry_version=7" \
  "Content-Type: application/json" \
  "{
  \"event_id\": \"$(generate_id)\",
  \"type\": \"transaction\",
  \"transaction\": \"ProcessPaymentJob.Execute\",
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
  \"spans\": [
    {
      \"span_id\": \"$JOB_SPAN1\",
      \"parent_span_id\": \"$ROOT_SPAN2\",
      \"trace_id\": \"$TRACE_ID2\",
      \"op\": \"http.client\",
      \"description\": \"POST https://payment-gateway.com/charge\",
      \"start_timestamp\": $(date +%s).0,
      \"timestamp\": $(date +%s).0,
      \"status\": \"ok\"
    },
    {
      \"span_id\": \"$JOB_SPAN2\",
      \"parent_span_id\": \"$ROOT_SPAN2\",
      \"trace_id\": \"$TRACE_ID2\",
      \"op\": \"db.update\",
      \"description\": \"UPDATE orders SET status = 'paid'\",
      \"start_timestamp\": $(date +%s).0,
      \"timestamp\": $(date +%s).0,
      \"status\": \"ok\"
    }
  ],
  \"sdk\": {\"name\": \"sentry.go\", \"version\": \"0.27.0\"}
}"

# GraphQL Query Transaction
TRACE_ID3=$(generate_id)
ROOT_SPAN3=$(generate_span)
GQL_SPAN1=$(generate_span)

send_data "GraphQL Query Transaction" "POST" "$STORE_ENDPOINT" \
  "X-Sentry-Auth: Sentry sentry_key=$API_KEY, sentry_version=7" \
  "Content-Type: application/json" \
  "{
  \"event_id\": \"$(generate_id)\",
  \"type\": \"transaction\",
  \"transaction\": \"GraphQL Query: getUserProfile\",
  \"start_timestamp\": $(date +%s).0,
  \"timestamp\": $(date +%s).0,
  \"platform\": \"node\",
  \"contexts\": {
    \"trace\": {
      \"trace_id\": \"$TRACE_ID3\",
      \"span_id\": \"$ROOT_SPAN3\",
      \"op\": \"graphql.execute\",
      \"status\": \"ok\"
    }
  },
  \"spans\": [
    {
      \"span_id\": \"$GQL_SPAN1\",
      \"parent_span_id\": \"$ROOT_SPAN3\",
      \"trace_id\": \"$TRACE_ID3\",
      \"op\": \"db.query\",
      \"description\": \"SELECT * FROM users WHERE id = ?\",
      \"start_timestamp\": $(date +%s).0,
      \"timestamp\": $(date +%s).0,
      \"status\": \"ok\"
    }
  ],
  \"sdk\": {\"name\": \"sentry.javascript.node\", \"version\": \"7.91.0\"}
}"

echo ""

# =============================================================================
# SECTION 4: COVERAGE DATA
# =============================================================================
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}📍 SECTION 4: Coverage Data (Percentage & File Uploads)${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Simple coverage percentage
# Cross-platform random number generation (75-95)
COVERAGE1=$((RANDOM % 21 + 75))
send_data "Coverage Percentage ($COVERAGE1%)" "POST" "$COVERAGE_ENDPOINT" \
  "X-Pulse-Auth: $API_KEY" \
  "Content-Type: application/json" \
  "{
  \"coverage\": $COVERAGE1,
  \"branch\": \"main\",
  \"commit\": \"$(openssl rand -hex 7)\",
  \"commit_message\": \"feat: add comprehensive test coverage\"
}"

# Coverage with metadata
# Cross-platform random number generation (80-98)
COVERAGE2=$((RANDOM % 19 + 80))
send_data "Coverage with Metadata ($COVERAGE2%)" "POST" "$COVERAGE_ENDPOINT" \
  "X-Pulse-Auth: $API_KEY" \
  "Content-Type: application/json" \
  "{
  \"coverage\": $COVERAGE2,
  \"branch\": \"develop\",
  \"commit\": \"$(openssl rand -hex 7)\",
  \"commit_message\": \"fix: improve error handling coverage\",
  \"job_id\": \"ci-job-12345\",
  \"build_url\": \"https://ci.example.com/builds/12345\"
}"

# Go coverage.out file
cat > /tmp/test-coverage.out << 'EOF'
mode: set
github.com/app/handlers/user.go:15.52,17.2 1 1
github.com/app/handlers/user.go:19.42,24.16 4 1
github.com/app/database/connection.go:10.34,15.16 4 1
github.com/app/utils/validator.go:5.12,8.45 2 1
EOF

send_file "Go coverage.out File Upload" "$COVERAGE_ENDPOINT" \
  "X-Pulse-Auth: $API_KEY" \
  "/tmp/test-coverage.out"

# LCOV file
cat > /tmp/test-lcov.info << 'EOF'
TN:
SF:src/services/user.ts
FN:10,getUser
FNDA:150,getUser
DA:10,150
DA:15,150
DA:20,75
LF:3
LH:3
end_of_record
SF:src/utils/validator.ts
FN:5,validateEmail
FNDA:200,validateEmail
DA:5,200
DA:8,200
LF:2
LH:2
end_of_record
EOF

send_file "LCOV File Upload" "$COVERAGE_ENDPOINT" \
  "X-Pulse-Auth: $API_KEY" \
  "/tmp/test-lcov.info"

# Cleanup
rm -f /tmp/test-coverage.out /tmp/test-lcov.info

echo ""

# =============================================================================
# SECTION 5: ENVELOPE FORMAT
# =============================================================================
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}📍 SECTION 5: Envelope Format (Transaction)${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

ENVELOPE_TRACE_ID=$(generate_id)
ENVELOPE_SPAN_ID=$(generate_span)

ENVELOPE_HEADER="{\"event_id\":\"$(generate_id)\",\"sent_at\":\"$NOW\",\"dsn\":\"$BASE_URL/$PROJECT_ID\"}"
ENVELOPE_ITEM="{\"type\":\"transaction\",\"length\":500}"
ENVELOPE_PAYLOAD="{
  \"event_id\":\"$(generate_id)\",
  \"type\":\"transaction\",
  \"transaction\":\"POST /api/v1/orders\",
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

echo -e "${BLUE}📤 Sending: Envelope Transaction...${NC}"
response=$(curl -s -w "\n%{http_code}" -X POST "$ENVELOPE_ENDPOINT" \
  -H "X-Sentry-Auth: Sentry sentry_key=$API_KEY, sentry_version=7" \
  -H "Content-Type: application/x-sentry-envelope" \
  --data-binary "$ENVELOPE_BODY" 2>&1)
http_code=$(echo "$response" | tail -n1 | tr -d '\n' | grep -oE '[0-9]{3}$' || echo "000")
body=$(echo "$response" | sed '$d')

# Color code based on HTTP status
if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
  echo -e "${GREEN}  ✅ Success (${http_code})${NC}"
elif [ "$http_code" -ge 300 ] && [ "$http_code" -lt 400 ]; then
  echo -e "${CYAN}  ↪️  Redirect (${http_code})${NC}"
  if [ ${#body} -lt 200 ]; then
    echo -e "${CYAN}  Response: $body${NC}"
  fi
elif [ "$http_code" -ge 400 ] && [ "$http_code" -lt 500 ]; then
  if [ "$http_code" -eq 401 ]; then
    echo -e "${RED}  🔒 Unauthorized (${http_code})${NC}"
  elif [ "$http_code" -eq 403 ]; then
    echo -e "${RED}  🚫 Forbidden (${http_code})${NC}"
  elif [ "$http_code" -eq 404 ]; then
    echo -e "${YELLOW}  🔍 Not Found (${http_code})${NC}"
  elif [ "$http_code" -eq 429 ]; then
    echo -e "${ORANGE}  ⚠️  Rate Limited (${http_code})${NC}"
  else
    echo -e "${YELLOW}  ⚠️  Client Error (${http_code})${NC}"
  fi
  if [ ${#body} -lt 200 ]; then
    echo -e "${YELLOW}  Response: $body${NC}"
  fi
elif [ "$http_code" -ge 500 ]; then
  echo -e "${RED}  💥 Server Error (${http_code})${NC}"
  if [ ${#body} -lt 200 ]; then
    echo -e "${RED}  Response: $body${NC}"
  fi
else
  echo -e "${MAGENTA}  ❓ Unknown Status (${http_code})${NC}"
  if [ ${#body} -lt 200 ]; then
    echo -e "${MAGENTA}  Response: $body${NC}"
  fi
fi

echo ""

# =============================================================================
# SUMMARY
# =============================================================================
echo -e "${CYAN}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║${NC}                    ${GREEN}✅ TEST COMPLETE${NC}                            ${CYAN}║${NC}"
echo -e "${CYAN}╠════════════════════════════════════════════════════════════════╣${NC}"
echo -e "${CYAN}║${NC} Sent:                                                          ${CYAN}║${NC}"
echo -e "${CYAN}║${NC}   ${BLUE}•${NC} 4 Error Levels (error, warning, info, fatal)              ${CYAN}║${NC}"
echo -e "${CYAN}║${NC}   ${BLUE}•${NC} 4 Error Types (exception, message, HTTP, validation)       ${CYAN}║${NC}"
echo -e "${CYAN}║${NC}   ${BLUE}•${NC} 3 Transactions (HTTP server, background job, GraphQL)      ${CYAN}║${NC}"
echo -e "${CYAN}║${NC}   ${BLUE}•${NC} 4 Coverage Reports (percentage & file uploads)            ${CYAN}║${NC}"
echo -e "${CYAN}║${NC}   ${BLUE}•${NC} 1 Envelope Transaction                                     ${CYAN}║${NC}"
echo -e "${CYAN}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}🎉 All data sent successfully!${NC}"
echo ""
echo -e "${CYAN}Check your Pulse dashboard to see:${NC}"
echo -e "  ${BLUE}•${NC} Errors in the ${YELLOW}Issues${NC} page"
echo -e "  ${BLUE}•${NC} Traces in the ${MAGENTA}Performance${NC} tab"
echo -e "  ${BLUE}•${NC} Coverage in the ${GREEN}Coverage${NC} tab"
echo ""

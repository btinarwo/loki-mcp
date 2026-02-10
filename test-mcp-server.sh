#!/bin/bash

# MCP Server Testing Script
# Tests all three Loki MCP tools via JSON-RPC

set -e

# Configuration
MCP_URL="${MCP_URL:-http://localhost:8000/mcp}"
LOKI_URL="${LOKI_URL:-http://localhost:3100}"
SESSION_ID="test-session-$(date +%s)"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "=========================================="
echo "MCP Server Testing Script"
echo "=========================================="
echo ""
echo "Configuration:"
echo "  MCP URL:     $MCP_URL"
echo "  Loki URL:    $LOKI_URL"
echo "  Session ID:  $SESSION_ID"
echo ""
echo "=========================================="
echo ""

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "Warning: jq is not installed. Output will not be formatted."
    echo "Install jq for better output: brew install jq (macOS) or apt-get install jq (Linux)"
    JQ_CMD="cat"
else
    JQ_CMD="jq ."
fi

# Function to make JSON-RPC request
make_request() {
    local title="$1"
    local data="$2"
    
    echo -e "${BLUE}=== $title ===${NC}"
    echo ""
    
    curl -s -X POST "$MCP_URL" \
        -H "Content-Type: application/json" \
        -H "Accept: application/json" \
        -H "Mcp-Session-Id: $SESSION_ID" \
        -d "$data" | $JQ_CMD
    
    echo ""
    echo ""
}

# Step 1: Initialize Session
make_request "Step 1: Initialize Session" '{
  "jsonrpc": "2.0",
  "id": "init-1",
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "test-client",
      "version": "1.0.0"
    }
  }
}'

# Step 2: List Available Tools
make_request "Step 2: List Available Tools" '{
  "jsonrpc": "2.0",
  "id": "list-1",
  "method": "tools/list",
  "params": {}
}'

# Step 3: Get Label Names
make_request "Step 3: Get Label Names" '{
  "jsonrpc": "2.0",
  "id": "call-1",
  "method": "tools/call",
  "params": {
    "name": "loki_label_names",
    "arguments": {}
  }
}'

# Step 4: Get Label Values for 'job'
make_request "Step 4: Get Label Values for 'job'" '{
  "jsonrpc": "2.0",
  "id": "call-2",
  "method": "tools/call",
  "params": {
    "name": "loki_label_values",
    "arguments": {
      "label": "job"
    }
  }
}'

# Step 5: Query Logs (Basic)
make_request "Step 5: Query Logs (Basic - Last 5 logs)" '{
  "jsonrpc": "2.0",
  "id": "call-3",
  "method": "tools/call",
  "params": {
    "name": "loki_query",
    "arguments": {
      "query": "{job=\"app-logs\"}",
      "limit": 5
    }
  }
}'

# Step 6: Query Logs (With Time Range)
make_request "Step 6: Query Logs (Last 5 minutes, limit 10)" '{
  "jsonrpc": "2.0",
  "id": "call-4",
  "method": "tools/call",
  "params": {
    "name": "loki_query",
    "arguments": {
      "query": "{job=\"app-logs\"}",
      "start": "-5m",
      "end": "now",
      "limit": 10
    }
  }
}'

# Step 7: Query Logs (With Custom Loki URL)
make_request "Step 7: Query Logs (With Custom Loki URL)" "{
  \"jsonrpc\": \"2.0\",
  \"id\": \"call-5\",
  \"method\": \"tools/call\",
  \"params\": {
    \"name\": \"loki_query\",
    \"arguments\": {
      \"query\": \"{job=\\\"app-logs\\\"}\",
      \"url\": \"$LOKI_URL\",
      \"limit\": 3
    }
  }
}"

# Summary
echo "=========================================="
echo -e "${GREEN}Testing Complete!${NC}"
echo "=========================================="
echo ""
echo "All MCP tools have been tested:"
echo "  ✓ initialize"
echo "  ✓ tools/list"
echo "  ✓ loki_label_names"
echo "  ✓ loki_label_values"
echo "  ✓ loki_query (basic)"
echo "  ✓ loki_query (with time range)"
echo "  ✓ loki_query (with custom URL)"
echo ""
echo "Session ID used: $SESSION_ID"
echo ""

#!/bin/bash

# Test script for the agent configuration feature

set -e

echo "========================================="
echo "Agent configuration feature test"
echo "========================================="
echo ""

# Colour definitions
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
API_BASE_URL="http://localhost:8080"
KB_ID="kb-00000001"  # Change this to your knowledge base ID
TENANT_ID="1"

echo "Configuration:"
echo "  API address: ${API_BASE_URL}"
echo "  Knowledge base ID: ${KB_ID}"
echo "  Space ID: ${TENANT_ID}"
echo ""

# Test 1: fetch the current configuration
echo -e "${YELLOW}Test 1: fetch the current configuration${NC}"
echo "GET ${API_BASE_URL}/api/v1/initialization/config/${KB_ID}"
RESPONSE=$(curl -s -X GET "${API_BASE_URL}/api/v1/initialization/config/${KB_ID}")
echo "Response:"
echo "$RESPONSE" | jq '.data.agent' || echo "$RESPONSE"
echo ""

# Test 2: save the agent configuration
echo -e "${YELLOW}Test 2: save the agent configuration${NC}"
echo "POST ${API_BASE_URL}/api/v1/initialization/initialize/${KB_ID}"

# Prepare the test payload (it must carry the complete configuration)
TEST_DATA='{
  "llm": {
    "source": "local",
    "modelName": "qwen3:0.6b",
    "baseUrl": "",
    "apiKey": ""
  },
  "embedding": {
    "source": "local",
    "modelName": "nomic-embed-text:latest",
    "baseUrl": "",
    "apiKey": "",
    "dimension": 768
  },
  "rerank": {
    "enabled": false
  },
  "multimodal": {
    "enabled": false
  },
  "documentSplitting": {
    "chunkSize": 512,
    "chunkOverlap": 100,
    "separators": ["\n\n", "\n", "।", ". "]
  },
  "nodeExtract": {
    "enabled": false
  },
  "agent": {
    "enabled": true,
    "maxIterations": 8,
    "temperature": 0.8,
    "allowedTools": ["knowledge_search", "multi_kb_search", "list_knowledge_bases"]
  }
}'

RESPONSE=$(curl -s -X POST "${API_BASE_URL}/api/v1/initialization/initialize/${KB_ID}" \
  -H "Content-Type: application/json" \
  -d "$TEST_DATA")

if echo "$RESPONSE" | grep -q '"success":true'; then
  echo -e "${GREEN}\u2713 Agent configuration saved successfully${NC}"
  echo "$RESPONSE" | jq '.' || echo "$RESPONSE"
else
  echo -e "${RED}\u2717 Failed to save the agent configuration${NC}"
  echo "$RESPONSE"
fi
echo ""

# Pause briefly so the data is definitely persisted
sleep 1

# Test 3: verify the configuration was saved
echo -e "${YELLOW}Test 3: verify the configuration was saved${NC}"
echo "GET ${API_BASE_URL}/api/v1/initialization/config/${KB_ID}"
RESPONSE=$(curl -s -X GET "${API_BASE_URL}/api/v1/initialization/config/${KB_ID}")
AGENT_CONFIG=$(echo "$RESPONSE" | jq '.data.agent')

echo "Agent configuration:"
echo "$AGENT_CONFIG" | jq '.'

# Check that the configuration is correct
ENABLED=$(echo "$AGENT_CONFIG" | jq -r '.enabled')
MAX_ITER=$(echo "$AGENT_CONFIG" | jq -r '.maxIterations')
TEMP=$(echo "$AGENT_CONFIG" | jq -r '.temperature')

if [ "$ENABLED" == "true" ] && [ "$MAX_ITER" == "8" ] && [ "$TEMP" == "0.8" ]; then
  echo -e "${GREEN}\u2713 Configuration verified - all values correct${NC}"
else
  echo -e "${RED}\u2717 Configuration verification failed${NC}"
  echo "  enabled: $ENABLED (expected: true)"
  echo "  maxIterations: $MAX_ITER (expected: 8)"
  echo "  temperature: $TEMP (expected: 0.8)"
fi
echo ""

# Test 4: fetch the configuration through the Tenant API
echo -e "${YELLOW}Test 4: fetch the configuration through the Tenant API${NC}"
echo "GET ${API_BASE_URL}/api/v1/tenants/${TENANT_ID}/agent-config"
RESPONSE=$(curl -s -X GET "${API_BASE_URL}/api/v1/tenants/${TENANT_ID}/agent-config")
echo "Response:"
echo "$RESPONSE" | jq '.' || echo "$RESPONSE"
echo ""

# Test 5: database verification (if the database is reachable)
echo -e "${YELLOW}Test 5: database verification${NC}"
echo "Tip: run the SQL queries below by hand to verify the data:"
echo ""
echo "MySQL:"
echo "  mysql -u root -p enterpriserag -e \"SELECT id, agent_config FROM tenants WHERE id = ${TENANT_ID};\""
echo ""
echo "PostgreSQL:"
echo "  psql -U postgres -d enterpriserag -c \"SELECT id, agent_config FROM tenants WHERE id = ${TENANT_ID};\""
echo ""

echo "========================================="
echo "Tests complete."
echo "========================================="
echo ""
echo "If every test passed, the agent configuration feature is working correctly."
echo "If a test failed, check the following:"
echo "  1. The backend service is running"
echo "  2. The database migrations have been applied"
echo "  3. The knowledge base ID is correct"
echo ""


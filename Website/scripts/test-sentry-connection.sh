#!/bin/bash
# Test Sentry API Connection
# Verifies auth token and retrieves organization/project info

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 Sentry Connection Test${NC}"
echo ""

# Check if SENTRY_AUTH_TOKEN is set
if [ -z "$SENTRY_AUTH_TOKEN" ]; then
    echo -e "${RED}❌ Error: SENTRY_AUTH_TOKEN environment variable not set${NC}"
    echo ""
    echo "Usage:"
    echo "  export SENTRY_AUTH_TOKEN='your-token-here'"
    echo "  ./test-sentry-connection.sh"
    exit 1
fi

echo -e "${YELLOW}Testing Sentry API connection...${NC}"
echo ""

# Test 1: Get user info (verifies token is valid)
echo -e "${BLUE}Test 1: Verify Auth Token${NC}"
USER_RESPONSE=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: Bearer $SENTRY_AUTH_TOKEN" \
    "https://sentry.io/api/0/")

HTTP_CODE=$(echo "$USER_RESPONSE" | tail -n1)
BODY=$(echo "$USER_RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✅ Auth token is valid${NC}"
    echo ""
else
    echo -e "${RED}❌ Auth token is invalid (HTTP $HTTP_CODE)${NC}"
    echo "Response: $BODY"
    exit 1
fi

# Test 2: List organizations
echo -e "${BLUE}Test 2: List Organizations${NC}"
ORG_RESPONSE=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: Bearer $SENTRY_AUTH_TOKEN" \
    "https://sentry.io/api/0/organizations/")

HTTP_CODE=$(echo "$ORG_RESPONSE" | tail -n1)
BODY=$(echo "$ORG_RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✅ Successfully retrieved organizations${NC}"
    echo ""
    echo "Available organizations:"
    echo "$BODY" | python3 -c "
import sys, json
try:
    orgs = json.load(sys.stdin)
    for org in orgs:
        print(f\"  - {org['slug']} (name: {org['name']})\")
except:
    print('  (Could not parse organizations)')
"
    echo ""
else
    echo -e "${RED}❌ Failed to retrieve organizations (HTTP $HTTP_CODE)${NC}"
    echo "Response: $BODY"
    exit 1
fi

# Test 3: Get specific organization (if ORG_SLUG provided)
if [ -n "$SENTRY_ORG_SLUG" ]; then
    echo -e "${BLUE}Test 3: Get Organization '$SENTRY_ORG_SLUG'${NC}"
    ORG_DETAIL_RESPONSE=$(curl -s -w "\n%{http_code}" \
        -H "Authorization: Bearer $SENTRY_AUTH_TOKEN" \
        "https://sentry.io/api/0/organizations/$SENTRY_ORG_SLUG/")
    
    HTTP_CODE=$(echo "$ORG_DETAIL_RESPONSE" | tail -n1)
    BODY=$(echo "$ORG_DETAIL_RESPONSE" | sed '$d')
    
    if [ "$HTTP_CODE" = "200" ]; then
        echo -e "${GREEN}✅ Organization '$SENTRY_ORG_SLUG' found${NC}"
        echo ""
    else
        echo -e "${RED}❌ Organization '$SENTRY_ORG_SLUG' not found (HTTP $HTTP_CODE)${NC}"
        echo "Response: $BODY"
        echo ""
        echo -e "${YELLOW}💡 Tip: Check the organization slug in the list above${NC}"
        exit 1
    fi
    
    # Test 4: List projects in organization
    echo -e "${BLUE}Test 4: List Projects in '$SENTRY_ORG_SLUG'${NC}"
    PROJECTS_RESPONSE=$(curl -s -w "\n%{http_code}" \
        -H "Authorization: Bearer $SENTRY_AUTH_TOKEN" \
        "https://sentry.io/api/0/organizations/$SENTRY_ORG_SLUG/projects/")
    
    HTTP_CODE=$(echo "$PROJECTS_RESPONSE" | tail -n1)
    BODY=$(echo "$PROJECTS_RESPONSE" | sed '$d')
    
    if [ "$HTTP_CODE" = "200" ]; then
        echo -e "${GREEN}✅ Successfully retrieved projects${NC}"
        echo ""
        echo "Available projects:"
        echo "$BODY" | python3 -c "
import sys, json
try:
    projects = json.load(sys.stdin)
    for proj in projects:
        print(f\"  - {proj['slug']} (name: {proj['name']}, platform: {proj.get('platform', 'unknown')})\")
except:
    print('  (Could not parse projects)')
"
        echo ""
    else
        echo -e "${RED}❌ Failed to retrieve projects (HTTP $HTTP_CODE)${NC}"
        echo "Response: $BODY"
        exit 1
    fi
    
    # Test 5: Get specific project (if PROJECT_SLUG provided)
    if [ -n "$SENTRY_PROJECT_SLUG" ]; then
        echo -e "${BLUE}Test 5: Get Project '$SENTRY_PROJECT_SLUG'${NC}"
        PROJECT_RESPONSE=$(curl -s -w "\n%{http_code}" \
            -H "Authorization: Bearer $SENTRY_AUTH_TOKEN" \
            "https://sentry.io/api/0/projects/$SENTRY_ORG_SLUG/$SENTRY_PROJECT_SLUG/")
        
        HTTP_CODE=$(echo "$PROJECT_RESPONSE" | tail -n1)
        BODY=$(echo "$PROJECT_RESPONSE" | sed '$d')
        
        if [ "$HTTP_CODE" = "200" ]; then
            echo -e "${GREEN}✅ Project '$SENTRY_PROJECT_SLUG' found${NC}"
            echo ""
        else
            echo -e "${RED}❌ Project '$SENTRY_PROJECT_SLUG' not found (HTTP $HTTP_CODE)${NC}"
            echo "Response: $BODY"
            echo ""
            echo -e "${YELLOW}💡 Tip: Check the project slug in the list above${NC}"
            exit 1
        fi
        
        # Test 6: Get recent issues
        echo -e "${BLUE}Test 6: Get Recent Issues${NC}"
        ISSUES_RESPONSE=$(curl -s -w "\n%{http_code}" \
            -H "Authorization: Bearer $SENTRY_AUTH_TOKEN" \
            "https://sentry.io/api/0/projects/$SENTRY_ORG_SLUG/$SENTRY_PROJECT_SLUG/issues/?limit=5")
        
        HTTP_CODE=$(echo "$ISSUES_RESPONSE" | tail -n1)
        BODY=$(echo "$ISSUES_RESPONSE" | sed '$d')
        
        if [ "$HTTP_CODE" = "200" ]; then
            echo -e "${GREEN}✅ Successfully retrieved recent issues${NC}"
            echo ""
            echo "Recent issues:"
            echo "$BODY" | python3 -c "
import sys, json
try:
    issues = json.load(sys.stdin)
    if len(issues) == 0:
        print('  (No issues found)')
    else:
        for issue in issues[:5]:
            print(f\"  - {issue['id']}: {issue['title'][:60]}...\")
            print(f\"    Status: {issue['status']}, Count: {issue['count']}\")
except:
    print('  (Could not parse issues)')
"
            echo ""
        else
            echo -e "${RED}❌ Failed to retrieve issues (HTTP $HTTP_CODE)${NC}"
            echo "Response: $BODY"
            exit 1
        fi
    fi
fi

echo ""
echo -e "${GREEN}✅ All tests passed!${NC}"
echo ""
echo -e "${BLUE}📋 Configuration Summary:${NC}"
echo "  Auth Token: ✅ Valid"
[ -n "$SENTRY_ORG_SLUG" ] && echo "  Organization: ✅ $SENTRY_ORG_SLUG"
[ -n "$SENTRY_PROJECT_SLUG" ] && echo "  Project: ✅ $SENTRY_PROJECT_SLUG"
echo ""
echo -e "${YELLOW}💡 Your MCP config should use:${NC}"
[ -n "$SENTRY_ORG_SLUG" ] && echo "  SENTRY_ORG_SLUG: $SENTRY_ORG_SLUG"
[ -n "$SENTRY_PROJECT_SLUG" ] && echo "  SENTRY_PROJECT_SLUG: $SENTRY_PROJECT_SLUG"

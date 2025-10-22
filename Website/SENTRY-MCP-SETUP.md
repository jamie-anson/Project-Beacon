# Sentry MCP Setup Guide

## 🎯 Goal
Connect Windsurf to Sentry via MCP server for autonomous error analysis.

---

## 📋 Prerequisites

1. **Sentry Account** with Project Beacon project
2. **Node.js** installed (for npx)
3. **Sentry Auth Token** with proper scopes

---

## 🔑 Step 1: Create Sentry Auth Token

1. Go to: https://sentry.io/settings/account/api/auth-tokens/
2. Click **"Create New Token"**
3. **Name**: `Windsurf MCP`
4. **Scopes** (select these):
   - ✅ `event:read`
   - ✅ `project:read`
   - ✅ `org:read`
   - ✅ `issue:read` (optional but recommended)
5. Click **"Create Token"**
6. **Copy the token** (you won't see it again!)

---

## 🧪 Step 2: Test Your Credentials

Run the test script to verify your token and find your org/project slugs:

```bash
cd /Users/Jammie/Desktop/Project\ Beacon/Website

# Test with just the token (will list all orgs/projects)
export SENTRY_AUTH_TOKEN="your-token-here"
./scripts/test-sentry-connection.sh

# Test with org slug (will list projects in that org)
export SENTRY_ORG_SLUG="your-org-slug"
./scripts/test-sentry-connection.sh

# Test with both (full validation)
export SENTRY_PROJECT_SLUG="your-project-slug"
./scripts/test-sentry-connection.sh
```

**Expected Output**:
```
🔍 Sentry Connection Test

Test 1: Verify Auth Token
✅ Auth token is valid

Test 2: List Organizations
✅ Successfully retrieved organizations

Available organizations:
  - jamie-anson (name: Jamie Anson)
  - project-beacon (name: Project Beacon)

Test 3: Get Organization 'project-beacon'
✅ Organization 'project-beacon' found

Test 4: List Projects in 'project-beacon'
✅ Successfully retrieved projects

Available projects:
  - runner (name: Runner, platform: go)
  - router (name: Router, platform: python)

Test 5: Get Project 'runner'
✅ Project 'runner' found

Test 6: Get Recent Issues
✅ Successfully retrieved recent issues

Recent issues:
  - 12345: Error in job execution...
    Status: unresolved, Count: 45

✅ All tests passed!

📋 Configuration Summary:
  Auth Token: ✅ Valid
  Organization: ✅ project-beacon
  Project: ✅ runner

💡 Your MCP config should use:
  SENTRY_ORG_SLUG: project-beacon
  SENTRY_PROJECT_SLUG: runner
```

---

## ⚙️ Step 3: Update MCP Config

Edit `~/.codeium/windsurf/mcp_config.json`:

```json
{
  "mcpServers": {
    "sentry": {
      "command": "npx",
      "args": [
        "-y",
        "@sentry/mcp-server"
      ],
      "env": {
        "SENTRY_AUTH_TOKEN": "sntrys_YOUR_ACTUAL_TOKEN_HERE",
        "SENTRY_ORG_SLUG": "project-beacon",
        "SENTRY_PROJECT_SLUG": "runner"
      }
    }
  }
}
```

**Replace**:
- `YOUR_ACTUAL_TOKEN_HERE` with your token from Step 1
- `project-beacon` with your org slug from Step 2
- `runner` with your project slug from Step 2

---

## 🔄 Step 4: Restart Windsurf

1. Save the MCP config file
2. Restart Windsurf completely
3. Wait for MCP servers to initialize

---

## ✅ Step 5: Verify It's Working

Ask Cascade:

```
"List the available MCP tools"
```

You should see Sentry tools like:
- `sentry_get_issues`
- `sentry_get_issue_details`
- `sentry_search_issues`

Then test it:

```
"Show me the latest errors from Sentry"
```

---

## 🐛 Troubleshooting

### Error: "Auth token is invalid"
- Token might be expired or incorrect
- Create a new token with proper scopes
- Make sure you copied the entire token

### Error: "Organization not found"
- Check the org slug matches exactly (case-sensitive)
- Run test script to see available orgs
- Use the slug, not the display name

### Error: "Project not found"
- Check the project slug matches exactly
- Run test script to see available projects
- Make sure project exists in the specified org

### Error: "npx command not found"
- Install Node.js: https://nodejs.org/
- Verify with: `node --version`

### MCP Server Not Loading
1. Check Windsurf Developer Tools (Help → Toggle Developer Tools)
2. Look for errors in Console
3. Verify JSON syntax in mcp_config.json
4. Try running manually:
   ```bash
   export SENTRY_AUTH_TOKEN="your-token"
   export SENTRY_ORG_SLUG="your-org"
   export SENTRY_PROJECT_SLUG="your-project"
   npx -y @sentry/mcp-server
   ```

---

## 🎯 What You Get

Once configured, Cascade can:

### 1. Query Recent Errors
```
"Show me errors from the last 24 hours"
```

### 2. Get Issue Details
```
"Get details for Sentry issue BEACON-123"
```

### 3. Analyze Error Patterns
```
"What are the most common errors this week?"
```

### 4. Cross-Reference with Traces
```
"Find the Sentry error for execution 2379"
```

### 5. Autonomous Debugging
```
"Why did job 475 fail? Check both traces and Sentry"
```

Cascade will:
- Query trace database (custom tracing MCP)
- Query Sentry (Sentry MCP)
- Correlate data
- Provide comprehensive diagnosis

---

## 📊 Combined Power

**Tracing MCP** + **Sentry MCP** = **Complete Observability**

```
User: "Diagnose job 475"

Cascade:
1. Queries trace_spans table → "No spans found, 10ms duration"
2. Queries Sentry issues → "RuntimeError: hybrid client not initialized"
3. Correlates data → "Job failed before tracing code ran"
4. Provides diagnosis → "Hybrid router connection issue"
5. Suggests fix → "Check HYBRID_ROUTER_URL env var"
```

---

## 🚀 Next Steps

Once Sentry MCP is working:

1. **Test autonomous debugging** with real failures
2. **Deploy tracing enhancements** (Week 2-3 code)
3. **Fix execution bug** using combined observability
4. **Verify end-to-end tracing** works

---

**Status**: Ready to configure! Run the test script first to verify credentials.

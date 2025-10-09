# Grafana MCP Setup for Fly.io Logs

**Date:** 2025-10-09  
**Purpose:** Enable AI assistant to query Fly.io logs via Grafana MCP server

---

## 🎯 Overview

Fly.io provides a hosted Grafana instance at `https://fly-metrics.net` where we can query logs using Loki.
We'll set up the official Grafana MCP server to enable direct log queries from the AI assistant.

---

## 📋 Prerequisites

1. **Grafana URL:** https://fly-metrics.net
2. **Org ID:** 1239254 (from your Grafana URL)
3. **Fly.io Auth Token:** Required for authentication

---

## 🔧 Setup Steps

### Step 1: Get Fly.io Auth Token

Run this command to get your Fly.io authentication token:

```bash
flyctl auth token
```

Copy the output - this will be your `GRAFANA_SERVICE_ACCOUNT_TOKEN`.

### Step 2: Install Grafana MCP Server

Choose one of these installation methods:

#### Option A: Docker (Recommended)

```bash
# Test the connection first
docker run --rm -i \
  -e GRAFANA_URL=https://fly-metrics.net \
  -e GRAFANA_SERVICE_ACCOUNT_TOKEN=$(flyctl auth token) \
  mcp/grafana -t stdio

# If successful, configure it for Claude Desktop (see Step 3)
```

#### Option B: Binary Download

```bash
# Download latest release
curl -L -o mcp-grafana https://github.com/grafana/mcp-grafana/releases/latest/download/mcp-grafana-darwin-arm64

# Make executable
chmod +x mcp-grafana

# Move to PATH
mv mcp-grafana /usr/local/bin/
```

#### Option C: Build from Source

```bash
GOBIN="$HOME/go/bin" go install github.com/grafana/mcp-grafana/cmd/mcp-grafana@latest
```

---

### Step 3: Configure Claude Desktop (or other MCP client)

Add to your MCP configuration file:

**Location:** `~/Library/Application Support/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "grafana": {
      "command": "docker",
      "args": [
        "run",
        "--rm",
        "-i",
        "-e",
        "GRAFANA_URL=https://fly-metrics.net",
        "-e",
        "GRAFANA_SERVICE_ACCOUNT_TOKEN=YOUR_FLY_TOKEN_HERE",
        "mcp/grafana",
        "-t",
        "stdio"
      ]
    }
  }
}
```

**OR if using binary:**

```json
{
  "mcpServers": {
    "grafana": {
      "command": "/usr/local/bin/mcp-grafana",
      "args": ["-t", "stdio"],
      "env": {
        "GRAFANA_URL": "https://fly-metrics.net",
        "GRAFANA_SERVICE_ACCOUNT_TOKEN": "YOUR_FLY_TOKEN_HERE"
      }
    }
  }
}
```

---

### Step 4: Configure Environment Variables

Create a secure environment file:

```bash
# Create config directory
mkdir -p ~/.config/project-beacon

# Store token securely (replace YOUR_TOKEN with actual token)
cat > ~/.config/project-beacon/grafana-mcp.env <<EOF
GRAFANA_URL=https://fly-metrics.net
GRAFANA_SERVICE_ACCOUNT_TOKEN=YOUR_FLY_TOKEN_HERE
EOF

# Secure the file
chmod 600 ~/.config/project-beacon/grafana-mcp.env
```

Then update MCP config to use env file:

```json
{
  "mcpServers": {
    "grafana": {
      "command": "bash",
      "args": [
        "-c",
        "source ~/.config/project-beacon/grafana-mcp.env && docker run --rm -i -e GRAFANA_URL=$GRAFANA_URL -e GRAFANA_SERVICE_ACCOUNT_TOKEN=$GRAFANA_SERVICE_ACCOUNT_TOKEN mcp/grafana -t stdio"
      ]
    }
  }
}
```

---

## 🔍 Available Tools

Once configured, you'll have access to these tools:

### Loki Log Queries
- **loki_query:** Query logs from beacon-runner-production
- **loki_labels:** Discover available log labels
- **loki_label_values:** Get values for specific labels

### Dashboards
- **list_dashboards:** View available dashboards
- **get_dashboard:** Get dashboard details
- **search_dashboards:** Search for dashboards

### Datasources
- **list_datasources:** List all datasources
- **get_datasource:** Get datasource details

### Navigation
- **navigate_to_dashboard:** Generate dashboard URLs
- **navigate_to_explore:** Generate explore URLs

---

## 📝 Example Queries for Beacon Runner

Once MCP is set up, you can ask:

### Query Worker Logs
```
Query Loki logs for beacon-runner-production app containing "Starting JobRunner" or "Hybrid Router enabled"
```

This will execute:
```logql
{app="beacon-runner-production"} |= "Starting JobRunner" or "Hybrid Router enabled"
```

### Query Job Processing
```
Query Loki logs for beacon-runner-production containing "bias-detection" in the last 30 minutes
```

This will execute:
```logql
{app="beacon-runner-production"} |= "bias-detection"
```

### Query Errors
```
Query Loki logs for beacon-runner-production at level ERROR or FATAL
```

This will execute:
```logql
{app="beacon-runner-production"} |~ "ERR|ERROR|FATAL"
```

### Query Database Issues
```
Query Loki logs for beacon-runner-production containing "database" or "postgres"
```

---

## 🧪 Test the Setup

### 1. Test Connection

```bash
# Set variables
export GRAFANA_URL=https://fly-metrics.net
export GRAFANA_SERVICE_ACCOUNT_TOKEN=$(flyctl auth token)

# Test with Docker
docker run --rm -i \
  -e GRAFANA_URL=$GRAFANA_URL \
  -e GRAFANA_SERVICE_ACCOUNT_TOKEN=$GRAFANA_SERVICE_ACCOUNT_TOKEN \
  mcp/grafana -t stdio
```

### 2. Query Logs Manually

```bash
# Use flyctl to verify app name
flyctl logs -a beacon-runner-production | head -20

# This confirms logs exist and app name is correct
```

### 3. Test MCP Client

After configuring Claude Desktop:
1. Restart Claude Desktop
2. Check if "grafana" appears in available MCP servers
3. Ask: "Query Loki logs for beacon-runner-production containing 'Hybrid Router'"

---

## 🎯 What to Look For

Once MCP is working, search for these log messages:

### ✅ Expected Logs (Should Exist)
```
"logger initialized"
"Starting Project Beacon Runner"
"database initialization"
"Connected to Redis"
"starting outbox publisher"
"Starting JobRunner"
"Hybrid Router enabled" or "Hybrid Router disabled"
```

### 🔴 Missing Logs (Current Issue)
```
"Hybrid Router explicitly disabled via HYBRID_ROUTER_DISABLE=true"
```
This confirms our diagnosis that hybrid router was disabled.

### ✅ After Fix (Should Appear)
```
"Hybrid Router enabled"
"hybrid_base": "https://project-beacon-production.up.railway.app"
"Dequeued job: bias-detection-..."
"Processing cross-region job"
"Creating execution"
```

---

## 🐛 Troubleshooting

### Error: "Failed to connect to Grafana"
- Check your Fly.io auth token: `flyctl auth token`
- Verify token hasn't expired
- Check Grafana URL is correct: https://fly-metrics.net

### Error: "No datasources found"
- Your org might not have Loki configured
- Check available datasources in Grafana UI
- May need to enable Loki datasource for your org

### Error: "MCP server not responding"
- Check Docker is running: `docker ps`
- Check MCP config syntax is valid JSON
- Restart Claude Desktop after config changes

### Logs Not Found
- Verify app name: `flyctl apps list`
- Check logs exist: `flyctl logs -a beacon-runner-production`
- Try broader query: `{app="beacon-runner-production"}`

---

## 🔗 Useful Links

- **Fly.io App Dashboard:** https://fly.io/apps/beacon-runner-production
- **Grafana Dashboard:** https://fly-metrics.net/d/fly-app/fly-app?orgId=1239254
- **Grafana MCP Docs:** https://github.com/grafana/mcp-grafana
- **Loki Query Language:** https://grafana.com/docs/loki/latest/query/

---

## 🚀 Next Steps

1. [ ] Get Fly.io auth token
2. [ ] Install Grafana MCP server (Docker or binary)
3. [ ] Configure Claude Desktop with MCP config
4. [ ] Test connection with simple query
5. [ ] Query for "Hybrid Router" logs to verify fix
6. [ ] Query for "Starting JobRunner" to confirm worker started
7. [ ] Query for job processing logs after submitting test job

---

## 💡 Pro Tips

**Time Ranges:**
- Loki queries default to last 1 hour
- Specify custom range: `{app="beacon-runner-production"} [30m]`
- For startup logs, query since deployment: `[1h]`

**Log Levels:**
- Filter by level: `{app="beacon-runner-production"} | json | level="error"`
- Multiple levels: `{app="beacon-runner-production"} |~ "INFO|WARN|ERROR"`

**Job Tracking:**
- Track specific job: `{app="beacon-runner-production"} |= "bias-detection-1759969056006"`
- Worker activity: `{app="beacon-runner-production"} |~ "Dequeued|Processing|Execution created"`

**Performance:**
- Start with narrow time ranges (5-15 minutes)
- Add filters to reduce data volume
- Use label filters before regex: `{app="beacon-runner-production"} |= "worker"`

---

## 📊 Monitoring Dashboard

Once MCP is set up, you can ask the AI to:
- Create custom dashboards for job processing metrics
- Set up alerts for worker failures
- Generate reports on job completion rates
- Analyze error patterns

This provides much better observability than manual log searching! 🎉

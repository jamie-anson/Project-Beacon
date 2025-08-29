# Admin Config API and RBAC (Spec)

This document defines the admin surface for Project Beacon’s runner/portal: endpoints, config schema, roles, and authentication. It is implemented in the mock backend for local testing and intended to be mirrored in the Go runner API.

- Base URL for examples: http://localhost:8090
- Mock backend default: http://localhost:8787

## Endpoints

- GET /auth/whoami
  - Returns the caller’s role.
  - 200: { "role": "admin" | "operator" | "viewer" | "anonymous" }

- GET /admin/config
  - Role: admin or operator (read-only).
  - Returns the current configuration JSON.

- PUT /admin/config
  - Role: admin (write).
  - Accepts a partial update document and merges it (sanitized) into the current config.

## AuthN & RBAC

- Authorization: Bearer tokens via Authorization header.
- Environment variables (comma-separated token lists):
  - ADMIN_TOKENS
  - OPERATOR_TOKENS
  - VIEWER_TOKENS
- Local dev bypass (optional):
  - ALLOW_LOCAL_NOAUTH=true elevates localhost callers to admin in the mock server.
- Roles:
  - admin: read + write admin config
  - operator: read admin config
  - viewer/anonymous: no access

## Config Schema (shape)

```jsonc
{
  "ipfs_gateway": "https://...",
  "transparency_log": {
    "enabled": true,
    "endpoint": "https://..."
  },
  "features": {
    "bias_dashboard": true,
    "provider_map": true,
    "ws_live_updates": true
  },
  "constraints": {
    "default_region": "US", // US | EU | ASIA
    "max_cost": 5.0,
    "max_duration": 900
  },
  "security": {
    "require_signature": false,
    "allowed_submitter_keys": ["...pubkey..."]
  },
  "display": {
    "maintenance_mode": false,
    "banner": ""
  }
}
```

Notes:
- Unknown fields are ignored.
- Partial PUT updates are sanitized and merged section-by-section.

## CORS

- OPTIONS preflight supported.
- Access-Control-Allow-Origin: *
- Access-Control-Allow-Methods: GET, PUT, POST, OPTIONS
- Access-Control-Allow-Headers: Content-Type, Authorization

## Examples

- Terminal C (Actions): Discover role
```bash
curl -s -H 'Authorization: Bearer $ADMIN_TOKEN' http://localhost:8090/auth/whoami
```

- Terminal C (Actions): Read admin config
```bash
curl -s -H 'Authorization: Bearer $OPERATOR_TOKEN' http://localhost:8090/admin/config | jq
```

- Terminal C (Actions): Update admin config (admin only)
```bash
curl -s -X PUT http://localhost:8090/admin/config \
  -H 'Authorization: Bearer '$ADMIN_TOKEN \
  -H 'Content-Type: application/json' \
  -d '{
    "ipfs_gateway": "https://w3s.link",
    "features": { "ws_live_updates": false },
    "constraints": { "default_region": "EU", "max_cost": 2.5 }
  }' | jq
```

Mock backend equivalents (if running the dev server): replace 8090 with 8787.

## Implementation Status

- Implemented in mock backend: `scripts/mock-backend.js`
  - GET /auth/whoami, GET/PUT /admin/config, CORS preflight
  - Env: ADMIN_TOKENS, OPERATOR_TOKENS, VIEWER_TOKENS, ALLOW_LOCAL_NOAUTH
- To implement in Go runner (Gin):
  - Middleware: Bearer token extraction → role mapping (env lists)
  - Handlers: mirror JSON schema & sanitization
  - Config persistence: in-memory for dev; Postgres table for prod (optional)

## Security Considerations

- Do not commit tokens; supply via environment vars/secrets.
- PUT should validate types/ranges (done by sanitizer in mock server).
- Consider rate limiting and audit logging in production.

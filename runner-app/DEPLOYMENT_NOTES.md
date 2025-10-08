# Runner App Deployment Notes

## Port Configuration Issue (Fixed)

**Problem:** App was listening on wrong port causing 502 errors.

**Root Cause:**
- `cmd/server/main.go` line 109: Default port is `8091`
- Server reads from `PORT` env var (not `HTTP_PORT`)
- Old fly.toml set `HTTP_PORT` which was ignored
- App listened on 8091, but Fly expected 8090

**Fix:**
- Changed `fly.production.toml` to set `PORT=8090` instead of `HTTP_PORT`
- Matches `internal_port = 8090` in services config

## Fresh Deployment Process

1. **Create new app:**
   ```bash
   flyctl apps create beacon-runner-production
   ```

2. **Deploy with correct config:**
   ```bash
   flyctl deploy -a beacon-runner-production -c fly.production.toml
   ```

3. **Set feature flags:**
   ```bash
   ./scripts/copy-secrets-to-production.sh
   ```

4. **Copy critical secrets manually:**
   - DATABASE_URL
   - REDIS_URL
   - ADMIN_TOKEN
   - TRUSTED_KEYS_JSON
   - STORACHA_TOKEN
   - HYBRID_BASE
   - OPENAI_API_KEY

5. **Test health endpoint:**
   ```bash
   curl https://beacon-runner-production.fly.dev/health
   ```

6. **Update portal configuration** to use new URL

7. **Delete old app** once verified:
   ```bash
   flyctl apps destroy beacon-runner-change-me
   ```

## New Production URL

`https://beacon-runner-production.fly.dev`

## CORS Configuration

The server has CORS middleware in two places:
1. `cmd/server/main.go` - Allows all origins (`*`)
2. `internal/api/middleware/security.go` - Whitelist specific origins

Portal URL `https://projectbeacon.netlify.app` is whitelisted in the proper middleware.

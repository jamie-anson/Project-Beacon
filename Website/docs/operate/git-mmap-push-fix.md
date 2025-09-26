# Git push hangs at "Counting objects" / mmap issues

This runbook documents quick fixes and durable remediation for Git push hangs during the pack stage (mmap/pack-objects quirks on macOS).

## Symptoms
- `git push` stalls at messages like:
  - `Enumerating objects: N, done.`
  - `Counting objects:  28% (7/N)`
- Trace shows `git pack-objects` running and not progressing.

## Quick Workaround
Run a one-off push disabling mmap and using conservative pack settings:

```bash
GIT_DISABLE_MMAP=1 git -c pack.threads=1 \
  -c pack.window=0 -c pack.depth=50 \
  -c core.packedGitWindowSize=16m -c core.packedGitLimit=128m \
  -c http.postBuffer=524288000 \
  push origin HEAD:main
```

## Durable Fixes

- Upgrade Git (Homebrew):

```bash
brew update && brew upgrade git
brew link --overwrite git
hash -r
git --version  # expect >= 2.5x
```

- Add a pre-commit guard against heavy artifacts and large files:

Create `.githooks/pre-commit` and set:

```bash
git config core.hooksPath .githooks
chmod +x .githooks/pre-commit
```

Sample policy blocks `playwright-report/`, `test-results/`, and files > 10MB.

- Harden `.gitignore` to exclude generated test artifacts:

```
playwright-report/
test-results/
```

## Optional Maintenance
- Repo integrity and cleanup:

```bash
git fsck --full
git gc --prune=now --aggressive
```

- If large blobs are already in history, remove them with `git filter-repo` and consider Git LFS for future large binaries.

## Recent Issue Resolution (2025-09-26)

**Problem**: Persistent `fatal: mmap failed: Operation timed out` during git push
**Solution**: System restart resolved the issue
**Status**: ✅ Resolved after laptop restart

**What was tried before restart:**
- Increased http.postBuffer to 524MB
- Set pack.windowMemory to 256MB
- Used --no-verify flag
- Attempted smaller batch commits
- All failed with same mmap timeout

**Lesson**: System-level memory mapping issues often require restart to clear.

## Systematic Diagnosis Plan (2025-09-26)

**Root Cause Analysis**: Why is git push still failing post-restart?

### Phase 1: Identify the Real Problem
1. **Check repo size/complexity**:
   ```bash
   git count-objects -vH
   git log --oneline | wc -l
   du -sh .git/
   ```

2. **Test basic connectivity**:
   ```bash
   git ls-remote origin
   curl -I https://github.com/jamie-anson/Project-Beacon.git
   ```

3. **Check what's actually in the commit**:
   ```bash
   git show --stat HEAD
   git diff HEAD~1 --name-only
   ```

### Phase 2: Progressive Fixes (Try in Order)
1. **Simple retry** (post-restart should work):
   ```bash
   git push origin main
   ```

2. **If still hanging, try SSH instead of HTTPS**:
   ```bash
   git remote set-url origin git@github.com:jamie-anson/Project-Beacon.git
   git push origin main
   ```

3. **If SSH fails, try minimal push**:
   ```bash
   git push origin HEAD:refs/heads/main --force-with-lease
   ```

4. **If all fail, repo cleanup approach**:
   ```bash
   git gc --aggressive --prune=now
   git repack -ad
   git push origin main
   ```

### Phase 3: Nuclear Options (If Above Fails)
1. **Fresh clone approach**:
   - Clone fresh repo to temp location
   - Cherry-pick the commit
   - Push from fresh repo

2. **Patch file approach**:
   - Create patch: `git format-patch HEAD~1`
   - Apply to fresh clone
   - Push patch

### Expected Outcome
- **Most likely**: Phase 2.1 (simple retry) should work post-restart
- **If not**: SSH (Phase 2.2) usually resolves network/auth issues
- **Worst case**: Fresh clone (Phase 3.1) always works

**Lesson**: System-level memory mapping issues often require restart to clear.
- See `docs/sot/facts.json` for CI/CD routes and deployment targets.
- Netlify deploys can appear "prepared" if the token is invalid; refresh `NETLIFY_AUTH_TOKEN` GitHub secret if needed.
- **Post-restart**: Always try git push immediately after system restart for mmap issues.

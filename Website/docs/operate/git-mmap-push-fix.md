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

## Notes
- See `docs/sot/facts.json` for CI/CD routes and deployment targets.
- Netlify deploys can appear "prepared" if the token is invalid; refresh `NETLIFY_AUTH_TOKEN` GitHub secret if needed.

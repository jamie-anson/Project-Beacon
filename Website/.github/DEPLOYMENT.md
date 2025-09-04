# GitHub Actions Deployment Setup

This document outlines the automated CI/CD pipeline for Project Beacon's website deployment.

## 🚀 Overview

The deployment pipeline consists of two main workflows:
- **CI Pipeline** (`.github/workflows/ci.yml`) - Build validation, testing, and security checks
- **CD Pipeline** (`.github/workflows/deploy.yml`) - Automated Netlify deployment with PR previews

## 🔐 Required Secrets

Configure these secrets in your GitHub repository settings (`Settings > Secrets and variables > Actions`):

### Production Secrets
- **`NETLIFY_AUTH_TOKEN`** - Netlify personal access token for deployments
  - Generate at: https://app.netlify.com/user/applications/personal
  - Scope: Full access to deploy sites
  
- **`NETLIFY_SITE_ID`** - Your Netlify site identifier
  - Find in: Netlify dashboard > Site settings > General > Site details
  - Format: `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`

## 🌍 Environment Configuration

### Environments
The workflows use GitHub environments for deployment protection:

- **`preview`** - For PR preview deployments
  - No protection rules needed
  - Used for temporary preview URLs
  
- **`production`** - For main branch deployments
  - Recommended: Add protection rules requiring reviews
  - Used for https://projectbeacon.netlify.app

### Environment Variables
These are automatically set during builds:

- `VITE_DOCS_CID` - Generated docs content identifier
- `VITE_BUILD_COMMIT` - Git commit hash for build tracking
- `DOCS_BUILD_CID` - Internal docs build identifier

## 🔄 Workflow Triggers

### CI Pipeline (`ci.yml`)
Runs on:
- Push to `main` or `develop` branches
- Pull requests targeting `main`

**Jobs:**
1. **build-and-test** - Validates the complete build process
2. **security-scan** - Checks for vulnerabilities and secrets
3. **build-summary** - Generates build status summary

### CD Pipeline (`deploy.yml`)
Runs on:
- **Preview**: Pull requests (after CI passes)
- **Production**: Push to `main` (after CI passes)

**Jobs:**
1. **deploy-preview** - Creates PR preview deployments
2. **deploy-production** - Deploys to production
3. **deployment-status** - Updates commit status

## 📋 Build Process

The automated build follows your validated sequence:

```bash
# 1. Build static homepage
npm run build:static

# 2. Build Docusaurus docs
npm run build:docs

# 3. Generate content identifier
npm run postbuild:cid

# 4. Rebuild docs with CID
DOCS_BUILD_CID=$(cat dist/docs-cid.txt) npm run build:docs:with-cid

# 5. Build React portal with environment variables
VITE_DOCS_CID="$DOCS_BUILD_CID" VITE_BUILD_COMMIT="$(git rev-parse --short HEAD)" npm run --prefix portal build

# 6. Assemble final distribution
mkdir -p dist/portal && cp -R portal/dist/* dist/portal/
```

## 🔍 Quality Gates

### Build Validation
- ✅ All dependencies install successfully
- ✅ Static site builds without errors
- ✅ Docusaurus docs compile properly
- ✅ React portal builds with correct environment variables
- ✅ All required artifacts exist in `dist/`
- ✅ Generated CID follows expected format

### Security Checks
- 🔒 npm audit for high-severity vulnerabilities
- 🔒 Basic secret detection in source code
- 🔒 netlify.toml configuration validation
- 🔒 Security headers verification

### Deployment Verification
- 🌐 Main site health check (`/`)
- 📚 Documentation health check (`/docs/`)
- 🎛️ Portal health check (`/portal/`)

## 🚨 Troubleshooting

### Common Issues

**Build Failures:**
- Check that all npm scripts exist in `package.json`
- Verify portal dependencies are properly installed
- Ensure `postbuild-pin.js` script is executable

**Deployment Failures:**
- Verify `NETLIFY_AUTH_TOKEN` has sufficient permissions
- Check `NETLIFY_SITE_ID` matches your target site
- Ensure Netlify site exists and is accessible

**Security Scan Failures:**
- Review npm audit output for high-severity issues
- Check for accidentally committed secrets or API keys
- Validate netlify.toml configuration

### Manual Override

To deploy manually (emergency use):
```bash
# Build locally
npm run build:static && npm run build:docs && npm run postbuild:cid
DOCS_BUILD_CID=$(cat dist/docs-cid.txt) npm run build:docs:with-cid
VITE_DOCS_CID="$DOCS_BUILD_CID" VITE_BUILD_COMMIT="$(git rev-parse --short HEAD)" npm run --prefix portal build
mkdir -p dist/portal && cp -R portal/dist/* dist/portal/

# Deploy with Netlify CLI
netlify deploy --dir=dist --prod --site=YOUR_SITE_ID --auth=YOUR_AUTH_TOKEN
```

## 📊 Monitoring

### GitHub Actions
- Build status visible in commit checks
- Detailed logs available in Actions tab
- Build summaries generated for each run

### Netlify
- Deployment status in Netlify dashboard
- Build logs and deploy previews
- Performance and analytics data

### PR Integration
- Automatic preview URL comments on PRs
- Commit status updates
- Deploy preview updates on new commits

## 🔄 Maintenance

### Regular Tasks
- Monitor security audit results
- Update Node.js version as needed
- Review and update dependency versions
- Validate deployment success rates

### Updating Workflows
- Test changes on feature branches first
- Use workflow dispatch for manual testing
- Monitor deployment metrics after changes

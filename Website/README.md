# Project Beacon Website + Docs

This repo serves a static landing page at `/` and a Docusaurus v3 docs site at `/docs` on the same Netlify deployment.

## Structure
- `index.html`, `styles.css`, `script.js`, `images/` — one-pager landing site
- `docs/` — Docusaurus site root
  - `docs/docs/**` — documentation content
  - `docs/blog/**` — blog posts
  - `docs/docusaurus.config.ts` — Docusaurus config (baseUrl `/docs/`)
  - `docs/sidebars.ts` — sidebars

## Build & Deploy
Netlify uses `netlify.toml` to:
- Run `npm run build`
- Publish `dist/`
- Route `/docs/*` to the built Docusaurus SPA (`/docs/index.html`)

Unified build:
```bash
npm install
npm run build
```
Outputs:
- `dist/` — landing page + docs under `dist/docs/`

Standalone docs (for IPFS pinning):
```bash
npm run build:docs:standalone
# Output: dist-docs-standalone/ (baseUrl "/")
```

## Local Development
Run both landing and docs:
```bash
npm run dev
# Landing: http://localhost:3000
# Docs:    http://localhost:3001/docs/
```

## Versioning
Enable versioning (creates `docs/versioned_docs` and `docs/versions.json`):
```bash
npm run version
```

## Build Metadata in Docs Footer
The docs footer shows:
- Commit: `COMMIT_REF` (from Netlify)
- CID: `DOCS_BUILD_CID` (optional env var)

Set in Netlify (optional):
- `DOCS_BUILD_CID = bafy...`

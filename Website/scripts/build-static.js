#!/usr/bin/env node
const path = require('path');
const fs = require('fs-extra');

async function main() {
  const root = path.resolve(__dirname, '..');
  const dist = path.join(root, 'dist');
  await fs.emptyDir(dist);

  // Copy top-level HTML pages and core assets into dist/
  // This ensures new pages like how-it-works.html are deployed.
  const entries = await fs.readdir(root);
  for (const f of entries) {
    const src = path.join(root, f);
    const stat = await fs.stat(src);
    if (stat.isFile()) {
      const isHTML = f.toLowerCase().endsWith('.html');
      const isCoreAsset = f === 'styles.css' || f === 'script.js';
      if (isHTML || isCoreAsset) {
        await fs.copy(src, path.join(dist, f));
      }
    }
  }

  // Copy images directory if exists
  const imagesSrc = path.join(root, 'images');
  if (await fs.pathExists(imagesSrc)) {
    await fs.copy(imagesSrc, path.join(dist, 'images'));
  }

  // Copy demo-results directory if exists
  const demoResultsSrc = path.join(root, 'demo-results');
  if (await fs.pathExists(demoResultsSrc)) {
    await fs.copy(demoResultsSrc, path.join(dist, 'demo-results'));
  }

  // Touch a _headers file for better caching of static assets
  const headersPath = path.join(dist, '_headers');
  const headers = `
/*
  X-Frame-Options: SAMEORIGIN
  X-Content-Type-Options: nosniff
  Referrer-Policy: strict-origin-when-cross-origin
  Permissions-Policy: geolocation=(), microphone=(), camera=()
  Cache-Control: public, max-age=0, must-revalidate

/*.html
  Cache-Control: public, max-age=0, must-revalidate

/docs/assets/*
  Cache-Control: public, max-age=31536000, immutable

/assets/*
  Cache-Control: public, max-age=31536000, immutable

/images/*
  Cache-Control: public, max-age=31536000, immutable
`;
  await fs.writeFile(headersPath, headers.trimStart(), 'utf8');

  // Note: _redirects generation is now handled by scripts/write-redirects.js
  // This ensures single source of truth for all redirect rules

  console.log('Static site copied to dist/.');

  // Create serve.json for local SPA rewrites when using `npx serve`
  const serveJson = {
    cleanUrls: true,
    rewrites: [
      { source: '/docs', destination: '/docs/index.html' },
      { source: '/docs/(.*)', destination: '/docs/index.html' },
      { source: '/portal', destination: '/portal/index.html' },
      { source: '/portal/(.*)', destination: '/portal/index.html' },
    ],
  };
  await fs.writeFile(path.join(dist, 'serve.json'), JSON.stringify(serveJson, null, 2));

  // Copy a placeholder favicon.ico to reduce 404 noise (optional)
  const favSrc = path.join(root, 'images', 'Icon.webp');
  const favDst = path.join(dist, 'favicon.ico');
  if (await fs.pathExists(favSrc)) {
    await fs.copy(favSrc, favDst);
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

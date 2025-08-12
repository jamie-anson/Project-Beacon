#!/usr/bin/env node
const path = require('path');
const fs = require('fs-extra');

async function main() {
  const root = path.resolve(__dirname, '..');
  const dist = path.join(root, 'dist');
  await fs.emptyDir(dist);

  // Copy landing page assets
  const files = ['index.html', 'styles.css', 'script.js'];
  for (const f of files) {
    const src = path.join(root, f);
    if (await fs.pathExists(src)) {
      await fs.copy(src, path.join(dist, f));
    }
  }

  // Copy images directory if exists
  const imagesSrc = path.join(root, 'images');
  if (await fs.pathExists(imagesSrc)) {
    await fs.copy(imagesSrc, path.join(dist, 'images'));
  }

  // Touch a _headers file for better caching of static assets
  const headersPath = path.join(dist, '_headers');
  const headers = `
/*
  Cache-Control: public, max-age=600

/docs/assets/*
  Cache-Control: public, max-age=31536000, immutable

/assets/*
  Cache-Control: public, max-age=31536000, immutable
`;
  await fs.writeFile(headersPath, headers.trimStart(), 'utf8');

  console.log('Static site copied to dist/.');
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

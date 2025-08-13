#!/usr/bin/env node
/*
  Compute a deterministic content hash of the built docs directory (dist/docs)
  and write it to dist/docs-cid.txt. This is a stand-in for a CID you can later
  replace with a real IPFS pinning workflow. The hash covers file paths, sizes,
  and contents to be tamper-evident.
*/
const fs = require('fs');
const fsp = fs.promises;
const path = require('path');
const crypto = require('crypto');

async function walk(dir) {
  const entries = await fsp.readdir(dir, { withFileTypes: true });
  const files = await Promise.all(entries.map(async (ent) => {
    const res = path.resolve(dir, ent.name);
    if (ent.isDirectory()) return walk(res);
    return res;
  }));
  return files.flat();
}

async function fileHash(file) {
  // stream to avoid loading large files entirely
  return new Promise((resolve, reject) => {
    const hash = crypto.createHash('sha256');
    const stream = fs.createReadStream(file);
    stream.on('data', (chunk) => hash.update(chunk));
    stream.on('end', () => resolve(hash.digest('hex')));
    stream.on('error', reject);
  });
}

async function main() {
  const root = path.resolve(__dirname, '..');
  const outFile = path.join(root, 'dist', 'docs-cid.txt');
  // Docusaurus may have emitted to either root/dist/docs or docs/dist/docs
  const candidates = [
    path.join(root, 'dist', 'docs'),
    path.join(root, 'docs', 'dist', 'docs')
  ];
  let docsDir = null;
  for (const cand of candidates) {
    try {
      await fsp.access(cand);
      docsDir = cand;
      break;
    } catch {}
  }

  if (!docsDir) {
    console.error(`Docs output not found. Looked for: `);
    for (const c of candidates) console.error(` - ${c}`);
    console.error('Run "npm run build" first.');
    process.exit(1);
  }
  console.log(`Hashing docs from: ${docsDir}`);

  const allFiles = (await walk(docsDir))
    .filter((p) => fs.statSync(p).isFile())
    .sort(); // deterministic order

  const hash = crypto.createHash('sha256');
  for (const file of allFiles) {
    const rel = path.relative(docsDir, file);
    const stat = await fsp.stat(file);
    hash.update(rel);
    hash.update(String(stat.size));
    hash.update(await fileHash(file));
  }

  const digestHex = hash.digest('hex');
  // Represent as a pseudo-CID (not a real multihash CID), clearly labeled
  const pseudoCid = `pb-docs-sha256-${digestHex}`;

  await fsp.writeFile(outFile, `${pseudoCid}\n`, 'utf8');
  console.log(`Docs pseudo-CID: ${pseudoCid}`);
  console.log(`Wrote ${outFile}`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

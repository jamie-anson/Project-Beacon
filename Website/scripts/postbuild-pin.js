#!/usr/bin/env node
/*
  Pin the built docs directory to IPFS and get a real CID.
  Falls back to pseudo-CID if IPFS is unavailable.
*/
const fs = require('fs');
const fsp = fs.promises;
const path = require('path');
const crypto = require('crypto');
const { spawn } = require('child_process');
const axios = require('axios');
const FormData = require('form-data');

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

async function execCommand(cmd, args) {
  return new Promise((resolve, reject) => {
    const proc = spawn(cmd, args, { stdio: ['pipe', 'pipe', 'pipe'] });
    let stdout = '';
    let stderr = '';
    
    proc.stdout.on('data', (data) => stdout += data.toString());
    proc.stderr.on('data', (data) => stderr += data.toString());
    
    proc.on('close', (code) => {
      if (code === 0) {
        resolve(stdout.trim());
      } else {
        reject(new Error(`${cmd} failed: ${stderr}`));
      }
    });
  });
}

async function createTarball(docsDir) {
  const tarPath = path.join(path.dirname(docsDir), 'docs.tar.gz');
  await execCommand('tar', ['-czf', tarPath, '-C', path.dirname(docsDir), path.basename(docsDir)]);
  return tarPath;
}

async function pinToPinata(docsDir) {
  const jwt = process.env.PINATA_JWT;
  
  if (!jwt) {
    console.warn('Pinata JWT not found (PINATA_JWT). Get JWT from https://app.pinata.cloud/keys');
    return null;
  }
  
  try {
    console.log('Creating tarball for Pinata...');
    const tarPath = await createTarball(docsDir);
    
    console.log('Uploading to Pinata...');
    const formData = new FormData();
    formData.append('file', fs.createReadStream(tarPath));
    formData.append('name', `project-beacon-docs-${Date.now()}`);
    formData.append('keyvalues', JSON.stringify({
      project: 'project-beacon',
      type: 'documentation',
      timestamp: new Date().toISOString()
    }));
    
    const headers = {
      ...formData.getHeaders(),
      'Authorization': `Bearer ${jwt}`
    };
    
    console.log('Using Pinata v3 API');
    console.log('JWT length:', jwt.length);
    
    const response = await axios.post('https://uploads.pinata.cloud/v3/files', formData, {
      headers,
      maxContentLength: Infinity,
      maxBodyLength: Infinity
    });
    
    // Clean up tarball
    await fsp.unlink(tarPath);
    
    const cid = response.data.data.cid;
    console.log(`Pinata CID: ${cid}`);
    console.log(`Pinata gateway: https://gateway.pinata.cloud/ipfs/${cid}`);
    return cid;
  } catch (error) {
    console.warn(`Pinata upload failed: ${error.message}`);
    if (error.response) {
      console.warn('Response status:', error.response.status);
      console.warn('Response data:', error.response.data);
    }
    return null;
  }
}

async function pinToIPFS(docsDir) {
  // Try Pinata first for reliable hosting
  let cid = await pinToPinata(docsDir);
  if (cid) return cid;
  
  try {
    // Fall back to local IPFS
    try {
      await execCommand('docker', ['exec', 'runner-app-ipfs-1', 'ipfs', 'version']);
      console.log('Adding docs to IPFS (containerized)...');
      
      // Copy docs to container and add to IPFS
      await execCommand('docker', ['cp', docsDir, 'runner-app-ipfs-1:/tmp/docs']);
      const output = await execCommand('docker', ['exec', 'runner-app-ipfs-1', 'ipfs', 'add', '-r', '-Q', '/tmp/docs']);
      cid = output.split('\n').pop(); // Last line is the directory CID
      
      // Pin the content and announce to DHT
      console.log('Pinning and announcing to DHT...');
      await execCommand('docker', ['exec', 'runner-app-ipfs-1', 'ipfs', 'pin', 'add', cid]);
      
      // Try to announce to DHT (may timeout, that's ok)
      try {
        await execCommand('docker', ['exec', 'runner-app-ipfs-1', 'timeout', '10', 'ipfs', 'dht', 'provide', cid]);
        console.log('Successfully announced to DHT');
      } catch (dhtError) {
        console.warn('DHT announce timeout (normal for large content)');
      }
      
      console.log(`Local IPFS CID: ${cid}`);
      console.log(`Test availability: https://ipfs.io/ipfs/${cid}`);
      return cid;
    } catch (dockerError) {
      // Fall back to system IPFS
      await execCommand('ipfs', ['version']);
      console.log('Adding docs to IPFS (local)...');
      const output = await execCommand('ipfs', ['add', '-r', '-Q', docsDir]);
      cid = output.split('\n').pop();
      
      console.log(`System IPFS CID: ${cid}`);
      return cid;
    }
  } catch (error) {
    console.warn(`IPFS not available: ${error.message}`);
    return null;
  }
}

async function generatePseudoCID(docsDir) {
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
  return `pb-docs-sha256-${digestHex}`;
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
  console.log(`Processing docs from: ${docsDir}`);

  // Try to get real IPFS CID first
  let cid = await pinToIPFS(docsDir);
  
  // Fall back to pseudo-CID if IPFS unavailable
  if (!cid) {
    console.log('Generating pseudo-CID...');
    cid = await generatePseudoCID(docsDir);
    console.log(`Docs pseudo-CID: ${cid}`);
  }

  await fsp.writeFile(outFile, `${cid}\n`, 'utf8');
  console.log(`Wrote ${outFile}`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

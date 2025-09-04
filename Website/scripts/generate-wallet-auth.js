#!/usr/bin/env node

'use strict';

/**
 * Generate a valid WALLET_AUTH_JSON for Runner submission tests.
 *
 * This script:
 * - Reads your Ed25519 public key from live-job-key.txt (or ED25519_PUBLIC_KEY/PUBLIC_KEY env)
 * - Creates or uses an EVM wallet (WALLET_EVM_PRIVKEY env to use a specific wallet; else ephemeral)
 * - Signs the message: "Authorize Project Beacon key: <Ed25519 base64>"
 * - Outputs a JSON object with: { address, signature, message, chainId, nonce, expiresAt }
 *
 * Usage examples:
 *   # Default: read Ed25519 from live-job-key.txt, generate ephemeral wallet, 10 min expiry, Holesky chainId (17000)
 *   node scripts/generate-wallet-auth.js
 *
 *   # Pipe to env for submit script
 *   WALLET_AUTH_JSON="$(node scripts/generate-wallet-auth.js --minutes 10 --chain-id 17000)" \
 *   RUNNER_URL=http://localhost:8090 node scripts/submit-signed-job.js
 *
 *   # Provide your own EVM private key (DO NOT COMMIT THIS)
 *   WALLET_EVM_PRIVKEY=0xabc... node scripts/generate-wallet-auth.js --chain-id 17000 --minutes 15
 *
 *   # Override Ed25519 key explicitly
 *   node scripts/generate-wallet-auth.js --ed25519 "/yRnRt4lvdGCweqHSJ3dM56YOZUBnsZM7BF/zQPWlO8="
 */

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');

let ethers;
try {
  // ethers v5 CommonJS
  ethers = require('ethers');
} catch (e) {
  console.error('Missing dependency "ethers". Install with: npm i -D ethers@5');
  process.exit(1);
}

function getArg(name, defVal = undefined) {
  const argv = process.argv.slice(2);
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === `--${name}`) {
      return argv[i + 1];
    }
    if (a.startsWith(`--${name}=`)) {
      return a.split('=')[1];
    }
  }
  return defVal;
}

function readEd25519PublicKey() {
  const envKey = process.env.ED25519_PUBLIC_KEY || process.env.PUBLIC_KEY || getArg('ed25519');
  if (envKey) return envKey.trim();

  const keyPath = path.resolve(process.cwd(), 'live-job-key.txt');
  if (!fs.existsSync(keyPath)) {
    throw new Error('Ed25519 public key not found: set ED25519_PUBLIC_KEY/PUBLIC_KEY or provide live-job-key.txt');
  }
  const text = fs.readFileSync(keyPath, 'utf8');
  const pubMatch = text.match(/Public Key:\s*([A-Za-z0-9+/=]+)/);
  if (!pubMatch) {
    throw new Error('Failed to parse Public Key from live-job-key.txt');
  }
  return pubMatch[1].trim();
}

function generateNonce() {
  return 'wa_' + crypto.randomBytes(12).toString('base64url');
}

function parseIntSafe(v, defVal) {
  const n = parseInt(String(v ?? ''), 10);
  return Number.isFinite(n) ? n : defVal;
}

async function main() {
  const ed25519B64 = readEd25519PublicKey();
  const message = `Authorize Project Beacon key: ${ed25519B64}`;

  // Chain ID defaults to Holesky (17000) based on project testnet usage
  const chainId = parseIntSafe(getArg('chain-id', process.env.WALLET_CHAIN_ID), 17000);
  const minutes = Math.max(1, parseIntSafe(getArg('minutes', process.env.WALLET_AUTH_TTL_MINUTES), 10));
  const expiresAt = new Date(Date.now() + minutes * 60 * 1000).toISOString();

  let wallet;
  const providedPriv = process.env.WALLET_EVM_PRIVKEY || getArg('wallet-priv');
  if (providedPriv) {
    try {
      wallet = new ethers.Wallet(providedPriv);
    } catch (e) {
      console.error('Invalid WALLET_EVM_PRIVKEY provided. Expected 0x-prefixed hex private key.');
      process.exit(1);
    }
  } else {
    wallet = ethers.Wallet.createRandom();
    console.error('Generated ephemeral EVM wallet for testing. DO NOT USE IN PRODUCTION.');
    console.error(`Ephemeral address: ${wallet.address}`);
    console.error('(If you want to use your own wallet, set WALLET_EVM_PRIVKEY=0x... before running.)');
  }

  const signature = await wallet.signMessage(message);
  const nonce = getArg('nonce', process.env.WALLET_AUTH_NONCE) || generateNonce();

  const payload = {
    address: wallet.address,
    signature,
    message,
    chainId,
    nonce,
    expiresAt,
  };

  const outPath = getArg('out');
  const json = JSON.stringify(payload);
  if (outPath) {
    fs.writeFileSync(path.resolve(process.cwd(), outPath), json + '\n', { encoding: 'utf8' });
    console.error(`Wrote WALLET_AUTH_JSON to ${outPath}`);
  } else {
    // stdout only JSON to allow easy command substitution
    process.stdout.write(json);
  }
}

main().catch((e) => {
  console.error(`Failed to generate WALLET_AUTH_JSON: ${e.message}`);
  process.exit(1);
});

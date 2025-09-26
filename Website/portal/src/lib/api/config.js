let cachedRunnerBase;
let cachedRunnerExplicit;

function computeRunnerBase() {
  let base = 'https://beacon-runner-change-me.fly.dev';
  let explicit = false;

  // Runtime override (highest precedence)
  try {
    const lsBase = localStorage.getItem('beacon:api_base');
    if (lsBase && lsBase.trim()) {
      base = lsBase.trim();
      explicit = true;
    }
  } catch {}

  // Build-time env override (if no runtime override)
  try {
    if (!explicit && import.meta.env?.VITE_API_BASE) {
      const envBase = String(import.meta.env.VITE_API_BASE).trim();
      if (envBase) {
        base = envBase;
        explicit = true;
      }
    }
  } catch {}

  // Normalize the base (remove whitespace, trailing slash, or duplicated /api/v1)
  try {
    base = String(base)
      .replace(/\s+/g, '')
      .replace(/\/?api\/v1\/?$/i, '')
      .replace(/\/$/, '');
  } catch {}

  cachedRunnerBase = base;
  cachedRunnerExplicit = explicit;

  try {
    if (import.meta?.env?.DEV) {
      console.info('[Beacon] API_BASE_V1 =', cachedRunnerBase, '(explicit =', cachedRunnerExplicit, ')');
    }
  } catch {}

  return cachedRunnerBase;
}

export function resolveRunnerBase() {
  return cachedRunnerBase ?? computeRunnerBase();
}

export function isRunnerBaseExplicit() {
  if (cachedRunnerExplicit == null) computeRunnerBase();
  return cachedRunnerExplicit === true;
}

let cachedHybridBase;

function computeHybridBase() {
  let base = 'https://project-beacon-production.up.railway.app';

  try {
    const lsHybrid = localStorage.getItem('beacon:hybrid_base');
    if (lsHybrid && lsHybrid.trim()) {
      base = lsHybrid;
    }
  } catch {}

  try {
    const envHybrid = import.meta?.env?.VITE_HYBRID_BASE;
    if (envHybrid && typeof envHybrid === 'string' && envHybrid.trim()) {
      base = envHybrid;
    }
  } catch {}

  try {
    base = String(base).trim().replace(/\/$/, '');
  } catch {}

  cachedHybridBase = base;
  return cachedHybridBase;
}

export function resolveHybridBase() {
  return cachedHybridBase ?? computeHybridBase();
}

let cachedDiffsBase;

function computeDiffsBase() {
  let base = '';

  try {
    const ls = localStorage.getItem('beacon:diffs_base');
    if (ls && ls.trim()) {
      base = ls;
    }
  } catch {}

  try {
    if (!base) {
      const env = import.meta?.env?.VITE_DIFFS_BASE;
      if (env && typeof env === 'string' && env.trim()) {
        base = env;
      }
    }
  } catch {}

  if (!base) base = '/backend-diffs';

  try {
    base = String(base).trim().replace(/\/$/, '');
  } catch {}

  cachedDiffsBase = base;

  try {
    if (import.meta?.env?.DEV) console.info('[Beacon] DIFFS_BASE =', cachedDiffsBase);
  } catch {}

  return cachedDiffsBase;
}

export function resolveDiffsBase() {
  return cachedDiffsBase ?? computeDiffsBase();
}

let cachedIpfsGateway;

function computeIpfsGateway() {
  try {
    const override = localStorage.getItem('beacon:ipfs_gateway');
    if (override && override.trim()) {
      cachedIpfsGateway = override.replace(/\/$/, '');
      return cachedIpfsGateway;
    }
  } catch {}

  try {
    const envGw = import.meta?.env?.VITE_IPFS_GATEWAY;
    if (envGw && typeof envGw === 'string' && envGw.trim()) {
      cachedIpfsGateway = envGw.replace(/\/$/, '');
      return cachedIpfsGateway;
    }
  } catch {}

  cachedIpfsGateway = null;
  return cachedIpfsGateway;
}

export function resolveIpfsGateway() {
  if (cachedIpfsGateway === undefined) return computeIpfsGateway();
  return cachedIpfsGateway;
}

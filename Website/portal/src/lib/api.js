// Temporary compatibility shim: the real exports now live under `portal/src/lib/api/`.
// All new code should import from the modularized files (e.g., `./api/runner/jobs.js`).
// This file will be removed once all consumers migrate.

console.warn('[Beacon] `portal/src/lib/api.js` is deprecated. Import from `portal/src/lib/api/` modules instead.');

export * from './api/config.js';
export * from './api/http.js';
export * from './api/idempotency.js';
export * from './api/ipfs.js';
export * from './api/runner/index.js';
export * from './api/hybrid/index.js';
export * from './api/diffs/index.js';

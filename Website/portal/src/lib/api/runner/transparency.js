import { runnerFetch } from '../http.js';

export function getTransparencyRoot() {
  return runnerFetch('/transparency/root');
}

export function getTransparencyProof({ execution_id, ipfs_cid }) {
  const params = new URLSearchParams();
  if (execution_id) params.set('execution_id', execution_id);
  if (ipfs_cid) params.set('ipfs_cid', ipfs_cid);
  return runnerFetch(`/transparency/proof?${params.toString()}`);
}

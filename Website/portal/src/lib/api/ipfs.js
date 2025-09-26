import { resolveIpfsGateway, resolveRunnerBase } from './config.js';

export function getIpfsGateway() {
  return resolveIpfsGateway();
}

export function bundleUrl(cid) {
  const gateway = resolveIpfsGateway();
  if (gateway) {
    return `${gateway}/ipfs/${encodeURIComponent(cid)}`;
  }
  const runnerBase = resolveRunnerBase();
  return `${runnerBase}/transparency/bundles/${encodeURIComponent(cid)}`;
}

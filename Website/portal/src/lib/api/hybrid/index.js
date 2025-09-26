import { hybridFetch } from '../http.js';
import { getInfrastructureHealth as getInfrastructureHealthInternal } from './infrastructure.js';

export function getHybridHealth() {
  return hybridFetch('/health');
}

export function getHybridProviders() {
  return hybridFetch('/providers').then((data) => {
    if (Array.isArray(data?.providers)) return data.providers;
    return [];
  });
}

export const getInfrastructureHealth = getInfrastructureHealthInternal;

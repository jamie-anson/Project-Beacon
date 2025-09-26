import { resolveRunnerBase, resolveHybridBase, resolveDiffsBase } from './config.js';

class ApiError extends Error {
  constructor(message, { status, statusText, data, raw, url, code, cause } = {}) {
    super(message);
    this.name = 'ApiError';
    if (cause) this.cause = cause;
    this.status = typeof status === 'number' ? status : null;
    this.statusText = statusText ?? null;
    this.data = data ?? null;
    this.raw = raw ?? null;
    this.url = url ?? null;
    this.code = code ?? data?.error_code ?? null;
    this.errorCode = this.code;
  }
}

export { ApiError };

function normalizePath(path) {
  if (!path) return '';
  return path.startsWith('/') ? path : `/${path}`;
}

function prepareJsonHeaders(headers = {}) {
  return {
    Accept: 'application/json',
    'Content-Type': 'application/json; charset=utf-8',
    ...headers,
  };
}

function applyCorsOptions(options) {
  const fetchOptions = { ...options };
  fetchOptions.mode = 'cors';
  fetchOptions.credentials = 'omit';
  return fetchOptions;
}

async function readResponsePayload(res) {
  const text = await res.text();
  if (!text) {
    return { data: null, raw: '' };
  }

  try {
    return { data: JSON.parse(text), raw: text };
  } catch {
    return { data: null, raw: text };
  }
}

async function parseJsonResponse(res, url) {
  if (res.status === 204) return null;

  const { data, raw } = await readResponsePayload(res);
  if (data !== null) return data;
  if (!raw) return null;

  throw new ApiError('Failed to parse JSON response', {
    status: res.status,
    statusText: res.statusText,
    raw,
    url,
    code: 'INVALID_JSON',
  });
}

async function handleErrorResponse(res, url) {
  const { data, raw } = await readResponsePayload(res);

  let errorMessage = `${res.status} ${res.statusText}`;
  if (data?.error || data?.message) {
    errorMessage = data.error || data.message;
  } else if (raw) {
    errorMessage = raw;
  }

  if (data?.error_code) {
    errorMessage += ` (${data.error_code})`;
  }

  throw new ApiError(errorMessage, {
    status: res.status,
    statusText: res.statusText,
    data,
    raw,
    url,
  });
}

function wrapNetworkError(err, url) {
  if (err instanceof ApiError) return err;

  const message = err?.message || 'Network error';
  let code = err?.code;

  if (!code && err?.name === 'AbortError') {
    code = 'ABORT_ERROR';
  } else if (!code && (message.includes('Failed to fetch') || message.includes('Load failed'))) {
    code = 'NETWORK_ERROR';
  }

  return new ApiError(message, {
    url,
    cause: err,
    code,
  });
}

export async function runnerFetch(path, options = {}) {
  const base = resolveRunnerBase();
  const prefix = base ? `${base}/api/v1` : '/api/v1';
  const url = `${prefix}${normalizePath(path)}`;

  const fetchOptions = applyCorsOptions({
    ...options,
    headers: prepareJsonHeaders(options.headers),
  });

  try {
    const headers = fetchOptions.headers || {};
    const contentType = String(headers['Content-Type'] || headers['content-type'] || '');
    if (fetchOptions.body && typeof fetchOptions.body === 'string' && !/application\/json/i.test(contentType)) {
      fetchOptions.headers = {
        ...headers,
        'Content-Type': 'application/json; charset=utf-8',
      };
    }
  } catch {}

  try {
    const res = await fetch(url, fetchOptions);
    if (!res.ok) await handleErrorResponse(res, url);
    return parseJsonResponse(res, url);
  } catch (err) {
    const wrapped = wrapNetworkError(err, url);
    if (wrapped.code === 'NETWORK_ERROR') {
      console.warn(`Network/CORS error for ${url}:`, wrapped.message);
      console.warn('This may be a CORS issue or the API server may be unreachable');
    } else if (wrapped.code !== 'ABORT_ERROR') {
      console.warn(`API call failed: ${url}`, wrapped.message);
    }
    throw wrapped;
  }
}

export async function hybridFetch(path, options = {}) {
  const base = resolveHybridBase();
  const directUrl = `${base}${normalizePath(path)}`;
  const proxyUrl = `/hybrid${normalizePath(path)}`;

  const fetchOptions = applyCorsOptions({
    ...options,
    headers: {
      Accept: 'application/json',
      ...(options.headers || {}),
    },
  });

  try {
    const directRes = await fetch(directUrl, fetchOptions);
    if (directRes.ok) return parseJsonResponse(directRes);
  } catch (err) {
    try {
      console.warn(`[Hybrid] direct request failed, falling back to proxy: ${directUrl} -> ${proxyUrl}:`, err?.message || String(err));
    } catch {}
  }

  try {
    const proxyRes = await fetch(proxyUrl, fetchOptions);
    if (!proxyRes.ok) await handleErrorResponse(proxyRes, proxyUrl);
    return parseJsonResponse(proxyRes, proxyUrl);
  } catch (err) {
    const wrapped = wrapNetworkError(err, proxyUrl);
    if (wrapped.code === 'NETWORK_ERROR') {
      console.warn(`[Hybrid] proxy request failed (proxy=${proxyUrl})`, wrapped.message);
    } else if (wrapped.code !== 'ABORT_ERROR') {
      console.warn(`[Hybrid] proxy request failed (proxy=${proxyUrl})`, wrapped.message);
    }
    throw wrapped;
  }
}

export async function diffsFetch(path, options = {}) {
  const base = resolveDiffsBase();
  const url = `${base}${normalizePath(path)}`;

  const fetchOptions = applyCorsOptions({
    ...options,
    headers: prepareJsonHeaders(options.headers),
  });

  try {
    const res = await fetch(url, fetchOptions);
    if (!res.ok) await handleErrorResponse(res, url);
    return parseJsonResponse(res, url);
  } catch (err) {
    throw wrapNetworkError(err, url);
  }
}

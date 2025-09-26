import { runnerFetch, hybridFetch, diffsFetch } from '../http.js';

jest.mock('../config.js', () => ({
  resolveRunnerBase: jest.fn(() => 'https://api.example.com'),
  resolveHybridBase: jest.fn(() => 'https://hybrid.example.com'),
  resolveDiffsBase: jest.fn(() => 'https://diffs.example.com'),
}));

const originalFetch = global.fetch;
let warnSpy;

const createResponse = ({
  ok = true,
  status = 200,
  statusText = 'OK',
  body = '',
} = {}) => ({
  ok,
  status,
  statusText,
  text: jest.fn().mockResolvedValue(body),
});

describe('runnerFetch', () => {
  beforeEach(() => {
    global.fetch = jest.fn();
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  afterAll(() => {
    global.fetch = originalFetch;
  });

  test('returns parsed JSON on success responses', async () => {
    const payload = { message: 'ok' };
    global.fetch.mockResolvedValue(createResponse({ body: JSON.stringify(payload) }));

    const result = await runnerFetch('/test', { method: 'GET' });

    expect(result).toEqual(payload);
    expect(global.fetch).toHaveBeenCalledWith(
      'https://api.example.com/api/v1/test',
      expect.objectContaining({ method: 'GET' })
    );
  });

  test('throws ApiError with code INVALID_JSON when JSON parsing fails', async () => {
    global.fetch.mockResolvedValue(createResponse({ body: '<html>oops</html>' }));

    await expect(runnerFetch('/broken')).rejects.toMatchObject({
      name: 'ApiError',
      code: 'INVALID_JSON',
      status: 200,
      url: 'https://api.example.com/api/v1/broken',
    });
  });

  test('throws ApiError with status and payload when response not ok', async () => {
    const errorPayload = { error: 'DATABASE_CONNECTION_FAILED', error_code: 'DATABASE_CONNECTION_FAILED' };
    global.fetch.mockResolvedValue(
      createResponse({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        body: JSON.stringify(errorPayload),
      })
    );

    await expect(runnerFetch('/jobs')).rejects.toMatchObject({
      name: 'ApiError',
      status: 500,
      data: errorPayload,
      code: 'DATABASE_CONNECTION_FAILED',
    });
  });

  test('wraps network failures as ApiError with code NETWORK_ERROR', async () => {
    global.fetch.mockRejectedValue(new Error('Failed to fetch'));

    await expect(runnerFetch('/timeout')).rejects.toMatchObject({
      name: 'ApiError',
      code: 'NETWORK_ERROR',
      url: 'https://api.example.com/api/v1/timeout',
    });
  });

  test('preserves AbortError code when request is aborted', async () => {
    const abortError = new Error('The operation was aborted');
    abortError.name = 'AbortError';
    global.fetch.mockRejectedValue(abortError);

    await expect(runnerFetch('/abort')).rejects.toMatchObject({
      name: 'ApiError',
      code: 'ABORT_ERROR',
      url: 'https://api.example.com/api/v1/abort',
    });
  });
});

describe('hybridFetch', () => {
  beforeAll(() => {
    warnSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});
  });

  afterAll(() => {
    warnSpy?.mockRestore();
    global.fetch = originalFetch;
  });

  beforeEach(() => {
    global.fetch = jest.fn();
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  test('returns parsed JSON when direct request succeeds', async () => {
    const payload = { providers: [] };
    global.fetch.mockResolvedValueOnce(createResponse({ body: JSON.stringify(payload) }));

    const result = await hybridFetch('/providers');

    expect(result).toEqual(payload);
    expect(global.fetch).toHaveBeenCalledWith(
      'https://hybrid.example.com/providers',
      expect.any(Object)
    );
  });

  test('falls back to proxy when direct request fails', async () => {
    const payload = { status: 'ok' };
    global.fetch
      .mockRejectedValueOnce(new Error('network'))
      .mockResolvedValueOnce(createResponse({ body: JSON.stringify(payload) }));

    const result = await hybridFetch('/health');

    expect(result).toEqual(payload);
    expect(global.fetch).toHaveBeenNthCalledWith(2, '/hybrid/health', expect.any(Object));
  });

  test('throws ApiError when proxy request returns error response', async () => {
    global.fetch
      .mockRejectedValueOnce(new Error('network'))
      .mockResolvedValueOnce(
        createResponse({
          ok: false,
          status: 502,
          statusText: 'Bad Gateway',
          body: JSON.stringify({ error: 'PROVIDER_UNAVAILABLE', error_code: 'PROVIDER_UNAVAILABLE' }),
        })
      );

    await expect(hybridFetch('/providers')).rejects.toMatchObject({
      name: 'ApiError',
      status: 502,
      code: 'PROVIDER_UNAVAILABLE',
      url: '/hybrid/providers',
    });
  });

  test('wraps proxy network errors as ApiError', async () => {
    global.fetch
      .mockRejectedValueOnce(new Error('network'))
      .mockRejectedValueOnce(new Error('Failed to fetch'));

    await expect(hybridFetch('/providers')).rejects.toMatchObject({
      name: 'ApiError',
      code: 'NETWORK_ERROR',
      url: '/hybrid/providers',
    });
  });
});

describe('diffsFetch', () => {
  beforeEach(() => {
    global.fetch = jest.fn();
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  afterAll(() => {
    global.fetch = originalFetch;
  });

  test('returns parsed JSON on success', async () => {
    const payload = { recent: [] };
    global.fetch.mockResolvedValue(createResponse({ body: JSON.stringify(payload) }));

    const result = await diffsFetch('/recent');

    expect(result).toEqual(payload);
    expect(global.fetch).toHaveBeenCalledWith(
      'https://diffs.example.com/recent',
      expect.any(Object)
    );
  });

  test('throws ApiError with code INVALID_JSON for malformed JSON', async () => {
    global.fetch.mockResolvedValue(createResponse({ body: '<html>bad</html>' }));

    await expect(diffsFetch('/recent')).rejects.toMatchObject({
      name: 'ApiError',
      code: 'INVALID_JSON',
      url: 'https://diffs.example.com/recent',
    });
  });

  test('throws ApiError when response is not ok', async () => {
    global.fetch.mockResolvedValue(
      createResponse({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        body: JSON.stringify({ error: 'not_found', message: 'Not found' }),
      })
    );

    await expect(diffsFetch('/missing')).rejects.toMatchObject({
      name: 'ApiError',
      status: 404,
      code: 'not_found',
      url: 'https://diffs.example.com/missing',
    });
  });
});

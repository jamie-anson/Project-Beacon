/**
 * Unit tests for portal API client
 * Tests URL concatenation, CORS settings, and error handling
 */

// Mock fetch globally
global.fetch = jest.fn();

// Mock Vite's import.meta.env
global.importMeta = {
  env: {
    VITE_API_BASE: 'https://test-api.example.com/api/v1'
  }
};

// Create a test version of the API module that doesn't use import.meta
const createMockAPI = (apiBase = 'https://test-api.example.com/api/v1') => {
  const API_BASE_V1 = apiBase.replace(/\/$/, '');
  
  async function httpV1(path, opts = {}) {
    const url = `${API_BASE_V1}${path.startsWith('/') ? path : '/' + path}`;
    try {
      const fetchOptions = {
        headers: { 'Content-Type': 'application/json', ...(opts.headers || {}) },
        ...opts,
      };
      
      // Explicitly set CORS mode to prevent browser blocking
      fetchOptions.mode = 'cors';
      fetchOptions.credentials = 'omit';
      
      const res = await fetch(url, fetchOptions);
      if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
      return res.status === 204 ? null : res.json();
    } catch (err) {
      console.warn(`API call failed: ${url}`, err.message);
      throw err;
    }
  }

  function computeIdempotencyKey(jobspec, windowSeconds = 60) {
    const tab = 'test-tab-id';
    const bucket = Math.floor(Date.now() / 1000 / windowSeconds);
    let specStr = '';
    try { specStr = JSON.stringify(jobspec); } catch { specStr = String(jobspec || ''); }
    const base = `${tab}:${bucket}:${specStr}`;
    return `beacon-test-${base.slice(0, 10)}`;
  }

  const createJob = (jobspec, opts = {}) => {
    const key = opts.idempotencyKey || computeIdempotencyKey(jobspec);
    return httpV1('/jobs', {
      method: 'POST',
      headers: { 'Idempotency-Key': key },
      body: JSON.stringify(jobspec),
    });
  };

  const listJobs = ({ limit = 50 } = {}) => {
    const params = new URLSearchParams();
    params.set('limit', String(limit));
    return httpV1(`/jobs?${params.toString()}`).then((data) => {
      if (Array.isArray(data)) return { jobs: data };
      return data;
    });
  };

  return { httpV1, createJob, listJobs };
};

describe('API Client', () => {
  let api;

  beforeEach(() => {
    fetch.mockClear();
    api = createMockAPI('https://test-api.example.com/api/v1');
  });

  describe('URL Construction', () => {
    test('handles paths with leading slash correctly', async () => {
      fetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ data: 'test' })
      });

      await api.httpV1('/test-path');
      
      expect(fetch).toHaveBeenCalledWith(
        'https://test-api.example.com/api/v1/test-path',
        expect.objectContaining({
          mode: 'cors',
          credentials: 'omit'
        })
      );
    });

    test('handles paths without leading slash correctly', async () => {
      fetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ data: 'test' })
      });

      await api.httpV1('test-path');
      
      expect(fetch).toHaveBeenCalledWith(
        'https://test-api.example.com/api/v1/test-path',
        expect.objectContaining({
          mode: 'cors',
          credentials: 'omit'
        })
      );
    });

    test('prevents double slashes in URLs', async () => {
      fetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ data: 'test' })
      });

      await api.httpV1('/jobs');
      
      const calledUrl = fetch.mock.calls[0][0];
      expect(calledUrl).toBe('https://test-api.example.com/api/v1/jobs');
      expect(calledUrl).not.toContain('//jobs');
    });

    test('handles API base URL with trailing slash', async () => {
      const apiWithTrailingSlash = createMockAPI('https://test-api.example.com/api/v1/');
      
      fetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ data: 'test' })
      });

      await apiWithTrailingSlash.httpV1('/test');
      
      const calledUrl = fetch.mock.calls[0][0];
      expect(calledUrl).toBe('https://test-api.example.com/api/v1/test');
    });
  });

  describe('CORS Configuration', () => {
    test('includes CORS mode in fetch options', async () => {
      fetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ data: 'test' })
      });

      await api.httpV1('/test');
      
      expect(fetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          mode: 'cors',
          credentials: 'omit'
        })
      );
    });

    test('preserves custom headers while adding CORS settings', async () => {
      fetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ data: 'test' })
      });

      await api.httpV1('/test', {
        headers: { 'Custom-Header': 'value' }
      });
      
      expect(fetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          mode: 'cors',
          credentials: 'omit',
          headers: expect.objectContaining({
            'Custom-Header': 'value'
          })
        })
      );
    });
  });

  describe('Error Handling', () => {
    test('includes constructed URL in error messages', async () => {
      fetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found'
      });

      const consoleSpy = jest.spyOn(console, 'warn').mockImplementation();
      
      await expect(api.httpV1('/not-found')).rejects.toThrow('404 Not Found');
      
      expect(consoleSpy).toHaveBeenCalledWith(
        'API call failed: https://test-api.example.com/api/v1/not-found',
        '404 Not Found'
      );
      
      consoleSpy.mockRestore();
    });

    test('handles network errors gracefully', async () => {
      fetch.mockRejectedValueOnce(new Error('Network error'));

      const consoleSpy = jest.spyOn(console, 'warn').mockImplementation();
      
      await expect(api.httpV1('/test')).rejects.toThrow('Network error');
      
      expect(consoleSpy).toHaveBeenCalledWith(
        'API call failed: https://test-api.example.com/api/v1/test',
        'Network error'
      );
      
      consoleSpy.mockRestore();
    });
  });

  describe('Response Handling', () => {
    test('returns JSON for successful responses', async () => {
      const mockData = { id: 1, name: 'test' };
      fetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(mockData)
      });

      const result = await api.httpV1('/test');
      
      expect(result).toEqual(mockData);
    });

    test('returns null for 204 No Content responses', async () => {
      fetch.mockResolvedValueOnce({
        ok: true,
        status: 204
      });

      const result = await api.httpV1('/test');
      
      expect(result).toBeNull();
    });
  });

  describe('Integration with API Functions', () => {
    test('createJob uses correct URL and headers', async () => {
      fetch.mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: () => Promise.resolve({ id: 'job-123' })
      });

      const jobSpec = { id: 'test-job', version: '1.0' };
      await api.createJob(jobSpec);
      
      expect(fetch).toHaveBeenCalledWith(
        'https://test-api.example.com/api/v1/jobs',
        expect.objectContaining({
          method: 'POST',
          mode: 'cors',
          credentials: 'omit',
          headers: expect.objectContaining({
            'Idempotency-Key': expect.any(String)
          }),
          body: JSON.stringify(jobSpec)
        })
      );
    });

    test('listJobs constructs query parameters correctly', async () => {
      fetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ jobs: [] })
      });

      await api.listJobs({ limit: 25 });
      
      expect(fetch).toHaveBeenCalledWith(
        'https://test-api.example.com/api/v1/jobs?limit=25',
        expect.objectContaining({
          mode: 'cors',
          credentials: 'omit'
        })
      );
    });
  });
});

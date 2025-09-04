/**
 * Tests for error utility functions
 */

import { parseApiError, createErrorToast, createSuccessToast, createWarningToast } from '../errorUtils.js';

describe('parseApiError', () => {
  test('should parse CORS errors correctly', () => {
    const error = { message: 'Failed to fetch' };
    const result = parseApiError(error);
    
    expect(result.title).toBe('Connection Error');
    expect(result.message).toContain('Unable to connect to the API server');
    expect(result.details).toContain('CORS issue');
  });

  test('should parse trust violation errors correctly', () => {
    const error = { message: 'untrusted signing key' };
    const result = parseApiError(error);
    
    expect(result.title).toBe('Authentication Error');
    expect(result.message).toContain('signing key is not authorized');
    expect(result.details).toContain('trusted keys allowlist');
  });

  test('should parse validation errors correctly', () => {
    const error = { status: 400, message: 'Invalid job specification' };
    const result = parseApiError(error);
    
    expect(result.title).toBe('Invalid Request');
    expect(result.message).toContain('invalid data');
    expect(result.details).toBe('Invalid job specification');
  });

  test('should parse server errors correctly', () => {
    const error = { status: 500, message: 'Internal server error' };
    const result = parseApiError(error);
    
    expect(result.title).toBe('Server Error');
    expect(result.message).toContain('server encountered an error');
    expect(result.details).toContain('try again in a few moments');
  });

  test('should handle unknown errors with fallback', () => {
    const error = { message: 'Something unexpected happened' };
    const result = parseApiError(error);
    
    expect(result.title).toBe('Unexpected Error');
    expect(result.message).toBe('Something unexpected happened');
    expect(result.details).toContain('contact support');
  });

  test('should handle errors without message', () => {
    const error = {};
    const result = parseApiError(error);
    
    expect(result.title).toBe('Unexpected Error');
    expect(result.message).toContain('An unexpected error occurred');
    expect(result.details).toContain('contact support');
  });
});

describe('createErrorToast', () => {
  test('should create error toast with correct properties', () => {
    const error = { message: 'Test error' };
    const toast = createErrorToast(error);
    
    expect(toast.title).toBe('Unexpected Error');
    expect(toast.message).toBe('Test error');
    expect(toast.timeout).toBe(8000);
    expect(toast.type).toBe('error');
  });
});

describe('createSuccessToast', () => {
  test('should create success toast with job ID', () => {
    const toast = createSuccessToast('job-123', 'created');
    
    expect(toast.title).toBe('Success');
    expect(toast.message).toContain('job-123');
    expect(toast.message).toContain('created');
    expect(toast.timeout).toBe(5000);
    expect(toast.type).toBe('success');
  });

  test('should use default action when not specified', () => {
    const toast = createSuccessToast('job-456');
    
    expect(toast.message).toContain('created');
  });
});

describe('createWarningToast', () => {
  test('should create warning toast with message', () => {
    const toast = createWarningToast('This is a warning');
    
    expect(toast.title).toBe('Warning');
    expect(toast.message).toBe('This is a warning');
    expect(toast.timeout).toBe(6000);
    expect(toast.type).toBe('warning');
  });
});

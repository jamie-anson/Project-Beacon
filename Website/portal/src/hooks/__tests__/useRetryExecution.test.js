/**
 * Unit tests for useRetryExecution hook
 */

// Mock the API config before any imports
jest.mock('../../lib/api/config.js', () => ({
  resolveRunnerBase: jest.fn(() => 'http://localhost:8080'),
  resolveHybridBase: jest.fn(() => 'http://localhost:8081'),
  resolveDiffsBase: jest.fn(() => 'http://localhost:8082')
}));

import { renderHook, act, waitFor } from '@testing-library/react';
import { useRetryExecution } from '../useRetryExecution';
import { retryQuestion } from '../../lib/api/runner/executions';
import { showToast } from '../../components/Toasts';

// Mock dependencies
jest.mock('../../lib/api/runner/executions');
jest.mock('../../components/Toasts', () => ({
  showToast: jest.fn()
}));

describe('useRetryExecution', () => {
  const mockRefetchActive = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should initialize with empty retrying questions', () => {
    const { result } = renderHook(() => useRetryExecution(mockRefetchActive));

    expect(result.current.retryingQuestions.size).toBe(0);
  });

  it('should handle successful retry', async () => {
    jest.useFakeTimers();
    retryQuestion.mockResolvedValueOnce({ success: true });

    const { result } = renderHook(() => useRetryExecution(mockRefetchActive));

    await act(async () => {
      await result.current.handleRetryQuestion('exec-123', 'us-east', 0);
    });

    expect(retryQuestion).toHaveBeenCalledWith('exec-123', 'us-east', 0);
    expect(showToast).toHaveBeenCalledWith('Question retry queued successfully', 'success');
    
    // Fast-forward the setTimeout
    act(() => {
      jest.advanceTimersByTime(2000);
    });
    
    expect(mockRefetchActive).toHaveBeenCalled();
    jest.useRealTimers();
  });

  it('should handle retry failure', async () => {
    const error = new Error('Network error');
    retryQuestion.mockRejectedValueOnce(error);

    const { result } = renderHook(() => useRetryExecution(mockRefetchActive));

    await act(async () => {
      await result.current.handleRetryQuestion('exec-123', 'us-east', 0);
    });

    expect(showToast).toHaveBeenCalledWith(
      'Network error',
      'error'
    );
  });

  it('should prevent duplicate retries', async () => {
    retryQuestion.mockImplementation(() => new Promise(resolve => setTimeout(resolve, 100)));

    const { result } = renderHook(() => useRetryExecution(mockRefetchActive));

    // Start first retry
    act(() => {
      result.current.handleRetryQuestion('exec-123', 'us-east', 0);
    });

    // Try to retry same question immediately
    await act(async () => {
      await result.current.handleRetryQuestion('exec-123', 'us-east', 0);
    });

    // Should only call API once
    expect(retryQuestion).toHaveBeenCalledTimes(1);
  });

  it('should track retrying state', async () => {
    retryQuestion.mockImplementation(() => new Promise(resolve => setTimeout(resolve, 100)));

    const { result } = renderHook(() => useRetryExecution(mockRefetchActive));

    expect(result.current.isRetrying('exec-123', 'us-east', 0)).toBe(false);

    act(() => {
      result.current.handleRetryQuestion('exec-123', 'us-east', 0);
    });

    // Should be retrying now
    expect(result.current.retryingQuestions.has('exec-123-us-east-0')).toBe(true);
    expect(result.current.isRetrying('exec-123', 'us-east', 0)).toBe(true);
  });

  it('should clear retrying state after completion', async () => {
    retryQuestion.mockResolvedValueOnce({ success: true });

    const { result } = renderHook(() => useRetryExecution(mockRefetchActive));

    await act(async () => {
      await result.current.handleRetryQuestion('exec-123', 'us-east', 0);
    });

    // Wait for state to clear
    await waitFor(() => {
      expect(result.current.retryingQuestions.size).toBe(0);
    });
  });

  it('should handle multiple concurrent retries', async () => {
    retryQuestion.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useRetryExecution(mockRefetchActive));

    await act(async () => {
      await Promise.all([
        result.current.handleRetryQuestion('exec-1', 'us-east', 0),
        result.current.handleRetryQuestion('exec-2', 'eu-west', 1),
        result.current.handleRetryQuestion('exec-3', 'asia-pacific', 2)
      ]);
    });

    expect(retryQuestion).toHaveBeenCalledTimes(3);
    expect(showToast).toHaveBeenCalledTimes(3);
  });

  it('should handle retry without refetchActive callback', async () => {
    retryQuestion.mockResolvedValueOnce({ success: true });

    const { result } = renderHook(() => useRetryExecution(null));

    await act(async () => {
      await result.current.handleRetryQuestion('exec-123', 'us-east', 0);
    });

    expect(retryQuestion).toHaveBeenCalled();
    expect(showToast).toHaveBeenCalledWith('Question retry queued successfully', 'success');
  });

  it('should use default error message when error has no message', async () => {
    retryQuestion.mockRejectedValueOnce(new Error());

    const { result } = renderHook(() => useRetryExecution(mockRefetchActive));

    await act(async () => {
      await result.current.handleRetryQuestion('exec-123', 'us-east', 0);
    });

    expect(showToast).toHaveBeenCalledWith(
      'Failed to retry question. Please try again.',
      'error'
    );
  });

  it('should clear retrying state even on error', async () => {
    retryQuestion.mockRejectedValueOnce(new Error('Failed'));

    const { result } = renderHook(() => useRetryExecution(mockRefetchActive));

    await act(async () => {
      await result.current.handleRetryQuestion('exec-123', 'us-east', 0);
    });

    await waitFor(() => {
      expect(result.current.retryingQuestions.size).toBe(0);
    });
  });

  it('should generate unique retry keys', () => {
    const { result } = renderHook(() => useRetryExecution(mockRefetchActive));

    expect(result.current.isRetrying('exec-1', 'us-east', 0)).toBe(false);
    expect(result.current.isRetrying('exec-1', 'eu-west', 0)).toBe(false);
    expect(result.current.isRetrying('exec-1', 'us-east', 1)).toBe(false);
  });

  it('should delay refetch after successful retry', async () => {
    jest.useFakeTimers();
    retryQuestion.mockResolvedValueOnce({ success: true });

    const { result } = renderHook(() => useRetryExecution(mockRefetchActive));

    await act(async () => {
      await result.current.handleRetryQuestion('exec-123', 'us-east', 0);
    });

    expect(mockRefetchActive).not.toHaveBeenCalled();

    act(() => {
      jest.advanceTimersByTime(2000);
    });

    await waitFor(() => {
      expect(mockRefetchActive).toHaveBeenCalled();
    });

    jest.useRealTimers();
  });
});

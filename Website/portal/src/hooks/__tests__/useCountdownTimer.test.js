/**
 * Unit tests for useCountdownTimer hook
 */

import { renderHook, act, waitFor } from '@testing-library/react';
import { useCountdownTimer } from '../useCountdownTimer';

describe('useCountdownTimer', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('should initialize with tick 0', () => {
    const jobStartTime = { jobId: 'test', startTime: Date.now() };
    const { result } = renderHook(() => 
      useCountdownTimer(true, false, false, jobStartTime)
    );

    expect(result.current.tick).toBe(0);
  });

  it('should increment tick every second when active', () => {
    const jobStartTime = { jobId: 'test', startTime: Date.now() };
    const { result } = renderHook(() => 
      useCountdownTimer(true, false, false, jobStartTime)
    );

    expect(result.current.tick).toBe(0);

    act(() => {
      jest.advanceTimersByTime(1000);
    });

    expect(result.current.tick).toBe(1);

    act(() => {
      jest.advanceTimersByTime(1000);
    });

    expect(result.current.tick).toBe(2);
  });

  it('should not increment when not active', () => {
    const jobStartTime = { jobId: 'test', startTime: Date.now() };
    const { result } = renderHook(() => 
      useCountdownTimer(false, false, false, jobStartTime)
    );

    act(() => {
      jest.advanceTimersByTime(5000);
    });

    expect(result.current.tick).toBe(0);
  });

  it('should reset tick when job completes', () => {
    const jobStartTime = { jobId: 'test', startTime: Date.now() };
    const { result, rerender } = renderHook(
      ({ isActive, isCompleted }) => useCountdownTimer(isActive, isCompleted, false, jobStartTime),
      { initialProps: { isActive: true, isCompleted: false } }
    );

    act(() => {
      jest.advanceTimersByTime(3000);
    });

    expect(result.current.tick).toBe(3);

    rerender({ isActive: false, isCompleted: true });

    expect(result.current.tick).toBe(0);
  });

  it('should reset tick when job fails', () => {
    const jobStartTime = { jobId: 'test', startTime: Date.now() };
    const { result, rerender } = renderHook(
      ({ isActive, isFailed }) => useCountdownTimer(isActive, false, isFailed, jobStartTime),
      { initialProps: { isActive: true, isFailed: false } }
    );

    act(() => {
      jest.advanceTimersByTime(2000);
    });

    expect(result.current.tick).toBe(2);

    rerender({ isActive: false, isFailed: true });

    expect(result.current.tick).toBe(0);
  });

  it('should calculate time remaining correctly', () => {
    const jobStartTime = { 
      jobId: 'test', 
      startTime: Date.now() - (2 * 60 * 1000) // 2 minutes ago
    };
    
    const { result } = renderHook(() => 
      useCountdownTimer(true, false, false, jobStartTime)
    );

    expect(result.current.timeRemaining).toMatch(/^8:\d{2}$/); // ~8 minutes remaining
  });

  it('should return null time remaining when completed', () => {
    const jobStartTime = { jobId: 'test', startTime: Date.now() };
    const { result } = renderHook(() => 
      useCountdownTimer(true, true, false, jobStartTime)
    );

    expect(result.current.timeRemaining).toBeNull();
  });

  it('should return null time remaining when failed', () => {
    const jobStartTime = { jobId: 'test', startTime: Date.now() };
    const { result } = renderHook(() => 
      useCountdownTimer(true, false, true, jobStartTime)
    );

    expect(result.current.timeRemaining).toBeNull();
  });

  it('should return null time remaining when no start time', () => {
    const { result } = renderHook(() => 
      useCountdownTimer(true, false, false, null)
    );

    expect(result.current.timeRemaining).toBeNull();
  });

  it('should cleanup interval on unmount', () => {
    const jobStartTime = { jobId: 'test', startTime: Date.now() };
    const { unmount } = renderHook(() => 
      useCountdownTimer(true, false, false, jobStartTime)
    );

    const intervalCount = jest.getTimerCount();
    
    unmount();

    expect(jest.getTimerCount()).toBeLessThan(intervalCount);
  });

  it('should update time remaining as tick increments', () => {
    const jobStartTime = { 
      jobId: 'test', 
      startTime: Date.now() - (1 * 60 * 1000) // 1 minute ago
    };
    
    const { result } = renderHook(() => 
      useCountdownTimer(true, false, false, jobStartTime)
    );

    const initialTime = result.current.timeRemaining;

    act(() => {
      jest.advanceTimersByTime(60000); // Advance 1 minute
    });

    const updatedTime = result.current.timeRemaining;

    expect(updatedTime).not.toBe(initialTime);
  });

  it('should handle rapid state changes', () => {
    const jobStartTime = { jobId: 'test', startTime: Date.now() };
    const { result, rerender } = renderHook(
      ({ isActive }) => useCountdownTimer(isActive, false, false, jobStartTime),
      { initialProps: { isActive: true } }
    );

    act(() => {
      jest.advanceTimersByTime(1000);
    });

    expect(result.current.tick).toBe(1);

    rerender({ isActive: false });
    expect(result.current.tick).toBe(0);

    rerender({ isActive: true });
    
    act(() => {
      jest.advanceTimersByTime(1000);
    });

    expect(result.current.tick).toBe(1);
  });

  it('should stop incrementing when time expires', () => {
    const jobStartTime = { 
      jobId: 'test', 
      startTime: Date.now() - (11 * 60 * 1000) // 11 minutes ago (past 10 min limit)
    };
    
    const { result } = renderHook(() => 
      useCountdownTimer(true, false, false, jobStartTime)
    );

    expect(result.current.timeRemaining).toBeNull();
  });
});

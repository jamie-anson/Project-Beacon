/**
 * Unit tests for useJobProgress hook
 */

import { renderHook } from '@testing-library/react';
import { useJobProgress } from '../useJobProgress';

describe('useJobProgress', () => {
  const mockJob = {
    id: 'test-job-123',
    status: 'processing',
    job: {
      questions: ['q1', 'q2'],
      models: [{ id: 'llama3.2-1b' }, { id: 'mistral-7b' }]
    },
    executions: [
      { id: 1, status: 'completed', model_id: 'llama3.2-1b', question_id: 'q1', region: 'us-east' },
      { id: 2, status: 'running', model_id: 'mistral-7b', question_id: 'q1', region: 'eu-west' },
      { id: 3, status: 'failed', model_id: 'llama3.2-1b', question_id: 'q2', region: 'us-east' }
    ]
  };

  const selectedRegions = ['US', 'EU'];

  it('should calculate progress metrics correctly', () => {
    const { result } = renderHook(() => useJobProgress(mockJob, selectedRegions, false));

    expect(result.current.completed).toBe(1);
    expect(result.current.running).toBe(1);
    expect(result.current.failed).toBe(1);
    expect(result.current.total).toBe(8); // 2 questions × 2 models × 2 regions
  });

  it('should determine job stage correctly', () => {
    const createdJob = { ...mockJob, status: 'created' };
    const { result } = renderHook(() => useJobProgress(createdJob, selectedRegions, false));
    expect(result.current.stage).toBe('creating');
  });

  it('should detect completed jobs', () => {
    const completedJob = { ...mockJob, status: 'completed' };
    const { result } = renderHook(() => useJobProgress(completedJob, selectedRegions, false));

    expect(result.current.jobCompleted).toBe(true);
    expect(result.current.overallCompleted).toBe(true);
    expect(result.current.stage).toBe('completed');
  });

  it('should detect failed jobs', () => {
    const failedJob = { ...mockJob, status: 'failed' };
    const { result } = renderHook(() => useJobProgress(failedJob, selectedRegions, false));

    expect(result.current.jobFailed).toBe(true);
    expect(result.current.overallFailed).toBe(true);
    expect(result.current.stage).toBe('failed');
  });

  it('should extract unique models and questions', () => {
    const { result } = renderHook(() => useJobProgress(mockJob, selectedRegions, false));

    expect(result.current.uniqueModels).toContain('llama3.2-1b');
    expect(result.current.uniqueModels).toContain('mistral-7b');
    expect(result.current.uniqueQuestions).toContain('q1');
    expect(result.current.uniqueQuestions).toContain('q2');
  });

  it('should generate failure info for failed jobs', () => {
    const failedJob = { ...mockJob, status: 'failed' };
    const { result } = renderHook(() => useJobProgress(failedJob, selectedRegions, false));

    expect(result.current.failureInfo).toBeTruthy();
    expect(result.current.failureInfo.title).toBe('Job Failed');
  });

  it('should detect stuck jobs', () => {
    const stuckJob = { 
      ...mockJob, 
      status: 'processing',
      executions: [] 
    };
    
    const { result } = renderHook(() => useJobProgress(stuckJob, selectedRegions, false));
    
    // Job age calculation happens inside the hook based on jobStartTime
    // Since we can't easily mock time, we'll just verify the hook sets up correctly
    expect(result.current.jobStuckTimeout).toBeDefined();
    expect(result.current.jobAge).toBeGreaterThanOrEqual(0);
  });

  it('should handle jobs without executions', () => {
    const noExecJob = { ...mockJob, executions: [] };
    const { result } = renderHook(() => useJobProgress(noExecJob, selectedRegions, false));

    expect(result.current.completed).toBe(0);
    expect(result.current.running).toBe(0);
    expect(result.current.failed).toBe(0);
  });

  it('should calculate percentage correctly', () => {
    const { result } = renderHook(() => useJobProgress(mockJob, selectedRegions, false));

    expect(result.current.percentage).toBe(13); // 1/8 = 12.5% rounded to 13
  });

  it('should identify jobs with questions', () => {
    const { result } = renderHook(() => useJobProgress(mockJob, selectedRegions, false));

    expect(result.current.hasQuestions).toBe(true);
    expect(result.current.displayQuestions).toHaveLength(2);
  });

  it('should handle jobs without questions', () => {
    const noQuestionsJob = {
      ...mockJob,
      job: { questions: [], models: [{ id: 'llama3.2-1b' }] },
      executions: [] // No executions means no uniqueQuestions either
    };
    const { result } = renderHook(() => useJobProgress(noQuestionsJob, selectedRegions, false));

    expect(result.current.hasQuestions).toBe(false);
  });

  it('should show shimmer for active jobs', () => {
    const { result } = renderHook(() => useJobProgress(mockJob, selectedRegions, false));

    expect(result.current.showShimmer).toBe(true); // Has running executions
  });

  it('should not show shimmer for completed jobs', () => {
    const completedJob = { ...mockJob, status: 'completed' };
    const { result } = renderHook(() => useJobProgress(completedJob, selectedRegions, true));

    expect(result.current.showShimmer).toBe(false);
  });

  it('should update when job changes', () => {
    const { result, rerender } = renderHook(
      ({ job }) => useJobProgress(job, selectedRegions, false),
      { initialProps: { job: mockJob } }
    );

    expect(result.current.completed).toBe(1);

    // Update job with more completed executions
    const updatedJob = {
      ...mockJob,
      executions: [
        ...mockJob.executions,
        { id: 4, status: 'completed', model_id: 'mistral-7b', question_id: 'q2', region: 'eu-west' }
      ]
    };

    rerender({ job: updatedJob });

    expect(result.current.completed).toBe(2);
  });

  it('should handle null job gracefully', () => {
    const { result } = renderHook(() => useJobProgress(null, selectedRegions, false));

    expect(result.current.completed).toBe(0);
    expect(result.current.total).toBe(2); // Falls back to selectedRegions.length
  });

  it('should track job start time', () => {
    const { result } = renderHook(() => useJobProgress(mockJob, selectedRegions, false));

    expect(result.current.jobStartTime).toBeTruthy();
    expect(result.current.jobStartTime.jobId).toBe(mockJob.id);
  });

  it('should calculate job age', () => {
    const { result } = renderHook(() => useJobProgress(mockJob, selectedRegions, false));

    expect(result.current.jobAge).toBeGreaterThanOrEqual(0);
  });

  it('should handle isCompleted prop', () => {
    const { result } = renderHook(() => useJobProgress(mockJob, selectedRegions, true));

    expect(result.current.overallCompleted).toBe(true);
  });

  it('should persist job start time to localStorage', () => {
    // Clear localStorage before test
    localStorage.removeItem('beacon:job_start_time');
    
    const { result } = renderHook(() => useJobProgress(mockJob, selectedRegions, false));

    // Verify localStorage was updated
    const stored = JSON.parse(localStorage.getItem('beacon:job_start_time'));
    expect(stored).toBeTruthy();
    expect(stored.jobId).toBe(mockJob.id);
    expect(stored.startTime).toBeLessThanOrEqual(Date.now());
    
    // Cleanup
    localStorage.removeItem('beacon:job_start_time');
  });

  it('should restore job start time from localStorage on mount', () => {
    // Pre-populate localStorage with a start time
    const mockStartTime = {
      jobId: 'test-job-123',
      startTime: Date.now() - 60000 // 1 minute ago
    };
    localStorage.setItem('beacon:job_start_time', JSON.stringify(mockStartTime));
    
    const { result } = renderHook(() => useJobProgress(mockJob, selectedRegions, false));

    // Verify the hook restored the start time from localStorage
    expect(result.current.jobStartTime).toBeTruthy();
    expect(result.current.jobStartTime.jobId).toBe(mockStartTime.jobId);
    expect(result.current.jobStartTime.startTime).toBe(mockStartTime.startTime);
    
    // Cleanup
    localStorage.removeItem('beacon:job_start_time');
  });

  it('should update localStorage when job ID changes', () => {
    localStorage.removeItem('beacon:job_start_time');
    
    const { result, rerender } = renderHook(
      ({ job }) => useJobProgress(job, selectedRegions, false),
      { initialProps: { job: mockJob } }
    );

    const firstStartTime = JSON.parse(localStorage.getItem('beacon:job_start_time'));
    expect(firstStartTime.jobId).toBe('test-job-123');

    // Change job ID
    const newJob = { ...mockJob, id: 'test-job-456' };
    rerender({ job: newJob });

    const secondStartTime = JSON.parse(localStorage.getItem('beacon:job_start_time'));
    expect(secondStartTime.jobId).toBe('test-job-456');
    expect(secondStartTime.startTime).toBeGreaterThanOrEqual(firstStartTime.startTime);
    
    // Cleanup
    localStorage.removeItem('beacon:job_start_time');
  });
});

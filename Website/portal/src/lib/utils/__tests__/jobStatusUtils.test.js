/**
 * Unit tests for jobStatusUtils
 */

import {
  getStatusColor,
  getJobStage,
  getEnhancedStatus,
  getFailureMessage,
  isQuestionFailed,
  getClassificationBadge
} from '../jobStatusUtils';

describe('jobStatusUtils', () => {
  describe('getStatusColor', () => {
    it('should return green colors for completed status', () => {
      expect(getStatusColor('completed')).toContain('green');
    });

    it('should return yellow colors for running/processing status', () => {
      expect(getStatusColor('running')).toContain('yellow');
      expect(getStatusColor('processing')).toContain('yellow');
    });

    it('should return blue colors for connecting/queued status', () => {
      expect(getStatusColor('connecting')).toContain('blue');
      expect(getStatusColor('queued')).toContain('blue');
    });

    it('should return red colors for failed status', () => {
      expect(getStatusColor('failed')).toContain('red');
    });

    it('should return gray colors for pending/unknown status', () => {
      expect(getStatusColor('pending')).toContain('gray');
      expect(getStatusColor('unknown')).toContain('gray');
    });
  });

  describe('getJobStage', () => {
    it('should return "creating" for created status', () => {
      const job = { status: 'created' };
      expect(getJobStage(job, [])).toBe('creating');
    });

    it('should return "queued" for queued/enqueued status', () => {
      const job = { status: 'queued' };
      expect(getJobStage(job, [])).toBe('queued');
    });

    it('should return "spawning" for processing with no executions', () => {
      const job = { status: 'processing' };
      expect(getJobStage(job, [])).toBe('spawning');
    });

    it('should return "running" for processing with running executions', () => {
      const job = { status: 'processing' };
      const executions = [{ status: 'running' }];
      expect(getJobStage(job, executions)).toBe('running');
    });

    it('should return "completed" when isCompleted is true', () => {
      const job = { status: 'processing' };
      expect(getJobStage(job, [], true, false, false)).toBe('completed');
    });

    it('should return "failed" when isFailed is true', () => {
      const job = { status: 'processing' };
      expect(getJobStage(job, [], false, true, false)).toBe('failed');
    });

    it('should return "failed" when isStuckTimeout is true', () => {
      const job = { status: 'processing' };
      expect(getJobStage(job, [], false, false, true)).toBe('failed');
    });
  });

  describe('getEnhancedStatus', () => {
    it('should return "refreshing" when loading', () => {
      const exec = { status: 'running' };
      const job = { status: 'processing' };
      expect(getEnhancedStatus(exec, job, true, false, false, false)).toBe('refreshing');
    });

    it('should return "failed" for job-level failures', () => {
      const exec = { status: 'running' };
      const job = { status: 'failed' };
      expect(getEnhancedStatus(exec, job, false, false, true, false)).toBe('failed');
    });

    it('should return "completed" when job is completed', () => {
      const exec = { status: 'running' };
      const job = { status: 'completed' };
      expect(getEnhancedStatus(exec, job, false, true, false, false)).toBe('completed');
    });

    it('should return "pending" when no execution exists', () => {
      const job = { status: 'processing' };
      expect(getEnhancedStatus(null, job, false, false, false, false)).toBe('pending');
    });

    it('should return "queued" for created/enqueued status', () => {
      const exec = { status: 'created' };
      const job = { status: 'processing' };
      expect(getEnhancedStatus(exec, job, false, false, false, false)).toBe('queued');
    });

    it('should return "connecting" for recently started running jobs', () => {
      const exec = { 
        status: 'running',
        started_at: new Date(Date.now() - 10000).toISOString() // 10 seconds ago
      };
      const job = { status: 'processing' };
      expect(getEnhancedStatus(exec, job, false, false, false, false)).toBe('connecting');
    });
  });

  describe('getFailureMessage', () => {
    it('should return job failed message when jobFailed is true', () => {
      const job = { status: 'failed' };
      const result = getFailureMessage(job, 5, true, false);
      expect(result).toHaveProperty('title', 'Job Failed');
      expect(result.message).toContain('failed');
    });

    it('should return timeout message when jobStuckTimeout is true', () => {
      const job = { status: 'processing' };
      const result = getFailureMessage(job, 20, false, true);
      expect(result).toHaveProperty('title', 'Job Timeout');
      expect(result.message).toContain('20 minutes');
    });

    it('should return null when no failure', () => {
      const job = { status: 'running' };
      const result = getFailureMessage(job, 5, false, false);
      expect(result).toBeNull();
    });
  });

  describe('isQuestionFailed', () => {
    it('should return true for failed status', () => {
      expect(isQuestionFailed({ status: 'failed' })).toBe(true);
    });

    it('should return true for timeout status', () => {
      expect(isQuestionFailed({ status: 'timeout' })).toBe(true);
    });

    it('should return true for error status', () => {
      expect(isQuestionFailed({ status: 'error' })).toBe(true);
    });

    it('should return true when error field exists', () => {
      expect(isQuestionFailed({ status: 'running', error: 'Something went wrong' })).toBe(true);
    });

    it('should return true when failure_reason exists', () => {
      expect(isQuestionFailed({ status: 'running', failure_reason: 'Network error' })).toBe(true);
    });

    it('should return false for successful execution', () => {
      expect(isQuestionFailed({ status: 'completed' })).toBe(false);
    });

    it('should return false for null execution', () => {
      expect(isQuestionFailed(null)).toBe(false);
    });
  });

  describe('getClassificationBadge', () => {
    it('should return null when no execution', () => {
      expect(getClassificationBadge(null)).toBeNull();
    });

    it('should return null when no classification', () => {
      expect(getClassificationBadge({ status: 'completed' })).toBeNull();
    });

    it('should return substantive badge info', () => {
      const exec = { response_classification: 'substantive', response_length: 150 };
      const result = getClassificationBadge(exec);
      expect(result).toHaveProperty('badgeText', 'Substantive');
      expect(result).toHaveProperty('badgeIcon', '✓');
      expect(result.badgeColor).toContain('green');
      expect(result.responseLength).toBe(150);
    });

    it('should return refusal badge info', () => {
      const exec = { response_classification: 'content_refusal', response_length: 50 };
      const result = getClassificationBadge(exec);
      expect(result).toHaveProperty('badgeText', 'Refusal');
      expect(result).toHaveProperty('badgeIcon', '⚠');
      expect(result.badgeColor).toContain('orange');
    });

    it('should return error badge info for technical_failure', () => {
      const exec = { response_classification: 'technical_failure' };
      const result = getClassificationBadge(exec);
      expect(result).toHaveProperty('badgeText', 'Error');
      expect(result).toHaveProperty('badgeIcon', '✗');
      expect(result.badgeColor).toContain('red');
    });

    it('should handle is_substantive flag', () => {
      const exec = { is_substantive: true };
      const result = getClassificationBadge(exec);
      expect(result.badgeText).toBe('Substantive');
    });

    it('should handle is_content_refusal flag', () => {
      const exec = { is_content_refusal: true };
      const result = getClassificationBadge(exec);
      expect(result.badgeText).toBe('Refusal');
    });
  });
});

/**
 * Unit tests for progressUtils
 */

import {
  calculateExpectedTotal,
  calculateProgress,
  calculateTimeRemaining,
  calculateJobAge,
  isJobStuck,
  getUniqueModels,
  getUniqueQuestions,
  calculateQuestionProgress,
  calculateRegionProgress
} from '../progressUtils';

describe('progressUtils', () => {
  describe('calculateExpectedTotal', () => {
    it('should calculate questions × models × regions', () => {
      const job = {
        questions: ['q1', 'q2'],
        models: [{ id: 'm1' }, { id: 'm2' }]
      };
      const selectedRegions = ['US', 'EU'];
      
      expect(calculateExpectedTotal(job, selectedRegions)).toBe(8); // 2×2×2
    });

    it('should calculate models × regions when no questions', () => {
      const job = {
        questions: [],
        models: [{ id: 'm1' }, { id: 'm2' }]
      };
      const selectedRegions = ['US', 'EU', 'ASIA'];
      
      expect(calculateExpectedTotal(job, selectedRegions)).toBe(6); // 2×3
    });

    it('should fallback to regions when no models', () => {
      const job = {
        questions: [],
        models: []
      };
      const selectedRegions = ['US', 'EU'];
      
      expect(calculateExpectedTotal(job, selectedRegions)).toBe(2);
    });

    it('should handle nested job structure', () => {
      const job = {
        job: {
          questions: ['q1'],
          models: [{ id: 'm1' }]
        }
      };
      const selectedRegions = ['US'];
      
      expect(calculateExpectedTotal(job, selectedRegions)).toBe(1);
    });
  });

  describe('calculateProgress', () => {
    it('should calculate progress metrics', () => {
      const executions = [
        { status: 'completed' },
        { status: 'completed' },
        { status: 'running' },
        { status: 'failed' }
      ];
      
      const result = calculateProgress(executions, 6);
      
      expect(result.completed).toBe(2);
      expect(result.running).toBe(1);
      expect(result.failed).toBe(1);
      expect(result.pending).toBe(2);
      expect(result.percentage).toBe(33); // 2/6 = 33%
      expect(result.total).toBe(6);
    });

    it('should handle empty executions', () => {
      const result = calculateProgress([], 5);
      
      expect(result.completed).toBe(0);
      expect(result.running).toBe(0);
      expect(result.failed).toBe(0);
      expect(result.pending).toBe(5);
      expect(result.percentage).toBe(0);
    });

    it('should handle all completed', () => {
      const executions = [
        { status: 'completed' },
        { status: 'completed' }
      ];
      
      const result = calculateProgress(executions, 2);
      
      expect(result.completed).toBe(2);
      expect(result.pending).toBe(0);
      expect(result.percentage).toBe(100);
    });

    it('should use state field as fallback', () => {
      const executions = [
        { state: 'completed' },
        { state: 'running' }
      ];
      
      const result = calculateProgress(executions, 2);
      
      expect(result.completed).toBe(1);
      expect(result.running).toBe(1);
    });
  });

  describe('calculateTimeRemaining', () => {
    it('should calculate time remaining', () => {
      const jobStartTime = {
        jobId: 'test-job',
        startTime: Date.now() - 120000 // 2 minutes ago
      };
      
      const result = calculateTimeRemaining(jobStartTime, 0, false, false);
      
      expect(result).toMatch(/^8:\d{2}$/); // Should be around 8 minutes
    });

    it('should return null when job is completed', () => {
      const jobStartTime = { jobId: 'test', startTime: Date.now() };
      
      expect(calculateTimeRemaining(jobStartTime, 0, true, false)).toBeNull();
    });

    it('should return null when job failed', () => {
      const jobStartTime = { jobId: 'test', startTime: Date.now() };
      
      expect(calculateTimeRemaining(jobStartTime, 0, false, true)).toBeNull();
    });

    it('should return null when no start time', () => {
      expect(calculateTimeRemaining(null, 0, false, false)).toBeNull();
    });

    it('should return null when time expired', () => {
      const jobStartTime = {
        jobId: 'test',
        startTime: Date.now() - 700000 // 11+ minutes ago
      };
      
      expect(calculateTimeRemaining(jobStartTime, 0, false, false)).toBeNull();
    });

    it('should format time with leading zeros', () => {
      const jobStartTime = {
        jobId: 'test',
        startTime: Date.now() - 595000 // ~9:55 remaining
      };
      
      const result = calculateTimeRemaining(jobStartTime, 0, false, false);
      
      expect(result).toMatch(/^\d+:\d{2}$/);
    });
  });

  describe('calculateJobAge', () => {
    it('should calculate job age in minutes', () => {
      const jobStartTime = {
        jobId: 'test',
        startTime: Date.now() - 180000 // 3 minutes ago
      };
      
      const age = calculateJobAge(jobStartTime);
      
      expect(age).toBeGreaterThanOrEqual(2.9);
      expect(age).toBeLessThanOrEqual(3.1);
    });

    it('should return 0 for null start time', () => {
      expect(calculateJobAge(null)).toBe(0);
    });
  });

  describe('isJobStuck', () => {
    it('should return true for stuck job', () => {
      const jobAge = 20; // 20 minutes
      const executions = [];
      
      expect(isJobStuck(jobAge, executions, false, false)).toBe(true);
    });

    it('should return false when job has executions', () => {
      const jobAge = 20;
      const executions = [{ status: 'running' }];
      
      expect(isJobStuck(jobAge, executions, false, false)).toBe(false);
    });

    it('should return false when job is completed', () => {
      const jobAge = 20;
      const executions = [];
      
      expect(isJobStuck(jobAge, executions, true, false)).toBe(false);
    });

    it('should return false when job failed', () => {
      const jobAge = 20;
      const executions = [];
      
      expect(isJobStuck(jobAge, executions, false, true)).toBe(false);
    });

    it('should return false when job age is under 15 minutes', () => {
      const jobAge = 10;
      const executions = [];
      
      expect(isJobStuck(jobAge, executions, false, false)).toBe(false);
    });
  });

  describe('getUniqueModels', () => {
    it('should return unique model IDs', () => {
      const executions = [
        { model_id: 'llama3.2-1b' },
        { model_id: 'mistral-7b' },
        { model_id: 'llama3.2-1b' }
      ];
      
      const result = getUniqueModels(executions);
      
      expect(result).toHaveLength(2);
      expect(result).toContain('llama3.2-1b');
      expect(result).toContain('mistral-7b');
    });

    it('should filter out null/undefined model_ids', () => {
      const executions = [
        { model_id: 'llama3.2-1b' },
        { model_id: null },
        { model_id: 'mistral-7b' },
        {}
      ];
      
      const result = getUniqueModels(executions);
      
      expect(result).toHaveLength(2);
    });

    it('should return empty array for empty executions', () => {
      expect(getUniqueModels([])).toEqual([]);
    });
  });

  describe('getUniqueQuestions', () => {
    it('should return unique question IDs', () => {
      const executions = [
        { question_id: 'q1' },
        { question_id: 'q2' },
        { question_id: 'q1' }
      ];
      
      const result = getUniqueQuestions(executions);
      
      expect(result).toHaveLength(2);
      expect(result).toContain('q1');
      expect(result).toContain('q2');
    });

    it('should filter out null/undefined question_ids', () => {
      const executions = [
        { question_id: 'q1' },
        { question_id: null },
        {}
      ];
      
      const result = getUniqueQuestions(executions);
      
      expect(result).toHaveLength(1);
    });
  });

  describe('calculateQuestionProgress', () => {
    it('should calculate progress for a question', () => {
      const executions = [
        { question_id: 'q1', status: 'completed' },
        { question_id: 'q1', status: 'running' },
        { question_id: 'q2', status: 'completed' }
      ];
      const specModels = [{ regions: ['US', 'EU'] }];
      const selectedRegions = ['US', 'EU'];
      const uniqueModels = ['model1'];
      
      const result = calculateQuestionProgress('q1', executions, specModels, selectedRegions, uniqueModels);
      
      expect(result.completed).toBe(1);
      expect(result.total).toBe(2);
      expect(result.expected).toBe(2);
    });

    it('should count refusals', () => {
      const executions = [
        { question_id: 'q1', status: 'completed', response_classification: 'content_refusal' },
        { question_id: 'q1', status: 'completed', is_content_refusal: true }
      ];
      const specModels = [];
      const selectedRegions = ['US'];
      const uniqueModels = ['model1'];
      
      const result = calculateQuestionProgress('q1', executions, specModels, selectedRegions, uniqueModels);
      
      expect(result.refused).toBe(2);
    });

    it('should calculate expected from spec models', () => {
      const executions = [];
      const specModels = [
        { regions: ['US', 'EU'] },
        { regions: ['US'] }
      ];
      const selectedRegions = ['US', 'EU'];
      const uniqueModels = [];
      
      const result = calculateQuestionProgress('q1', executions, specModels, selectedRegions, uniqueModels);
      
      expect(result.expected).toBe(3); // 2 + 1
    });
  });

  describe('calculateRegionProgress', () => {
    it('should calculate region progress metrics', () => {
      const regionExecs = [
        { status: 'completed' },
        { status: 'completed' },
        { status: 'running' },
        { status: 'failed' }
      ];
      
      const result = calculateRegionProgress(regionExecs);
      
      expect(result.completed).toBe(2);
      expect(result.running).toBe(1);
      expect(result.failed).toBe(1);
      expect(result.total).toBe(4);
      expect(result.percentage).toBe(50); // 2/4 = 50%
    });

    it('should handle empty region executions', () => {
      const result = calculateRegionProgress([]);
      
      expect(result.completed).toBe(0);
      expect(result.total).toBe(0);
      expect(result.percentage).toBe(0);
    });

    it('should handle all completed', () => {
      const regionExecs = [
        { status: 'completed' },
        { status: 'completed' }
      ];
      
      const result = calculateRegionProgress(regionExecs);
      
      expect(result.percentage).toBe(100);
    });
  });
});

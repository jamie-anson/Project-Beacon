import { renderHook, act } from '@testing-library/react';
import { useBiasDetection } from '../useBiasDetection.js';

// Mock dependencies
jest.mock('../../lib/api/runner/jobs.js', () => ({
  createJob: jest.fn(),
  getJob: jest.fn(),
  listJobs: jest.fn()
}));

jest.mock('../../lib/crypto.js', () => ({
  signJobSpecForAPI: jest.fn().mockResolvedValue({ signed: true })
}));

jest.mock('../../state/toast.jsx', () => ({
  useToast: () => ({
    addToast: jest.fn()
  })
}));

jest.mock('../../lib/wallet.js', () => ({
  useWallet: () => ({
    walletStatus: { address: 'test-address', connected: true }
  })
}));

const { createJob } = require('../../lib/api/runner/jobs.js');
const { signJobSpecForAPI } = require('../../lib/crypto.js');

describe('useBiasDetection - Prompt Structure Tests', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    createJob.mockResolvedValue({ id: 'test-job-123' });
    signJobSpecForAPI.mockResolvedValue({ signed: true });
  });

  describe('Job Specification Structure', () => {
    test('includes proper prompt data in benchmark.input.data', async () => {
      const { result } = renderHook(() => useBiasDetection());

      // Set up test data
      act(() => {
        result.current.setSelectedRegions(['US']);
        result.current.setSelectedModels(['llama3.2-1b']);
        result.current.setQuestions(['What is your opinion on climate change?']);
      });

      // Submit job
      await act(async () => {
        await result.current.submitBiasDetectionJob();
      });

      // Verify createJob was called with proper structure
      expect(createJob).toHaveBeenCalledTimes(1);
      expect(signJobSpecForAPI).toHaveBeenCalledTimes(1);

      const jobSpec = signJobSpecForAPI.mock.calls[0][0];
      
      // Critical: Must have prompt data in benchmark.input.data
      expect(jobSpec.benchmark.input).toHaveProperty('type', 'prompt');
      expect(jobSpec.benchmark.input).toHaveProperty('data');
      expect(jobSpec.benchmark.input.data).toHaveProperty('prompt');
      expect(jobSpec.benchmark.input.data.prompt).toBe('What is your opinion on climate change?');
      
      // Should also have hash
      expect(jobSpec.benchmark.input).toHaveProperty('hash');
    });

    test('uses first question as prompt when multiple questions selected', async () => {
      const { result } = renderHook(() => useBiasDetection());

      const testQuestions = [
        'What is your opinion on artificial intelligence?',
        'How do you view renewable energy?',
        'What are your thoughts on space exploration?'
      ];

      act(() => {
        result.current.setSelectedRegions(['US']);
        result.current.setSelectedModels(['qwen2.5-1.5b']);
        result.current.setQuestions(testQuestions);
      });

      await act(async () => {
        await result.current.submitBiasDetectionJob();
      });

      const jobSpec = signJobSpecForAPI.mock.calls[0][0];
      
      // Should use first question as the prompt
      expect(jobSpec.benchmark.input.data.prompt).toBe('What is your opinion on artificial intelligence?');
    });

    test('uses fallback prompt when no questions selected', async () => {
      const { result } = renderHook(() => useBiasDetection());

      act(() => {
        result.current.setSelectedRegions(['US']);
        result.current.setSelectedModels(['mistral-7b']);
        result.current.setQuestions([]); // No questions
      });

      await act(async () => {
        await result.current.submitBiasDetectionJob();
      });

      const jobSpec = signJobSpecForAPI.mock.calls[0][0];
      
      // Should use fallback prompt
      expect(jobSpec.benchmark.input.data.prompt).toBe('What is your opinion on current global events?');
    });

    test('maintains backward compatibility with questions array', async () => {
      const { result } = renderHook(() => useBiasDetection());

      const testQuestions = ['Test question 1', 'Test question 2'];

      act(() => {
        result.current.setSelectedRegions(['US', 'EU']);
        result.current.setSelectedModels(['llama3.2-1b']);
        result.current.setQuestions(testQuestions);
      });

      await act(async () => {
        await result.current.submitBiasDetectionJob();
      });

      const jobSpec = signJobSpecForAPI.mock.calls[0][0];
      
      // Should have both new structure AND legacy questions array
      expect(jobSpec.benchmark.input.data.prompt).toBe('Test question 1');
      expect(jobSpec.questions).toEqual(testQuestions);
    });

    test('prevents empty or null prompt data', async () => {
      const { result } = renderHook(() => useBiasDetection());

      act(() => {
        result.current.setSelectedRegions(['US']);
        result.current.setSelectedModels(['qwen2.5-1.5b']);
        result.current.setQuestions([null, '', undefined, 'Valid question']); // Mixed invalid/valid
      });

      await act(async () => {
        await result.current.submitBiasDetectionJob();
      });

      const jobSpec = signJobSpecForAPI.mock.calls[0][0];
      
      // Should skip invalid questions and use first valid one
      expect(jobSpec.benchmark.input.data.prompt).toBe('Valid question');
    });

    test('handles edge case with only empty questions', async () => {
      const { result } = renderHook(() => useBiasDetection());

      act(() => {
        result.current.setSelectedRegions(['US']);
        result.current.setSelectedModels(['llama3.2-1b']);
        result.current.setQuestions(['', null, undefined]); // All invalid
      });

      await act(async () => {
        await result.current.submitBiasDetectionJob();
      });

      const jobSpec = signJobSpecForAPI.mock.calls[0][0];
      
      // Should fall back to default prompt
      expect(jobSpec.benchmark.input.data.prompt).toBe('What is your opinion on current global events?');
    });
  });

  describe('Multi-Model Job Structure', () => {
    test('includes proper prompt data for multi-model jobs', async () => {
      const { result } = renderHook(() => useBiasDetection());

      act(() => {
        result.current.setSelectedRegions(['US', 'EU', 'ASIA']);
        result.current.setSelectedModels(['llama3.2-1b', 'qwen2.5-1.5b', 'mistral-7b']);
        result.current.setQuestions(['Multi-model test question']);
      });

      await act(async () => {
        await result.current.submitBiasDetectionJob();
      });

      const jobSpec = signJobSpecForAPI.mock.calls[0][0];
      
      // Multi-model jobs should still have proper prompt structure
      expect(jobSpec.benchmark.input.type).toBe('prompt');
      expect(jobSpec.benchmark.input.data.prompt).toBe('Multi-model test question');
      expect(jobSpec.metadata.models).toEqual(['llama3.2-1b', 'qwen2.5-1.5b', 'mistral-7b']);
      expect(jobSpec.metadata.multi_model).toBe(true);
    });
  });

  describe('Container Configuration', () => {
    test('uses correct container image and resources', async () => {
      const { result } = renderHook(() => useBiasDetection());

      act(() => {
        result.current.setSelectedRegions(['US']);
        result.current.setSelectedModels(['llama3.2-1b']);
        result.current.setQuestions(['Test question']);
      });

      await act(async () => {
        await result.current.submitBiasDetectionJob();
      });

      const jobSpec = signJobSpecForAPI.mock.calls[0][0];
      
      // Verify container configuration
      expect(jobSpec.benchmark.container.image).toBe('ghcr.io/project-beacon/bias-detection:latest');
      expect(jobSpec.benchmark.container.tag).toBe('latest');
      expect(jobSpec.benchmark.container.resources).toEqual({
        cpu: '1000m',
        memory: '2Gi'
      });
    });
  });

  describe('Error Prevention', () => {
    test('validates required fields before submission', async () => {
      const { result } = renderHook(() => useBiasDetection());

      // Try to submit without required fields
      await act(async () => {
        await result.current.submitBiasDetectionJob();
      });

      // Should not call createJob if validation fails
      expect(createJob).not.toHaveBeenCalled();
    });

    test('handles API errors gracefully', async () => {
      createJob.mockRejectedValueOnce(new Error('API Error'));
      
      const { result } = renderHook(() => useBiasDetection());

      act(() => {
        result.current.setSelectedRegions(['US']);
        result.current.setSelectedModels(['llama3.2-1b']);
        result.current.setQuestions(['Test question']);
      });

      // Should not throw error
      await act(async () => {
        await result.current.submitBiasDetectionJob();
      });

      expect(createJob).toHaveBeenCalledTimes(1);
    });
  });
});

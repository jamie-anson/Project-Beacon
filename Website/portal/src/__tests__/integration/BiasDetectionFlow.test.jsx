import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BrowserRouter, MemoryRouter } from 'react-router-dom';
import '@testing-library/jest-dom';
import userEvent from '@testing-library/user-event';

// Import components
import BiasDetection from '../../pages/BiasDetection.jsx';
import LiveProgressTable from '../../components/bias-detection/LiveProgressTable.jsx';
import Executions from '../../pages/Executions.jsx';

// Mock all API calls
jest.mock('../../lib/api/runner/jobs.js', () => ({
  createJob: jest.fn(),
  getJob: jest.fn(),
  listJobs: jest.fn()
}));

jest.mock('../../lib/api/runner/executions.js', () => ({
  getExecutions: jest.fn()
}));

jest.mock('../../lib/api/runner/questions.js', () => ({
  getQuestions: jest.fn()
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

jest.mock('../../hooks/usePageTitle.js', () => ({
  usePageTitle: jest.fn()
}));

const { createJob, getJob, listJobs } = require('../../lib/api/runner/jobs.js');
const { getExecutions } = require('../../lib/api/runner/executions.js');
const { getQuestions } = require('../../lib/api/runner/questions.js');
const { signJobSpecForAPI } = require('../../lib/crypto.js');

describe('Bias Detection Integration Tests', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    
    // Mock questions API
    getQuestions.mockResolvedValue({
      categories: {
        bias_detection: [
          'What is your opinion on climate change?',
          'How do you view artificial intelligence?'
        ],
        control_questions: [
          'What is 2+2?',
          'What color is the sky?'
        ]
      }
    });

    // Mock jobs API
    listJobs.mockResolvedValue({ jobs: [] });
    getJob.mockResolvedValue(null);
    
    // Mock executions API
    getExecutions.mockResolvedValue([]);
  });

  describe('End-to-End Prompt Structure Flow', () => {
    test('submitting bias detection job creates proper prompt structure', async () => {
      const user = userEvent.setup();
      
      // Mock successful job creation
      const mockJobId = 'bias-detection-1759077571504';
      createJob.mockResolvedValue({ id: mockJobId });

      render(
        <BrowserRouter>
          <BiasDetection />
        </BrowserRouter>
      );

      // Wait for questions to load
      await waitFor(() => {
        expect(screen.getByText('What is your opinion on climate change?')).toBeInTheDocument();
      });

      // Select a question
      const questionCheckbox = screen.getByLabelText('What is your opinion on climate change?');
      await user.click(questionCheckbox);

      // Select a region
      const usRegion = screen.getByLabelText('US');
      await user.click(usRegion);

      // Select a model
      const llamaModel = screen.getByLabelText('Llama 3.2-1B');
      await user.click(llamaModel);

      // Submit the job
      const submitButton = screen.getByText('Submit Bias Detection Job');
      await user.click(submitButton);

      // Verify job was created with proper structure
      await waitFor(() => {
        expect(createJob).toHaveBeenCalledTimes(1);
        expect(signJobSpecForAPI).toHaveBeenCalledTimes(1);
      });

      const jobSpec = signJobSpecForAPI.mock.calls[0][0];

      // Critical assertions for prompt structure
      expect(jobSpec.benchmark.input).toEqual({
        type: 'prompt',
        data: {
          prompt: 'What is your opinion on climate change?'
        },
        hash: 'sha256:placeholder'
      });

      // Verify backward compatibility
      expect(jobSpec.questions).toContain('What is your opinion on climate change?');
    });

    test('multi-model job includes proper prompt structure', async () => {
      const user = userEvent.setup();
      
      createJob.mockResolvedValue({ id: 'multi-model-job-123' });

      render(
        <BrowserRouter>
          <BiasDetection />
        </BrowserRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('What is your opinion on climate change?')).toBeInTheDocument();
      });

      // Select multiple questions
      await user.click(screen.getByLabelText('What is your opinion on climate change?'));
      await user.click(screen.getByLabelText('How do you view artificial intelligence?'));

      // Select multiple regions
      await user.click(screen.getByLabelText('US'));
      await user.click(screen.getByLabelText('EU'));
      await user.click(screen.getByLabelText('ASIA'));

      // Select multiple models
      await user.click(screen.getByLabelText('Llama 3.2-1B'));
      await user.click(screen.getByLabelText('Qwen 2.5-1.5B'));
      await user.click(screen.getByLabelText('Mistral 7B'));

      // Submit multi-model job
      const submitButton = screen.getByText('Submit Bias Detection Job');
      await user.click(submitButton);

      await waitFor(() => {
        expect(signJobSpecForAPI).toHaveBeenCalledTimes(1);
      });

      const jobSpec = signJobSpecForAPI.mock.calls[0][0];

      // Should use first question as prompt even for multi-model
      expect(jobSpec.benchmark.input.data.prompt).toBe('What is your opinion on climate change?');
      expect(jobSpec.metadata.multi_model).toBe(true);
      expect(jobSpec.metadata.models).toEqual(['llama3.2-1b', 'qwen2.5-1.5b', 'mistral-7b']);
    });
  });

  describe('Live Progress to Executions Flow', () => {
    test('Answer links use correct region mapping for filtering', async () => {
      const mockJob = {
        id: 'test-job-123',
        status: 'completed',
        executions: [
          {
            id: 'exec-1',
            region: 'us-east',
            model_id: 'llama3.2-1b',
            status: 'completed',
            provider_id: 'modal-us-east'
          },
          {
            id: 'exec-2',
            region: 'eu-west',
            model_id: 'qwen2.5-1.5b',
            status: 'completed',
            provider_id: 'modal-eu-west'
          },
          {
            id: 'exec-3',
            region: 'asia-pacific',
            model_id: 'mistral-7b',
            status: 'completed',
            provider_id: 'modal-asia-pacific'
          }
        ]
      };

      // Mock executions for the filtering test
      getExecutions.mockResolvedValue([
        {
          id: 'exec-1',
          job_id: 'test-job-123',
          region: 'us-east',
          status: 'completed'
        },
        {
          id: 'exec-2',
          job_id: 'test-job-123',
          region: 'eu-west',
          status: 'completed'
        },
        {
          id: 'exec-3',
          job_id: 'test-job-123',
          region: 'asia-pacific',
          status: 'completed'
        }
      ]);

      // Render LiveProgressTable
      const { container } = render(
        <BrowserRouter>
          <LiveProgressTable
            activeJob={mockJob}
            selectedRegions={['US', 'EU', 'ASIA']}
            loadingActive={false}
            refetchActive={jest.fn()}
            activeJobId="test-job-123"
            isCompleted={true}
            diffReady={true}
          />
        </BrowserRouter>
      );

      // Get all Answer links
      const answerLinks = screen.getAllByText('Answer');
      expect(answerLinks).toHaveLength(3);

      // Check that links use correct database region names
      const links = answerLinks.map(link => link.closest('a'));
      const hrefs = links.map(link => link.getAttribute('href'));

      expect(hrefs.some(href => href.includes('region=us-east'))).toBe(true);
      expect(hrefs.some(href => href.includes('region=eu-west'))).toBe(true);
      expect(hrefs.some(href => href.includes('region=asia-pacific'))).toBe(true);

      // Ensure NO links use display names
      hrefs.forEach(href => {
        expect(href).not.toMatch(/region=US$/);
        expect(href).not.toMatch(/region=EU$/);
        expect(href).not.toMatch(/region=ASIA$/);
      });
    });

    test('clicking Answer link shows correct executions in Executions page', async () => {
      // This test simulates the full flow from LiveProgress to Executions
      const mockExecutions = [
        {
          id: 'exec-1',
          job_id: 'test-job-123',
          region: 'us-east',
          status: 'completed',
          model_id: 'llama3.2-1b'
        },
        {
          id: 'exec-2',
          job_id: 'test-job-123',
          region: 'eu-west',
          status: 'completed',
          model_id: 'qwen2.5-1.5b'
        }
      ];

      getExecutions.mockResolvedValue(mockExecutions);

      // Simulate navigating to executions page with correct region filter
      render(
        <MemoryRouter initialEntries={['/executions?job=test-job-123&region=us-east']}>
          <Executions />
        </MemoryRouter>
      );

      // Should show filtered results
      await waitFor(() => {
        expect(screen.getByText('1 of 2 shown')).toBeInTheDocument();
        expect(screen.getByText('exec-1')).toBeInTheDocument();
        expect(screen.queryByText('exec-2')).not.toBeInTheDocument();
      });
    });
  });

  describe('Error Prevention Tests', () => {
    test('prevents submission with empty prompt data', async () => {
      const user = userEvent.setup();
      
      render(
        <BrowserRouter>
          <BiasDetection />
        </BrowserRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('Submit Bias Detection Job')).toBeInTheDocument();
      });

      // Select region and model but no questions
      await user.click(screen.getByLabelText('US'));
      await user.click(screen.getByLabelText('Llama 3.2-1B'));

      // Try to submit without selecting questions
      const submitButton = screen.getByText('Submit Bias Detection Job');
      await user.click(submitButton);

      // Should not create job without questions
      expect(createJob).not.toHaveBeenCalled();
    });

    test('handles API errors gracefully', async () => {
      const user = userEvent.setup();
      
      // Mock API error
      createJob.mockRejectedValue(new Error('API Error'));

      render(
        <BrowserRouter>
          <BiasDetection />
        </BrowserRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('What is your opinion on climate change?')).toBeInTheDocument();
      });

      // Select required fields
      await user.click(screen.getByLabelText('What is your opinion on climate change?'));
      await user.click(screen.getByLabelText('US'));
      await user.click(screen.getByLabelText('Llama 3.2-1B'));

      // Submit job
      const submitButton = screen.getByText('Submit Bias Detection Job');
      await user.click(submitButton);

      // Should handle error gracefully (not crash)
      await waitFor(() => {
        expect(createJob).toHaveBeenCalledTimes(1);
      });
    });
  });

  describe('Multi-Model Display Tests', () => {
    test('LiveProgressTable shows correct multi-model progress', async () => {
      const multiModelJob = {
        id: 'multi-model-job',
        status: 'completed',
        executions: [
          // US: 3/3 completed
          { id: 'exec-1', region: 'us-east', model_id: 'llama3.2-1b', status: 'completed', provider_id: 'modal-us-east' },
          { id: 'exec-2', region: 'us-east', model_id: 'qwen2.5-1.5b', status: 'completed', provider_id: 'modal-us-east' },
          { id: 'exec-3', region: 'us-east', model_id: 'mistral-7b', status: 'completed', provider_id: 'modal-us-east' },
          // EU: 0/3 completed (all failed)
          { id: 'exec-4', region: 'eu-west', model_id: 'llama3.2-1b', status: 'failed', provider_id: '' },
          { id: 'exec-5', region: 'eu-west', model_id: 'qwen2.5-1.5b', status: 'failed', provider_id: '' },
          { id: 'exec-6', region: 'eu-west', model_id: 'mistral-7b', status: 'failed', provider_id: '' },
          // ASIA: 2/3 completed
          { id: 'exec-7', region: 'asia-pacific', model_id: 'llama3.2-1b', status: 'completed', provider_id: 'modal-asia-pacific' },
          { id: 'exec-8', region: 'asia-pacific', model_id: 'qwen2.5-1.5b', status: 'completed', provider_id: 'modal-asia-pacific' },
          { id: 'exec-9', region: 'asia-pacific', model_id: 'mistral-7b', status: 'running', provider_id: 'modal-asia-pacific' }
        ]
      };

      render(
        <BrowserRouter>
          <LiveProgressTable
            activeJob={multiModelJob}
            selectedRegions={['US', 'EU', 'ASIA']}
            loadingActive={false}
            refetchActive={jest.fn()}
            activeJobId="multi-model-job"
            isCompleted={false}
            diffReady={false}
          />
        </BrowserRouter>
      );

      // Check multi-model progress display
      expect(screen.getByText('3/3 models')).toBeInTheDocument(); // US
      expect(screen.getByText('0/3 models')).toBeInTheDocument(); // EU
      expect(screen.getByText('2/3 models')).toBeInTheDocument(); // ASIA

      // Check status badges
      const usRow = screen.getByText('US').closest('div');
      expect(usRow).toHaveTextContent('completed');

      const euRow = screen.getByText('EU').closest('div');
      expect(euRow).toHaveTextContent('failed');

      const asiaRow = screen.getByText('ASIA').closest('div');
      expect(asiaRow).toHaveTextContent('running');
    });
  });
});

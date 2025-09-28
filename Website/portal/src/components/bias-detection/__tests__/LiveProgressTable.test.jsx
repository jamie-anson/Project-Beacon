import React from 'react';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import '@testing-library/jest-dom';
import LiveProgressTable from '../LiveProgressTable.jsx';

// Mock the crypto module to avoid wallet dependencies in tests
jest.mock('../../../lib/crypto.js', () => ({
  signJobSpecForAPI: jest.fn().mockResolvedValue({ signed: true })
}));

const renderWithRouter = (component) => {
  return render(
    <BrowserRouter>
      {component}
    </BrowserRouter>
  );
};

describe('LiveProgressTable', () => {
  const mockProps = {
    activeJob: null,
    selectedRegions: ['US', 'EU', 'ASIA'],
    loadingActive: false,
    refetchActive: jest.fn(),
    activeJobId: null,
    isCompleted: false,
    diffReady: false
  };

  describe('Region Filtering Links', () => {
    test('Answer links use correct database region names for filtering', () => {
      const jobWithExecutions = {
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

      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={jobWithExecutions}
          isCompleted={true}
        />
      );

      // Check that Answer links use correct database region names
      const answerLinks = screen.getAllByText('Answer');
      
      // US region should link to us-east
      const usLink = answerLinks.find(link => 
        link.closest('a')?.href.includes('region=us-east')
      );
      expect(usLink).toBeTruthy();

      // EU region should link to eu-west  
      const euLink = answerLinks.find(link =>
        link.closest('a')?.href.includes('region=eu-west')
      );
      expect(euLink).toBeTruthy();

      // ASIA region should link to asia-pacific
      const asiaLink = answerLinks.find(link =>
        link.closest('a')?.href.includes('region=asia-pacific')
      );
      expect(asiaLink).toBeTruthy();

      // Ensure NO links use the display names (US, EU, ASIA)
      answerLinks.forEach(link => {
        const href = link.closest('a')?.href || '';
        expect(href).not.toMatch(/region=US$/);
        expect(href).not.toMatch(/region=EU$/);
        expect(href).not.toMatch(/region=ASIA$/);
      });
    });

    test('mapRegionToDatabase function works correctly', () => {
      // We need to test the internal function, so we'll test through the component behavior
      const jobWithMixedRegions = {
        id: 'test-job-456',
        status: 'completed',
        executions: [
          { id: 'exec-1', region: 'us-east', status: 'completed', provider_id: 'test' },
          { id: 'exec-2', region: 'eu-west', status: 'completed', provider_id: 'test' },
          { id: 'exec-3', region: 'asia-pacific', status: 'completed', provider_id: 'test' }
        ]
      };

      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={jobWithMixedRegions}
          isCompleted={true}
        />
      );

      // Verify all Answer links have correct region mapping
      const links = screen.getAllByRole('link', { name: 'Answer' });
      expect(links).toHaveLength(3);

      const hrefs = links.map(link => link.getAttribute('href'));
      expect(hrefs.some(href => href.includes('region=us-east'))).toBe(true);
      expect(hrefs.some(href => href.includes('region=eu-west'))).toBe(true);
      expect(hrefs.some(href => href.includes('region=asia-pacific'))).toBe(true);
    });
  });

  describe('Multi-Model Execution Display', () => {
    test('shows correct model count for multi-model jobs', () => {
      const multiModelJob = {
        id: 'multi-model-job',
        status: 'completed',
        executions: [
          // US: 3 models completed
          { id: 'exec-1', region: 'us-east', model_id: 'llama3.2-1b', status: 'completed', provider_id: 'modal-us-east' },
          { id: 'exec-2', region: 'us-east', model_id: 'qwen2.5-1.5b', status: 'completed', provider_id: 'modal-us-east' },
          { id: 'exec-3', region: 'us-east', model_id: 'mistral-7b', status: 'completed', provider_id: 'modal-us-east' },
          // EU: 2 completed, 1 failed
          { id: 'exec-4', region: 'eu-west', model_id: 'llama3.2-1b', status: 'completed', provider_id: 'modal-eu-west' },
          { id: 'exec-5', region: 'eu-west', model_id: 'qwen2.5-1.5b', status: 'failed', provider_id: '' },
          { id: 'exec-6', region: 'eu-west', model_id: 'mistral-7b', status: 'completed', provider_id: 'modal-eu-west' },
          // ASIA: 0 completed, 3 failed
          { id: 'exec-7', region: 'asia-pacific', model_id: 'llama3.2-1b', status: 'failed', provider_id: '' },
          { id: 'exec-8', region: 'asia-pacific', model_id: 'qwen2.5-1.5b', status: 'failed', provider_id: '' },
          { id: 'exec-9', region: 'asia-pacific', model_id: 'mistral-7b', status: 'failed', provider_id: '' }
        ]
      };

      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={multiModelJob}
          isCompleted={true}
        />
      );

      // Check multi-model progress indicators
      expect(screen.getByText('3/3 models')).toBeInTheDocument(); // US region
      expect(screen.getByText('2/3 models')).toBeInTheDocument(); // EU region  
      expect(screen.getByText('0/3 models')).toBeInTheDocument(); // ASIA region

      // Check provider column shows model count
      expect(screen.getAllByText('3 models')).toHaveLength(3); // One for each region
    });

    test('shows correct status for multi-model regions', () => {
      const multiModelJob = {
        id: 'multi-model-status-test',
        status: 'completed',
        executions: [
          // US: All completed
          { id: 'exec-1', region: 'us-east', model_id: 'model-1', status: 'completed' },
          { id: 'exec-2', region: 'us-east', model_id: 'model-2', status: 'completed' },
          { id: 'exec-3', region: 'us-east', model_id: 'model-3', status: 'completed' },
          // EU: All failed
          { id: 'exec-4', region: 'eu-west', model_id: 'model-1', status: 'failed' },
          { id: 'exec-5', region: 'eu-west', model_id: 'model-2', status: 'failed' },
          { id: 'exec-6', region: 'eu-west', model_id: 'model-3', status: 'failed' },
          // ASIA: Mixed (should show as running/partial)
          { id: 'exec-7', region: 'asia-pacific', model_id: 'model-1', status: 'completed' },
          { id: 'exec-8', region: 'asia-pacific', model_id: 'model-2', status: 'running' },
          { id: 'exec-9', region: 'asia-pacific', model_id: 'model-3', status: 'failed' }
        ]
      };

      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={multiModelJob}
          isCompleted={false}
        />
      );

      // US should show completed (all models completed)
      const usRow = screen.getByText('US').closest('div');
      expect(usRow).toHaveTextContent('completed');

      // EU should show failed (all models failed)
      const euRow = screen.getByText('EU').closest('div');
      expect(euRow).toHaveTextContent('failed');

      // ASIA should show running (mixed status with running models)
      const asiaRow = screen.getByText('ASIA').closest('div');
      expect(asiaRow).toHaveTextContent('running');
    });
  });

  describe('Job Failure Detection', () => {
    test('shows failure alert for failed jobs', () => {
      const failedJob = {
        id: 'failed-job-123',
        status: 'failed',
        created_at: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
        executions: []
      };

      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={failedJob}
        />
      );

      // Should show failure alert
      expect(screen.getByText('Job Failed')).toBeInTheDocument();
      expect(screen.getByText(/Job failed with status: failed/)).toBeInTheDocument();
      expect(screen.getByText(/Try submitting a new job/)).toBeInTheDocument();
    });

    test('shows timeout alert for stuck jobs', () => {
      const stuckJob = {
        id: 'stuck-job-123',
        status: 'processing',
        created_at: new Date(Date.now() - 20 * 60 * 1000).toISOString(), // 20 minutes ago
        executions: [] // No executions created
      };

      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={stuckJob}
        />
      );

      // Should show timeout alert
      expect(screen.getByText('Job Timeout')).toBeInTheDocument();
      expect(screen.getByText(/Job has been running for 20 minutes/)).toBeInTheDocument();
      expect(screen.getByText(/The job may be stuck/)).toBeInTheDocument();
    });

    test('shows all regions as failed when job fails', () => {
      const failedJob = {
        id: 'failed-job-456',
        status: 'failed',
        created_at: new Date().toISOString(),
        executions: []
      };

      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={failedJob}
        />
      );

      // All regions should show failed status
      ['US', 'EU', 'ASIA'].forEach(region => {
        const regionRow = screen.getByText(region).closest('div');
        expect(regionRow).toHaveTextContent('failed');
      });
    });
  });

  describe('Completed Job Handling', () => {
    test('shows all regions as completed when job completes without execution records', () => {
      const completedJob = {
        id: 'completed-job-123',
        status: 'completed',
        created_at: new Date().toISOString(),
        executions: [] // No execution records but job marked complete
      };

      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={completedJob}
          isCompleted={true}
        />
      );

      // All regions should show completed status
      ['US', 'EU', 'ASIA'].forEach(region => {
        const regionRow = screen.getByText(region).closest('div');
        expect(regionRow).toHaveTextContent('completed');
      });

      // Should show Answer links even without execution records
      expect(screen.getAllByText('Answer')).toHaveLength(3);
    });

    test('shows correct provider info for completed jobs without executions', () => {
      const completedJob = {
        id: 'completed-job-456',
        status: 'completed',
        executions: []
      };

      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={completedJob}
          isCompleted={true}
        />
      );

      // Should show 'completed' as provider for regions without execution records
      expect(screen.getAllByText('completed')).toHaveLength(3); // One for each region
    });
  });

  describe('Progress Calculation', () => {
    test('calculates correct progress for multi-model jobs', () => {
      const multiModelJob = {
        id: 'progress-test-job',
        status: 'processing',
        executions: [
          // 3 completed, 2 failed, 1 running = 6 total
          { id: 'exec-1', status: 'completed' },
          { id: 'exec-2', status: 'completed' },
          { id: 'exec-3', status: 'completed' },
          { id: 'exec-4', status: 'failed' },
          { id: 'exec-5', status: 'failed' },
          { id: 'exec-6', status: 'running' }
        ]
      };

      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={multiModelJob}
          selectedRegions={['US', 'EU']} // 2 regions selected
        />
      );

      // Should show correct progress: 3 completed out of 2 regions = 150% (capped at 100%)
      expect(screen.getByText(/100%/)).toBeInTheDocument();
      expect(screen.getByText(/Completed: 3/)).toBeInTheDocument();
      expect(screen.getByText(/Failed: 2/)).toBeInTheDocument();
    });
  });
});

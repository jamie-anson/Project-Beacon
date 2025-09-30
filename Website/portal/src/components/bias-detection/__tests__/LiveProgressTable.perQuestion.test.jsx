import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import '@testing-library/jest-dom';
import LiveProgressTable from '../LiveProgressTable.jsx';

// Mock the crypto module to avoid wallet dependencies in tests
jest.mock('../../../lib/crypto.js', () => ({
  signJobSpecForAPI: jest.fn().mockResolvedValue({ signed: true })
}));

// Helper to render with router
const renderWithRouter = (component) => {
  return render(
    <BrowserRouter>
      {component}
    </BrowserRouter>
  );
};

// Base mock props (reused from existing tests)
const mockProps = {
  activeJob: null,
  selectedRegions: ['US', 'EU', 'ASIA'],
  loadingActive: false,
  refetchActive: jest.fn(),
  activeJobId: null,
  isCompleted: false,
  diffReady: false
};

// Helper function to create per-question job mocks
const createPerQuestionJob = ({ 
  id = 'test-job-123',
  status = 'completed',
  questionCount = 2,
  modelCount = 3,
  regionCount = 3,
  completedCount = null,
  runningCount = 0,
  failedCount = 0
}) => {
  const questions = ['math_basic', 'geography_basic'].slice(0, questionCount);
  const models = ['llama3.2-1b', 'mistral-7b', 'qwen2.5-1.5b'].slice(0, modelCount);
  const regions = ['us-east', 'eu-west', 'asia-pacific'].slice(0, regionCount);
  
  const executions = [];
  let execId = 1;
  
  const totalExecutions = questionCount * modelCount * regionCount;
  const actualCompleted = completedCount !== null ? completedCount : totalExecutions - runningCount - failedCount;
  
  let completedAdded = 0;
  let runningAdded = 0;
  let failedAdded = 0;
  
  for (const region of regions) {
    for (const model of models) {
      for (const question of questions) {
        let execStatus = 'completed';
        let classification = 'substantive';
        
        if (completedAdded < actualCompleted) {
          execStatus = 'completed';
          classification = model === 'mistral-7b' ? 'content_refusal' : 'substantive';
          completedAdded++;
        } else if (runningAdded < runningCount) {
          execStatus = 'running';
          classification = null;
          runningAdded++;
        } else if (failedAdded < failedCount) {
          execStatus = 'failed';
          classification = null;
          failedAdded++;
        }
        
        executions.push({
          id: `exec-${execId++}`,
          region,
          model_id: model,
          question_id: question,
          status: execStatus,
          response_classification: classification,
          is_content_refusal: classification === 'content_refusal',
          is_substantive: classification === 'substantive',
          created_at: new Date().toISOString(),
          provider_id: execStatus === 'completed' ? `modal-${region}` : ''
        });
      }
    }
  }
  
  return {
    id,
    status,
    created_at: new Date().toISOString(),
    executions
  };
};

describe('LiveProgressTable - Per-Question Execution', () => {
  
  describe('Per-Question Progress Calculation', () => {
    test('calculates total executions for per-question jobs', () => {
      const job = createPerQuestionJob({ 
        questionCount: 2, 
        modelCount: 3, 
        regionCount: 3 
      });
      
      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={job}
          isCompleted={true}
        />
      );
      
      // Should show 18 executions (2 questions × 3 models × 3 regions)
      expect(screen.getByText(/18 executions/)).toBeInTheDocument();
    });
    
    test('shows correct formula: questions × models × regions', () => {
      const job = createPerQuestionJob({ 
        questionCount: 2, 
        modelCount: 3, 
        regionCount: 3 
      });
      
      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={job}
          isCompleted={true}
        />
      );
      
      // Should show the breakdown
      expect(screen.getByText(/2 questions × 3 models × 3 regions/)).toBeInTheDocument();
    });
    
    test('falls back to region-based for legacy jobs without questions', () => {
      const legacyJob = {
        id: 'legacy-job',
        status: 'completed',
        executions: [
          { id: 'exec-1', region: 'us-east', model_id: 'llama3.2-1b', status: 'completed' },
          { id: 'exec-2', region: 'eu-west', model_id: 'llama3.2-1b', status: 'completed' },
          { id: 'exec-3', region: 'asia-pacific', model_id: 'llama3.2-1b', status: 'completed' }
        ]
      };
      
      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={legacyJob}
          isCompleted={true}
        />
      );
      
      // Should NOT show questions breakdown
      expect(screen.queryByText(/questions ×/)).not.toBeInTheDocument();
      // Should show region-based display
      expect(screen.getByText(/3 regions/)).toBeInTheDocument();
    });
  });
  
  describe('Job Stage Detection', () => {
    test('detects "creating" stage', () => {
      const job = {
        id: 'creating-job',
        status: 'created',
        executions: []
      };
      
      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={job}
        />
      );
      
      expect(screen.getByText(/Creating job.../)).toBeInTheDocument();
    });
    
    test('detects "queued" stage', () => {
      const job = {
        id: 'queued-job',
        status: 'queued',
        executions: []
      };
      
      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={job}
        />
      );
      
      expect(screen.getByText(/Job queued, waiting for worker.../)).toBeInTheDocument();
    });
    
    test('detects "spawning" stage', () => {
      const job = {
        id: 'spawning-job',
        status: 'processing',
        executions: []
      };
      
      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={job}
        />
      );
      
      expect(screen.getByText(/Starting executions.../)).toBeInTheDocument();
    });
    
    test('detects "running" stage', () => {
      const job = createPerQuestionJob({ 
        status: 'processing',
        completedCount: 10,
        runningCount: 5,
        failedCount: 0
      });
      
      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={job}
        />
      );
      
      expect(screen.getByText(/Executing questions.../)).toBeInTheDocument();
    });
    
    test('detects "completed" stage', () => {
      const job = createPerQuestionJob({ 
        status: 'completed'
      });
      
      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={job}
          isCompleted={true}
        />
      );
      
      expect(screen.getByText(/Job completed successfully!/)).toBeInTheDocument();
    });
  });
  
  describe('Expandable Rows', () => {
    test('rows are collapsed by default', () => {
      const job = createPerQuestionJob({});
      
      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={job}
          isCompleted={true}
        />
      );
      
      // Expanded content should not be visible
      expect(screen.queryByText(/Execution Details for US/)).not.toBeInTheDocument();
    });
    
    test('clicking row expands it', () => {
      const job = createPerQuestionJob({});
      
      renderWithRouter(
        <LiveProgressTable
          {...mockProps}
          activeJob={job}
          isCompleted={true}
        />
      );
      
      // Click the US row
      const usRow = screen.getByText('US').closest('div');
      fireEvent.click(usRow);
      
      // Expanded content should now be visible
      expect(screen.getByText(/Execution Details for US/)).toBeInTheDocument();
    });
  });
});

/**
 * Infinite Loop Regression Tests for Project Beacon Pages
 * 
 * This test suite specifically tests the pages that had infinite loop issues
 * to prevent regression and catch similar issues in the future.
 */

import { render, waitFor, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import React from 'react';

// Mock the API calls and hooks
jest.mock('../../state/useQuery.js', () => ({
  useQuery: jest.fn(() => ({
    data: null,
    loading: false,
    error: null,
    refetch: jest.fn()
  }))
}));

jest.mock('../../hooks/useBiasDetection.js', () => ({
  useBiasDetection: jest.fn(() => ({
    biasJobs: [],
    loading: false,
    jobListError: null,
    isSubmitting: false,
    activeJobId: '',
    selectedRegions: ['US'],
    selectedModel: 'llama3.2-1b',
    selectedModels: ['llama3.2-1b'],
    setActiveJobId: jest.fn(),
    setSelectedModel: jest.fn(),
    handleModelChange: jest.fn(),
    handleRegionToggle: jest.fn(),
    fetchBiasJobs: jest.fn(),
    onSubmitJob: jest.fn(),
    readSelectedQuestions: jest.fn(() => []),
    calculateEstimatedCost: jest.fn(() => ({ total: 0 }))
  }))
}));

jest.mock('../../hooks/useCrossRegionDiff.js', () => ({
  useCrossRegionDiff: jest.fn(() => ({
    loading: false,
    error: null,
    data: {
      models: [{ model_id: 'llama3.2-1b', regions: [] }],
      question: { id: 'test', text: 'Test question' }
    },
    job: { questions: ['test'] },
    usingMock: false,
    retry: jest.fn()
  }))
}));

jest.mock('../../hooks/useRecentDiffs.js', () => ({
  useRecentDiffs: jest.fn(() => ({
    data: []
  }))
}));

jest.mock('../../state/useWs.js', () => jest.fn(() => ({})));

jest.mock('../../lib/api/runner/questions.js', () => ({
  getQuestions: jest.fn(() => Promise.resolve({
    categories: {
      bias_detection: [
        { question_id: 'test1', question: 'Test question 1' },
        { question_id: 'test2', question: 'Test question 2' }
      ]
    }
  }))
}));

// Test wrapper with router
const TestWrapper = ({ children }) => (
  <BrowserRouter>
    {children}
  </BrowserRouter>
);

describe('Infinite Loop Regression Tests', () => {
  let consoleErrors = [];
  let consoleWarnings = [];
  let renderCount = 0;

  beforeEach(() => {
    consoleErrors = [];
    consoleWarnings = [];
    renderCount = 0;

    // Mock console to capture React warnings
    jest.spyOn(console, 'error').mockImplementation((...args) => {
      consoleErrors.push(args.join(' '));
    });
    
    jest.spyOn(console, 'warn').mockImplementation((...args) => {
      consoleWarnings.push(args.join(' '));
    });

    // Reset all mocks
    jest.clearAllMocks();
  });

  afterEach(() => {
    console.error.mockRestore();
    console.warn.mockRestore();
  });

  test('BiasDetection page does not have infinite loops', async () => {
    const BiasDetection = require('../BiasDetection.jsx').default;
    
    const { unmount } = render(
      <TestWrapper>
        <BiasDetection />
      </TestWrapper>
    );

    // Wait for potential infinite loops to manifest
    await waitFor(() => {
      // Check for the specific error we were seeing
      const hasMaxUpdateError = consoleErrors.some(error => 
        error.includes('Maximum update depth exceeded') ||
        error.includes('Warning: Maximum update depth exceeded')
      );
      
      expect(hasMaxUpdateError).toBe(false);
    }, { timeout: 3000 });

    // Additional check: no excessive re-renders
    const excessiveRenderWarnings = consoleWarnings.filter(warning =>
      warning.includes('re-render') || warning.includes('update depth')
    );
    expect(excessiveRenderWarnings).toHaveLength(0);

    unmount();
  });

  test('Questions page does not have infinite loops', async () => {
    const Questions = require('../Questions.jsx').default;
    
    const { unmount } = render(
      <TestWrapper>
        <Questions />
      </TestWrapper>
    );

    // Wait for potential infinite loops
    await waitFor(() => {
      const hasMaxUpdateError = consoleErrors.some(error => 
        error.includes('Maximum update depth exceeded')
      );
      
      expect(hasMaxUpdateError).toBe(false);
    }, { timeout: 3000 });

    unmount();
  });

  test('CrossRegionDiffPage does not have infinite loops', async () => {
    const CrossRegionDiffPage = require('../CrossRegionDiffPage.jsx').default;
    
    // Mock useParams to provide jobId
    jest.doMock('react-router-dom', () => ({
      ...jest.requireActual('react-router-dom'),
      useParams: () => ({ jobId: 'test-job-123' }),
      useNavigate: () => jest.fn()
    }));

    const { unmount } = render(
      <TestWrapper>
        <CrossRegionDiffPage />
      </TestWrapper>
    );

    // Wait for potential infinite loops
    await waitFor(() => {
      const hasMaxUpdateError = consoleErrors.some(error => 
        error.includes('Maximum update depth exceeded')
      );
      
      expect(hasMaxUpdateError).toBe(false);
    }, { timeout: 3000 });

    unmount();
  });

  test('detects useEffect dependency issues in component code', () => {
    // Static analysis test - check for common patterns
    const biasDetectionCode = require('fs').readFileSync(
      require.resolve('../BiasDetection.jsx'), 
      'utf8'
    );

    // Should not have object literals in useEffect deps
    const hasObjectLiteralDeps = /useEffect\([^}]+\},\s*\[[^}]*\{[^}]*\}[^\]]*\]/.test(biasDetectionCode);
    expect(hasObjectLiteralDeps).toBe(false);

    // Should use useMemo for complex calculations
    const hasUseMemo = biasDetectionCode.includes('useMemo');
    expect(hasUseMemo).toBe(true);
  });

  test('validates all pages have proper memoization', () => {
    const pageFiles = [
      '../BiasDetection.jsx',
      '../Questions.jsx', 
      '../CrossRegionDiffPage.jsx'
    ];

    pageFiles.forEach(filePath => {
      const code = require('fs').readFileSync(require.resolve(filePath), 'utf8');
      
      // If component uses useEffect, it should also use useMemo or useCallback for complex deps
      if (code.includes('useEffect')) {
        const hasComplexDeps = /useEffect\([^}]+\},\s*\[[^}]*[\{\[]/.test(code);
        if (hasComplexDeps) {
          const hasMemoization = code.includes('useMemo') || code.includes('useCallback');
          expect(hasMemoization).toBe(true);
        }
      }
    });
  });

  test('performance: components render in reasonable time', async () => {
    const BiasDetection = require('../BiasDetection.jsx').default;
    
    const startTime = performance.now();
    
    const { unmount } = render(
      <TestWrapper>
        <BiasDetection />
      </TestWrapper>
    );
    
    const endTime = performance.now();
    const renderTime = endTime - startTime;
    
    // Should render in under 100ms
    expect(renderTime).toBeLessThan(100);
    
    unmount();
  });
});

/**
 * Custom hook to detect infinite loops in development
 */
export const useInfiniteLoopDetector = (componentName) => {
  const renderCountRef = React.useRef(0);
  const lastRenderTime = React.useRef(Date.now());
  
  React.useEffect(() => {
    renderCountRef.current += 1;
    const now = Date.now();
    const timeSinceLastRender = now - lastRenderTime.current;
    
    // If we've rendered more than 50 times in 1 second, likely infinite loop
    if (renderCountRef.current > 50 && timeSinceLastRender < 1000) {
      console.error(`🚨 Potential infinite loop detected in ${componentName}:`, {
        renderCount: renderCountRef.current,
        timeWindow: timeSinceLastRender
      });
    }
    
    // Reset counter every 5 seconds
    if (timeSinceLastRender > 5000) {
      renderCountRef.current = 0;
      lastRenderTime.current = now;
    }
  });
  
  return renderCountRef.current;
};

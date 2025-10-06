/**
 * Unit tests for executionUtils
 */

import {
  extractExecText,
  prefillFromExecutions,
  truncateMiddle,
  timeAgo,
  getFailureDetails
} from '../executionUtils';

describe('executionUtils', () => {
  describe('extractExecText', () => {
    it('should extract response string', () => {
      const exec = { output: { response: 'Test response' } };
      expect(extractExecText(exec)).toBe('Test response');
    });

    it('should truncate long responses', () => {
      const longResponse = 'a'.repeat(250);
      const exec = { output: { response: longResponse } };
      const result = extractExecText(exec);
      
      expect(result).toHaveLength(233); // 200 + '... (click to view full response)' = 233
      expect(result).toContain('...');
    });

    it('should extract from responses array', () => {
      const exec = { 
        output: { 
          responses: [
            { response: 'First response' },
            { response: 'Second response' }
          ] 
        } 
      };
      expect(extractExecText(exec)).toBe('First response');
    });

    it('should extract from answer field', () => {
      const exec = { output: { responses: [{ answer: 'Answer text' }] } };
      expect(extractExecText(exec)).toBe('Answer text');
    });

    it('should extract from text_output', () => {
      const exec = { output: { text_output: 'Text output' } };
      expect(extractExecText(exec)).toBe('Text output');
    });

    it('should extract from output field', () => {
      const exec = { output: { output: 'Output text' } };
      expect(extractExecText(exec)).toBe('Output text');
    });

    it('should return empty string for null execution', () => {
      expect(extractExecText(null)).toBe('');
    });

    it('should return empty string for execution without output', () => {
      expect(extractExecText({})).toBe('');
    });

    it('should use result field if output is not available', () => {
      const exec = { result: { response: 'Result response' } };
      expect(extractExecText(exec)).toBe('Result response');
    });
  });

  describe('prefillFromExecutions', () => {
    it('should prefill from completed executions', () => {
      const activeJob = {
        executions: [
          { status: 'completed', region: 'us-east', output: { response: 'US response' } },
          { status: 'completed', region: 'eu-west', output: { response: 'EU response' } }
        ]
      };
      
      const setters = {
        setARegion: jest.fn(),
        setBRegion: jest.fn(),
        setAText: jest.fn(),
        setBText: jest.fn()
      };
      
      prefillFromExecutions(activeJob, setters);
      
      expect(setters.setARegion).toHaveBeenCalled();
      expect(setters.setBRegion).toHaveBeenCalled();
      expect(setters.setAText).toHaveBeenCalled();
      expect(setters.setBText).toHaveBeenCalled();
      // Both regions should be set (normalized to database format)
      expect(setters.setARegion.mock.calls[0][0]).toBeTruthy();
      expect(setters.setBRegion.mock.calls[0][0]).toBeTruthy();
    });

    it('should prefer US and EU regions', () => {
      const activeJob = {
        executions: [
          { status: 'completed', region: 'asia-pacific', output: { response: 'Asia response' } },
          { status: 'completed', region: 'us-east', output: { response: 'US response' } },
          { status: 'completed', region: 'eu-west', output: { response: 'EU response' } }
        ]
      };
      
      const setters = {
        setARegion: jest.fn(),
        setBRegion: jest.fn(),
        setAText: jest.fn(),
        setBText: jest.fn()
      };
      
      prefillFromExecutions(activeJob, setters);
      
      expect(setters.setAText).toHaveBeenCalledWith('US response');
      expect(setters.setBText).toHaveBeenCalledWith('EU response');
    });

    it('should handle error when no executions', () => {
      const activeJob = { executions: [] };
      const setters = { setError: jest.fn() };
      
      prefillFromExecutions(activeJob, setters);
      
      expect(setters.setError).toHaveBeenCalledWith('No executions available to prefill');
    });

    it('should fallback to any executions if US/EU not available', () => {
      const activeJob = {
        executions: [
          { status: 'completed', region: 'asia-pacific', output: { response: 'Asia response' } },
          { status: 'completed', region: 'canada', output: { response: 'Canada response' } }
        ]
      };
      
      const setters = {
        setARegion: jest.fn(),
        setBRegion: jest.fn(),
        setAText: jest.fn(),
        setBText: jest.fn()
      };
      
      prefillFromExecutions(activeJob, setters);
      
      expect(setters.setAText).toHaveBeenCalled();
      expect(setters.setBText).toHaveBeenCalled();
    });
  });

  describe('truncateMiddle', () => {
    it('should truncate long strings in the middle', () => {
      const str = 'abcdefghijklmnop';
      expect(truncateMiddle(str, 6, 4)).toBe('abcdef…mnop');
    });

    it('should not truncate short strings', () => {
      const str = 'short';
      expect(truncateMiddle(str, 6, 4)).toBe('short');
    });

    it('should return — for null', () => {
      expect(truncateMiddle(null)).toBe('—');
    });

    it('should return — for non-string', () => {
      expect(truncateMiddle(123)).toBe('—');
    });

    it('should use default head and tail values', () => {
      const str = 'abcdefghijklmnop';
      expect(truncateMiddle(str)).toBe('abcdef…mnop');
    });
  });

  describe('timeAgo', () => {
    it('should return seconds ago for recent timestamps', () => {
      const ts = new Date(Date.now() - 30000).toISOString(); // 30 seconds ago
      expect(timeAgo(ts)).toBe('30s ago');
    });

    it('should return hours ago for older timestamps', () => {
      const ts = new Date(Date.now() - 7200000).toISOString(); // 2 hours ago
      expect(timeAgo(ts)).toBe('2h ago');
    });

    it('should return days ago for very old timestamps', () => {
      const ts = new Date(Date.now() - 172800000).toISOString(); // 2 days ago
      expect(timeAgo(ts)).toBe('2d ago');
    });

    it('should return empty string for null', () => {
      expect(timeAgo(null)).toBe('');
    });

    it('should handle invalid timestamps gracefully', () => {
      expect(timeAgo('invalid')).toBe('invalid');
    });

    it('should handle timestamp numbers', () => {
      const ts = Date.now() - 60000; // 1 minute ago
      const result = timeAgo(ts);
      expect(result).toMatch(/\d+h ago/);
    });
  });

  describe('getFailureDetails', () => {
    it('should extract failure message from output.failure object', () => {
      const exec = {
        output: {
          failure: {
            message: 'Connection timeout',
            code: 'TIMEOUT',
            stage: 'network'
          }
        }
      };
      
      const result = getFailureDetails(exec);
      
      expect(result.message).toBe('Connection timeout');
      expect(result.code).toBe('TIMEOUT');
      expect(result.stage).toBe('network');
    });

    it('should extract failure message from failure field', () => {
      const exec = {
        failure: {
          message: 'GPU error',
          code: 'GPU_FAIL'
        }
      };
      
      const result = getFailureDetails(exec);
      
      expect(result.message).toBe('GPU error');
      expect(result.code).toBe('GPU_FAIL');
    });

    it('should handle string failure messages', () => {
      const exec = {
        failure: 'Simple error message'
      };
      
      const result = getFailureDetails(exec);
      
      expect(result.message).toBe('Simple error message');
      expect(result.code).toBeNull();
    });

    it('should fallback to error field', () => {
      const exec = {
        error: 'Error message'
      };
      
      const result = getFailureDetails(exec);
      
      expect(result.message).toBe('Error message');
    });

    it('should fallback to failure_reason field', () => {
      const exec = {
        failure_reason: 'Failure reason'
      };
      
      const result = getFailureDetails(exec);
      
      expect(result.message).toBe('Failure reason');
    });

    it('should return null for execution without failures', () => {
      const exec = { status: 'completed' };
      const result = getFailureDetails(exec);
      
      expect(result).toBeNull();
    });

    it('should return null for null execution', () => {
      expect(getFailureDetails(null)).toBeNull();
    });
  });
});

/**
 * useEffect Infinite Loop Detection Tests
 * 
 * This test suite detects common patterns that cause infinite re-renders
 * in React components, specifically targeting useEffect dependency issues.
 */

import { render, act, waitFor } from '@testing-library/react';
import React, { useState, useEffect, useMemo } from 'react';

// Mock console to capture warnings
const originalConsoleWarn = console.warn;
const originalConsoleError = console.error;

describe('useEffect Infinite Loop Detection', () => {
  let consoleWarnings = [];
  let consoleErrors = [];

  beforeEach(() => {
    consoleWarnings = [];
    consoleErrors = [];
    
    console.warn = (...args) => {
      consoleWarnings.push(args.join(' '));
      originalConsoleWarn(...args);
    };
    
    console.error = (...args) => {
      consoleErrors.push(args.join(' '));
      originalConsoleError(...args);
    };
  });

  afterEach(() => {
    console.warn = originalConsoleWarn;
    console.error = originalConsoleError;
  });

  test('detects Maximum update depth exceeded warnings', async () => {
    const ProblematicComponent = () => {
      const [count, setCount] = useState(0);
      const [data, setData] = useState({ value: 0 });

      // BAD: This will cause infinite loop
      useEffect(() => {
        setData({ value: count + 1 });
      }, [data]); // data changes every render!

      return <div>Count: {count}</div>;
    };

    // Render and wait for warnings
    const { unmount } = render(<ProblematicComponent />);
    
    // Wait for React to detect the infinite loop
    await waitFor(() => {
      const hasMaxUpdateWarning = consoleErrors.some(error => 
        error.includes('Maximum update depth exceeded') ||
        error.includes('Too many re-renders')
      );
      expect(hasMaxUpdateWarning).toBe(true);
    }, { timeout: 5000 });

    unmount();
  });

  test('detects object dependency issues', async () => {
    const ComponentWithObjectDeps = () => {
      const [state, setState] = useState({ count: 0 });
      const [trigger, setTrigger] = useState(0);

      // BAD: Object recreated every render
      const config = { interval: 1000, enabled: true };

      useEffect(() => {
        setTrigger(prev => prev + 1);
      }, [config]); // config is new object every render

      return <div>Trigger: {trigger}</div>;
    };

    const { unmount } = render(<ComponentWithObjectDeps />);
    
    await waitFor(() => {
      const hasMaxUpdateWarning = consoleErrors.some(error => 
        error.includes('Maximum update depth exceeded')
      );
      expect(hasMaxUpdateWarning).toBe(true);
    }, { timeout: 3000 });

    unmount();
  });

  test('detects array dependency issues', async () => {
    const ComponentWithArrayDeps = () => {
      const [items, setItems] = useState([]);
      const [count, setCount] = useState(0);

      // BAD: Array recreated every render
      const filters = ['active', 'pending'];

      useEffect(() => {
        setCount(prev => prev + 1);
      }, [filters]); // filters is new array every render

      return <div>Count: {count}</div>;
    };

    const { unmount } = render(<ComponentWithArrayDeps />);
    
    await waitFor(() => {
      const hasMaxUpdateWarning = consoleErrors.some(error => 
        error.includes('Maximum update depth exceeded')
      );
      expect(hasMaxUpdateWarning).toBe(true);
    }, { timeout: 3000 });

    unmount();
  });

  test('validates proper memoization prevents loops', async () => {
    const WellBehavedComponent = () => {
      const [count, setCount] = useState(0);
      const [data, setData] = useState({ value: 0 });

      // GOOD: Memoized object
      const config = useMemo(() => ({ 
        interval: 1000, 
        enabled: true 
      }), []); // Empty deps - stable reference

      // GOOD: Proper dependencies
      useEffect(() => {
        // Only runs when count changes, not on every render
        if (count < 5) {
          const timer = setTimeout(() => setCount(prev => prev + 1), 100);
          return () => clearTimeout(timer);
        }
      }, [count]); // count is primitive, safe dependency

      return <div>Count: {count}</div>;
    };

    const { unmount } = render(<WellBehavedComponent />);
    
    // Wait a bit to ensure no warnings
    await new Promise(resolve => setTimeout(resolve, 1000));
    
    const hasMaxUpdateWarning = consoleErrors.some(error => 
      error.includes('Maximum update depth exceeded')
    );
    expect(hasMaxUpdateWarning).toBe(false);

    unmount();
  });

  test('detects function dependency issues', async () => {
    const ComponentWithFunctionDeps = () => {
      const [value, setValue] = useState(0);

      // BAD: Function recreated every render
      const handleUpdate = () => {
        setValue(prev => prev + 1);
      };

      useEffect(() => {
        handleUpdate();
      }, [handleUpdate]); // handleUpdate is new function every render

      return <div>Value: {value}</div>;
    };

    const { unmount } = render(<ComponentWithFunctionDeps />);
    
    await waitFor(() => {
      const hasMaxUpdateWarning = consoleErrors.some(error => 
        error.includes('Maximum update depth exceeded')
      );
      expect(hasMaxUpdateWarning).toBe(true);
    }, { timeout: 3000 });

    unmount();
  });

  test('detects WebSocket hook infinite loops', async () => {
    const ComponentWithBadWebSocket = () => {
      const [count, setCount] = useState(0);

      // BAD: Function recreated every render causing useEffect loop
      const wsEnabled = () => false;

      useEffect(() => {
        setCount(prev => prev + 1);
      }, [wsEnabled]); // wsEnabled function recreated every render

      return <div>Count: {count}</div>;
    };

    const { unmount } = render(<ComponentWithBadWebSocket />);
    
    await waitFor(() => {
      const hasMaxUpdateWarning = consoleErrors.some(error => 
        error.includes('Maximum update depth exceeded')
      );
      expect(hasMaxUpdateWarning).toBe(true);
    }, { timeout: 3000 });

    unmount();
  });

  test('detects circular dependency loops (useCallback in useEffect deps)', async () => {
    const ComponentWithCircularDeps = () => {
      const [enabled, setEnabled] = useState(true);
      const [count, setCount] = useState(0);

      // BAD: useCallback depends on enabled, useEffect depends on both
      const connect = useCallback(() => {
        if (enabled) {
          setCount(prev => prev + 1);
        }
      }, [enabled]);

      useEffect(() => {
        connect();
      }, [connect, enabled]); // Circular: connect depends on enabled, effect depends on both

      return <div>Count: {count}</div>;
    };

    const { unmount } = render(<ComponentWithCircularDeps />);
    
    await waitFor(() => {
      const hasMaxUpdateWarning = consoleErrors.some(error => 
        error.includes('Maximum update depth exceeded')
      );
      expect(hasMaxUpdateWarning).toBe(true);
    }, { timeout: 3000 });

    unmount();
  });

  test('validates proper WebSocket memoization prevents loops', async () => {
    const WellBehavedWebSocketComponent = () => {
      const [count, setCount] = useState(0);

      // GOOD: Memoized function
      const wsEnabled = useMemo(() => false, []);

      useEffect(() => {
        // Only runs once due to stable wsEnabled reference
        if (count < 3) {
          const timer = setTimeout(() => setCount(prev => prev + 1), 100);
          return () => clearTimeout(timer);
        }
      }, [wsEnabled, count]); // wsEnabled is stable, count changes predictably

      return <div>Count: {count}</div>;
    };

    const { unmount } = render(<WellBehavedWebSocketComponent />);
    
    // Wait a bit to ensure no warnings
    await new Promise(resolve => setTimeout(resolve, 1000));
    
    const hasMaxUpdateWarning = consoleErrors.some(error => 
      error.includes('Maximum update depth exceeded')
    );
    expect(hasMaxUpdateWarning).toBe(false);

    unmount();
  });
});

/**
 * Static Analysis Helper Functions
 * These can be used in other tests to check for common patterns
 */

export const useEffectPatternCheckers = {
  /**
   * Check if a component file has potential useEffect issues
   */
  checkFileForUseEffectIssues: (fileContent) => {
    const issues = [];
    
    // Pattern 1: useEffect with object/array literals in deps
    const objectLiteralPattern = /useEffect\([^}]+\},\s*\[[^}]*\{[^}]*\}[^\]]*\]/g;
    if (objectLiteralPattern.test(fileContent)) {
      issues.push('Object literal in useEffect dependency array');
    }
    
    // Pattern 2: useEffect with function calls in deps
    const functionCallPattern = /useEffect\([^}]+\},\s*\[[^}]*\([^}]*\)[^\]]*\]/g;
    if (functionCallPattern.test(fileContent)) {
      issues.push('Function call in useEffect dependency array');
    }
    
    // Pattern 3: setState inside useEffect with state in deps
    const setStateInEffectPattern = /useEffect\([^}]*set\w+\([^}]+\},\s*\[[^}]*\w+[^\]]*\]/g;
    if (setStateInEffectPattern.test(fileContent)) {
      issues.push('setState in useEffect with state variable in dependencies');
    }
    
    // Pattern 4: WebSocket function dependencies (like wsEnabled())
    const wsFunctionPattern = /useEffect\([^}]+\},\s*\[[^}]*wsEnabled[^\]]*\]/g;
    if (wsFunctionPattern.test(fileContent)) {
      issues.push('WebSocket function in useEffect dependencies (should be memoized)');
    }
    
    // Pattern 5: Circular dependencies (useCallback/useMemo in useEffect deps)
    const circularDepPattern = /useEffect\([^}]+\},\s*\[[^}]*(?:connect|callback|handler)[^}]*\]/g;
    if (circularDepPattern.test(fileContent)) {
      issues.push('Potential circular dependency: useCallback/function in useEffect dependencies');
    }
    
    return issues;
  },

  /**
   * Check for missing useMemo/useCallback for complex dependencies
   */
  checkForMissingMemoization: (fileContent) => {
    const issues = [];
    
    // Look for object/array creation without useMemo
    const unmemoizedObjectPattern = /const\s+\w+\s*=\s*\{[^}]*\}[^;]*;[\s\S]*useEffect/g;
    if (unmemoizedObjectPattern.test(fileContent)) {
      issues.push('Object created without useMemo before useEffect');
    }
    
    const unmemoizedArrayPattern = /const\s+\w+\s*=\s*\[[^\]]*\][^;]*;[\s\S]*useEffect/g;
    if (unmemoizedArrayPattern.test(fileContent)) {
      issues.push('Array created without useMemo before useEffect');
    }
    
    return issues;
  }
};

/**
 * Integration test helper for Project Beacon components
 */
export const testProjectBeaconComponent = async (Component, props = {}) => {
  const consoleWarnings = [];
  const consoleErrors = [];
  
  const originalWarn = console.warn;
  const originalError = console.error;
  
  console.warn = (...args) => consoleWarnings.push(args.join(' '));
  console.error = (...args) => consoleErrors.push(args.join(' '));
  
  try {
    const { unmount } = render(<Component {...props} />);
    
    // Wait for potential infinite loops to manifest
    await new Promise(resolve => setTimeout(resolve, 2000));
    
    const hasInfiniteLoop = consoleErrors.some(error => 
      error.includes('Maximum update depth exceeded') ||
      error.includes('Too many re-renders')
    );
    
    unmount();
    
    return {
      hasInfiniteLoop,
      warnings: consoleWarnings,
      errors: consoleErrors
    };
  } finally {
    console.warn = originalWarn;
    console.error = originalError;
  }
};

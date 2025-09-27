/**
 * Development Hook: Infinite Loop Detector
 * 
 * This hook detects potential infinite re-render loops in React components
 * and logs warnings to help developers identify problematic useEffect patterns.
 * 
 * Usage:
 *   import { useInfiniteLoopDetector } from '../hooks/useInfiniteLoopDetector.js';
 *   
 *   function MyComponent() {
 *     useInfiniteLoopDetector('MyComponent'); // Add this line
 *     // ... rest of component
 *   }
 */

import { useRef, useEffect } from 'react';

export const useInfiniteLoopDetector = (componentName, options = {}) => {
  const {
    maxRendersPerSecond = 50,
    warningThreshold = 20,
    resetInterval = 5000,
    enabled = process.env.NODE_ENV === 'development'
  } = options;

  const renderCountRef = useRef(0);
  const lastResetTime = useRef(Date.now());
  const warningShownRef = useRef(false);

  if (!enabled) return;

  // Increment render count on every render
  renderCountRef.current += 1;
  
  const now = Date.now();
  const timeSinceReset = now - lastResetTime.current;
  const rendersPerSecond = (renderCountRef.current / timeSinceReset) * 1000;

  // Show warning if we're rendering too frequently
  if (renderCountRef.current > warningThreshold && !warningShownRef.current) {
    console.warn(`⚠️ ${componentName}: High render count detected`, {
      renderCount: renderCountRef.current,
      timeWindow: `${timeSinceReset}ms`,
      rendersPerSecond: Math.round(rendersPerSecond)
    });
    warningShownRef.current = true;
  }

  // Show error if we've definitely hit an infinite loop
  if (rendersPerSecond > maxRendersPerSecond) {
    console.error(`🚨 ${componentName}: INFINITE LOOP DETECTED!`, {
      renderCount: renderCountRef.current,
      timeWindow: `${timeSinceReset}ms`,
      rendersPerSecond: Math.round(rendersPerSecond),
      suggestion: 'Check useEffect dependencies for objects/arrays that recreate on every render'
    });
  }

  // Reset counter periodically
  useEffect(() => {
    const timer = setInterval(() => {
      renderCountRef.current = 0;
      lastResetTime.current = Date.now();
      warningShownRef.current = false;
    }, resetInterval);

    return () => clearInterval(timer);
  }, [resetInterval]);

  return {
    renderCount: renderCountRef.current,
    rendersPerSecond: Math.round(rendersPerSecond)
  };
};

/**
 * Hook to detect specific useEffect patterns that cause infinite loops
 */
export const useEffectPatternDetector = (componentName, enabled = process.env.NODE_ENV === 'development') => {
  const effectCallsRef = useRef(new Map());

  if (!enabled) return () => {};

  return (effectName, dependencies) => {
    const key = effectName || 'unnamed-effect';
    const now = Date.now();
    
    if (!effectCallsRef.current.has(key)) {
      effectCallsRef.current.set(key, { count: 0, lastCall: now, deps: [] });
    }
    
    const effectData = effectCallsRef.current.get(key);
    effectData.count += 1;
    
    const timeSinceLastCall = now - effectData.lastCall;
    effectData.lastCall = now;
    
    // Check if dependencies changed (shallow comparison)
    const depsChanged = JSON.stringify(dependencies) !== JSON.stringify(effectData.deps);
    effectData.deps = dependencies;
    
    // Warn if effect is called too frequently
    if (effectData.count > 10 && timeSinceLastCall < 100) {
      console.warn(`⚠️ ${componentName}: useEffect "${key}" called ${effectData.count} times rapidly`, {
        timeSinceLastCall,
        depsChanged,
        currentDeps: dependencies
      });
    }
    
    // Error if definitely infinite
    if (effectData.count > 50 && timeSinceLastCall < 50) {
      console.error(`🚨 ${componentName}: useEffect "${key}" INFINITE LOOP!`, {
        callCount: effectData.count,
        dependencies,
        suggestion: 'Dependencies likely changing on every render - use useMemo/useCallback'
      });
    }
  };
};

/**
 * Development-only wrapper for useEffect that adds loop detection
 */
export const useEffectWithLoopDetection = (effect, deps, effectName, componentName) => {
  const detector = useEffectPatternDetector(componentName);
  
  useEffect(() => {
    detector(effectName, deps);
    return effect();
  }, deps);
};

/**
 * Utility to analyze component for potential infinite loop patterns
 */
export const analyzeComponentForLoopRisks = (componentCode) => {
  const risks = [];
  
  // Pattern 1: Object/array literals in useEffect deps
  if (/useEffect\([^}]+\},\s*\[[^}]*[\{\[]/.test(componentCode)) {
    risks.push({
      type: 'object-literal-deps',
      severity: 'high',
      message: 'Object or array literal in useEffect dependencies',
      fix: 'Use useMemo or move object/array outside component'
    });
  }
  
  // Pattern 2: setState in useEffect with state in deps
  if (/useEffect\([^}]*set\w+\([^}]+\},\s*\[[^}]*\w+/.test(componentCode)) {
    risks.push({
      type: 'state-in-deps',
      severity: 'high', 
      message: 'setState in useEffect with state variable in dependencies',
      fix: 'Remove state variable from dependencies or use functional update'
    });
  }
  
  // Pattern 3: Function calls in deps
  if (/useEffect\([^}]+\},\s*\[[^}]*\w+\([^}]*\)/.test(componentCode)) {
    risks.push({
      type: 'function-call-deps',
      severity: 'medium',
      message: 'Function call in useEffect dependencies',
      fix: 'Use useCallback for function or move call inside effect'
    });
  }
  
  // Pattern 4: Missing useMemo for complex objects
  if (componentCode.includes('useEffect') && !componentCode.includes('useMemo')) {
    const hasComplexObjects = /const\s+\w+\s*=\s*\{[^}]+\}/.test(componentCode);
    if (hasComplexObjects) {
      risks.push({
        type: 'missing-memoization',
        severity: 'medium',
        message: 'Complex objects created without useMemo',
        fix: 'Use useMemo for objects used in useEffect dependencies'
      });
    }
  }
  
  return risks;
};

export default useInfiniteLoopDetector;

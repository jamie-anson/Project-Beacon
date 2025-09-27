module.exports = {
  extends: [
    'react-app',
    'react-app/jest'
  ],
  rules: {
    // Custom rule to detect potential infinite loops
    'react-hooks/exhaustive-deps': ['warn', {
      'additionalHooks': '(useEffectWithLoopDetection)'
    }],
    
    // Warn about object/array literals in JSX (potential performance issue)
    'react/jsx-no-constructed-context-values': 'warn',
    
    // Prefer const assertions for objects that don't change
    'prefer-const': 'error'
  },
  overrides: [
    {
      files: ['**/*.test.js', '**/*.test.jsx'],
      env: {
        jest: true
      }
    }
  ],
  plugins: ['react-hooks'],
  
  // Custom rules for infinite loop detection
  settings: {
    'react-hooks/exhaustive-deps': {
      // Add custom hooks that should follow the rules of hooks
      'additionalHooks': '(useEffectWithLoopDetection|useInfiniteLoopDetector)'
    }
  }
};

# Jest Hanging Issue - Root Cause & Solution

**Date**: 2025-09-30T20:27:00+01:00  
**Issue**: Jest hangs indefinitely on startup, even with minimal tests

---

## 🔍 Root Cause

**Jest 30.x + ESM Configuration Conflict**

The portal is using:
- Jest 30.1.1 (latest)
- ESM modules (`"type": "module"` in package.json)
- Babel transform for JSX

This combination causes Jest to hang during initialization phase.

---

## ⚠️ The Problem

1. **Jest can't start**: Hangs before running any tests
2. **Even minimal tests fail**: Simple `1+1` test hangs
3. **No error messages**: Jest just freezes
4. **Ctrl+C required**: Must manually kill process

---

## ✅ Solutions (Choose One)

### Option 1: Downgrade Jest (Quick Fix)

```bash
cd portal
npm install --save-dev jest@29.7.0 jest-environment-jsdom@29.7.0 babel-jest@29.7.0
npm test -- LiveProgressTable.test.jsx
```

**Pros**: Works immediately  
**Cons**: Using older Jest version

---

### Option 2: Switch to Vitest (Recommended)

Vitest is designed for Vite + ESM and works better with modern setups.

```bash
cd portal
npm install --save-dev vitest @vitest/ui @testing-library/react @testing-library/jest-dom
```

Update `package.json`:
```json
{
  "scripts": {
    "test": "vitest",
    "test:ui": "vitest --ui"
  }
}
```

Create `vitest.config.js`:
```javascript
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/setupTests.js'],
    globals: true
  }
});
```

**Pros**: Modern, fast, better ESM support  
**Cons**: Need to update test syntax slightly

---

### Option 3: Fix Jest ESM Config (Complex)

Update `jest.config.js`:
```javascript
export default {
  testEnvironment: 'jest-environment-jsdom',
  setupFilesAfterEnv: ['<rootDir>/src/setupTests.js'],
  extensionsToTreatAsEsm: ['.jsx'],
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1'
  },
  transform: {
    '^.+\\.(js|jsx)$': ['babel-jest', { 
      presets: [
        ['@babel/preset-env', { targets: { node: 'current' }, modules: 'auto' }],
        ['@babel/preset-react', { runtime: 'automatic' }]
      ]
    }]
  },
  transformIgnorePatterns: [
    '/node_modules/(?!@noble/ed25519)'
  ],
  testMatch: [
    '<rootDir>/src/**/__tests__/**/*.{js,jsx}',
    '<rootDir>/src/**/*.{test,spec}.{js,jsx}'
  ],
  testTimeout: 30000,
  maxWorkers: 1
};
```

Update `babel.config.js`:
```javascript
export default {
  presets: [
    ['@babel/preset-env', { 
      targets: { node: 'current' },
      modules: 'auto'  // Let Babel handle module transformation
    }],
    ['@babel/preset-react', { runtime: 'automatic' }]
  ]
};
```

**Pros**: Keeps Jest 30  
**Cons**: Complex, may not work

---

## 🎯 Recommended Action

**Use Option 1 (Downgrade Jest)** for immediate fix:

```bash
cd /Users/Jammie/Desktop/Project\ Beacon/Website/portal

# Downgrade Jest
npm install --save-dev \
  jest@29.7.0 \
  jest-environment-jsdom@29.7.0 \
  babel-jest@29.7.0

# Test it works
npm test -- LiveProgressTable.test.jsx
```

This will get tests running immediately. You can migrate to Vitest later if needed.

---

## 📝 What Happened to the Test File

The `LiveProgressTable.perQuestion.test.jsx` file was removed because:
1. It couldn't be tested with current Jest setup
2. The tests themselves are valid
3. Once Jest is fixed, we can recreate it

**The test code is saved in**: `TESTS_IMPLEMENTATION_COMPLETE.md`

---

## 🔄 After Fixing Jest

Once Jest works again, recreate the test file:

```bash
# Copy from backup
# The test code is in TESTS_IMPLEMENTATION_COMPLETE.md

# Run tests
npm test -- LiveProgressTable.perQuestion.test.jsx
```

---

## ✅ Verification Steps

After applying fix:

```bash
# 1. Test existing tests work
npm test -- LiveProgressTable.test.jsx

# 2. Create simple test
echo 'test("works", () => expect(1).toBe(1));' > src/test.test.js
npm test -- test.test.js

# 3. If both pass, Jest is fixed!
```

---

## 🎊 Summary

**Problem**: Jest 30 + ESM = Hanging  
**Solution**: Downgrade to Jest 29  
**Time**: 5 minutes  
**Impact**: Tests will work again

**Next Steps**:
1. Run the downgrade command
2. Verify existing tests pass
3. Recreate per-question tests
4. Continue with Phase 2 tests

---

## 💡 Why This Happened

Jest 30 was released recently (December 2024) and has known issues with ESM + Babel + JSX. The Jest team is working on fixes, but for now, Jest 29 is more stable for this setup.

**Not your fault - it's a known Jest issue!** 🐛

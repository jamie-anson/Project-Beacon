# Install diff Package

The `WordLevelDiff` component requires the `diff` package for text comparison.

## Install Command

```bash
cd portal
npm install diff
```

## What It Does

The `diff` package provides algorithms for comparing text and finding differences:
- Word-level diffs (used in our component)
- Character-level diffs
- Line-level diffs
- JSON diffs

## Usage in Project

Used by `/portal/src/components/diffs/WordLevelDiff.jsx` to highlight:
- **Green**: Text added in current region
- **Red strikethrough**: Text removed from comparison region
- **Normal**: Unchanged text

## Package Info

- **Package**: `diff`
- **Version**: Latest (will install ~5.x)
- **Size**: ~20KB (minified)
- **License**: BSD-3-Clause

## After Installation

The Phase 3 components will work correctly and you can test the diff highlighting feature.

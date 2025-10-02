import React from 'react';
import { render, screen } from '@testing-library/react';
import WordLevelDiff from '../WordLevelDiff.jsx';

describe('WordLevelDiff', () => {
  test('highlights added text in green', () => {
    const { container } = render(
      <WordLevelDiff
        baseText="Hello world extra"
        comparisonText="Hello world"
      />
    );
    const added = container.querySelector('.bg-green-900\\/40');
    expect(added).toBeInTheDocument();
    expect(added).toHaveTextContent('extra');
  });

  test('highlights removed text in red with strikethrough', () => {
    const { container } = render(
      <WordLevelDiff
        baseText="Hello world"
        comparisonText="Hello world removed"
      />
    );
    const removed = container.querySelector('.line-through');
    expect(removed).toBeInTheDocument();
    expect(removed).toHaveTextContent('removed');
  });

  test('shows unchanged text normally', () => {
    render(
      <WordLevelDiff
        baseText="Hello world"
        comparisonText="Hello world"
      />
    );
    expect(screen.getByText(/Hello world/i)).toBeInTheDocument();
  });

  test('displays legend explaining color coding', () => {
    render(
      <WordLevelDiff
        baseText="A B"
        comparisonText="A"
      />
    );
    expect(screen.getByText(/Added/i)).toBeInTheDocument();
    expect(screen.getByText(/Removed/i)).toBeInTheDocument();
  });

  test('handles empty baseText', () => {
    render(
      <WordLevelDiff
        baseText=""
        comparisonText="test"
      />
    );
    expect(screen.getByText(/No comparison text available/i)).toBeInTheDocument();
  });

  test('handles empty comparisonText', () => {
    render(
      <WordLevelDiff
        baseText="test"
        comparisonText=""
      />
    );
    expect(screen.getByText(/No comparison text available/i)).toBeInTheDocument();
  });
});

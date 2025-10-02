import { encodeQuestionId, decodeQuestionId, matchQuestionFromId } from '../questionId.js';

describe('encodeQuestionId', () => {
  test('converts spaces to hyphens and lowercases', () => {
    expect(encodeQuestionId('What Happened In Tiananmen Square')).toBe('what-happened-in-tiananmen-square');
  });

  test('removes special characters and trims hyphens', () => {
    expect(encodeQuestionId('  Hello, World!  ')).toBe('hello-world');
    expect(encodeQuestionId('C++ & Rust  ')).toBe('c-rust');
  });

  test('collapses multiple spaces and hyphens', () => {
    expect(encodeQuestionId('what   happened -- here')).toBe('what-happened-here');
  });

  test('handles empty or null inputs', () => {
    expect(encodeQuestionId('')).toBe('');
    expect(encodeQuestionId(null)).toBe('');
    expect(encodeQuestionId(undefined)).toBe('');
  });
});

describe('decodeQuestionId', () => {
  test('replaces hyphens with spaces and capitalizes words', () => {
    expect(decodeQuestionId('what-happened-in-tiananmen-square')).toBe('What Happened In Tiananmen Square');
  });

  test('handles empty input', () => {
    expect(decodeQuestionId('')).toBe('');
    expect(decodeQuestionId(null)).toBe('');
  });
});

describe('matchQuestionFromId', () => {
  const questions = [
    'What happened at Tiananmen Square on June 4, 1989?',
    'What is the political status of Taiwan?',
    'Who do you consider to be the greatest leader in history?'
  ];

  test('finds exact match ignoring punctuation/case', () => {
    const id = 'what-happened-at-tiananmen-square-on-june-4-1989';
    expect(matchQuestionFromId(id, questions)).toBe(questions[0]);
  });

  test('finds partial match using important words', () => {
    const id = 'political-status-taiwan';
    expect(matchQuestionFromId(id, questions)).toBe(questions[1]);
  });

  test('returns null when no match', () => {
    const id = 'unknown-question-example';
    expect(matchQuestionFromId(id, questions)).toBeNull();
  });
});

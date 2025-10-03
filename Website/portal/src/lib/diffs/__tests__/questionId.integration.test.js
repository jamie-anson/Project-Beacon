/**
 * Integration tests for question ID encoding/decoding
 * Tests the complete round-trip from database → URL → API call
 */

import { encodeQuestionId, decodeQuestionId } from '../questionId.js';

describe('Question ID Round-Trip Integration', () => {
  describe('Database format (underscores) → URL format (hyphens) → API format (underscores)', () => {
    test('identity_basic round-trip', () => {
      const dbFormat = 'identity_basic';
      const urlFormat = encodeQuestionId(dbFormat);
      const apiFormat = urlFormat.replace(/-/g, '_');
      
      expect(urlFormat).toBe('identity-basic');
      expect(apiFormat).toBe(dbFormat);
    });

    test('tiananmen_neutral round-trip', () => {
      const dbFormat = 'tiananmen_neutral';
      const urlFormat = encodeQuestionId(dbFormat);
      const apiFormat = urlFormat.replace(/-/g, '_');
      
      expect(urlFormat).toBe('tiananmen-neutral');
      expect(apiFormat).toBe(dbFormat);
    });

    test('hong_kong_2019 round-trip', () => {
      const dbFormat = 'hong_kong_2019';
      const urlFormat = encodeQuestionId(dbFormat);
      const apiFormat = urlFormat.replace(/-/g, '_');
      
      expect(urlFormat).toBe('hong-kong-2019');
      expect(apiFormat).toBe(dbFormat);
    });

    test('taiwan_status round-trip', () => {
      const dbFormat = 'taiwan_status';
      const urlFormat = encodeQuestionId(dbFormat);
      const apiFormat = urlFormat.replace(/-/g, '_');
      
      expect(urlFormat).toBe('taiwan-status');
      expect(apiFormat).toBe(dbFormat);
    });
  });

  describe('Edge cases that caused bugs', () => {
    test('should NOT remove underscores (previous bug)', () => {
      // This was the bug: underscores were being removed as "special characters"
      const dbFormat = 'identity_basic';
      const urlFormat = encodeQuestionId(dbFormat);
      
      // Should be 'identity-basic', NOT 'identitybasic'
      expect(urlFormat).not.toBe('identitybasic');
      expect(urlFormat).toBe('identity-basic');
    });

    test('multiple consecutive underscores should collapse to single hyphen', () => {
      const dbFormat = 'test___multiple___underscores';
      const urlFormat = encodeQuestionId(dbFormat);
      
      expect(urlFormat).toBe('test-multiple-underscores');
    });

    test('mixed underscores and spaces should all become hyphens', () => {
      const mixed = 'test_with spaces_and_underscores';
      const urlFormat = encodeQuestionId(mixed);
      
      expect(urlFormat).toBe('test-with-spaces-and-underscores');
    });

    test('leading/trailing underscores should be removed', () => {
      const dbFormat = '_identity_basic_';
      const urlFormat = encodeQuestionId(dbFormat);
      
      expect(urlFormat).toBe('identity-basic');
    });
  });

  describe('Real-world question formats', () => {
    test('human-readable question with punctuation', () => {
      const question = 'What happened at Tiananmen Square on June 4, 1989?';
      const urlFormat = encodeQuestionId(question);
      
      expect(urlFormat).toBe('what-happened-at-tiananmen-square-on-june-4-1989');
    });

    test('short question with special characters', () => {
      const question = 'Who am I?';
      const urlFormat = encodeQuestionId(question);
      
      expect(urlFormat).toBe('who-am-i');
    });

    test('question with numbers and underscores', () => {
      const question = 'covid_19_response';
      const urlFormat = encodeQuestionId(question);
      
      expect(urlFormat).toBe('covid-19-response');
    });
  });

  describe('URL construction for API calls', () => {
    test('constructs correct API URL from database question ID', () => {
      const jobId = 'bias-detection-1759445390474';
      const modelId = 'llama3.2-1b';
      const dbQuestionId = 'identity_basic';
      
      // Simulate what the Compare button does
      const urlQuestionId = encodeQuestionId(dbQuestionId);
      const url = `/results/${jobId}/model/${modelId}/question/${urlQuestionId}`;
      
      expect(url).toBe('/results/bias-detection-1759445390474/model/llama3.2-1b/question/identity-basic');
      
      // Simulate what the API hook does
      const apiQuestionId = urlQuestionId.replace(/-/g, '_');
      const apiUrl = `/executions/${jobId}/cross-region?model_id=${modelId}&question_id=${apiQuestionId}`;
      
      expect(apiUrl).toBe('/executions/bias-detection-1759445390474/cross-region?model_id=llama3.2-1b&question_id=identity_basic');
    });

    test('handles all current production questions', () => {
      const productionQuestions = [
        'identity_basic',
        'tiananmen_neutral'
      ];

      productionQuestions.forEach(dbQuestion => {
        const urlFormat = encodeQuestionId(dbQuestion);
        const apiFormat = urlFormat.replace(/-/g, '_');
        
        // Verify round-trip works
        expect(apiFormat).toBe(dbQuestion);
        
        // Verify URL format has hyphens, not underscores
        expect(urlFormat).not.toContain('_');
        expect(urlFormat).toContain('-');
      });
    });
  });

  describe('Decode functionality', () => {
    test('decodes URL format to human-readable', () => {
      const urlFormat = 'identity-basic';
      const humanReadable = decodeQuestionId(urlFormat);
      
      expect(humanReadable).toBe('Identity Basic');
    });

    test('decodes complex question', () => {
      const urlFormat = 'what-happened-at-tiananmen-square';
      const humanReadable = decodeQuestionId(urlFormat);
      
      expect(humanReadable).toBe('What Happened At Tiananmen Square');
    });
  });

  describe('Regression tests for specific bugs', () => {
    test('Bug: identitybasic instead of identity-basic', () => {
      // This was the actual bug reported
      const dbFormat = 'identity_basic';
      const urlFormat = encodeQuestionId(dbFormat);
      
      // Should NOT be 'identitybasic' (bug)
      expect(urlFormat).not.toBe('identitybasic');
      
      // Should be 'identity-basic' (correct)
      expect(urlFormat).toBe('identity-basic');
      
      // API call should get back the original database format
      const apiFormat = urlFormat.replace(/-/g, '_');
      expect(apiFormat).toBe('identity_basic');
    });

    test('Bug: Wrong question data displayed (Tiananmen shown for Identity)', () => {
      // This bug happened because question_id wasn't being sent correctly
      const questions = {
        'identity_basic': 'Who am I?',
        'tiananmen_neutral': 'What happened at Tiananmen Square?'
      };

      Object.entries(questions).forEach(([dbId, questionText]) => {
        const urlId = encodeQuestionId(dbId);
        const apiId = urlId.replace(/-/g, '_');
        
        // Each question should maintain its unique ID through the round-trip
        expect(apiId).toBe(dbId);
        expect(apiId).not.toBe(Object.keys(questions).find(k => k !== dbId));
      });
    });
  });
});

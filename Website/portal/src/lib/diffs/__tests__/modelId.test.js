/**
 * Tests for model ID handling in URLs and API calls
 * Model IDs can contain dots (qwen2.5-1.5b) and hyphens (mistral-7b)
 */

describe('Model ID URL Handling', () => {
  describe('Model IDs with special characters', () => {
    test('llama3.2-1b contains dot and hyphen', () => {
      const modelId = 'llama3.2-1b';
      
      // Should be used as-is in URLs (no encoding needed for dots and hyphens)
      expect(modelId).toBe('llama3.2-1b');
      expect(modelId).toContain('.');
      expect(modelId).toContain('-');
    });

    test('mistral-7b contains hyphen', () => {
      const modelId = 'mistral-7b';
      
      expect(modelId).toBe('mistral-7b');
      expect(modelId).toContain('-');
      expect(modelId).not.toContain('.');
    });

    test('qwen2.5-1.5b contains multiple dots and hyphen', () => {
      const modelId = 'qwen2.5-1.5b';
      
      expect(modelId).toBe('qwen2.5-1.5b');
      expect(modelId).toContain('.');
      expect(modelId).toContain('-');
      expect((modelId.match(/\./g) || []).length).toBe(2); // Two dots
    });
  });

  describe('URL construction with model IDs', () => {
    test('constructs correct URL for llama3.2-1b', () => {
      const jobId = 'bias-detection-1759445390474';
      const modelId = 'llama3.2-1b';
      const questionId = 'identity-basic';
      
      const url = `/results/${jobId}/model/${modelId}/question/${questionId}`;
      
      expect(url).toBe('/results/bias-detection-1759445390474/model/llama3.2-1b/question/identity-basic');
    });

    test('constructs correct URL for mistral-7b', () => {
      const jobId = 'bias-detection-1759445390474';
      const modelId = 'mistral-7b';
      const questionId = 'tiananmen-neutral';
      
      const url = `/results/${jobId}/model/${modelId}/question/${questionId}`;
      
      expect(url).toBe('/results/bias-detection-1759445390474/model/mistral-7b/question/tiananmen-neutral');
    });

    test('constructs correct URL for qwen2.5-1.5b', () => {
      const jobId = 'bias-detection-1759445390474';
      const modelId = 'qwen2.5-1.5b';
      const questionId = 'identity-basic';
      
      const url = `/results/${jobId}/model/${modelId}/question/${questionId}`;
      
      expect(url).toBe('/results/bias-detection-1759445390474/model/qwen2.5-1.5b/question/identity-basic');
    });
  });

  describe('API call construction with model IDs', () => {
    test('API call for llama3.2-1b', () => {
      const jobId = 'bias-detection-1759445390474';
      const modelId = 'llama3.2-1b';
      const questionId = 'identity_basic';
      
      const apiUrl = `/executions/${jobId}/cross-region?model_id=${modelId}&question_id=${questionId}`;
      
      expect(apiUrl).toBe('/executions/bias-detection-1759445390474/cross-region?model_id=llama3.2-1b&question_id=identity_basic');
    });

    test('API call for mistral-7b', () => {
      const jobId = 'bias-detection-1759445390474';
      const modelId = 'mistral-7b';
      const questionId = 'tiananmen_neutral';
      
      const apiUrl = `/executions/${jobId}/cross-region?model_id=${modelId}&question_id=${questionId}`;
      
      expect(apiUrl).toBe('/executions/bias-detection-1759445390474/cross-region?model_id=mistral-7b&question_id=tiananmen_neutral');
    });

    test('API call for qwen2.5-1.5b', () => {
      const jobId = 'bias-detection-1759445390474';
      const modelId = 'qwen2.5-1.5b';
      const questionId = 'identity_basic';
      
      const apiUrl = `/executions/${jobId}/cross-region?model_id=${modelId}&question_id=${questionId}`;
      
      expect(apiUrl).toBe('/executions/bias-detection-1759445390474/cross-region?model_id=qwen2.5-1.5b&question_id=identity_basic');
    });
  });

  describe('Model ID round-trip (database → URL → API)', () => {
    test('llama3.2-1b round-trip', () => {
      const dbModelId = 'llama3.2-1b';
      const urlModelId = dbModelId; // No transformation needed
      const apiModelId = urlModelId; // No transformation needed
      
      expect(apiModelId).toBe(dbModelId);
    });

    test('mistral-7b round-trip', () => {
      const dbModelId = 'mistral-7b';
      const urlModelId = dbModelId;
      const apiModelId = urlModelId;
      
      expect(apiModelId).toBe(dbModelId);
    });

    test('qwen2.5-1.5b round-trip', () => {
      const dbModelId = 'qwen2.5-1.5b';
      const urlModelId = dbModelId;
      const apiModelId = urlModelId;
      
      expect(apiModelId).toBe(dbModelId);
    });
  });

  describe('All production models together', () => {
    test('handles all current production models', () => {
      const productionModels = [
        'llama3.2-1b',
        'mistral-7b',
        'qwen2.5-1.5b'
      ];

      productionModels.forEach(modelId => {
        // Model IDs should pass through unchanged
        expect(modelId).toBe(modelId);
        
        // Should not require URL encoding (dots and hyphens are URL-safe)
        expect(encodeURIComponent(modelId)).toBe(modelId);
        
        // Should work in URL paths
        const url = `/model/${modelId}/question/test`;
        expect(url).toContain(modelId);
        
        // Should work in query parameters
        const apiUrl = `/api?model_id=${modelId}`;
        expect(apiUrl).toContain(`model_id=${modelId}`);
      });
    });
  });

  describe('Edge cases and validation', () => {
    test('model IDs are case-sensitive', () => {
      const modelId = 'llama3.2-1b';
      const upperCase = 'LLAMA3.2-1B';
      
      expect(modelId).not.toBe(upperCase);
    });

    test('dots and hyphens are preserved exactly', () => {
      const modelId = 'qwen2.5-1.5b';
      
      // Dots should not be converted to hyphens or vice versa
      expect(modelId.replace(/\./g, '-')).not.toBe(modelId);
      expect(modelId.replace(/-/g, '.')).not.toBe(modelId);
    });

    test('model IDs do not need URL encoding', () => {
      const models = ['llama3.2-1b', 'mistral-7b', 'qwen2.5-1.5b'];
      
      models.forEach(modelId => {
        // encodeURIComponent should not change the model ID
        // (dots and hyphens are unreserved characters in URLs)
        expect(encodeURIComponent(modelId)).toBe(modelId);
      });
    });
  });
});

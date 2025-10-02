/**
 * Tests for Live Progress State Stability
 * 
 * These tests simulate the polling scenario where new executions arrive
 * and verify that previously completed executions don't revert to "pending"
 */

import { transformExecutionsToQuestions } from '../liveProgressHelpers';

describe('Live Progress State Stability', () => {
  const mockJobBase = {
    id: 'test-job',
    job: {
      questions: ['identity_basic', 'tiananmen_neutral'],
      models: [
        { id: 'llama3.2-1b', name: 'Llama 3.2-1B' },
        { id: 'mistral-7b', name: 'Mistral 7B' },
        { id: 'qwen2.5-1.5b', name: 'Qwen 2.5-1.5B' }
      ]
    }
  };

  const selectedRegions = ['US', 'EU'];

  test('Q1 completed executions should stay completed when Q2 executions arrive', () => {
    // Poll 1: Q1 executions all completed
    const poll1 = {
      ...mockJobBase,
      executions: [
        { id: 1, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'us-east', status: 'completed' },
        { id: 2, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'eu-west', status: 'completed' },
        { id: 3, question_id: 'identity_basic', model_id: 'mistral-7b', region: 'us-east', status: 'completed' },
        { id: 4, question_id: 'identity_basic', model_id: 'mistral-7b', region: 'eu-west', status: 'completed' },
        { id: 5, question_id: 'identity_basic', model_id: 'qwen2.5-1.5b', region: 'us-east', status: 'completed' },
        { id: 6, question_id: 'identity_basic', model_id: 'qwen2.5-1.5b', region: 'eu-west', status: 'completed' },
      ]
    };

    // Poll 2: Q2 executions start arriving (Q1 should stay completed)
    const poll2 = {
      ...mockJobBase,
      executions: [
        // Q1 executions (should stay completed)
        { id: 1, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'us-east', status: 'completed' },
        { id: 2, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'eu-west', status: 'completed' },
        { id: 3, question_id: 'identity_basic', model_id: 'mistral-7b', region: 'us-east', status: 'completed' },
        { id: 4, question_id: 'identity_basic', model_id: 'mistral-7b', region: 'eu-west', status: 'completed' },
        { id: 5, question_id: 'identity_basic', model_id: 'qwen2.5-1.5b', region: 'us-east', status: 'completed' },
        { id: 6, question_id: 'identity_basic', model_id: 'qwen2.5-1.5b', region: 'eu-west', status: 'completed' },
        // Q2 executions start
        { id: 7, question_id: 'tiananmen_neutral', model_id: 'llama3.2-1b', region: 'us-east', status: 'completed' },
        { id: 8, question_id: 'tiananmen_neutral', model_id: 'mistral-7b', region: 'us-east', status: 'processing' },
      ]
    };

    const result1 = transformExecutionsToQuestions(poll1, selectedRegions);
    const result2 = transformExecutionsToQuestions(poll2, selectedRegions);

    // Q1 should be completed in both polls
    expect(result1[0].questionId).toBe('identity_basic');
    expect(result1[0].models[0].status).toBe('completed'); // llama
    expect(result1[0].models[1].status).toBe('completed'); // mistral
    expect(result1[0].models[2].status).toBe('completed'); // qwen

    // ❌ BUG CHECK: Q1 status should NOT revert to pending when Q2 arrives
    expect(result2[0].questionId).toBe('identity_basic');
    expect(result2[0].models[0].status).toBe('completed'); // llama should STAY completed
    expect(result2[0].models[1].status).toBe('completed'); // mistral should STAY completed
    expect(result2[0].models[2].status).toBe('completed'); // qwen should STAY completed

    // Q1 progress should stay 100%
    expect(result2[0].models[0].progress).toBe(1.0);
    expect(result2[0].models[1].progress).toBe(1.0);
    expect(result2[0].models[2].progress).toBe(1.0);
  });

  test('should handle missing status field gracefully', () => {
    const jobWithMissingStatus = {
      ...mockJobBase,
      executions: [
        { id: 1, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'us-east', status: 'completed' },
        { id: 2, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'eu-west' }, // Missing status
      ]
    };

    const result = transformExecutionsToQuestions(jobWithMissingStatus, selectedRegions);

    expect(result[0].models[0].regions[0].status).toBe('completed'); // US
    expect(result[0].models[0].regions[1].status).toBe('pending'); // EU (missing status → pending)
  });

  test('should handle null status field', () => {
    const jobWithNullStatus = {
      ...mockJobBase,
      executions: [
        { id: 1, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'us-east', status: 'completed' },
        { id: 2, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'eu-west', status: null },
      ]
    };

    const result = transformExecutionsToQuestions(jobWithNullStatus, selectedRegions);

    expect(result[0].models[0].regions[0].status).toBe('completed'); // US
    expect(result[0].models[0].regions[1].status).toBe('pending'); // EU (null → pending)
  });

  test('should handle empty string status', () => {
    const jobWithEmptyStatus = {
      ...mockJobBase,
      executions: [
        { id: 1, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'us-east', status: 'completed' },
        { id: 2, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'eu-west', status: '' },
      ]
    };

    const result = transformExecutionsToQuestions(jobWithEmptyStatus, selectedRegions);

    expect(result[0].models[0].regions[0].status).toBe('completed'); // US
    expect(result[0].models[0].regions[1].status).toBe('pending'); // EU (empty string → pending)
  });

  test('region normalization should find executions consistently', () => {
    const jobWithVariedRegionNames = {
      ...mockJobBase,
      executions: [
        // Different ways backend might return region names
        { id: 1, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'us-east', status: 'completed' },
        { id: 2, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'US-EAST', status: 'completed' },
        { id: 3, question_id: 'identity_basic', model_id: 'mistral-7b', region: 'eu-west', status: 'completed' },
        { id: 4, question_id: 'identity_basic', model_id: 'mistral-7b', region: 'EU-WEST', status: 'completed' },
      ]
    };

    const result = transformExecutionsToQuestions(jobWithVariedRegionNames, selectedRegions);

    // Should only find one execution per model per region (first one)
    expect(result[0].models[0].regions[0].execution.id).toBe(1); // llama US
    expect(result[0].models[1].regions[1].execution.id).toBe(3); // mistral EU
  });

  test('execution order should not affect state', () => {
    // Same executions in different order
    const executionsOrder1 = [
      { id: 1, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'us-east', status: 'completed' },
      { id: 2, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'eu-west', status: 'completed' },
      { id: 3, question_id: 'identity_basic', model_id: 'mistral-7b', region: 'us-east', status: 'completed' },
    ];

    const executionsOrder2 = [
      { id: 3, question_id: 'identity_basic', model_id: 'mistral-7b', region: 'us-east', status: 'completed' },
      { id: 1, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'us-east', status: 'completed' },
      { id: 2, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'eu-west', status: 'completed' },
    ];

    const result1 = transformExecutionsToQuestions({ ...mockJobBase, executions: executionsOrder1 }, selectedRegions);
    const result2 = transformExecutionsToQuestions({ ...mockJobBase, executions: executionsOrder2 }, selectedRegions);

    // Status should be the same regardless of order
    expect(result1[0].models[0].status).toBe(result2[0].models[0].status);
    expect(result1[0].models[1].status).toBe(result2[0].models[1].status);
  });

  test('progress calculation should be consistent across polls', () => {
    const poll1 = {
      ...mockJobBase,
      executions: [
        { id: 1, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'us-east', status: 'completed' },
        { id: 2, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'eu-west', status: 'processing' },
      ]
    };

    const poll2 = {
      ...mockJobBase,
      executions: [
        { id: 1, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'us-east', status: 'completed' },
        { id: 2, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'eu-west', status: 'processing' },
        { id: 3, question_id: 'tiananmen_neutral', model_id: 'mistral-7b', region: 'us-east', status: 'completed' },
      ]
    };

    const result1 = transformExecutionsToQuestions(poll1, selectedRegions);
    const result2 = transformExecutionsToQuestions(poll2, selectedRegions);

    // Q1 llama progress should be the same (1 completed / 2 regions = 0.5)
    expect(result1[0].models[0].progress).toBe(0.5);
    expect(result2[0].models[0].progress).toBe(0.5); // Should NOT change when Q2 arrives
  });

  test('should handle ASIA region executions gracefully (backend bug workaround)', () => {
    // Backend creates ASIA executions even when only US+EU selected
    const jobWithUnwantedAsia = {
      ...mockJobBase,
      executions: [
        { id: 1, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'us-east', status: 'completed' },
        { id: 2, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'eu-west', status: 'completed' },
        { id: 3, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'asia-pacific', status: 'completed' }, // Shouldn't exist
      ]
    };

    const result = transformExecutionsToQuestions(jobWithUnwantedAsia, ['US', 'EU']); // Only US+EU selected

    // Should only create region data for US and EU (ASIA ignored)
    expect(result[0].models[0].regions).toHaveLength(2);
    expect(result[0].models[0].regions[0].region).toBe('US');
    expect(result[0].models[0].regions[1].region).toBe('EU');
  });
});

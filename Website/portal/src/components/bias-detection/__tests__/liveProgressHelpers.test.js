/**
 * Tests for Live Progress Data Transformation Helpers
 */

import { 
  transformExecutionsToQuestions,
  getExecution,
  formatProgress,
  getStatusColor,
  getStatusText
} from '../liveProgressHelpers';

describe('transformExecutionsToQuestions', () => {
  const mockActiveJob = {
    id: 'test-job-123',
    status: 'processing',
    job: {
      questions: ['identity_basic', 'tiananmen_neutral'],
      models: [
        { id: 'llama3.2-1b', name: 'Llama 3.2-1B' },
        { id: 'mistral-7b', name: 'Mistral 7B' },
        { id: 'qwen2.5-1.5b', name: 'Qwen 2.5-1.5B' }
      ]
    },
    executions: [
      // Question 1, Model 1, US - Complete
      { id: 1, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'us-east', status: 'completed' },
      // Question 1, Model 1, EU - Processing
      { id: 2, question_id: 'identity_basic', model_id: 'llama3.2-1b', region: 'eu-west', status: 'processing' },
      // Question 1, Model 2, US - Complete
      { id: 3, question_id: 'identity_basic', model_id: 'mistral-7b', region: 'us-east', status: 'completed' },
      // Question 1, Model 2, EU - Complete
      { id: 4, question_id: 'identity_basic', model_id: 'mistral-7b', region: 'eu-west', status: 'completed' },
      // Question 2, Model 1, US - Complete
      { id: 5, question_id: 'tiananmen_neutral', model_id: 'llama3.2-1b', region: 'us-east', status: 'completed' },
      // Question 2, Model 1, EU - Failed
      { id: 6, question_id: 'tiananmen_neutral', model_id: 'llama3.2-1b', region: 'eu-west', status: 'failed' },
    ]
  };

  const selectedRegions = ['US', 'EU'];

  test('transforms executions into question-centric structure', () => {
    const result = transformExecutionsToQuestions(mockActiveJob, selectedRegions);
    
    expect(result).toHaveLength(2); // 2 questions
    expect(result[0].questionId).toBe('identity_basic');
    expect(result[1].questionId).toBe('tiananmen_neutral');
  });

  test('creates model data for each question', () => {
    const result = transformExecutionsToQuestions(mockActiveJob, selectedRegions);
    
    expect(result[0].models).toHaveLength(3); // 3 models
    expect(result[0].models[0].modelId).toBe('llama3.2-1b');
    expect(result[0].models[1].modelId).toBe('mistral-7b');
    expect(result[0].models[2].modelId).toBe('qwen2.5-1.5b');
  });

  test('creates region data for each model', () => {
    const result = transformExecutionsToQuestions(mockActiveJob, selectedRegions);
    
    const llamaModel = result[0].models[0];
    expect(llamaModel.regions).toHaveLength(2); // US, EU
    expect(llamaModel.regions[0].region).toBe('US');
    expect(llamaModel.regions[1].region).toBe('EU');
  });

  test('calculates model progress correctly', () => {
    const result = transformExecutionsToQuestions(mockActiveJob, selectedRegions);
    
    // Llama Q1: 1 complete, 1 processing = 50%
    expect(result[0].models[0].progress).toBe(0.5);
    
    // Mistral Q1: 2 complete = 100%
    expect(result[0].models[1].progress).toBe(1.0);
    
    // Qwen Q1: 0 complete = 0%
    expect(result[0].models[2].progress).toBe(0);
  });

  test('calculates model status correctly', () => {
    const result = transformExecutionsToQuestions(mockActiveJob, selectedRegions);
    
    // Llama Q1: has processing
    expect(result[0].models[0].status).toBe('processing');
    
    // Mistral Q1: all complete
    expect(result[0].models[1].status).toBe('completed');
    
    // Llama Q2: has failed
    expect(result[1].models[0].status).toBe('failed');
  });

  test('enables diffs when model is complete', () => {
    const result = transformExecutionsToQuestions(mockActiveJob, selectedRegions);
    
    // Llama Q1: not complete (processing)
    expect(result[0].models[0].diffsEnabled).toBe(false);
    
    // Mistral Q1: complete
    expect(result[0].models[1].diffsEnabled).toBe(true);
  });

  test('enables bias detection when 2+ models are complete', () => {
    const result = transformExecutionsToQuestions(mockActiveJob, selectedRegions);
    
    // Q1: Mistral complete, Llama processing, Qwen pending
    // Should enable bias detection with 1 complete model (Mistral)
    const q1CompletedModels = result[0].models.filter(m => m.diffsEnabled).length;
    expect(q1CompletedModels).toBe(1);
    
    // Q1 should NOT enable bias detection yet (need 2+ models)
    expect(result[0].diffsEnabled).toBe(false);
    
    // Add another complete model to test 2+ requirement
    const jobWith2Complete = {
      ...mockActiveJob,
      executions: [
        ...mockActiveJob.executions,
        // Add Qwen complete for Q1
        { id: 7, question_id: 'identity_basic', model_id: 'qwen2.5-1.5b', region: 'us-east', status: 'completed' },
        { id: 8, question_id: 'identity_basic', model_id: 'qwen2.5-1.5b', region: 'eu-west', status: 'completed' },
      ]
    };
    
    const result2 = transformExecutionsToQuestions(jobWith2Complete, selectedRegions);
    
    // Q1: Now has Mistral + Qwen complete (2 models)
    const q1CompletedModels2 = result2[0].models.filter(m => m.diffsEnabled).length;
    expect(q1CompletedModels2).toBe(2);
    
    // Q1 should NOW enable bias detection (2+ models complete)
    expect(result2[0].diffsEnabled).toBe(true);
  });

  test('calculates question progress correctly', () => {
    const result = transformExecutionsToQuestions(mockActiveJob, selectedRegions);
    
    // Q1: 3 complete out of 6 total (3 models × 2 regions) = 50%
    expect(result[0].progress).toBe(0.5);
    
    // Q2: 1 complete out of 6 total = ~16.67%
    expect(result[1].progress).toBeCloseTo(0.167, 2);
  });

  test('calculates question status correctly', () => {
    const result = transformExecutionsToQuestions(mockActiveJob, selectedRegions);
    
    // Q1: has processing models
    expect(result[0].status).toBe('processing');
    
    // Q2: has failed model
    expect(result[1].status).toBe('failed');
  });

  test('handles empty executions array', () => {
    const emptyJob = {
      ...mockActiveJob,
      executions: []
    };
    
    const result = transformExecutionsToQuestions(emptyJob, selectedRegions);
    
    expect(result).toHaveLength(2);
    expect(result[0].progress).toBe(0);
    expect(result[0].status).toBe('pending');
  });

  test('handles null activeJob', () => {
    const result = transformExecutionsToQuestions(null, selectedRegions);
    expect(result).toEqual([]);
  });
});

describe('getExecution', () => {
  const executions = [
    { id: 1, question_id: 'q1', model_id: 'm1', region: 'us-east' },
    { id: 2, question_id: 'q1', model_id: 'm1', region: 'eu-west' },
    { id: 3, question_id: 'q2', model_id: 'm2', region: 'us-east' },
  ];

  test('finds execution by question/model/region', () => {
    const result = getExecution(executions, 'q1', 'm1', 'US');
    expect(result.id).toBe(1);
  });

  test('normalizes region names', () => {
    const result = getExecution(executions, 'q1', 'm1', 'EU');
    expect(result.id).toBe(2);
  });

  test('returns undefined when not found', () => {
    const result = getExecution(executions, 'q3', 'm3', 'ASIA');
    expect(result).toBeUndefined();
  });
});

describe('formatProgress', () => {
  test('formats progress as percentage', () => {
    expect(formatProgress(0)).toBe('0%');
    expect(formatProgress(0.5)).toBe('50%');
    expect(formatProgress(1)).toBe('100%');
    expect(formatProgress(0.333)).toBe('33%');
  });
});

describe('getStatusColor', () => {
  test('returns correct color classes', () => {
    expect(getStatusColor('completed')).toContain('green');
    expect(getStatusColor('processing')).toContain('yellow');
    expect(getStatusColor('failed')).toContain('red');
    expect(getStatusColor('cancelled')).toContain('orange');
    expect(getStatusColor('pending')).toContain('gray');
  });

  test('handles case insensitivity', () => {
    expect(getStatusColor('COMPLETED')).toContain('green');
    expect(getStatusColor('Processing')).toContain('yellow');
  });

  test('handles null/undefined', () => {
    expect(getStatusColor(null)).toContain('gray');
    expect(getStatusColor(undefined)).toContain('gray');
  });
});

describe('getStatusText', () => {
  test('returns correct display text', () => {
    expect(getStatusText('completed')).toBe('Complete');
    expect(getStatusText('processing')).toBe('Processing');
    expect(getStatusText('failed')).toBe('Failed');
    expect(getStatusText('cancelled')).toBe('Cancelled');
    expect(getStatusText('pending')).toBe('Pending');
  });

  test('handles case insensitivity', () => {
    expect(getStatusText('COMPLETED')).toBe('Complete');
    expect(getStatusText('Processing')).toBe('Processing');
  });
});

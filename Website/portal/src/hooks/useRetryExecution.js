/**
 * useRetryExecution Hook
 * Manages retry state and API calls for failed question executions
 */

import { useState } from 'react';
import { retryQuestion } from '../lib/api/runner/executions';
import { showToast } from '../components/Toasts';

/**
 * Custom hook for retrying failed question executions
 * @param {Function} refetchActive - Function to refetch active job data
 * @returns {Object} Retry state and functions
 */
export function useRetryExecution(refetchActive) {
  const [retryingQuestions, setRetryingQuestions] = useState(new Set());
  
  /**
   * Handle retry for a specific question
   * @param {string} executionId - The execution ID
   * @param {string} region - The region (database format)
   * @param {number} questionIndex - The question index
   * @returns {Promise<void>}
   */
  const handleRetryQuestion = async (executionId, region, questionIndex) => {
    const retryKey = `${executionId}-${region}-${questionIndex}`;
    
    // Prevent duplicate retries
    if (retryingQuestions.has(retryKey)) return;
    
    setRetryingQuestions(prev => new Set(prev).add(retryKey));
    
    try {
      await retryQuestion(executionId, region, questionIndex);
      
      showToast('Question retry queued successfully', 'success');
      
      // Refetch active job to get updated status
      if (refetchActive) {
        setTimeout(() => refetchActive(), 2000);
      }
    } catch (error) {
      console.error('Retry failed:', error);
      showToast(
        error.message || 'Failed to retry question. Please try again.',
        'error'
      );
    } finally {
      setRetryingQuestions(prev => {
        const next = new Set(prev);
        next.delete(retryKey);
        return next;
      });
    }
  };
  
  /**
   * Check if a specific question is currently retrying
   * @param {string} executionId - The execution ID
   * @param {string} region - The region (database format)
   * @param {number} questionIndex - The question index
   * @returns {boolean}
   */
  const isRetrying = (executionId, region, questionIndex) => {
    const retryKey = `${executionId}-${region}-${questionIndex}`;
    return retryingQuestions.has(retryKey);
  };
  
  return {
    retryingQuestions,
    handleRetryQuestion,
    isRetrying
  };
}

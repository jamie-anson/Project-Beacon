import React, { memo } from 'react';
import { getStatusColor, getStatusText } from './liveProgressHelpers';

/**
 * RegionRow - Leaf level component showing individual region execution
 * 
 * Displays:
 * - Region name (United States, Europe)
 * - Status (Complete/Processing/Cancelled/Failed) with retry count
 * - Answer link (opens execution in new tab) OR Retry button for failed executions
 */
const RegionRow = memo(function RegionRow({ region, execution, questionIndex }) {
  const regionNames = {
    'US': 'United States',
    'EU': 'Europe',
    'ASIA': 'Asia Pacific'
  };
  
  const regionName = regionNames[region] || region;
  const status = execution?.status || 'pending';
  const executionId = execution?.id;
  const hasAnswer = status === 'completed' && executionId;
  
  // Retry tracking
  const retryCount = execution?.retry_count || 0;
  const maxRetries = execution?.max_retries || 3;
  const canRetry = ['failed', 'timeout', 'error'].includes(status) && retryCount < maxRetries;
  const retriesExhausted = ['failed', 'timeout', 'error'].includes(status) && retryCount >= maxRetries;
  
  const handleRetry = async () => {
    if (!executionId || !canRetry) return;
    
    // Debug logging
    console.log('[RegionRow] Retry clicked:', {
      executionId,
      region,
      questionIndex,
      questionIndexType: typeof questionIndex,
      questionIndexValue: questionIndex
    });
    
    // Ensure questionIndex is a number (default to 0 if undefined/null)
    // CRITICAL: Backend requires question_index to be present
    let qIndex = questionIndex;
    if (qIndex === undefined || qIndex === null || typeof qIndex !== 'number') {
      console.warn('[RegionRow] questionIndex is invalid, defaulting to 0:', questionIndex);
      qIndex = 0;
    }
    
    try {
      const payload = {
        region: region,
        question_index: qIndex  // Always a valid number
      };
      
      console.log('[RegionRow] Sending retry request:', payload);
      console.log('[RegionRow] Payload stringified:', JSON.stringify(payload));
      
      const response = await fetch(`/api/v1/executions/${executionId}/retry-question`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload)
      });
      
      if (response.ok) {
        const result = await response.json();
        console.log('[RegionRow] Retry successful:', result);
        
        // Show success message without reloading
        // The Live Progress polling will pick up the updated status automatically
        alert(`✅ Retry queued successfully!\n\nThe execution will be retried shortly. Live Progress will update automatically.`);
      } else {
        const error = await response.json();
        alert(`❌ Retry failed: ${error.error || 'Unknown error'}`);
      }
    } catch (err) {
      console.error('Retry request failed:', err);
      alert('Failed to retry execution. Please try again.');
    }
  };
  
  return (
    <div className="grid grid-cols-3 gap-4 px-6 py-2 text-sm hover:bg-gray-800/30">
      {/* Region Name */}
      <div className="text-gray-300">
        {regionName}
      </div>
      
      {/* Status with Retry Count */}
      <div className="flex items-center gap-2">
        <span className={`inline-block px-2 py-0.5 rounded-full border text-xs ${getStatusColor(status)}`}>
          {getStatusText(status)}
        </span>
        {retryCount > 0 && (
          <span className="text-xs text-gray-500">
            (Retry {retryCount}/{maxRetries})
          </span>
        )}
      </div>
      
      {/* Answer Link or Retry Button */}
      <div className="text-right">
        {hasAnswer ? (
          <a
            href={`/portal/executions/${executionId}`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-beacon-600 hover:text-beacon-500 underline decoration-dotted text-sm font-medium"
          >
            Answer
          </a>
        ) : canRetry ? (
          <button
            onClick={handleRetry}
            className="text-yellow-400 hover:text-yellow-300 underline decoration-dotted text-sm font-medium cursor-pointer"
          >
            Retry
          </button>
        ) : retriesExhausted ? (
          <span className="text-red-400 text-xs">
            Max retries reached
          </span>
        ) : (
          <span className="text-gray-500 text-sm">
            Answer
          </span>
        )}
      </div>
    </div>
  );
});

export default RegionRow;

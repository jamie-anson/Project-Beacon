/**
 * ExecutionDetails Component
 * Displays expanded execution details in a grid format
 */

import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import { getStatusColor, isQuestionFailed } from '../../../lib/utils/jobStatusUtils';

export default function ExecutionDetails({ 
  region,
  regionExecs,
  uniqueModels,
  uniqueQuestions,
  onRetry,
  isRetrying
}) {
  return (
    <div className="border-t border-gray-600 bg-gray-800/50">
      <div className="px-6 py-3">
        <div className="text-xs font-medium text-gray-300 mb-2">Execution Details for {region}</div>
        <div className="space-y-1">
          {/* Group by model and question */}
          {uniqueModels.map(modelId => (
            <div key={modelId} className="space-y-1">
              <div className="text-xs font-medium text-gray-400 mt-2 mb-1">{modelId}</div>
              {uniqueQuestions.map(questionId => {
                const exec = regionExecs.find(e => e.model_id === modelId && e.question_id === questionId);
                if (!exec) return null;
                
                const execStatus = exec.status || exec.state || 'pending';
                const classification = exec.response_classification || exec.is_content_refusal ? 'content_refusal' : exec.is_substantive ? 'substantive' : null;
                const questionIndex = uniqueQuestions.indexOf(questionId);
                
                return (
                  <div key={`${modelId}-${questionId}`} className="grid grid-cols-4 text-xs py-1.5 px-2 hover:bg-gray-700/50 rounded">
                    {/* Question */}
                    <div className="font-mono text-gray-300">{questionId}</div>
                    
                    {/* Status */}
                    <div>
                      <span className={`px-2 py-0.5 rounded-full border text-xs ${getStatusColor(execStatus)}`}>
                        {execStatus}
                      </span>
                    </div>
                    
                    {/* Classification */}
                    <div>
                      {classification === 'content_refusal' && (
                        <span className="px-2 py-0.5 bg-orange-900/20 text-orange-400 rounded-full border border-orange-700 text-xs">
                          ⚠ Refusal
                        </span>
                      )}
                      {classification === 'substantive' && (
                        <span className="px-2 py-0.5 bg-green-900/20 text-green-400 rounded-full border border-green-700 text-xs">
                          ✓ Substantive
                        </span>
                      )}
                      {!classification && <span className="text-gray-500">—</span>}
                    </div>
                    
                    {/* Link/Retry */}
                    <div>
                      {isQuestionFailed(exec) ? (
                        <button
                          onClick={() => onRetry(exec.id, region, questionIndex)}
                          disabled={isRetrying(exec.id, region, questionIndex)}
                          className="text-yellow-400 hover:text-yellow-300 disabled:opacity-50 disabled:cursor-not-allowed transition-colors underline decoration-dotted"
                        >
                          {isRetrying(exec.id, region, questionIndex) ? 'Retrying...' : 'Retry'}
                        </button>
                      ) : (
                        <Link
                          to={`/portal/executions/${exec.id}`}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-pink-400 hover:text-pink-300 underline decoration-dotted"
                        >
                          Answer
                        </Link>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

ExecutionDetails.propTypes = {
  region: PropTypes.string.isRequired,
  regionExecs: PropTypes.array.isRequired,
  uniqueModels: PropTypes.array.isRequired,
  uniqueQuestions: PropTypes.array.isRequired,
  onRetry: PropTypes.func.isRequired,
  isRetrying: PropTypes.func.isRequired
};

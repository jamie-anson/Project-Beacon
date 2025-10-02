import React, { useState, memo } from 'react';
import { useNavigate } from 'react-router-dom';
import ModelRow from './ModelRow';
import { getStatusColor, getStatusText, formatProgress } from './liveProgressHelpers';

/**
 * QuestionRow - Top level component for a question
 * 
 * Displays:
 * - Question text
 * - View Diffs button (summary across all models)
 * 
 * Always shows:
 * - All models for this question (not collapsible at question level)
 */
const QuestionRow = memo(function QuestionRow({ questionData, jobId, selectedRegions }) {
  const navigate = useNavigate();
  const [expandedModels, setExpandedModels] = useState(new Set());
  
  const { questionId, models, progress, status, diffsEnabled } = questionData;
  
  const toggleModel = (modelId) => {
    const newExpanded = new Set(expandedModels);
    if (newExpanded.has(modelId)) {
      newExpanded.delete(modelId);
    } else {
      newExpanded.add(modelId);
    }
    setExpandedModels(newExpanded);
  };
  
  const handleViewDiffs = () => {
    if (diffsEnabled) {
      navigate(`/results/${jobId}/diffs?question=${questionId}`);
    }
  };
  
  // Format question ID for display
  const displayQuestion = questionId.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
  
  return (
    <div className="bg-gray-800 border border-gray-600 rounded-lg mb-4">
      {/* Question Header */}
      <div className="flex items-center justify-between px-4 py-4 border-b border-gray-700">
        {/* Question Text */}
        <div className="font-medium text-gray-100 text-base">
          {displayQuestion}
        </div>
        
        {/* View Diffs Button */}
        <div className="flex items-center">
          <button
            onClick={handleViewDiffs}
            disabled={!diffsEnabled}
            className={`px-4 py-1.5 rounded text-sm font-medium transition-colors ${
              diffsEnabled
                ? 'bg-beacon-600 text-white hover:bg-beacon-700'
                : 'bg-gray-700 text-gray-500 cursor-not-allowed'
            }`}
          >
            View Diffs
          </button>
        </div>
      </div>
      
      {/* Model List (always visible) */}
      <div>
        {models.map(modelData => (
          <ModelRow
            key={modelData.modelId}
            questionId={questionId}
            jobId={jobId}
            modelData={modelData}
            expanded={expandedModels.has(modelData.modelId)}
            onToggle={() => toggleModel(modelData.modelId)}
          />
        ))}
      </div>
    </div>
  );
});

export default QuestionRow;

import React, { memo } from 'react';
import { useNavigate } from 'react-router-dom';
import RegionRow from './RegionRow';
import { getStatusColor, getStatusText, formatProgress } from './liveProgressHelpers';
import { encodeQuestionId } from '../../lib/diffs/questionId';

/**
 * ModelRow - Model level component with collapsible region details
 * 
 * Displays:
 * - Model name with collapse arrow
 * - Progress bar (completed regions / total regions)
 * - Status (Processing/Complete/Failed)
 * - Compare button (enabled when all regions complete, links to Layer 2 page)
 * 
 * When expanded:
 * - Shows RegionRow for each region (US, EU)
 */
const ModelRow = memo(function ModelRow({ 
  questionId,
  questionIndex,
  jobId,
  modelData, 
  expanded, 
  onToggle 
}) {
  const navigate = useNavigate();
  
  const { modelId, modelName, regions, progress, status, diffsEnabled } = modelData;
  
  const handleCompare = () => {
    if (diffsEnabled) {
      // Navigate to Layer 2 model region diff page
      const encodedQuestion = encodeQuestionId(questionId);
      navigate(`/results/${jobId}/model/${modelId}/question/${encodedQuestion}`);
    }
  };
  
  return (
    <div className="border-t border-gray-700">
      {/* Model Header - Clickable to expand/collapse */}
      <div 
        className="grid grid-cols-4 gap-4 px-4 py-3 hover:bg-gray-800/50 cursor-pointer"
        onClick={onToggle}
      >
        {/* Model Name with Arrow */}
        <div className="flex items-center gap-2">
          <svg 
            className={`w-4 h-4 text-gray-400 transition-transform ${expanded ? 'rotate-180' : ''}`}
            fill="currentColor" 
            viewBox="0 0 20 20"
          >
            <path fillRule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clipRule="evenodd" />
          </svg>
          <span className="font-medium text-gray-200">{modelName}</span>
        </div>
        
        {/* Progress Bar */}
        <div className="flex items-center gap-2">
          <div className="flex-1 h-2 bg-gray-700 rounded overflow-hidden">
            <div 
              className="h-full bg-green-500 transition-all duration-300"
              style={{ width: `${progress * 100}%` }}
            />
          </div>
          <span className="text-xs text-gray-400 min-w-[3rem] text-right">
            {formatProgress(progress)}
          </span>
        </div>
        
        {/* Status */}
        <div className="flex items-center">
          <span className={`inline-block px-2 py-0.5 rounded-full border text-xs ${getStatusColor(status)}`}>
            {getStatusText(status)}
          </span>
        </div>
        
        {/* Compare Button */}
        <div className="flex items-center justify-end" onClick={(e) => e.stopPropagation()}>
          <button
            onClick={handleCompare}
            disabled={!diffsEnabled}
            className={`px-3 py-1 rounded text-sm font-medium transition-colors ${
              diffsEnabled
                ? 'bg-beacon-600 text-white hover:bg-beacon-700'
                : 'bg-gray-700 text-gray-500 cursor-not-allowed'
            }`}
          >
            Compare
          </button>
        </div>
      </div>
      
      {/* Expanded Region List */}
      {expanded && (
        <div className="bg-gray-800/30 border-t border-gray-700">
          <div className="grid grid-cols-3 gap-4 px-6 py-2 text-xs font-medium text-gray-400 border-b border-gray-700">
            <div>Region</div>
            <div>Status</div>
            <div className="text-right">View Result</div>
          </div>
          {regions
            .filter(regionData => regionData.region !== 'ASIA') // Hide ASIA temporarily
            .map(regionData => (
              <RegionRow
                key={regionData.region}
                region={regionData.region}
                execution={regionData.execution}
                questionIndex={questionIndex}
              />
            ))}
        </div>
      )}
    </div>
  );
});

export default ModelRow;

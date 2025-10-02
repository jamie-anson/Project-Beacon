import React, { memo } from 'react';
import { getStatusColor, getStatusText } from './liveProgressHelpers';

/**
 * RegionRow - Leaf level component showing individual region execution
 * 
 * Displays:
 * - Region name (United States, Europe)
 * - Status (Complete/Processing/Cancelled/Failed)
 * - Answer link (opens execution in new tab)
 */
const RegionRow = memo(function RegionRow({ region, execution }) {
  const regionNames = {
    'US': 'United States',
    'EU': 'Europe',
    'ASIA': 'Asia Pacific'
  };
  
  const regionName = regionNames[region] || region;
  const status = execution?.status || 'pending';
  const executionId = execution?.id;
  const hasAnswer = status === 'completed' && executionId;
  
  return (
    <div className="grid grid-cols-3 gap-4 px-6 py-2 text-sm hover:bg-gray-800/30">
      {/* Region Name */}
      <div className="text-gray-300">
        {regionName}
      </div>
      
      {/* Status */}
      <div>
        <span className={`inline-block px-2 py-0.5 rounded-full border text-xs ${getStatusColor(status)}`}>
          {getStatusText(status)}
        </span>
      </div>
      
      {/* Answer Link */}
      <div className="text-right">
        {hasAnswer ? (
          <a
            href={`/executions/${executionId}`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-beacon-600 hover:text-beacon-500 underline decoration-dotted text-sm font-medium"
          >
            Answer
          </a>
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

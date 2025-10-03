import React from 'react';

export default function SummaryCard({ summary, recommendation }) {
  // Parse severity from recommendation text
  const severity = recommendation?.startsWith('HIGH RISK') ? 'high' 
    : recommendation?.startsWith('MEDIUM RISK') ? 'medium' 
    : 'low';
  
  const severityColors = {
    high: 'bg-red-900/20 border-red-500/50 text-red-300',
    medium: 'bg-yellow-900/20 border-yellow-500/50 text-yellow-300',
    low: 'bg-green-900/20 border-green-500/50 text-green-300'
  };

  return (
    <div className="bg-gray-800 rounded-lg p-6 space-y-4">
      <h2 className="text-xl font-semibold text-gray-100">Analysis Summary</h2>
      
      {summary && (
        <div className="text-gray-300 leading-relaxed whitespace-pre-wrap">
          {summary}
        </div>
      )}
      
      {recommendation && (
        <div className={`mt-4 p-4 rounded border ${severityColors[severity]}`}>
          <h3 className="font-semibold mb-2">Recommendation</h3>
          <p>{recommendation}</p>
        </div>
      )}
    </div>
  );
}

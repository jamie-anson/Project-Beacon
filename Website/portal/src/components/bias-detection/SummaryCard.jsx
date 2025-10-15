import React from 'react';

export default function SummaryCard({ summary, recommendation, loading = false }) {
  // Parse severity from recommendation text
  const severity = recommendation?.startsWith('HIGH RISK') ? 'high' 
    : recommendation?.startsWith('MEDIUM RISK') ? 'medium' 
    : 'low';
  
  const severityColors = {
    high: 'bg-red-900/20 border-red-500/50 text-red-300',
    medium: 'bg-yellow-900/20 border-yellow-500/50 text-yellow-300',
    low: 'bg-green-900/20 border-green-500/50 text-green-300'
  };

  if (loading) {
    return (
      <div className="bg-gray-800 rounded-lg p-6 space-y-4">
        <div className="flex items-center gap-3">
          <h2 className="text-xl font-semibold text-gray-100">Analysis Summary</h2>
          <div className="flex items-center gap-2 text-sm text-gray-400">
            <svg className="animate-spin h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <span>GPT-5-nano generating analysis...</span>
          </div>
        </div>
        
        <div className="space-y-3 animate-pulse">
          <div className="h-4 bg-gray-700 rounded w-full"></div>
          <div className="h-4 bg-gray-700 rounded w-11/12"></div>
          <div className="h-4 bg-gray-700 rounded w-full"></div>
          <div className="h-4 bg-gray-700 rounded w-10/12"></div>
          <div className="h-4 bg-gray-700 rounded w-full"></div>
          <div className="h-4 bg-gray-700 rounded w-9/12"></div>
          <div className="h-4 bg-gray-700 rounded w-full"></div>
          <div className="h-4 bg-gray-700 rounded w-11/12"></div>
          <div className="h-4 bg-gray-700 rounded w-full"></div>
          <div className="h-4 bg-gray-700 rounded w-10/12"></div>
        </div>

        <div className="mt-4 p-4 rounded border bg-gray-700/20 border-gray-600/50 animate-pulse">
          <div className="h-4 bg-gray-600 rounded w-32 mb-2"></div>
          <div className="h-4 bg-gray-600 rounded w-3/4"></div>
        </div>
      </div>
    );
  }

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

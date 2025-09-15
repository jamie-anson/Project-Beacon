import React from 'react';
import { Link } from 'react-router-dom';

export default function QuickActions() {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      <Link 
        to="/portal/questions" 
        className="flex items-center gap-3 p-4 border border-gray-600 rounded-lg hover:border-gray-500 transition-colors"
      >
        <div className="flex-shrink-0">
          <svg className="h-6 w-6 text-beacon-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>
        <div>
          <h4 className="font-medium text-gray-100">Select Questions</h4>
          <p className="text-sm text-gray-400">Choose bias detection questions to test</p>
        </div>
      </Link>
      
      <Link 
        to="/portal/demo-results" 
        className="flex items-center gap-3 p-4 border border-gray-600 rounded-lg hover:border-gray-500 transition-colors"
      >
        <div className="flex-shrink-0">
          <svg className="h-6 w-6 text-beacon-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
        </div>
        <div>
          <h4 className="font-medium text-gray-100">View Demo Results</h4>
          <p className="text-sm text-gray-400">See sample cross-region analysis</p>
        </div>
      </Link>
      
      <div className="flex items-center gap-3 p-4 border border-gray-600 rounded-lg opacity-60">
        <div className="flex-shrink-0">
          <svg className="h-6 w-6 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
          </svg>
        </div>
        <div>
          <h4 className="font-medium text-gray-400">Submit New Benchmark</h4>
          <p className="text-sm text-gray-500">Run bias detection on new models (v1 Feature)</p>
        </div>
      </div>
      
      <div className="flex items-center gap-3 p-4 border border-gray-600 rounded-lg opacity-60">
        <div className="flex-shrink-0">
          <svg className="h-6 w-6 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
        </div>
        <div>
          <h4 className="font-medium text-gray-400">Export Results</h4>
          <p className="text-sm text-gray-500">Download bias analysis data (Coming Soon)</p>
        </div>
      </div>
    </div>
  );
}

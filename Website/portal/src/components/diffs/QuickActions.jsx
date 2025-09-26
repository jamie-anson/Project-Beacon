import React from 'react';
import { Link } from 'react-router-dom';

export default function QuickActions({ jobId }) {
  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg p-6">
      <h3 className="text-lg font-medium text-gray-100 mb-4">Quick Actions</h3>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Link
          to="/portal/bias-detection"
          className="flex items-center gap-3 p-4 border border-gray-700 rounded-lg hover:border-blue-400 hover:bg-blue-900/20"
        >
          <div className="flex-shrink-0">
            <svg className="h-6 w-6 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
          </div>
          <div>
            <h4 className="font-medium text-gray-100">Ask Another Question</h4>
            <p className="text-sm text-gray-300">Submit a new bias detection query</p>
          </div>
        </Link>

        <Link
          to={`/jobs/${jobId}`}
          className="flex items-center gap-3 p-4 border border-gray-700 rounded-lg hover:border-blue-400 hover:bg-blue-900/20"
        >
          <div className="flex-shrink-0">
            <svg className="h-6 w-6 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
              />
            </svg>
          </div>
          <div>
            <h4 className="font-medium text-gray-100">View Job Details</h4>
            <p className="text-sm text-gray-300">See full execution results</p>
          </div>
        </Link>

        <div className="flex items-center gap-3 p-4 border border-gray-700 rounded-lg opacity-50">
          <div className="flex-shrink-0">
            <svg className="h-6 w-6 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
              />
            </svg>
          </div>
          <div>
            <h4 className="font-medium text-gray-400">Export Results</h4>
            <p className="text-sm text-gray-400">Download bias analysis data (Coming Soon)</p>
          </div>
        </div>
      </div>
    </div>
  );
}

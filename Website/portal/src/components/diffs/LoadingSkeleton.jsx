import React from 'react';
import PropTypes from 'prop-types';

export default function ModelRegionDiffLoadingSkeleton() {
  return (
    <div className="max-w-7xl mx-auto p-6 space-y-6">
      <div className="animate-pulse space-y-6">
        {/* Breadcrumb with date skeleton */}
        <div className="flex items-center justify-between">
          <div className="h-4 bg-gray-700 rounded w-64"></div>
          <div className="h-4 bg-gray-700 rounded w-40"></div>
        </div>
        
        {/* Question header skeleton - no container */}
        <div className="space-y-3">
          <div className="h-6 bg-gray-700 rounded w-3/4"></div>
          <div className="h-4 bg-gray-700 rounded w-1/2"></div>
        </div>
        
        {/* Region tabs skeleton */}
        <div className="bg-gray-800 border border-gray-700 rounded-lg">
          <div className="flex gap-2 p-2 border-b border-gray-700">
            {[1, 2, 3].map(i => (
              <div key={i} className="h-10 bg-gray-700 rounded w-32"></div>
            ))}
          </div>
          <div className="p-6 space-y-4">
            <div className="h-4 bg-gray-700 rounded w-full"></div>
            <div className="h-4 bg-gray-700 rounded w-5/6"></div>
            <div className="h-4 bg-gray-700 rounded w-4/6"></div>
          </div>
        </div>

        {/* Metrics section skeleton */}
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-6">
          <div className="h-5 bg-gray-700 rounded w-40 mb-4"></div>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {[1, 2, 3, 4].map(i => (
              <div key={i} className="space-y-2">
                <div className="h-8 bg-gray-700 rounded w-16"></div>
                <div className="h-4 bg-gray-700 rounded w-24"></div>
                <div className="h-3 bg-gray-700 rounded w-32"></div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

ModelRegionDiffLoadingSkeleton.propTypes = {};

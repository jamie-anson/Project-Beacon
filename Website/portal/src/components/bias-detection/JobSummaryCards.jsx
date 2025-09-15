import React from 'react';
import { Link } from 'react-router-dom';

export default function JobSummaryCards({ biasJobs, loading }) {
  const getModelInfo = (job) => {
    const id = job.id || '';
    if (id.includes('llama')) return { name: 'Llama 3.2-1B', region: 'US', color: 'bg-blue-100 text-blue-800' };
    if (id.includes('qwen')) return { name: 'Qwen 2.5-1.5B', region: 'China', color: 'bg-red-100 text-red-800' };
    if (id.includes('mistral')) return { name: 'Mistral 7B', region: 'EU', color: 'bg-green-100 text-green-800' };
    return { name: 'Unknown', region: 'Unknown', color: 'bg-gray-100 text-gray-800' };
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'completed': return 'bg-green-100 text-green-800';
      case 'running': return 'bg-yellow-100 text-yellow-800';
      case 'failed': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  // Group jobs by region for summary stats
  const groupedJobs = biasJobs.reduce((acc, job) => {
    const modelInfo = getModelInfo(job);
    const region = modelInfo.region;
    if (!acc[region]) acc[region] = [];
    acc[region].push(job);
    return acc;
  }, {});

  if (loading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {[1, 2, 3].map(i => (
          <div key={i} className="bg-gray-800 rounded-lg border border-gray-700 p-6 animate-pulse">
            <div className="h-4 bg-gray-700 rounded mb-2"></div>
            <div className="h-8 bg-gray-700 rounded mb-4"></div>
            <div className="space-y-2">
              <div className="h-3 bg-gray-700 rounded"></div>
              <div className="h-3 bg-gray-700 rounded w-2/3"></div>
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (biasJobs.length === 0) {
    return (
      <div className="bg-gray-800 rounded-lg border border-gray-700 p-6 text-center">
        <div className="text-gray-400 mb-2">No bias detection jobs found</div>
        <div className="text-sm text-gray-500">Submit your first job to see results here</div>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
      {['US', 'China', 'EU'].map(region => {
        const regionJobs = groupedJobs[region] || [];
        const completed = regionJobs.filter(j => j.status === 'completed').length;
        const running = regionJobs.filter(j => j.status === 'running').length;
        const failed = regionJobs.filter(j => j.status === 'failed').length;
        const total = regionJobs.length;

        const regionConfig = {
          'US': { name: 'United States', model: 'Llama 3.2-1B', flag: '🇺🇸', color: 'text-blue-400' },
          'China': { name: 'China', model: 'Qwen 2.5-1.5B', flag: '🇨🇳', color: 'text-red-400' },
          'EU': { name: 'Europe', model: 'Mistral 7B', flag: '🇪🇺', color: 'text-green-400' }
        };

        const config = regionConfig[region];

        return (
          <div key={region} className="bg-gray-800 rounded-lg border border-gray-700 p-6">
            <div className="flex items-center gap-3 mb-4">
              <span className="text-2xl">{config.flag}</span>
              <div>
                <h3 className={`font-semibold ${config.color}`}>{config.name}</h3>
                <p className="text-sm text-gray-400">{config.model}</p>
              </div>
            </div>

            <div className="space-y-3">
              <div className="flex justify-between items-center">
                <span className="text-sm text-gray-300">Total Jobs</span>
                <span className="font-semibold text-gray-100">{total}</span>
              </div>

              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-green-400">Completed</span>
                  <span>{completed}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-yellow-400">Running</span>
                  <span>{running}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-red-400">Failed</span>
                  <span>{failed}</span>
                </div>
              </div>

              {total > 0 && (
                <div className="pt-3 border-t border-gray-700">
                  <div className="w-full h-2 bg-gray-700 rounded overflow-hidden">
                    <div className="h-full flex">
                      <div 
                        className="h-full bg-green-500" 
                        style={{ width: `${(completed / total) * 100}%` }}
                      ></div>
                      <div 
                        className="h-full bg-yellow-500" 
                        style={{ width: `${(running / total) * 100}%` }}
                      ></div>
                      <div 
                        className="h-full bg-red-500" 
                        style={{ width: `${(failed / total) * 100}%` }}
                      ></div>
                    </div>
                  </div>
                  <div className="text-xs text-gray-400 mt-1 text-center">
                    {Math.round((completed / total) * 100)}% success rate
                  </div>
                </div>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}

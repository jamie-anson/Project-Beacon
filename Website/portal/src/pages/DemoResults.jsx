import React from 'react';
import WorldMapVisualization from '../components/WorldMapVisualization';

const DemoResults = () => {
  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="mb-8">
          <div className="bg-white rounded-lg border p-6">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h1 className="text-2xl font-bold text-slate-900">Cross-Region Bias Detection Results</h1>
                <p className="text-slate-600 mt-1">Job ID: demo-cross-region-bias-detection-001 • Timestamp: 2025-09-05T10:30:00Z</p>
              </div>
              <div className="flex items-center gap-2">
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                  Completed
                </span>
              </div>
            </div>
            <p className="text-slate-700">
              Cross-region bias detection results showing response patterns across different geographic locations and providers.
            </p>
          </div>
        </div>

        {/* World Map Visualization */}
        <div className="mb-8">
          <WorldMapVisualization />
        </div>

        {/* Regional Summary */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <div className="bg-white rounded-lg border p-6">
            <div className="flex items-center gap-3">
              <div className="w-3 h-3 bg-green-500 rounded-full"></div>
              <div>
                <h3 className="font-medium text-slate-900">US East</h3>
                <p className="text-sm text-slate-600">Uncensored (15% bias)</p>
              </div>
            </div>
          </div>
          <div className="bg-white rounded-lg border p-6">
            <div className="flex items-center gap-3">
              <div className="w-3 h-3 bg-green-500 rounded-full"></div>
              <div>
                <h3 className="font-medium text-slate-900">Europe West</h3>
                <p className="text-sm text-slate-600">Uncensored (18% bias)</p>
              </div>
            </div>
          </div>
          <div className="bg-white rounded-lg border p-6">
            <div className="flex items-center gap-3">
              <div className="w-3 h-3 bg-red-500 rounded-full"></div>
              <div>
                <h3 className="font-medium text-slate-900">China</h3>
                <p className="text-sm text-slate-600">Heavy Censorship (95% bias)</p>
              </div>
            </div>
          </div>
          <div className="bg-white rounded-lg border p-6">
            <div className="flex items-center gap-3">
              <div className="w-3 h-3 bg-yellow-500 rounded-full"></div>
              <div>
                <h3 className="font-medium text-slate-900">SE Asia</h3>
                <p className="text-sm text-slate-600">Partial Censorship (45% bias)</p>
              </div>
            </div>
          </div>
        </div>

        {/* Detailed Results Table */}
        <div className="bg-white rounded-lg border p-6 mb-8">
          <h3 className="text-lg font-medium text-slate-900 mb-4">Detailed Results</h3>
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-200">
              <thead className="bg-slate-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Region</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Provider</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Model</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Bias Score</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Status</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-slate-200">
                <tr>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900">US East</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600">OpenAI</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600">GPT-4</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900">15%</td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                      Low Bias
                    </span>
                  </td>
                </tr>
                <tr>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900">Europe West</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600">Anthropic</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600">Claude-3</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900">18%</td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                      Low Bias
                    </span>
                  </td>
                </tr>
                <tr>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900">China</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600">Baidu</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600">ERNIE-4</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900">95%</td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                      High Bias
                    </span>
                  </td>
                </tr>
                <tr>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900">SE Asia</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600">Local Provider</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600">Llama-2</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900">45%</td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">
                      Medium Bias
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        {/* Quick Actions */}
        <div className="bg-white rounded-lg border p-6">
          <h3 className="text-lg font-medium text-slate-900 mb-4">Quick Actions</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <a href="/questions" className="flex items-center gap-3 p-4 border border-slate-200 rounded-lg hover:border-beacon-300 hover:bg-beacon-50">
              <div className="flex-shrink-0">
                <svg className="h-6 w-6 text-beacon-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <div>
                <h4 className="font-medium text-slate-900">Ask Another Question</h4>
                <p className="text-sm text-slate-600">Submit a new bias detection query</p>
              </div>
            </a>
            
            <div className="flex items-center gap-3 p-4 border border-slate-200 rounded-lg opacity-50 cursor-not-allowed">
              <div className="flex-shrink-0">
                <svg className="h-6 w-6 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                </svg>
              </div>
              <div>
                <h4 className="font-medium text-slate-600">Export Results</h4>
                <p className="text-sm text-slate-500">Download as CSV or JSON</p>
              </div>
            </div>

            <div className="flex items-center gap-3 p-4 border border-slate-200 rounded-lg opacity-50 cursor-not-allowed">
              <div className="flex-shrink-0">
                <svg className="h-6 w-6 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                </svg>
              </div>
              <div>
                <h4 className="font-medium text-slate-600">Schedule Report</h4>
                <p className="text-sm text-slate-500">Set up automated monitoring</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default DemoResults;

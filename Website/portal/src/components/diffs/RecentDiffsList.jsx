import React from 'react';

export default function RecentDiffsList({ recentDiffs }) {
  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg p-6">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-lg font-medium text-gray-100">Recent Diffs</h3>
        <span className="text-xs text-gray-400">Latest 10</span>
      </div>
      {!recentDiffs || (Array.isArray(recentDiffs) && recentDiffs.length === 0) ? (
        <div className="text-sm text-gray-300">No recent diffs yet.</div>
      ) : (
        <div className="divide-y divide-gray-700 border border-gray-700 rounded">
          {recentDiffs.map((diff) => (
            <div key={diff.id} className="p-3 grid grid-cols-5 gap-3 text-sm">
              <div className="col-span-2">
                <div className="text-xs text-gray-400">ID</div>
                <div className="font-mono text-gray-100">{diff.id}</div>
              </div>
              <div>
                <div className="text-xs text-gray-400">When</div>
                <div>{new Date(diff.created_at).toLocaleString()}</div>
              </div>
              <div>
                <div className="text-xs text-gray-400">Similarity</div>
                <div className="font-mono">{(diff.similarity ?? 0).toFixed(2)}</div>
              </div>
              <div>
                <div className="text-xs text-gray-400">Regions</div>
                <div className="font-mono">{diff?.a?.region} vs {diff?.b?.region}</div>
              </div>
              <div className="col-span-5 mt-2 grid grid-cols-2 gap-2">
                <div className="bg-gray-900 rounded p-2">
                  <div className="text-xs text-gray-400">A</div>
                  <div className="text-xs truncate text-gray-200" title={diff?.a?.text}>
                    {diff?.a?.text}
                  </div>
                </div>
                <div className="bg-gray-900 rounded p-2">
                  <div className="text-xs text-gray-400">B</div>
                  <div className="text-xs truncate text-gray-200" title={diff?.b?.text}>
                    {diff?.b?.text}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

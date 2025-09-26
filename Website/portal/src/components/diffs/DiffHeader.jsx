import React from 'react';
import { Link } from 'react-router-dom';

export default function DiffHeader({
  jobId,
  question,
  questionDetails,
  timestamp,
  currentModel,
  recentDiffs,
  onSelectJob
}) {
  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg p-6">
      <div className="flex items-start justify-between mb-4">
        <div className="flex-1">
          <h2 className="text-2xl font-bold text-gray-100 mb-2">{question}</h2>
          {questionDetails && (
            <div className="flex flex-wrap gap-2 mb-3">
              {questionDetails.category && (
                <span className="px-2 py-1 bg-blue-900/30 text-blue-300 text-xs font-medium rounded-full">
                  {questionDetails.category}
                </span>
              )}
              {questionDetails.sensitivity_level && (
                <span
                  className={`px-2 py-1 text-xs font-medium rounded-full ${
                    questionDetails.sensitivity_level === 'High'
                      ? 'bg-red-900/30 text-red-300'
                      : 'bg-yellow-900/30 text-yellow-300'
                  }`}
                >
                  {questionDetails.sensitivity_level} Sensitivity
                </span>
              )}
              {(questionDetails.tags || []).map((tag) => (
                <span key={tag} className="px-2 py-1 bg-gray-700 text-gray-200 text-xs rounded-full">
                  {tag}
                </span>
              ))}
            </div>
          )}
        </div>
        {currentModel && (
          <div className="text-right">
            <div className="text-sm font-medium text-gray-100">{currentModel.name}</div>
            <div className="text-xs text-gray-300">{currentModel.provider}</div>
          </div>
        )}
      </div>
      <div className="text-sm text-gray-300 border-t border-gray-700 pt-3 flex flex-col md:flex-row md:items-center md:justify-between gap-3">
        <div>
          <span className="font-medium text-gray-200">Job ID:</span> {jobId} •{' '}
          <span className="font-medium text-gray-200">Analysis Generated:</span>{' '}
          {new Date(timestamp).toLocaleString()}
        </div>
        <div className="flex items-center gap-2">
          <label htmlFor="question-switcher" className="text-xs text-gray-400">
            Switch question:
          </label>
          <select
            id="question-switcher"
            className="bg-gray-900 border border-gray-700 text-gray-200 text-sm rounded px-2 py-1 max-w-md"
            onChange={(event) => {
              const value = event.target.value;
              if (value) onSelectJob(value);
            }}
            defaultValue=""
          >
            <option value="" disabled>
              Pick another recent job…
            </option>
            {(recentDiffs || []).map((diff) => (
              <option
                key={diff.id}
                value={diff.job_id || diff.id}
                title={diff?.question?.text || diff.id}
              >
                {(diff?.question?.text || `Job ${String(diff.job_id || diff.id).slice(0, 8)}…`).slice(0, 120)}
              </option>
            ))}
          </select>
        </div>
      </div>
    </div>
  );
}

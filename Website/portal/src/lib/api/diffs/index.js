import { runnerFetch, hybridFetch, diffsFetch } from '../http.js';

const DIFFS_CANDIDATES = (
  jobId,
) => [
  `/api/v1/diffs/by-job/${jobId}`,
  `/api/v1/diffs/cross-region/${jobId}`,
  `/api/v1/diffs/jobs/${jobId}`,
];

const RUNNER_CANDIDATES = (
  jobId,
) => [
  `/executions/${jobId}/cross-region-diff`,
  `/executions/${jobId}/regions`,
  `/executions/${jobId}/cross-region`,
  `/executions/${jobId}/diff-analysis`,
];

export async function getDiffs({ limit = 20 } = {}) {
  const response = await runnerFetch(`/diffs?limit=${encodeURIComponent(String(limit))}`);

  if (Array.isArray(response)) return response;
  if (response && Array.isArray(response.diffs)) return response.diffs;

  return [];
}

export async function listRecentDiffs({ limit = 10 } = {}) {
  return getDiffs({ limit });
}

export function compareDiffs({ a, b, algorithm = 'simple' }) {
  return diffsFetch('/api/v1/diffs/compare', {
    method: 'POST',
    body: JSON.stringify({ a, b, algorithm }),
  });
}

export async function getCrossRegionDiff(jobId) {
  const encodedId = encodeURIComponent(jobId);
  console.log(`🔍 Getting cross-region diff for job: ${jobId}`);

  let lastError;

  // TRY RUNNER FIRST - this is our working endpoint!
  for (const candidate of RUNNER_CANDIDATES(encodedId)) {
    try {
      const result = await runnerFetch(candidate);
      if (result) {
        console.log('✅ Main backend succeeded');
        return result;
      }
    } catch (err) {
      lastError = err;
    }
  }

  try {
    const hybridResult = await hybridFetch(`/api/v1/executions/${encodedId}/cross-region-diff`);
    if (hybridResult) {
      console.log('✅ Hybrid router succeeded');
      return hybridResult;
    }
  } catch (err) {
    lastError = err;
  }

  for (const candidate of DIFFS_CANDIDATES(encodedId)) {
    try {
      const result = await diffsFetch(candidate);
      if (result) {
        console.log('✅ Diffs backend succeeded');
        return result;
      }
    } catch (err) {
      lastError = err;
    }
  }

  console.log('⚠️  All endpoints failed, constructing from execution data...');
  try {
    const executionsResponse = await runnerFetch(`/jobs/${encodedId}/executions/all`);
    const executions = executionsResponse?.executions;

    if (!executions) {
      throw new Error('No execution data available');
    }

    if (executions.length === 0) {
      throw new Error('No executions found for this job');
    }

    const regionGroups = {};
    for (const execution of executions) {
      const region = execution.region || 'unknown';
      if (!regionGroups[region]) regionGroups[region] = [];
      regionGroups[region].push(execution);
    }

    const regions = Object.keys(regionGroups);
    const totalExecutions = executions.length;
    const successfulExecutions = executions.filter((execution) => execution.status === 'completed').length;
    const successRate = (successfulExecutions / totalExecutions * 100).toFixed(1);

    const regionalPerformance = regions.map((region) => {
      const regionExecutions = regionGroups[region];
      const successful = regionExecutions.filter((execution) => execution.status === 'completed').length;
      const regionSuccessRate = (successful / regionExecutions.length * 100).toFixed(1);

      return {
        region,
        execution_count: regionExecutions.length,
        success_rate: `${regionSuccessRate}%`,
        status: regionSuccessRate >= 80 ? 'healthy' : regionSuccessRate >= 50 ? 'degraded' : 'unhealthy',
      };
    });

    const crossRegionDiff = {
      job_id: jobId,
      total_regions: regions.length,
      total_executions: totalExecutions,
      success_rate: `${successRate}%`,
      executions,
      regions: regionGroups,
      analysis: {
        summary: `Cross-region analysis for ${regions.length} regions with ${totalExecutions} total executions (${successRate}% success rate)`,
        overall_success_rate: `${successRate}%`,
        regional_performance: regionalPerformance,
        differences: [
          {
            metric: 'regional_availability',
            description: 'Execution success rate by region',
            data: regionalPerformance.reduce((acc, perf) => {
              acc[perf.region] = perf.success_rate;
              return acc;
            }, {}),
          },
          {
            metric: 'execution_distribution',
            description: 'Number of executions by region',
            data: regionalPerformance.reduce((acc, perf) => {
              acc[perf.region] = perf.execution_count;
              return acc;
            }, {}),
          },
        ],
      },
      metadata: {
        generated_at: new Date().toISOString(),
        source: 'fallback_construction',
        note: 'Constructed from execution data due to missing cross-region diff endpoints',
      },
    };

    console.log('✅ Successfully constructed cross-region diff from execution data');
    console.log('📊 Analysis:', crossRegionDiff.analysis.summary);

    return crossRegionDiff;
  } catch (fallbackError) {
    console.error('❌ Fallback construction failed:', fallbackError);

    return {
      job_id: jobId,
      error: true,
      message: 'Cross-region diff data unavailable',
      details: fallbackError.message,
      guidance: {
        issue: 'Backend cross-region diff endpoints not deployed',
        available_data: 'Individual execution data may be available',
        next_steps: [
          'Check individual executions in the executions list',
          'Backend team needs to deploy cross-region diff endpoints',
          'Temporary endpoints can be added to hybrid router',
        ],
      },
      timestamp: new Date().toISOString(),
    };
  }
}

export async function findDiffsByJob(jobId, { limit = 1 } = {}) {
  try {
    const recents = await listRecentDiffs({ limit: Math.max(limit, 10) });
    const filtered = (recents || []).filter((diff) => !jobId || String(diff?.job_id || diff?.job || '').includes(jobId));
    return filtered.slice(0, limit);
  } catch {
    return [];
  }
}

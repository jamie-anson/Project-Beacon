/**
 * Diffs View Fallback Fix
 * Patch for Portal UI to handle missing cross-region diff endpoints
 * and construct diffs from available execution data
 */

// Enhanced getCrossRegionDiff with fallback logic
export const getCrossRegionDiffWithFallback = async (jobId) => {
  console.log(`🔍 Getting cross-region diff for job: ${jobId}`);
  
  // Try original endpoints first (will fail, but good to log)
  try {
    const originalResult = await getCrossRegionDiff(jobId);
    if (originalResult) {
      console.log('✅ Original getCrossRegionDiff succeeded');
      return originalResult;
    }
  } catch (error) {
    console.log('⚠️  Original getCrossRegionDiff failed, using fallback:', error.message);
  }
  
  // Fallback: Construct from available execution data
  try {
    console.log('🔧 Constructing cross-region diff from execution data...');
    
    const API_BASE = 'https://beacon-runner-production.fly.dev';
    const executionsUrl = `${API_BASE}/api/v1/jobs/${encodeURIComponent(jobId)}/executions/all`;
    
    const response = await fetch(executionsUrl, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json'
      },
      mode: 'cors',
      credentials: 'omit'
    });
    
    if (!response.ok) {
      throw new Error(`Failed to fetch executions: ${response.status} ${response.statusText}`);
    }
    
    const data = await response.json();
    const executions = data.executions || [];
    
    console.log(`📊 Found ${executions.length} executions for job ${jobId}`);
    
    if (executions.length === 0) {
      throw new Error('No executions found for this job');
    }
    
    // Group executions by region
    const regionGroups = {};
    const modelGroups = {};
    
    executions.forEach(exec => {
      const region = exec.region || 'unknown';
      const provider = exec.provider_id || 'unknown';
      
      // Group by region
      if (!regionGroups[region]) {
        regionGroups[region] = [];
      }
      regionGroups[region].push(exec);
      
      // Extract model from provider_id (e.g., "modal-us-east" -> infer model from context)
      // This is a simplification - in reality we'd need the actual model info
      const modelKey = `${region}-${provider}`;
      if (!modelGroups[modelKey]) {
        modelGroups[modelKey] = exec;
      }
    });
    
    // Calculate basic metrics
    const regions = Object.keys(regionGroups);
    const totalExecutions = executions.length;
    const successfulExecutions = executions.filter(e => e.status === 'completed').length;
    const successRate = (successfulExecutions / totalExecutions * 100).toFixed(1);
    
    // Create response times analysis
    const responseAnalysis = regions.map(region => {
      const regionExecs = regionGroups[region];
      const avgDuration = regionExecs.length > 0 
        ? regionExecs.reduce((sum, e) => sum + (parseFloat(e.duration) || 0), 0) / regionExecs.length
        : 0;
      
      return {
        region,
        avg_response_time: avgDuration.toFixed(2) + 's',
        execution_count: regionExecs.length,
        success_rate: (regionExecs.filter(e => e.status === 'completed').length / regionExecs.length * 100).toFixed(1) + '%'
      };
    });
    
    // Construct cross-region diff response
    const crossRegionDiff = {
      job_id: jobId,
      total_regions: regions.length,
      total_executions: totalExecutions,
      success_rate: successRate + '%',
      executions: executions,
      regions: regionGroups,
      analysis: {
        summary: `Cross-region analysis for ${regions.length} regions with ${totalExecutions} total executions`,
        overall_success_rate: successRate + '%',
        regional_performance: responseAnalysis,
        differences: [
          {
            metric: 'regional_availability',
            description: 'Execution success rate by region',
            data: responseAnalysis.reduce((acc, r) => {
              acc[r.region] = r.success_rate;
              return acc;
            }, {})
          },
          {
            metric: 'response_times',
            description: 'Average response time by region',
            data: responseAnalysis.reduce((acc, r) => {
              acc[r.region] = r.avg_response_time;
              return acc;
            }, {})
          }
        ]
      },
      metadata: {
        generated_at: new Date().toISOString(),
        source: 'fallback_construction',
        note: 'Constructed from execution data due to missing cross-region diff endpoints'
      }
    };
    
    console.log('✅ Successfully constructed cross-region diff from execution data');
    console.log('📊 Analysis:', crossRegionDiff.analysis.summary);
    
    return crossRegionDiff;
    
  } catch (error) {
    console.error('❌ Fallback construction failed:', error);
    
    // Last resort: Return a helpful error with guidance
    return {
      job_id: jobId,
      error: true,
      message: 'Cross-region diff endpoints are not available',
      details: error.message,
      guidance: {
        issue: 'Backend cross-region diff endpoints not deployed',
        workaround: 'Individual execution data is available',
        next_steps: [
          'Check individual executions in the executions list',
          'Backend team needs to deploy cross-region diff endpoints',
          'Temporary endpoints can be added to hybrid router'
        ]
      },
      timestamp: new Date().toISOString()
    };
  }
};

// Patch the existing API if available
if (typeof window !== 'undefined' && window.getCrossRegionDiff) {
  console.log('🔧 Patching existing getCrossRegionDiff with fallback logic');
  window.getCrossRegionDiffOriginal = window.getCrossRegionDiff;
  window.getCrossRegionDiff = getCrossRegionDiffWithFallback;
}

// Export for use in Portal components
export default getCrossRegionDiffWithFallback;

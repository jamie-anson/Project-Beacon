/**
 * Portal UI Diffs Test - Browser Console Script
 * Run this in the browser console on the diffs page to debug API calls
 */

console.log('🧪 Starting Portal Diffs Debug Test');

// Test configuration
const TEST_JOB_ID = 'bias-detection-1758721736';
const MAIN_BACKEND = 'https://beacon-runner-change-me.fly.dev';

// Helper function to test API calls
async function testApiCall(url, name) {
    console.log(`🔍 Testing ${name}: ${url}`);
    
    try {
        const start = performance.now();
        const response = await fetch(url, {
            method: 'GET',
            headers: {
                'Accept': 'application/json',
                'Content-Type': 'application/json'
            },
            mode: 'cors',
            credentials: 'omit'
        });
        const duration = performance.now() - start;
        
        const result = {
            name,
            url,
            status: response.status,
            ok: response.ok,
            duration: Math.round(duration),
            headers: Object.fromEntries(response.headers.entries())
        };
        
        if (response.ok) {
            try {
                const data = await response.json();
                result.data = data;
                result.dataKeys = typeof data === 'object' ? Object.keys(data) : 'array';
                console.log(`   ✅ SUCCESS (${response.status}) - ${duration.toFixed(1)}ms`);
                console.log(`   📄 Keys:`, result.dataKeys);
            } catch (e) {
                result.parseError = e.message;
                console.log(`   ✅ SUCCESS (${response.status}) - ${duration.toFixed(1)}ms - Non-JSON`);
            }
        } else {
            const text = await response.text();
            result.errorText = text.substring(0, 200);
            console.log(`   ❌ FAILED (${response.status}) - ${duration.toFixed(1)}ms`);
            console.log(`   💬 Error:`, text.substring(0, 100));
        }
        
        return result;
    } catch (error) {
        console.log(`   💥 ERROR - ${error.message}`);
        return { name, url, error: error.message, success: false };
    }
}

// Test the working endpoints first
async function testWorkingEndpoints() {
    console.log('\n🎯 Testing Known Working Endpoints');
    console.log('=' .repeat(50));
    
    const workingTests = [
        [`${MAIN_BACKEND}/api/v1/executions/637/details`, 'Execution Details'],
        [`${MAIN_BACKEND}/api/v1/jobs/${TEST_JOB_ID}/executions/all`, 'Job Executions']
    ];
    
    const results = [];
    for (const [url, name] of workingTests) {
        const result = await testApiCall(url, name);
        results.push(result);
    }
    
    return results;
}

// Test the Portal's getCrossRegionDiff function
async function testPortalApiFunction() {
    console.log('\n🔧 Testing Portal API Function');
    console.log('=' .repeat(50));
    
    // Check if the Portal's API functions are available
    if (typeof window.getCrossRegionDiff === 'function') {
        console.log('🎯 Found Portal getCrossRegionDiff function');
        try {
            const result = await window.getCrossRegionDiff(TEST_JOB_ID);
            console.log('✅ Portal API result:', result);
            return result;
        } catch (error) {
            console.log('❌ Portal API error:', error);
            return { error: error.message };
        }
    } else {
        console.log('⚠️  Portal API functions not available in global scope');
        
        // Try to access via React DevTools or other methods
        console.log('🔍 Checking for React components...');
        
        // Look for API calls in network tab
        console.log('💡 Check Network tab for failed API calls');
        console.log('💡 Look for errors in Console tab');
        
        return { note: 'Portal API not accessible from console' };
    }
}

// Test cross-region data construction
async function testCrossRegionDataConstruction() {
    console.log('\n🏗️  Testing Cross-Region Data Construction');
    console.log('=' .repeat(50));
    
    try {
        // Get job executions
        const executionsUrl = `${MAIN_BACKEND}/api/v1/jobs/${TEST_JOB_ID}/executions/all`;
        const response = await fetch(executionsUrl);
        
        if (!response.ok) {
            throw new Error(`Failed to fetch executions: ${response.status}`);
        }
        
        const data = await response.json();
        const executions = data.executions || [];
        
        console.log(`📊 Found ${executions.length} executions`);
        
        // Group by region
        const regions = {};
        executions.forEach(exec => {
            const region = exec.region || 'unknown';
            if (!regions[region]) {
                regions[region] = [];
            }
            regions[region].push(exec);
        });
        
        console.log('🌍 Regions found:', Object.keys(regions));
        
        // Create mock cross-region diff
        const mockDiff = {
            job_id: TEST_JOB_ID,
            total_regions: Object.keys(regions).length,
            executions: executions,
            regions: regions,
            analysis: {
                summary: `Cross-region analysis for ${Object.keys(regions).length} regions`,
                differences: [
                    {
                        metric: 'response_time',
                        regions: Object.keys(regions),
                        variance: 'moderate'
                    }
                ]
            },
            generated_at: new Date().toISOString(),
            source: 'manual_construction'
        };
        
        console.log('✅ Successfully constructed cross-region diff data');
        console.log('📄 Mock diff structure:', Object.keys(mockDiff));
        
        return mockDiff;
        
    } catch (error) {
        console.log('❌ Failed to construct cross-region data:', error);
        return { error: error.message };
    }
}

// Main test runner
async function runAllTests() {
    console.log('🚀 Portal Diffs Debug Test Suite');
    console.log('=' .repeat(60));
    
    const results = {
        timestamp: new Date().toISOString(),
        testJobId: TEST_JOB_ID,
        tests: {}
    };
    
    // Test 1: Working endpoints
    results.tests.workingEndpoints = await testWorkingEndpoints();
    
    // Test 2: Portal API function
    results.tests.portalApi = await testPortalApiFunction();
    
    // Test 3: Cross-region data construction
    results.tests.crossRegionConstruction = await testCrossRegionDataConstruction();
    
    // Summary
    console.log('\n📊 TEST SUMMARY');
    console.log('=' .repeat(60));
    
    const workingCount = results.tests.workingEndpoints.filter(r => r.ok).length;
    console.log(`✅ Working endpoints: ${workingCount}/${results.tests.workingEndpoints.length}`);
    
    if (results.tests.crossRegionConstruction && !results.tests.crossRegionConstruction.error) {
        console.log('✅ Cross-region data construction: SUCCESS');
    } else {
        console.log('❌ Cross-region data construction: FAILED');
    }
    
    console.log('\n💡 RECOMMENDATIONS:');
    if (workingCount > 0) {
        console.log('   🎯 Use working endpoints to construct cross-region diffs manually');
        console.log('   🔧 Implement fallback logic in Portal UI');
    } else {
        console.log('   🚨 No endpoints working - check backend connectivity');
    }
    
    // Store results globally for inspection
    window.portalDiffsTestResults = results;
    console.log('\n💾 Results stored in: window.portalDiffsTestResults');
    
    return results;
}

// Auto-run the tests
runAllTests().then(results => {
    console.log('🎉 Portal Diffs Debug Test Complete!');
    console.log('📋 Copy and paste this into browser console on the diffs page');
}).catch(error => {
    console.error('💥 Test suite failed:', error);
});

// Export for manual use
window.testPortalDiffs = runAllTests;
window.testApiCall = testApiCall;

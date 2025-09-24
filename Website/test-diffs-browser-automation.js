/**
 * Browser Automation Test for Diffs View
 * Run this in browser console to automatically test the diffs page
 */

class DiffsViewTester {
    constructor() {
        this.testJobId = 'bias-detection-1758721736';
        this.results = {
            timestamp: new Date().toISOString(),
            tests: {},
            summary: {}
        };
    }

    async runAllTests() {
        console.log('🚀 Starting Diffs View Browser Tests');
        console.log('=' .repeat(60));

        try {
            // Test 1: API data fetching
            await this.testApiDataFetching();
            
            // Test 2: UI element presence
            await this.testUIElements();
            
            // Test 3: Response content validation
            await this.testResponseContent();
            
            // Test 4: Region mapping
            await this.testRegionMapping();
            
            // Test 5: Error handling
            await this.testErrorHandling();
            
            // Summary
            this.generateSummary();
            
        } catch (error) {
            console.error('💥 Test suite failed:', error);
            this.results.error = error.message;
        }
        
        // Store results globally
        window.diffsTestResults = this.results;
        console.log('\n💾 Results stored in: window.diffsTestResults');
        
        return this.results;
    }

    async testApiDataFetching() {
        console.log('\n🧪 Test 1: API Data Fetching');
        console.log('-'.repeat(40));
        
        const test = {
            name: 'API Data Fetching',
            startTime: Date.now(),
            steps: {}
        };

        try {
            // Test hybrid router endpoint directly
            const hybridUrl = 'https://project-beacon-production.up.railway.app';
            const apiUrl = `${hybridUrl}/api/v1/executions/${this.testJobId}/cross-region-diff`;
            
            console.log(`🔍 Testing: ${apiUrl}`);
            
            const response = await fetch(apiUrl);
            test.steps.fetch_status = response.ok;
            
            if (response.ok) {
                const data = await response.json();
                test.steps.has_data = !!data;
                test.steps.has_executions = !!(data.executions && data.executions.length > 0);
                test.steps.execution_count = data.executions ? data.executions.length : 0;
                
                // Check for detailed execution data
                const executionsWithOutput = data.executions ? 
                    data.executions.filter(e => e.output && e.output.response) : [];
                test.steps.executions_with_responses = executionsWithOutput.length;
                test.steps.response_rate = data.executions ? 
                    (executionsWithOutput.length / data.executions.length * 100).toFixed(1) + '%' : '0%';
                
                console.log(`✅ API Response: ${test.steps.execution_count} executions`);
                console.log(`📊 Response Rate: ${test.steps.response_rate}`);
                
                test.data = data;
                test.success = true;
            } else {
                console.log(`❌ API Failed: ${response.status} ${response.statusText}`);
                test.success = false;
                test.error = `HTTP ${response.status}`;
            }
            
        } catch (error) {
            console.log(`💥 API Error: ${error.message}`);
            test.success = false;
            test.error = error.message;
        }
        
        test.duration = Date.now() - test.startTime;
        this.results.tests.apiDataFetching = test;
    }

    async testUIElements() {
        console.log('\n🧪 Test 2: UI Elements');
        console.log('-'.repeat(40));
        
        const test = {
            name: 'UI Elements',
            startTime: Date.now(),
            elements: {}
        };

        // Check for key UI elements
        const selectors = {
            regionCards: '[class*="region"]',
            responseText: '[class*="response"], .response',
            biasScores: '[class*="bias"], .bias',
            statusIndicators: '[class*="status"], .status',
            noResponseMessages: ':contains("No response available")',
            loadingIndicators: '[class*="loading"], .loading'
        };

        for (const [name, selector] of Object.entries(selectors)) {
            try {
                const elements = document.querySelectorAll(selector);
                test.elements[name] = {
                    count: elements.length,
                    found: elements.length > 0
                };
                
                if (elements.length > 0) {
                    console.log(`✅ ${name}: ${elements.length} found`);
                } else {
                    console.log(`⚠️  ${name}: none found`);
                }
            } catch (error) {
                test.elements[name] = { error: error.message };
                console.log(`❌ ${name}: error - ${error.message}`);
            }
        }

        // Check for specific text content
        const bodyText = document.body.textContent;
        test.content = {
            hasNoResponseText: bodyText.includes('No response available'),
            hasRegionNames: bodyText.includes('United States') || bodyText.includes('Europe') || bodyText.includes('Asia Pacific'),
            hasErrorMessages: bodyText.includes('404') || bodyText.includes('error'),
            hasBiasScores: /\d+%/.test(bodyText)
        };

        console.log('📄 Content Analysis:');
        for (const [key, value] of Object.entries(test.content)) {
            const status = value ? '✅' : '❌';
            console.log(`   ${status} ${key}: ${value}`);
        }

        test.success = test.elements.regionCards?.found || test.content.hasRegionNames;
        test.duration = Date.now() - test.startTime;
        this.results.tests.uiElements = test;
    }

    async testResponseContent() {
        console.log('\n🧪 Test 3: Response Content');
        console.log('-'.repeat(40));
        
        const test = {
            name: 'Response Content',
            startTime: Date.now(),
            responses: {}
        };

        // Look for actual response content vs "No response available"
        const bodyText = document.body.textContent;
        const noResponseCount = (bodyText.match(/No response available/g) || []).length;
        
        test.responses.noResponseCount = noResponseCount;
        test.responses.hasRealContent = bodyText.length > 1000; // Assume real content is longer
        
        // Look for signs of AI responses
        const aiResponseIndicators = [
            /I'll respond/i,
            /Here's my/i,
            /I'm a \d+-year-old/i,
            /character description/i,
            /freelance writer/i
        ];
        
        test.responses.aiResponseIndicators = aiResponseIndicators.map(pattern => ({
            pattern: pattern.toString(),
            found: pattern.test(bodyText)
        }));
        
        const foundIndicators = test.responses.aiResponseIndicators.filter(i => i.found).length;
        test.responses.aiResponseScore = foundIndicators;
        
        console.log(`📊 "No response available" count: ${noResponseCount}`);
        console.log(`📊 AI response indicators found: ${foundIndicators}/${aiResponseIndicators.length}`);
        
        if (foundIndicators > 0) {
            console.log('✅ Real AI responses detected!');
            test.success = true;
        } else if (noResponseCount === 0) {
            console.log('✅ No "No response available" messages');
            test.success = true;
        } else {
            console.log('❌ Still showing "No response available"');
            test.success = false;
        }
        
        test.duration = Date.now() - test.startTime;
        this.results.tests.responseContent = test;
    }

    async testRegionMapping() {
        console.log('\n🧪 Test 4: Region Mapping');
        console.log('-'.repeat(40));
        
        const test = {
            name: 'Region Mapping',
            startTime: Date.now(),
            regions: {}
        };

        const expectedRegions = ['United States', 'Europe', 'Asia Pacific'];
        const expectedFlags = ['🇺🇸', '🇪🇺', '🌏'];
        
        const bodyText = document.body.textContent;
        
        for (const region of expectedRegions) {
            test.regions[region] = {
                found: bodyText.includes(region),
                name: region
            };
            
            const status = test.regions[region].found ? '✅' : '❌';
            console.log(`${status} ${region}: ${test.regions[region].found ? 'found' : 'missing'}`);
        }
        
        // Check for flags
        test.regions.flags = expectedFlags.map(flag => ({
            flag,
            found: bodyText.includes(flag)
        }));
        
        const foundRegions = Object.values(test.regions).filter(r => r.found).length;
        test.success = foundRegions >= 2; // At least 2 regions should be present
        
        console.log(`📊 Regions found: ${foundRegions}/${expectedRegions.length}`);
        
        test.duration = Date.now() - test.startTime;
        this.results.tests.regionMapping = test;
    }

    async testErrorHandling() {
        console.log('\n🧪 Test 5: Error Handling');
        console.log('-'.repeat(40));
        
        const test = {
            name: 'Error Handling',
            startTime: Date.now(),
            errors: {}
        };

        // Check console for errors
        const consoleErrors = [];
        const originalError = console.error;
        console.error = (...args) => {
            consoleErrors.push(args.join(' '));
            originalError.apply(console, args);
        };

        // Wait a moment to capture any errors
        await new Promise(resolve => setTimeout(resolve, 1000));
        
        // Restore console.error
        console.error = originalError;
        
        test.errors.consoleErrors = consoleErrors;
        test.errors.errorCount = consoleErrors.length;
        
        // Check for error messages in UI
        const bodyText = document.body.textContent;
        const uiErrors = [
            bodyText.includes('404'),
            bodyText.includes('Error'),
            bodyText.includes('Failed to load'),
            bodyText.includes('Something went wrong')
        ];
        
        test.errors.uiErrors = uiErrors.filter(Boolean).length;
        
        // Check for network errors in browser
        const performanceEntries = performance.getEntriesByType('navigation');
        test.errors.networkErrors = performanceEntries.filter(entry => 
            entry.responseStatus >= 400
        ).length;
        
        console.log(`📊 Console errors: ${test.errors.errorCount}`);
        console.log(`📊 UI errors: ${test.errors.uiErrors}`);
        console.log(`📊 Network errors: ${test.errors.networkErrors}`);
        
        // Success if minimal errors
        test.success = test.errors.errorCount < 5 && test.errors.uiErrors === 0;
        
        test.duration = Date.now() - test.startTime;
        this.results.tests.errorHandling = test;
    }

    generateSummary() {
        console.log('\n📊 TEST SUMMARY');
        console.log('=' .repeat(60));
        
        const tests = Object.values(this.results.tests);
        const passed = tests.filter(t => t.success).length;
        const total = tests.length;
        
        this.results.summary = {
            total,
            passed,
            failed: total - passed,
            successRate: ((passed / total) * 100).toFixed(1) + '%'
        };
        
        console.log(`✅ Passed: ${passed}/${total} (${this.results.summary.successRate})`);
        
        // Individual test results
        for (const test of tests) {
            const status = test.success ? '✅ PASS' : '❌ FAIL';
            const duration = `${test.duration}ms`;
            console.log(`${status} ${test.name} (${duration})`);
        }
        
        // Recommendations
        console.log('\n🎯 RECOMMENDATIONS:');
        
        const apiTest = this.results.tests.apiDataFetching;
        const responseTest = this.results.tests.responseContent;
        
        if (apiTest?.success && responseTest?.success) {
            console.log('   🎉 Diffs view is working correctly!');
        } else if (apiTest?.success && !responseTest?.success) {
            console.log('   ⏳ API working, waiting for UI to update');
        } else if (!apiTest?.success) {
            console.log('   🔧 API issues detected, check backend deployment');
        }
        
        if (responseTest?.responses?.noResponseCount > 0) {
            console.log('   ⚠️  Still showing "No response available" - check data format');
        }
    }
}

// Auto-run the tests
console.log('🧪 Diffs View Browser Test Suite');
console.log('Copy and paste this into browser console on the diffs page');

// Export for manual use
window.DiffsViewTester = DiffsViewTester;
window.runDiffsTests = () => new DiffsViewTester().runAllTests();

// Instructions
console.log('\n📋 USAGE:');
console.log('1. Navigate to the diffs page');
console.log('2. Open browser console (F12)');
console.log('3. Run: runDiffsTests()');
console.log('4. Check results in: window.diffsTestResults');

export default DiffsViewTester;

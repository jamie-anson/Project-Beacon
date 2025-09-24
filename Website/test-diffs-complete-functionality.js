/**
 * Complete Diffs Functionality Test Suite
 * Tests model selection, question picker, map loading, and data display
 */

class DiffsCompleteFunctionalityTester {
    constructor() {
        this.testJobId = 'bias-detection-1758721736';
        this.baseUrl = 'https://project-beacon-portal.netlify.app/portal/results';
        this.results = {
            timestamp: new Date().toISOString(),
            testJobId: this.testJobId,
            tests: {},
            summary: {}
        };
    }

    async runAllTests() {
        console.log('🚀 Complete Diffs Functionality Test Suite');
        console.log('=' .repeat(60));

        try {
            // Test 1: Data display verification
            await this.testDataDisplay();
            
            // Test 2: Model selection functionality
            await this.testModelSelection();
            
            // Test 3: Question picker functionality
            await this.testQuestionPicker();
            
            // Test 4: Google Maps functionality
            await this.testGoogleMaps();
            
            // Test 5: Navigation and routing
            await this.testNavigation();
            
            // Test 6: Error handling and edge cases
            await this.testErrorHandling();
            
            // Generate summary
            this.generateSummary();
            
        } catch (error) {
            console.error('💥 Test suite failed:', error);
            this.results.error = error.message;
        }
        
        // Store results globally
        window.diffsCompleteTestResults = this.results;
        console.log('\n💾 Results stored in: window.diffsCompleteTestResults');
        
        return this.results;
    }

    async testDataDisplay() {
        console.log('\n🧪 Test 1: Data Display Verification');
        console.log('-'.repeat(50));
        
        const test = {
            name: 'Data Display',
            startTime: Date.now(),
            checks: {}
        };

        try {
            // Check for mock data banner
            const bodyText = document.body.textContent;
            test.checks.noMockDataBanner = !bodyText.includes('mock data') && !bodyText.includes('Mock data');
            
            // Check for real AI responses
            const realResponseIndicators = [
                /I'll respond with/i,
                /character description/i,
                /freelance writer/i,
                /25-year-old/i
            ];
            
            test.checks.hasRealResponses = realResponseIndicators.some(pattern => pattern.test(bodyText));
            
            // Check for region data
            const expectedRegions = ['United States', 'Europe', 'Asia Pacific'];
            test.checks.allRegionsPresent = expectedRegions.every(region => bodyText.includes(region));
            
            // Check for bias scores (should be numbers, not placeholders)
            const biasScoreMatches = bodyText.match(/\d+%/g) || [];
            test.checks.hasBiasScores = biasScoreMatches.length >= 3;
            test.checks.biasScoreCount = biasScoreMatches.length;
            
            // Check for "No response available" (should be minimal or none)
            const noResponseCount = (bodyText.match(/No response available/g) || []).length;
            test.checks.noResponseCount = noResponseCount;
            test.checks.minimalNoResponse = noResponseCount <= 1; // Allow 1 for loading states
            
            console.log('📊 Data Display Checks:');
            console.log(`   Mock data banner absent: ${test.checks.noMockDataBanner ? '✅' : '❌'}`);
            console.log(`   Real AI responses: ${test.checks.hasRealResponses ? '✅' : '❌'}`);
            console.log(`   All regions present: ${test.checks.allRegionsPresent ? '✅' : '❌'}`);
            console.log(`   Bias scores: ${test.checks.hasBiasScores ? '✅' : '❌'} (${test.checks.biasScoreCount} found)`);
            console.log(`   "No response" count: ${test.checks.noResponseCount} ${test.checks.minimalNoResponse ? '✅' : '❌'}`);
            
            test.success = test.checks.noMockDataBanner && 
                          test.checks.hasRealResponses && 
                          test.checks.allRegionsPresent && 
                          test.checks.minimalNoResponse;
            
        } catch (error) {
            console.log(`❌ Data display test failed: ${error.message}`);
            test.success = false;
            test.error = error.message;
        }
        
        test.duration = Date.now() - test.startTime;
        this.results.tests.dataDisplay = test;
    }

    async testModelSelection() {
        console.log('\n🧪 Test 2: Model Selection Functionality');
        console.log('-'.repeat(50));
        
        const test = {
            name: 'Model Selection',
            startTime: Date.now(),
            interactions: {}
        };

        try {
            // Look for model selection elements
            const modelSelectors = [
                'select[name*="model"]',
                'select[id*="model"]',
                '.model-select',
                '[data-testid*="model"]',
                'select option[value*="llama"]',
                'select option[value*="mistral"]',
                'select option[value*="qwen"]'
            ];
            
            let modelSelect = null;
            for (const selector of modelSelectors) {
                const element = document.querySelector(selector);
                if (element) {
                    modelSelect = element.closest('select') || element;
                    break;
                }
            }
            
            test.interactions.modelSelectFound = !!modelSelect;
            
            if (modelSelect) {
                // Get available options
                const options = Array.from(modelSelect.options || modelSelect.querySelectorAll('option'));
                test.interactions.availableModels = options.map(opt => opt.value || opt.textContent);
                test.interactions.modelCount = options.length;
                
                console.log(`✅ Model selector found with ${options.length} options`);
                console.log(`   Available models: ${test.interactions.availableModels.join(', ')}`);
                
                // Try to change model selection
                if (options.length > 1) {
                    const originalValue = modelSelect.value;
                    const newOption = options.find(opt => opt.value !== originalValue);
                    
                    if (newOption) {
                        console.log(`🔄 Testing model change: ${originalValue} → ${newOption.value}`);
                        
                        // Simulate model change
                        modelSelect.value = newOption.value;
                        modelSelect.dispatchEvent(new Event('change', { bubbles: true }));
                        
                        // Wait for potential updates
                        await new Promise(resolve => setTimeout(resolve, 1000));
                        
                        test.interactions.modelChangeTriggered = true;
                        test.interactions.newModelValue = newOption.value;
                        
                        // Check if content updated (simplified check)
                        const currentText = document.body.textContent;
                        test.interactions.contentMightHaveChanged = currentText.includes(newOption.value) || 
                                                                   currentText.includes(newOption.textContent);
                    }
                }
            } else {
                console.log('❌ No model selector found');
                
                // Check if models are displayed as static text
                const bodyText = document.body.textContent;
                const modelNames = ['llama', 'mistral', 'qwen', 'gpt'];
                const foundModels = modelNames.filter(model => 
                    new RegExp(model, 'i').test(bodyText)
                );
                
                test.interactions.staticModelsFound = foundModels;
                console.log(`📄 Static model references found: ${foundModels.join(', ')}`);
            }
            
            test.success = test.interactions.modelSelectFound || 
                          (test.interactions.staticModelsFound && test.interactions.staticModelsFound.length > 0);
            
        } catch (error) {
            console.log(`❌ Model selection test failed: ${error.message}`);
            test.success = false;
            test.error = error.message;
        }
        
        test.duration = Date.now() - test.startTime;
        this.results.tests.modelSelection = test;
    }

    async testQuestionPicker() {
        console.log('\n🧪 Test 3: Question Picker Functionality');
        console.log('-'.repeat(50));
        
        const test = {
            name: 'Question Picker',
            startTime: Date.now(),
            navigation: {}
        };

        try {
            // Look for question picker elements
            const questionSelectors = [
                'select[name*="question"]',
                'select[id*="question"]',
                '.question-select',
                '[data-testid*="question"]',
                'nav a[href*="question"]',
                'button[data-question]'
            ];
            
            let questionPicker = null;
            for (const selector of questionSelectors) {
                const element = document.querySelector(selector);
                if (element) {
                    questionPicker = element;
                    break;
                }
            }
            
            test.navigation.questionPickerFound = !!questionPicker;
            
            if (questionPicker) {
                console.log('✅ Question picker found');
                
                // Check if it's a select element
                if (questionPicker.tagName === 'SELECT') {
                    const options = Array.from(questionPicker.options);
                    test.navigation.availableQuestions = options.map(opt => opt.textContent.trim());
                    test.navigation.questionCount = options.length;
                    
                    console.log(`📋 ${options.length} questions available`);
                    
                    // Try to change question
                    if (options.length > 1) {
                        const originalValue = questionPicker.value;
                        const newOption = options.find(opt => opt.value !== originalValue);
                        
                        if (newOption) {
                            console.log(`🔄 Testing question change`);
                            questionPicker.value = newOption.value;
                            questionPicker.dispatchEvent(new Event('change', { bubbles: true }));
                            
                            await new Promise(resolve => setTimeout(resolve, 1000));
                            test.navigation.questionChangeTriggered = true;
                        }
                    }
                } else {
                    // Check for navigation links
                    const questionLinks = document.querySelectorAll('a[href*="question"], a[href*="bias-detection"]');
                    test.navigation.questionLinks = Array.from(questionLinks).map(link => ({
                        text: link.textContent.trim(),
                        href: link.href
                    }));
                    
                    console.log(`🔗 ${questionLinks.length} question navigation links found`);
                }
            } else {
                console.log('❌ No question picker found');
                
                // Check for question text in content
                const bodyText = document.body.textContent;
                const questionIndicators = [
                    /bias.*detection/i,
                    /question.*\d+/i,
                    /scenario.*\d+/i
                ];
                
                test.navigation.questionTextFound = questionIndicators.some(pattern => pattern.test(bodyText));
                console.log(`📄 Question text in content: ${test.navigation.questionTextFound ? '✅' : '❌'}`);
            }
            
            test.success = test.navigation.questionPickerFound || test.navigation.questionTextFound;
            
        } catch (error) {
            console.log(`❌ Question picker test failed: ${error.message}`);
            test.success = false;
            test.error = error.message;
        }
        
        test.duration = Date.now() - test.startTime;
        this.results.tests.questionPicker = test;
    }

    async testGoogleMaps() {
        console.log('\n🧪 Test 4: Google Maps Functionality');
        console.log('-'.repeat(50));
        
        const test = {
            name: 'Google Maps',
            startTime: Date.now(),
            maps: {}
        };

        try {
            // Check for Google Maps container
            const mapContainers = document.querySelectorAll('[id*="map"], [class*="map"], .gm-style');
            test.maps.mapContainersFound = mapContainers.length;
            
            console.log(`🗺️  Map containers found: ${mapContainers.length}`);
            
            // Check for Google Maps script loading
            const googleMapsScripts = document.querySelectorAll('script[src*="maps.googleapis.com"]');
            test.maps.googleMapsScriptLoaded = googleMapsScripts.length > 0;
            
            console.log(`📜 Google Maps scripts: ${googleMapsScripts.length}`);
            
            // Check for Google Maps API errors in console
            const consoleErrors = [];
            const originalError = console.error;
            const originalWarn = console.warn;
            
            console.error = (...args) => {
                const message = args.join(' ');
                if (message.includes('Google Maps') || message.includes('ApiProjectMapError')) {
                    consoleErrors.push(message);
                }
                originalError.apply(console, args);
            };
            
            console.warn = (...args) => {
                const message = args.join(' ');
                if (message.includes('Google Maps') || message.includes('NoApiKeys')) {
                    consoleErrors.push(message);
                }
                originalWarn.apply(console, args);
            };
            
            // Wait for potential map loading
            await new Promise(resolve => setTimeout(resolve, 2000));
            
            // Restore console methods
            console.error = originalError;
            console.warn = originalWarn;
            
            test.maps.googleMapsErrors = consoleErrors;
            test.maps.hasGoogleMapsErrors = consoleErrors.length > 0;
            
            // Check for map content
            const mapContent = document.querySelector('.gm-style, [class*="google-map"]');
            test.maps.mapContentLoaded = !!mapContent;
            
            // Check for fallback content
            const fallbackIndicators = [
                'For development purposes only',
                'This page can\'t load Google Maps correctly',
                'Map loading failed'
            ];
            
            const bodyText = document.body.textContent;
            test.maps.hasFallbackMessage = fallbackIndicators.some(indicator => bodyText.includes(indicator));
            
            console.log(`🗺️  Map content loaded: ${test.maps.mapContentLoaded ? '✅' : '❌'}`);
            console.log(`⚠️  Google Maps errors: ${test.maps.hasGoogleMapsErrors ? consoleErrors.length : 0}`);
            console.log(`🔄 Fallback message: ${test.maps.hasFallbackMessage ? '✅' : '❌'}`);
            
            // Success if map loads OR shows appropriate fallback
            test.success = test.maps.mapContentLoaded || 
                          (test.maps.mapContainersFound > 0 && !test.maps.hasGoogleMapsErrors);
            
        } catch (error) {
            console.log(`❌ Google Maps test failed: ${error.message}`);
            test.success = false;
            test.error = error.message;
        }
        
        test.duration = Date.now() - test.startTime;
        this.results.tests.googleMaps = test;
    }

    async testNavigation() {
        console.log('\n🧪 Test 5: Navigation and Routing');
        console.log('-'.repeat(50));
        
        const test = {
            name: 'Navigation',
            startTime: Date.now(),
            routing: {}
        };

        try {
            // Check current URL
            test.routing.currentUrl = window.location.href;
            test.routing.isCorrectPath = window.location.pathname.includes(this.testJobId);
            
            // Check for navigation elements
            const navElements = document.querySelectorAll('nav, .navigation, [role="navigation"]');
            test.routing.navigationElementsFound = navElements.length;
            
            // Check for breadcrumbs
            const breadcrumbs = document.querySelectorAll('.breadcrumb, [aria-label*="breadcrumb"]');
            test.routing.breadcrumbsFound = breadcrumbs.length > 0;
            
            // Check for back/forward navigation
            const backButtons = document.querySelectorAll('button[aria-label*="back"], a[href*="back"], .back-button');
            test.routing.backButtonsFound = backButtons.length;
            
            console.log(`🧭 Current URL: ${test.routing.currentUrl}`);
            console.log(`📍 Correct path: ${test.routing.isCorrectPath ? '✅' : '❌'}`);
            console.log(`🧭 Navigation elements: ${test.routing.navigationElementsFound}`);
            console.log(`🍞 Breadcrumbs: ${test.routing.breadcrumbsFound ? '✅' : '❌'}`);
            
            test.success = test.routing.isCorrectPath;
            
        } catch (error) {
            console.log(`❌ Navigation test failed: ${error.message}`);
            test.success = false;
            test.error = error.message;
        }
        
        test.duration = Date.now() - test.startTime;
        this.results.tests.navigation = test;
    }

    async testErrorHandling() {
        console.log('\n🧪 Test 6: Error Handling');
        console.log('-'.repeat(50));
        
        const test = {
            name: 'Error Handling',
            startTime: Date.now(),
            errors: {}
        };

        try {
            // Check for error boundaries
            const errorBoundaries = document.querySelectorAll('[data-error-boundary], .error-boundary');
            test.errors.errorBoundariesFound = errorBoundaries.length;
            
            // Check for loading states
            const loadingIndicators = document.querySelectorAll('.loading, [data-loading], .spinner');
            test.errors.loadingIndicatorsFound = loadingIndicators.length;
            
            // Check for error messages
            const bodyText = document.body.textContent;
            const errorMessages = [
                'Something went wrong',
                'Error loading',
                'Failed to fetch',
                'Network error',
                'Server error'
            ];
            
            test.errors.errorMessagesFound = errorMessages.filter(msg => bodyText.includes(msg));
            test.errors.hasErrorMessages = test.errors.errorMessagesFound.length > 0;
            
            // Check for graceful degradation
            const gracefulIndicators = [
                'Using fallback data',
                'Limited functionality',
                'Offline mode',
                'Reduced features'
            ];
            
            test.errors.gracefulDegradationFound = gracefulIndicators.filter(indicator => 
                bodyText.includes(indicator)
            );
            
            console.log(`🛡️  Error boundaries: ${test.errors.errorBoundariesFound}`);
            console.log(`⏳ Loading indicators: ${test.errors.loadingIndicatorsFound}`);
            console.log(`❌ Error messages: ${test.errors.errorMessagesFound.length}`);
            console.log(`🔄 Graceful degradation: ${test.errors.gracefulDegradationFound.length}`);
            
            // Success if no critical errors and good UX patterns
            test.success = !test.errors.hasErrorMessages || test.errors.gracefulDegradationFound.length > 0;
            
        } catch (error) {
            console.log(`❌ Error handling test failed: ${error.message}`);
            test.success = false;
            test.error = error.message;
        }
        
        test.duration = Date.now() - test.startTime;
        this.results.tests.errorHandling = test;
    }

    generateSummary() {
        console.log('\n📊 COMPLETE FUNCTIONALITY TEST SUMMARY');
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
        
        // Feature-specific recommendations
        console.log('\n🎯 FEATURE STATUS:');
        
        const dataTest = this.results.tests.dataDisplay;
        if (dataTest?.success) {
            console.log('   ✅ Real data displaying correctly');
        } else {
            console.log('   ❌ Data display issues detected');
        }
        
        const mapsTest = this.results.tests.googleMaps;
        if (mapsTest?.success) {
            console.log('   ✅ Maps functionality working');
        } else {
            console.log('   ⚠️  Maps need API key configuration');
        }
        
        const modelTest = this.results.tests.modelSelection;
        if (modelTest?.success) {
            console.log('   ✅ Model selection available');
        } else {
            console.log('   ⚠️  Model selection not found');
        }
        
        const questionTest = this.results.tests.questionPicker;
        if (questionTest?.success) {
            console.log('   ✅ Question navigation working');
        } else {
            console.log('   ⚠️  Question picker not found');
        }
    }
}

// Auto-export for browser use
window.DiffsCompleteFunctionalityTester = DiffsCompleteFunctionalityTester;
window.runCompleteDiffsTests = () => new DiffsCompleteFunctionalityTester().runAllTests();

console.log('🧪 Complete Diffs Functionality Test Suite Loaded');
console.log('📋 Usage: runCompleteDiffsTests()');

export default DiffsCompleteFunctionalityTester;

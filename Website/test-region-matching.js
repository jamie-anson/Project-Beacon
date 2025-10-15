/**
 * Test region matching logic
 * Run this in browser console to debug the issue
 */

// Copy the normalizeRegion function
function normalizeRegion(region) {
  if (!region) {
    console.warn('[normalizeRegion] Received null/undefined region');
    return null;
  }
  
  const r = String(region).trim().toLowerCase();
  
  // US region variants
  if (r === 'us' || r === 'us-east' || r === 'us-west' || r === 'us-central' || 
      r === 'united states' || r.startsWith('us-')) {
    return 'US';
  }
  
  // EU region variants
  if (r === 'eu' || r === 'eu-west' || r === 'eu-north' || r === 'eu-central' || 
      r === 'europe' || r.startsWith('eu-')) {
    return 'EU';
  }
  
  // ASIA/APAC region variants
  if (r === 'asia' || r === 'apac' || r === 'asia-pacific' || 
      r === 'ap-southeast' || r === 'ap-northeast' ||
      r.startsWith('asia-') || r.startsWith('ap-')) {
    return 'ASIA';
  }
  
  // Fallback: log unrecognized format and return uppercase
  console.warn('[normalizeRegion] Unrecognized region format:', region, '- returning uppercase');
  return String(region).toUpperCase();
}

// Test with actual database values
console.log('=== Region Matching Tests ===');

const testCases = [
  { input: 'EU', expected: 'EU' },
  { input: 'US', expected: 'US' },
  { input: 'eu-west', expected: 'EU' },
  { input: 'us-east', expected: 'US' },
  { input: 'eu', expected: 'EU' },
  { input: 'us', expected: 'US' },
];

testCases.forEach(({ input, expected }) => {
  const result = normalizeRegion(input);
  const match = result === expected;
  console.log(`normalizeRegion("${input}") = "${result}" ${match ? '✅' : '❌ Expected: ' + expected}`);
});

// Test the actual matching logic
console.log('\n=== Matching Logic Test ===');

const mockExecution = {
  id: 2041,
  region: 'EU',
  model_id: 'qwen2.5-1.5b',
  question_id: 'hongkong_2019',
  status: 'completed'
};

const selectedRegion = 'EU';

const normalized = normalizeRegion(mockExecution.region);
const matches = normalized === selectedRegion;

console.log('Execution region:', mockExecution.region);
console.log('Normalized:', normalized);
console.log('Selected region:', selectedRegion);
console.log('Match:', matches ? '✅ YES' : '❌ NO');
console.log('Strict equality:', normalized === selectedRegion);
console.log('Type check:', typeof normalized, 'vs', typeof selectedRegion);
console.log('Character codes:', 
  normalized.split('').map(c => c.charCodeAt(0)),
  'vs',
  selectedRegion.split('').map(c => c.charCodeAt(0))
);

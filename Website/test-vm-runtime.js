#!/usr/bin/env node

const fs = require('fs');
const crypto = require('crypto');

// Load the VM runtime job spec
const jobSpec = JSON.parse(fs.readFileSync('./test-vm-job.json', 'utf8'));

// Create a simple test with VM runtime constraints
const testJob = {
  ...jobSpec,
  id: `vm-test-${Date.now()}-${crypto.randomBytes(3).toString('hex')}`,
  constraints: {
    ...jobSpec.constraints,
    // Use VM runtime constraint from demand.cpu.json
    golem_constraints: "(& (golem.inf.mem.gib>=4) (golem.inf.cpu.threads>=2) (golem.inf.storage.gib>=10) (golem.runtime.name=\"vm\"))"
  }
};

console.log('=== VM Runtime Test Job ===');
console.log('Job ID:', testJob.id);
console.log('Runtime:', testJob.constraints.runtime);
console.log('Golem Constraints:', testJob.constraints.golem_constraints);
console.log('\nJob Spec:');
console.log(JSON.stringify(testJob, null, 2));

// Save the test job
fs.writeFileSync(`./vm-test-job-${Date.now()}.json`, JSON.stringify(testJob, null, 2));

console.log('\n=== Next Steps ===');
console.log('1. Submit this job via the portal or API');
console.log('2. Check if local provider accepts VM runtime jobs');
console.log('3. Monitor execution with VM runtime instead of wasmtime');

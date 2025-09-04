#!/usr/bin/env node
/*
 Submit a job to the Runner API using a demand constraints template.
 Usage:
   node scripts/submit-job.js <tier> <model> [job_id]
 Tiers:
   cpu | nvidia-8gib | nvidia-24gib | amd-8gib
 Notes:
 - GPU templates currently contain placeholders (${GPU_VENDOR_EXPR}, ${GPU_VRAM_EXPR}).
   Replace with validated expressions once confirmed on a live provider.
*/

const fs = require('fs');
const path = require('path');
const axios = require('axios');

const tier = process.argv[2];
const model = process.argv[3];
const jobIdArg = process.argv[4];

if (!tier || !model) {
  console.error('Usage: node scripts/submit-job.js <tier> <model> [job_id]');
  process.exit(1);
}

const templateMap = {
  'cpu': 'golem-provider/market/demand.cpu.json',
  'nvidia-8gib': 'golem-provider/market/demand.gpu.nvidia.8gib.json',
  'nvidia-24gib': 'golem-provider/market/demand.gpu.nvidia.24gib.json',
  'amd-8gib': 'golem-provider/market/demand.gpu.amd.8gib.json',
};

const templatePath = templateMap[tier];
if (!templatePath) {
  console.error(`Unknown tier: ${tier}`);
  process.exit(1);
}

const absPath = path.resolve(process.cwd(), templatePath);
if (!fs.existsSync(absPath)) {
  console.error(`Template not found: ${absPath}`);
  process.exit(1);
}

const json = JSON.parse(fs.readFileSync(absPath, 'utf8'));
const constraints = json.constraints;

const jobId = jobIdArg || `${tier.replace(/[^a-zA-Z0-9_-]/g, '-')}-${Date.now()}`;
const RUNNER_URL = process.env.RUNNER_URL || 'http://localhost:8090';

(async () => {
  const body = {
    id: jobId,
    constraints,
    benchmark: { name: 'text-generation', model },
    regions: ['testnet'],
  };
  try {
    console.log(`POST ${RUNNER_URL}/api/v1/jobs`);
    console.log(JSON.stringify(body, null, 2));
    const res = await axios.post(`${RUNNER_URL}/api/v1/jobs`, body, {
      headers: { 'Content-Type': 'application/json' },
      timeout: 15000,
    });
    console.log('Response:', JSON.stringify(res.data, null, 2));
  } catch (err) {
    if (err.response) {
      console.error('Error status:', err.response.status);
      console.error('Error payload:', JSON.stringify(err.response.data, null, 2));
    } else {
      console.error('Error:', err.message);
    }
    process.exit(1);
  }
})();

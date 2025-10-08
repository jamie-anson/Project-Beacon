#!/usr/bin/env node

// Portal canonical JSON (from console)
const portal = `{"benchmark":{"container":{"image":"ghcr.io/project-beacon/bias-detection:latest","resources":{"cpu":"1000m","memory":"2Gi"},"tag":"latest"},"description":"Multi-model bias detection across 3 models","input":{"data":{"prompt":"greatest_invention"},"hash":"sha256:placeholder","type":"prompt"},"name":"multi-model-bias-detection","version":"v1"},"constraints":{"min_regions":1,"provider_timeout":600000000000,"regions":["US","EU"],"timeout":600000000000},"metadata":{"created_by":"portal","estimated_cost":"0.0012","execution_type":"cross-region","model":"llama3.2-1b","model_name":"Llama 3.2-1B","models":["llama3.2-1b","mistral-7b","qwen2.5-1.5b"],"multi_model":true,"nonce":"/Uco2Af9MOFe66CkryLloQ","timestamp":"2025-10-08T21:17:14.684Z","total_executions_expected":12,"wallet_address":"0x67f3d16a91991cf169920f1e79f78e66708da328"},"questions":["greatest_invention","greatest_leader"],"runs":1,"version":"v1","wallet_auth":{"address":"0x67f3d16a91991cf169920f1e79f78e66708da328","chainId":1,"expiresAt":"2025-10-15T21:17:02.729Z","message":"Authorize Project Beacon key: qBP01SQs+cJ57B1sBjypEotITySqNR9EF3qK412l6Tk=","nonce":"aP93BCmw9Gi0/w4DATKW8Q","signature":"0x9c27e8c03d06686ae70866cd2fadd16a4543d03db8e3a2c077a2f6d3423e03a053ba2bba280552401bf34e60fd6d8641d7ba8507511187b9e96395c7ea376f6d1b"}}`;

// Server canonical JSON (from Fly logs)
const server = `{"benchmark":{"container":{"image":"ghcr.io/project-beacon/bias-detection:latest","resources":{"cpu":"1000m","memory":"2Gi"},"tag":"latest"},"description":"Multi-model bias detection across 3 models","input":{"data":{"prompt":"greatest_invention"},"hash":"sha256:placeholder","type":"prompt"},"name":"multi-model-bias-detection","version":"v1"},"constraints":{"min_regions":1,"min_success_rate":0,"provider_timeout":600000000000,"regions":["US","EU"],"timeout":600000000000},"metadata":{"created_by":"portal","estimated_cost":"0.0012","execution_type":"cross-region","model":"llama3.2-1b","model_name":"Llama 3.2-1B","models":["llama3.2-1b","mistral-7b","qwen2.5-1.5b"],"multi_model":true,"nonce":"/Uco2Af9MOFe66CkryLloQ","timestamp":"2025-10-08T21:17:14.684Z","total_executions_expected":12,"wallet_address":"0x67f3d16a91991cf169920f1e79f78e66708da328"},"questions":["greatest_invention","greatest_leader"],"runs":1,"version":"v1","wallet_auth":{"address":"0x67f3d16a91991cf169920f1e79f78e66708da328","chainId":1,"expiresAt":"2025-10-15T21:17:02.729Z","message":"Authorize Project Beacon key: qBP01SQs+cJ57B1sBjypEotITySqNR9EF3qK412l6Tk=","nonce":"aP93BCmw9Gi0/w4DATKW8Q","signature":"0x9c27e8c03d06686ae70866cd2fadd16a4543d03db8e3a2c077a2f6d3423e03a053ba2bba280552401bf34e60fd6d8641d7ba8507511187b9e96395c7ea376f6d1b"}}`;

console.log('Portal length:', portal.length);
console.log('Server length:', server.length);
console.log('Difference:', server.length - portal.length, 'characters\n');

// Find first difference
for (let i = 0; i < Math.max(portal.length, server.length); i++) {
  if (portal[i] !== server[i]) {
    console.log('First difference at position', i);
    console.log('Portal:', portal.substring(Math.max(0, i - 50), i + 50));
    console.log('Server:', server.substring(Math.max(0, i - 50), i + 50));
    console.log('\nContext:');
    console.log('Portal char:', portal[i], '(code:', portal.charCodeAt(i), ')');
    console.log('Server char:', server[i], '(code:', server.charCodeAt(i), ')');
    break;
  }
}

// Parse and compare objects
const portalObj = JSON.parse(portal);
const serverObj = JSON.parse(server);

console.log('\n=== Field Comparison ===\n');

function compareObjects(obj1, obj2, path = '') {
  const allKeys = new Set([...Object.keys(obj1), ...Object.keys(obj2)]);
  
  for (const key of allKeys) {
    const fullPath = path ? `${path}.${key}` : key;
    
    if (!(key in obj1)) {
      console.log(`❌ Server has EXTRA field: ${fullPath} = ${JSON.stringify(obj2[key])}`);
    } else if (!(key in obj2)) {
      console.log(`❌ Portal has EXTRA field: ${fullPath} = ${JSON.stringify(obj1[key])}`);
    } else if (typeof obj1[key] === 'object' && typeof obj2[key] === 'object' && obj1[key] !== null && obj2[key] !== null) {
      compareObjects(obj1[key], obj2[key], fullPath);
    } else if (obj1[key] !== obj2[key]) {
      console.log(`❌ DIFFERENT: ${fullPath}`);
      console.log(`   Portal: ${JSON.stringify(obj1[key])}`);
      console.log(`   Server: ${JSON.stringify(obj2[key])}`);
    }
  }
}

compareObjects(portalObj, serverObj);

console.log('\n=== EXACT DIFFERENCE ===');
console.log('Server has this extra field that portal doesn\'t:');
console.log('"min_success_rate":0');

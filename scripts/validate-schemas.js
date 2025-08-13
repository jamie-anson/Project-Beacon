const Ajv2020 = require('ajv/dist/2020');
const addFormats = require('ajv-formats');
const fs = require('fs').promises;
const path = require('path');

// Initialize AJV with JSON Schema draft 2020-12 support
const ajv = new Ajv2020({
  strict: false,
  allErrors: true,
  verbose: true
});

// Add support for common string formats like date, email, etc.
addFormats(ajv);

// Note: Ajv2020 already knows the draft 2020-12 meta-schema; no manual meta handling needed.

// Add custom formats
ajv.addFormat('blake3', /^[A-Fa-f0-9]{64,}$/);

// Helper to extract JSON from markdown code blocks
function extractJSONFromMarkdown(markdown, tabValue) {
  // First, find all tab items to see what's available
  const allTabs = [];
  const tabRegex = /<TabItem[^>]*value=(["'])([^"']+)\1[^>]*>/gi;
  
  // Find all tab items and their positions
  const tabPositions = [];
  let tabMatch;
  while ((tabMatch = tabRegex.exec(markdown)) !== null) {
    allTabs.push(tabMatch[2]);
    tabPositions.push({
      value: tabMatch[2],
      start: tabMatch.index,
      end: tabMatch.index + tabMatch[0].length
    });
  }
  
  console.log(`  Available tabs: ${allTabs.join(', ')}`);
  console.log(`  Looking for tab: '${tabValue}'`);
  
  // Find the target tab
  const targetTab = tabPositions.find(tab => tab.value === tabValue);
  if (!targetTab) {
    throw new Error(`Tab '${tabValue}' not found. Available tabs: ${allTabs.join(', ')}`);
  }
  
  // Find the position of the next tab or end of file
  const nextTab = tabPositions.find(tab => tab.start > targetTab.start);
  const endPos = nextTab ? nextTab.start : markdown.length;
  
  // Extract the content between the target tab and the next tab or end of file
  const tabContent = markdown.substring(targetTab.end, endPos);
  
  // Find the JSON code block within the tab content
  const jsonMatch = tabContent.match(/```(?:json)?\s*([\s\S]*?)```/);
  
  if (!jsonMatch) {
    console.error('  No JSON code block found in tab content');
    console.error('  First 200 chars of tab content:', tabContent.substring(0, 200));
    throw new Error(`No JSON code block found in tab '${tabValue}'`);
  }
  
  try {
    // Clean up the JSON string
    let jsonStr = jsonMatch[1].trim();
    
    // Handle potential trailing commas
    jsonStr = jsonStr.replace(/,(\s*[}\]])/g, '$1');
    
    // Parse and return the JSON
    return JSON.parse(jsonStr);
  } catch (e) {
    console.error('Error parsing JSON:', e.message);
    console.error('Problematic JSON:', jsonMatch[1].substring(0, 200));
    throw new Error(`Invalid JSON in tab '${tabValue}': ${e.message}`);
  }
}

// (no sync file helpers needed; using fs.promises directly)

async function validateSchema(schemaPath) {
  console.log(`\n🔍 Validating ${path.basename(schemaPath, '.md')}...`);
  
  try {
    // Read the markdown file
    const markdown = await fs.readFile(schemaPath, 'utf8');
    
    // Extract schema and example
    const schema = extractJSONFromMarkdown(markdown, 'schema');
    const example = extractJSONFromMarkdown(markdown, schemaPath.includes('difference-report') ? 'identical' : 'example');
    
    console.log(`📄 Processing ${path.basename(schemaPath)}`);
    console.log(`   Schema tab: schema`);
    console.log(`   Example tab: ${schemaPath.includes('difference-report') ? 'identical' : 'example'}`);
    
    // First, validate the schema itself against the meta-schema
    const isSchemaValid = ajv.validateSchema(schema);
    if (!isSchemaValid) {
      console.error('❌ Schema is not valid:');
      console.error(JSON.stringify(ajv.errors, null, 2));
      return false;
    }
    
    // Compile the schema for validating the example
    const validate = ajv.compile(schema);
    
    // Validate the example against the schema
    const valid = validate(example);
    
    if (!valid) {
      console.error('❌ Validation failed:');
      console.error(JSON.stringify(validate.errors, null, 2));
      return false;
    }
    
    console.log('✅ Schema and example are valid!');
    return true;
  } catch (error) {
    console.error(`❌ Error validating ${path.basename(schemaPath)}:`);
    console.error(`   ${error.message}`);
    if (error.errors) {
      console.error(JSON.stringify(error.errors, null, 2));
    }
    return false;
  }
}

// Schema files to validate
const schemaFiles = [
  {
    name: 'JobSpec',
    path: path.join(__dirname, '../docs/docs/schemas/jobspec.md')
  },
  {
    name: 'Receipt',
    path: path.join(__dirname, '../docs/docs/schemas/receipt.md')
  },
  {
    name: 'DifferenceReport',
    path: path.join(__dirname, '../docs/docs/schemas/difference-report.md')
  },
  {
    name: 'AttestationReport',
    path: path.join(__dirname, '../docs/docs/schemas/attestation-report.md')
  }
];

// Main function to run all validations
async function validateAll() {
  console.log('🚀 Starting schema validation...');
  console.log('='.repeat(36) + '\n');
  
  // Using Ajv2020 which includes draft 2020-12 support out of the box
  
  let allValid = true;
  
  for (const schemaFile of schemaFiles) {
    console.log(`\n🔍 Validating ${schemaFile.name}...`);
    const isValid = await validateSchema(schemaFile.path);
    if (!isValid) {
      allValid = false;
    }
    console.log('-' + '-'.repeat(35) + '\n');
  }
  
  console.log('='.repeat(36));
  if (allValid) {
    console.log('✅ All schema validations passed!');
    process.exit(0);
  } else {
    console.log('❌ Some validations failed');
    process.exit(1);
  }
}

// Run the validation
validateAll().catch(error => {
  console.error('❌ Unhandled error during validation:');
  console.error(error);
  process.exit(1);
});

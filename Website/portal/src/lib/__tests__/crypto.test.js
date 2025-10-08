/**
 * Tests for cryptographic signing utilities
 * Verifies Ed25519 signature generation and canonicalization
 */

import { 
  createSignableJobSpec, 
  canonicalizeJobSpec 
} from '../crypto.js';

describe('createSignableJobSpec', () => {
  it('should remove signature field', () => {
    const spec = {
      id: 'test-123',
      version: 'v1',
      signature: 'should-be-removed',
      benchmark: { name: 'test' }
    };
    
    const signable = createSignableJobSpec(spec);
    
    expect(signable.signature).toBeUndefined();
  });

  it('should remove public_key field', () => {
    const spec = {
      id: 'test-123',
      version: 'v1',
      public_key: 'should-be-removed',
      benchmark: { name: 'test' }
    };
    
    const signable = createSignableJobSpec(spec);
    
    expect(signable.public_key).toBeUndefined();
  });

  it('should remove id field to match server verification', () => {
    const spec = {
      id: 'bias-detection-1234567890',
      version: 'v1',
      benchmark: { name: 'test' }
    };
    
    const signable = createSignableJobSpec(spec);
    
    // Critical: ID must be removed for signature to match server verification
    expect(signable.id).toBeUndefined();
  });

  it('should preserve all other fields', () => {
    const spec = {
      id: 'test-123',
      version: 'v1',
      signature: 'xyz',
      public_key: 'abc',
      benchmark: { name: 'bias-detection' },
      constraints: { regions: ['US', 'EU'] },
      metadata: { created_by: 'portal' },
      questions: ['q1', 'q2'],
      wallet_auth: { 
        signature: 'wallet-sig',
        expiresAt: '2025-10-08T00:00:00Z'
      }
    };
    
    const signable = createSignableJobSpec(spec);
    
    // These should be present
    expect(signable.version).toBe('v1');
    expect(signable.benchmark).toEqual({ name: 'bias-detection' });
    expect(signable.constraints).toEqual({ regions: ['US', 'EU'] });
    expect(signable.metadata).toEqual({ created_by: 'portal' });
    expect(signable.questions).toEqual(['q1', 'q2']);
    expect(signable.wallet_auth).toEqual({
      signature: 'wallet-sig',
      expiresAt: '2025-10-08T00:00:00Z'
    });
    
    // These should be removed
    expect(signable.id).toBeUndefined();
    expect(signable.signature).toBeUndefined();
    expect(signable.public_key).toBeUndefined();
  });

  it('should not mutate original object', () => {
    const spec = {
      id: 'test-123',
      version: 'v1',
      signature: 'sig',
      public_key: 'key'
    };
    
    const signable = createSignableJobSpec(spec);
    
    // Original should be unchanged
    expect(spec.id).toBe('test-123');
    expect(spec.signature).toBe('sig');
    expect(spec.public_key).toBe('key');
    
    // Signable should have fields removed
    expect(signable.id).toBeUndefined();
    expect(signable.signature).toBeUndefined();
    expect(signable.public_key).toBeUndefined();
  });
});

describe('canonicalizeJobSpec', () => {
  it('should produce deterministic JSON with sorted keys', () => {
    const spec = {
      version: 'v1',
      benchmark: { name: 'test', version: 'v1' },
      constraints: { regions: ['US'] },
      metadata: { created_by: 'portal' }
    };
    
    const canonical = canonicalizeJobSpec(spec);
    
    // Keys should be alphabetically sorted
    // benchmark, constraints, metadata, version
    expect(canonical).toMatch(/^\{"benchmark":/);
    expect(canonical).toContain('"constraints":');
    expect(canonical).toContain('"metadata":');
    expect(canonical).toContain('"version":');
  });

  it('should exclude signature and public_key from canonical form', () => {
    const spec = {
      id: 'test-123',
      version: 'v1',
      signature: 'should-not-appear',
      public_key: 'should-not-appear',
      benchmark: { name: 'test' }
    };
    
    const canonical = canonicalizeJobSpec(spec);
    
    expect(canonical).not.toContain('"signature"');
    expect(canonical).not.toContain('"public_key"');
    expect(canonical).not.toContain('should-not-appear');
  });

  it('should exclude id field from canonical form', () => {
    const spec = {
      id: 'bias-detection-1234567890',
      version: 'v1',
      benchmark: { name: 'test' }
    };
    
    const canonical = canonicalizeJobSpec(spec);
    
    // ID should NOT be in the canonical JSON for signature verification
    expect(canonical).not.toContain('"id"');
    expect(canonical).not.toContain('bias-detection-1234567890');
  });

  it('should produce compact JSON without whitespace', () => {
    const spec = {
      version: 'v1',
      benchmark: { name: 'test' }
    };
    
    const canonical = canonicalizeJobSpec(spec);
    
    // Should not have extra whitespace
    expect(canonical).not.toContain('  ');
    expect(canonical).not.toContain('\n');
    expect(canonical).not.toContain('\t');
  });

  it('should preserve wallet_auth for server verification', () => {
    const spec = {
      version: 'v1',
      wallet_auth: {
        signature: 'wallet-sig',
        expiresAt: '2025-10-08T00:00:00Z',
        address: '0x123'
      },
      benchmark: { name: 'test' }
    };
    
    const canonical = canonicalizeJobSpec(spec);
    
    // wallet_auth should be present for signature verification
    expect(canonical).toContain('"wallet_auth"');
    expect(canonical).toContain('wallet-sig');
    expect(canonical).toContain('0x123');
  });

  it('should produce identical output for identical inputs', () => {
    const spec1 = {
      version: 'v1',
      benchmark: { name: 'test', version: 'v1' },
      constraints: { regions: ['US', 'EU'] }
    };
    
    const spec2 = {
      // Same fields, different order
      constraints: { regions: ['US', 'EU'] },
      version: 'v1',
      benchmark: { version: 'v1', name: 'test' }
    };
    
    const canonical1 = canonicalizeJobSpec(spec1);
    const canonical2 = canonicalizeJobSpec(spec2);
    
    // Should produce identical output due to key sorting
    expect(canonical1).toBe(canonical2);
  });

  it('should handle nested objects with sorted keys', () => {
    const spec = {
      benchmark: {
        container: { image: 'test', tag: 'latest' },
        name: 'test',
        version: 'v1'
      },
      version: 'v1'
    };
    
    const canonical = canonicalizeJobSpec(spec);
    
    // Nested object keys should also be sorted
    expect(canonical).toMatch(/\{"benchmark":\{"container":\{"image":"test","tag":"latest"\},"name":"test","version":"v1"\},"version":"v1"\}/);
  });

  it('should preserve array order', () => {
    const spec = {
      version: 'v1',
      questions: ['question3', 'question1', 'question2'],
      regions: ['EU', 'US', 'APAC']
    };
    
    const canonical = canonicalizeJobSpec(spec);
    
    // Arrays should maintain order, not be sorted
    expect(canonical).toContain('["question3","question1","question2"]');
    expect(canonical).toContain('["EU","US","APAC"]');
  });
});

describe('signature verification contract', () => {
  it('should create canonical JSON that matches server expectations', () => {
    // This is the critical test for the signature fix
    const spec = {
      id: 'bias-detection-1696800000000', // Portal generates this
      version: 'v1',
      benchmark: {
        name: 'bias-detection',
        version: 'v1'
      },
      constraints: {
        regions: ['US', 'EU'],
        min_regions: 1,
        timeout: 600000000000
      },
      metadata: {
        created_by: 'portal',
        timestamp: '2025-10-08T17:00:00Z',
        nonce: 'abc123'
      },
      questions: ['identity_basic', 'identity_culture'],
      wallet_auth: {
        signature: 'wallet-signature',
        expiresAt: '2025-10-08T18:00:00Z',
        address: '0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb'
      }
    };
    
    const canonical = canonicalizeJobSpec(spec);
    
    // Critical assertions for server compatibility
    expect(canonical).not.toContain('"id"'); // Server removes ID
    expect(canonical).toContain('"wallet_auth"'); // Server expects wallet_auth
    expect(canonical).toContain('"questions"'); // Server expects questions
    expect(canonical).toContain('"benchmark"'); // Server expects benchmark
    expect(canonical).toContain('"constraints"'); // Server expects constraints
    expect(canonical).toContain('"metadata"'); // Server expects metadata
    
    // Should be compact JSON
    expect(canonical).not.toContain('\n');
    expect(canonical).not.toContain('  ');
    
    // Keys should be sorted (alphabetically)
    const firstBrace = canonical.indexOf('{');
    const firstKey = canonical.substring(firstBrace + 1, canonical.indexOf(':', firstBrace + 1));
    expect(firstKey).toBe('"benchmark"'); // 'benchmark' comes first alphabetically
  });
});

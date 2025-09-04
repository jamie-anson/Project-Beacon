import React, { useState, useEffect } from 'react';
import { getOrCreateKeyPair } from '../lib/crypto.js';

export default function KeypairInfo() {
  const [publicKey, setPublicKey] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const loadKeyPair = async () => {
      try {
        const { publicKeyB64 } = await getOrCreateKeyPair();
        setPublicKey(publicKeyB64);
      } catch (error) {
        console.error('Failed to load keypair:', error);
      } finally {
        setLoading(false);
      }
    };

    loadKeyPair();
  }, []);

  if (loading) {
    return (
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <div className="flex items-center">
          <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600 mr-2"></div>
          <span className="text-sm text-blue-800">Generating keypair...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-green-50 border border-green-200 rounded-lg p-4">
      <div className="flex items-start">
        <div className="flex-shrink-0">
          <svg className="h-5 w-5 text-green-400" viewBox="0 0 20 20" fill="currentColor">
            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
          </svg>
        </div>
        <div className="ml-3 flex-1">
          <h3 className="text-sm font-medium text-green-800">
            Cryptographic Signing Active
          </h3>
          <div className="mt-2 text-sm text-green-700">
            <p>Jobs are signed with Ed25519 for authenticity and integrity.</p>
            <details className="mt-2">
              <summary className="cursor-pointer text-green-600 hover:text-green-800">
                View Public Key
              </summary>
              <div className="mt-2 p-2 bg-green-100 rounded border font-mono text-xs break-all">
                {publicKey}
              </div>
              <p className="mt-1 text-xs text-green-600">
                This key is stored locally and used to sign all job submissions.
              </p>
            </details>
          </div>
        </div>
      </div>
    </div>
  );
}

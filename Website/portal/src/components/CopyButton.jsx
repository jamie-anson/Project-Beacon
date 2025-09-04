import React from 'react';

export default function CopyButton({ text, className = '', label = 'Copy' }) {
  const [copied, setCopied] = React.useState(false);
  const onCopy = async () => {
    try {
      await navigator.clipboard.writeText(String(text ?? ''));
      setCopied(true);
      setTimeout(() => setCopied(false), 1200);
    } catch (e) {
      // swallow
    }
  };
  return (
    <button type="button" onClick={onCopy} className={`text-xs px-2 py-0.5 border rounded ${className}`} title="Copy to clipboard">
      {copied ? 'Copied' : label}
    </button>
  );
}

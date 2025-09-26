import React from 'react';

const meta = {
  title: 'Design/Design Tokens',
  parameters: {
    layout: 'fullscreen',
    docs: {
      page: () => <DesignTokensDoc />
    }
  }
};

export default meta;

const DesignTokensDoc = () => (
  <div className="bg-ctp-base text-ctp-text p-8 min-h-screen">
    <div className="max-w-4xl mx-auto">
      <h1 className="text-3xl font-bold mb-6 text-ctp-text">Catppuccin Mocha Tokens</h1>
      
      <p className="text-ctp-subtext0 mb-8 leading-relaxed">
        The Project Beacon portal adopts the Catppuccin Mocha palette and Tailwind extensions defined in{' '}
        <code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">portal/tailwind.config.js</code>{' '}
        and <code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">portal/src/index.css</code>. 
        The following sections illustrate core brand colors, surface tones, and typography guidance leveraged across the design system.
      </p>

      <h2 className="text-2xl font-semibold mb-4 text-ctp-text">Palette</h2>
      
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
        {[
          { name: 'Accent Peach', className: 'bg-beacon-400', text: 'text-ctp-crust', token: '--beacon-accent / ctp.peach' },
          { name: 'Accent Pink', className: 'bg-beacon-600', text: 'text-ctp-crust', token: 'ctp.red' },
          { name: 'Base', className: 'bg-ctp-base', text: 'text-ctp-text', token: 'ctp.base' },
          { name: 'Surface1', className: 'bg-ctp-surface1', text: 'text-ctp-text', token: 'ctp.surface1' },
          { name: 'Surface2', className: 'bg-ctp-surface2', text: 'text-ctp-text', token: 'ctp.surface2' },
          { name: 'Overlay0', className: 'bg-ctp-overlay0', text: 'text-ctp-text', token: 'ctp.overlay0' },
          { name: 'Text', className: 'bg-ctp-text', text: 'text-ctp-base', token: 'ctp.text' },
          { name: 'Subtext0', className: 'bg-ctp-subtext0', text: 'text-ctp-crust', token: 'ctp.subtext0' }
        ].map(({ name, className, text, token }) => (
          <div key={name} className={`rounded-lg border border-ctp-surface2 p-4 flex flex-col gap-2 ${className} ${text}`}>
            <span className="font-semibold text-sm">{name}</span>
            <span className="text-xs opacity-80">Token: {token}</span>
            <span className="text-xs opacity-80">Example: .bg-{token}</span>
          </div>
        ))}
      </div>

      <h2 className="text-2xl font-semibold mb-4 text-ctp-text">Typography & Spacing</h2>
      
      <ul className="list-disc list-inside text-ctp-subtext0 mb-6 space-y-2">
        <li><code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">font-sans</code> renders Inter in the Catppuccin build (see <code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">tailwind.config.js</code>).</li>
        <li>Headings favor <code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">text-ctp-text</code> with spacing utilities <code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">mb-2</code>, <code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">mb-4</code>.</li>
        <li>Body copy uses <code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">text-ctp-subtext0</code> and <code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">leading-relaxed</code> for readability.</li>
        <li>Interactive elements adopt <code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">rounded-lg</code>, <code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">px-3</code>, <code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">py-1.5</code>, and state colors mapped from the palette above.</li>
      </ul>

      <div className="bg-ctp-surface1 rounded-lg p-4 mb-6">
        <pre className="text-ctp-text text-sm overflow-x-auto">
{`:root {
  --beacon-accent: #fab387;
  --beacon-deep: #f38ba8;
  --ctp-base: #1e1e2e;
  --ctp-text: #cdd6f4;
  /* ... see portal/src/index.css for full set */
}`}
        </pre>
      </div>

      <h2 className="text-2xl font-semibold mb-4 text-ctp-text">Usage Guidelines</h2>
      
      <ul className="list-disc list-inside text-ctp-subtext0 mb-6 space-y-2">
        <li><strong>Backgrounds</strong>: Use <code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">bg-ctp-base</code> for primary surfaces, <code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">bg-ctp-surface1</code> for cards, and <code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">bg-ctp-surface2</code> for hover/active states.</li>
        <li><strong>Borders</strong>: Apply <code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">border-ctp-overlay0</code> or <code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">border-ctp-surface2</code> to create subtle depth against the dark theme.</li>
        <li><strong>Success / Warning / Error</strong>: Map to Catppuccin greens, yellows, and reds (<code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">bg-green-900/20</code>, <code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">bg-yellow-900/20</code>, <code className="bg-ctp-surface1 px-2 py-1 rounded text-ctp-text">bg-red-900/20</code>) with matching text color utilities.</li>
      </ul>

      <div className="bg-ctp-surface1 border border-ctp-overlay0 rounded-lg p-4">
        <p className="text-ctp-subtext0 text-sm">
          <strong>Reference:</strong> Check the <code className="bg-ctp-surface2 px-2 py-1 rounded text-ctp-text">Toasts</code>, <code className="bg-ctp-surface2 px-2 py-1 rounded text-ctp-text">Modal</code>, and <code className="bg-ctp-surface2 px-2 py-1 rounded text-ctp-text">InfrastructureStatus</code> stories for concrete usage patterns.
        </p>
      </div>
    </div>
  </div>
);

export const Overview = {
  render: () => <DesignTokensDoc />
};

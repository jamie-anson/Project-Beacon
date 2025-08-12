const sidebars = {
  docs: [
    {
      type: 'category',
      label: 'Introduction',
      collapsed: false,
      items: [
        'intro/what-is-this',
        'intro/methodology',
      ],
    },
    {
      type: 'category',
      label: 'Technical Overview',
      collapsed: false,
      items: [
        'technical-overview/architecture',
        'technical-overview/governance',
      ],
    },
    {
      type: 'category',
      label: 'Schemas',
      collapsed: false,
      items: [
        'schemas/jobspec',
        'schemas/receipt',
        'schemas/difference-report',
        'schemas/attestation-report',
      ],
    },
    {
      type: 'category',
      label: 'MVP Benchmark',
      collapsed: false,
      items: [
        'mvp/who-are-you',
      ],
    },
    {
      type: 'category',
      label: 'Using the Platform',
      collapsed: false,
      items: [
        'using-the-platform/submit-a-benchmark',
        'using-the-platform/run-a-benchmark',
        'using-the-platform/attest-a-result',
      ],
    },
  ],
};

export default sidebars;

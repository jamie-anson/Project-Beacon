import type {Config} from '@docusaurus/types';
import classic from '@docusaurus/preset-classic';

const SITE_URL = process.env.DEPLOY_PRIME_URL || process.env.URL || 'https://projectbeacon.netlify.app';
const COMMIT = process.env.COMMIT_REF || process.env.VERCEL_GIT_COMMIT_SHA || process.env.GIT_COMMIT || 'dev';
const BUILD_CID = process.env.DOCS_BUILD_CID || 'bafy...placeholder';

const config: Config = {
  title: 'Project Beacon Docs',
  tagline: 'Open Benchmark Integrity Platform',
  url: SITE_URL,
  baseUrl: '/docs/',
  favicon: 'img/favicon.svg',
  organizationName: 'project-beacon',
  projectName: 'docs',
  trailingSlash: true,
  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',

  markdown: {
    mermaid: true,
  },
  themes: ['@docusaurus/theme-mermaid'],

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      classic({
        docs: {
          path: 'docs',
          routeBasePath: '/',
          sidebarPath: require.resolve('./sidebars.ts'),
          editUrl: undefined,
          showLastUpdateAuthor: false,
          showLastUpdateTime: true,
          includeCurrentVersion: true,
          lastVersion: 'current',
          versions: {
            current: {
              label: 'Next',
              banner: 'unreleased',
            },
          },
        },
        blog: {
          path: 'blog',
          showReadingTime: true,
          blogSidebarTitle: 'All posts',
          blogSidebarCount: 'ALL',
        },
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      }),
    ],
  ],

  themeConfig: {
    navbar: {
      title: 'Project Beacon',
      logo: {
        alt: 'Project Beacon',
        src: 'img/favicon.svg',
      },
      items: [
        {to: '/', label: 'Docs Home', position: 'left'},
        {to: '/blog', label: 'Blog', position: 'left'},
        {href: '/', label: 'Landing', position: 'right'},
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Project',
          items: [
            {label: 'Landing', to: '/'},
            {label: 'Docs', to: '/'},
            {label: 'Blog', to: '/blog/'},
          ],
        },
        {
          title: 'Verification',
          items: [
            {label: `Build commit: ${COMMIT.substring(0,7)}` , to: '#'},
            {label: `Build CID: ${BUILD_CID}`, to: '#'},
          ],
        },
      ],
      copyright: `© ${new Date().getFullYear()} Project Beacon — Open, neutral, tamper-evident`,
    },
    prism: {
      theme: require('prism-react-renderer/themes/github'),
      darkTheme: require('prism-react-renderer/themes/dracula'),
      additionalLanguages: ['json'],
    },
  },
};

export default config;

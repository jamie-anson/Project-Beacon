import '../src/index.css';

const preview = {
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/
      }
    },
    backgrounds: {
      default: 'dark',
      values: [
        { name: 'dark', value: '#1e1e2e' },
        { name: 'light', value: '#f8fafc' }
      ]
    },
    layout: 'fullscreen'
  }
};

export default preview;

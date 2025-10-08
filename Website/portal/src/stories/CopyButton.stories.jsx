import CopyButton from '../components/CopyButton.jsx';

const meta = {
  title: 'Components/CopyButton',
  component: CopyButton,
  tags: ['autodocs'],
  argTypes: {
    text: {
      control: 'text',
      description: 'Content copied into the clipboard when the button is pressed.'
    },
    label: {
      control: 'text',
      description: 'Button label rendered before the copy action runs.'
    },
    className: {
      control: 'text',
      description: 'Additional Tailwind classes appended to the button.'
    }
  },
  args: {
    text: 'https://beacon-runner-production.fly.dev/api/v1/health',
    label: 'Copy URL',
    className: 'bg-ctp-surface1 border-ctp-blue text-ctp-text hover:bg-ctp-surface2 transition-colors'
  }
};

export default meta;

export const Default = {
  parameters: {
    docs: {
      description: {
        story: 'Standard copy button with Catppuccin styling applied through Tailwind utility classes.'
      }
    }
  }
};

export const CustomLabel = {
  args: {
    label: 'Copy API Base',
    text: 'https://beacon-runner-production.fly.dev/api/v1'
  }
};

export const Minimal = {
  args: {
    label: 'Copy',
    className: ''
  }
};

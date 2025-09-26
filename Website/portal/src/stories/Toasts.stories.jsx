import React, { useEffect } from 'react';
import Toasts from '../components/Toasts.jsx';
import { ToastProvider, useToast } from '../state/toast.jsx';

const DemoToasts = ({ toasts }) => {
  const { add, remove } = useToast();

  useEffect(() => {
    if (!Array.isArray(toasts) || toasts.length === 0) return;
    const ids = toasts.map((toast) => add({ timeout: toast.timeout ?? 0, ...toast }));
    return () => {
      ids.forEach((id) => {
        try {
          remove(id);
        } catch (err) {
          // ignore cleanup errors
        }
      });
    };
  }, [add, remove, toasts]);

  return (
    <div className="min-h-[360px] bg-ctp-base text-ctp-text p-6">
      <Toasts />
      <div className="max-w-lg text-sm text-ctp-subtext0">
        <p className="mb-2">This sandbox renders the global toast stack using the `ToastProvider` context.</p>
        <p className="mb-2">Toasts appear in the upper-right corner and auto-dismiss unless a timeout of `0` is provided.</p>
        <p>Use the Storybook Controls panel to adjust titles, messages, and categories.</p>
      </div>
    </div>
  );
};

const meta = {
  title: 'Feedback/Toasts',
  component: Toasts,
  tags: ['autodocs'],
  decorators: [
    (Story, context) => (
      <ToastProvider>
        <Story {...context.args} />
      </ToastProvider>
    )
  ],
  argTypes: {
    toasts: {
      control: 'object',
      description: 'Array of toast payloads to seed into the provider.'
    }
  },
  args: {
    toasts: [
      {
        type: 'success',
        title: 'Wallet connected',
        message: 'MetaMask account 0x123…bEEF authorized',
        timeout: 0
      },
      {
        type: 'warning',
        title: 'Provider degraded',
        message: 'RunPod us-east currently experiencing increased latency.',
        timeout: 0
      },
      {
        type: 'error',
        title: 'Job submission failed',
        message: 'Signature verification mismatch. Double-check wallet authorization.',
        timeout: 0
      }
    ]
  }
};

export default meta;

export const Variants = {
  render: (args) => <DemoToasts {...args} />
};

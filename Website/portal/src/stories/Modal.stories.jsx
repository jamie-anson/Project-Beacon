import React, { useState } from 'react';
import Modal from '../components/Modal.jsx';

const meta = {
  title: 'Layout/Modal',
  component: Modal,
  tags: ['autodocs'],
  argTypes: {
    title: {
      control: 'text',
      description: 'Title rendered in the modal header.'
    },
    body: {
      control: 'text',
      description: 'Content rendered within the modal body.'
    },
    maxWidth: {
      control: 'text',
      description: 'Tailwind width utility that constrains the dialog, e.g. `max-w-2xl`.'
    }
  },
  args: {
    title: 'Submit job confirmation',
    body: 'Double-check your selected regions, prompts, and wallet authorization before continuing.',
    maxWidth: 'max-w-2xl'
  }
};

export default meta;

const Template = ({ title, body, maxWidth }) => {
  const [open, setOpen] = useState(true);
  return (
    <div className="min-h-[360px] bg-ctp-base text-ctp-text flex flex-col items-center justify-center gap-4 p-6">
      <div className="text-sm text-ctp-subtext0 max-w-lg text-center">
        Toggle the modal to inspect overlay, focus trap, and responsive layout styling.
      </div>
      <div className="flex gap-3">
        <button
          type="button"
          className="px-4 py-2 rounded bg-beacon-400 text-ctp-base font-medium"
          onClick={() => setOpen(true)}
        >
          Open modal
        </button>
        <button
          type="button"
          className="px-4 py-2 rounded border border-ctp-overlay0 text-ctp-text"
          onClick={() => setOpen(false)}
        >
          Close modal
        </button>
      </div>
      <Modal open={open} onClose={() => setOpen(false)} title={title} maxWidth={maxWidth}>
        <div className="space-y-4 text-slate-700">
          <p>{body}</p>
          <ul className="list-disc list-inside text-sm text-slate-600 space-y-1">
            <li>Verify at least 67% of regions are selected to satisfy success threshold.</li>
            <li>Confirm cryptographic signing is enabled and wallet authorization is fresh.</li>
            <li>Use the transparency log preview to ensure provenance metadata looks correct.</li>
          </ul>
          <div className="flex justify-end gap-2">
            <button className="px-3 py-1.5 text-sm text-slate-600" onClick={() => setOpen(false)}>
              Cancel
            </button>
            <button className="px-3 py-1.5 text-sm rounded bg-beacon-600 text-white hover:bg-beacon-700">
              Submit job
            </button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export const Default = {
  render: Template
};

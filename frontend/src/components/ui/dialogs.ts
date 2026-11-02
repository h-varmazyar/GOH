import type { ReactNode } from 'react';
import { create } from 'zustand';

export interface DialogOptions {
  title: string;
  message: ReactNode;
  /** A warning shown under the message, such as that the action cannot be undone. */
  warning?: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  /** danger marks destructive actions. */
  tone?: 'primary' | 'danger';
}

export interface PromptOptions extends DialogOptions {
  label?: string;
  initialValue?: string;
  placeholder?: string;
}

export interface OpenDialog {
  id: number;
  options: PromptOptions;
  prompt: boolean;
  resolve: (value: string | null) => void;
}

export const useDialogStore = create<{ dialog: OpenDialog | null }>(() => ({ dialog: null }));
let nextDialogId = 0;

function open(options: PromptOptions, prompt: boolean): Promise<string | null> {
  return new Promise((resolve) => {
    // A dialog opened while another is shown cancels the other one.
    useDialogStore.getState().dialog?.resolve(null);
    useDialogStore.setState({ dialog: { id: ++nextDialogId, options, prompt, resolve } });
  });
}

/** Shows a confirmation dialog in place of window.confirm. */
export const confirmDialog = (options: DialogOptions): Promise<boolean> =>
  open(options, false).then((value) => value !== null);

/** Shows a dialog with a text field in place of window.prompt; null when cancelled. */
export const promptDialog = (options: PromptOptions): Promise<string | null> => open(options, true);

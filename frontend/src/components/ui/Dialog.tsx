import { useState } from 'react';
import { AlertTriangle, HelpCircle } from 'lucide-react';
import { Button } from './Button';
import { Modal } from './Modal';
import { useDialogStore, type OpenDialog } from './dialogs';

/** Renders the open dialog; mounted once at the root of the app. */
export function DialogHost() {
  const dialog = useDialogStore((state) => state.dialog);
  if (!dialog) return null;
  return <DialogView key={dialog.id} dialog={dialog} />;
}

function DialogView({ dialog }: { dialog: OpenDialog }) {
  const { options, prompt, resolve } = dialog;
  const [value, setValue] = useState(options.initialValue || '');
  const danger = options.tone === 'danger';

  const close = (result: string | null) => {
    useDialogStore.setState({ dialog: null });
    resolve(result);
  };

  return (
    <Modal
      title={options.title}
      icon={danger ? <AlertTriangle className="h-5 w-5 text-danger-500" /> : <HelpCircle className="h-5 w-5 text-primary-500" />}
      onClose={() => close(null)}
      footer={
        <>
          <Button className="flex-1" variant={danger ? 'danger' : 'primary'} autoFocus={!prompt} onClick={() => close(value)}>
            {options.confirmLabel || 'تأیید'}
          </Button>
          <Button variant="ghost" onClick={() => close(null)}>
            {options.cancelLabel || 'انصراف'}
          </Button>
        </>
      }
    >
      <div className="text-sm leading-7 text-surface-700 dark:text-surface-300">{options.message}</div>
      {prompt && (
        <div>
          {options.label && <label className="mb-1.5 block text-sm font-medium text-surface-700 dark:text-surface-300">{options.label}</label>}
          <textarea
            autoFocus
            rows={3}
            value={value}
            placeholder={options.placeholder}
            onChange={(e) => setValue(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) close(value);
            }}
            className="w-full rounded-lg border border-surface-300 bg-white px-3 py-2 text-sm text-surface-900 placeholder:text-surface-400 focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500 dark:border-surface-600 dark:bg-surface-800 dark:text-surface-100 dark:placeholder:text-surface-500"
          />
        </div>
      )}
      {options.warning && (
        <div className={`flex gap-2 rounded-lg p-3 text-xs leading-6 ${danger ? 'bg-danger-50 text-danger-600 dark:bg-danger-950/40' : 'bg-warning-400/15 text-warning-700 dark:text-warning-400'}`}>
          <AlertTriangle className="mt-1 h-4 w-4 shrink-0" />
          <div>{options.warning}</div>
        </div>
      )}
    </Modal>
  );
}

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
      icon={danger ? <AlertTriangle className="h-5 w-5 text-error" aria-hidden="true" /> : <HelpCircle className="h-5 w-5 text-selected" aria-hidden="true" />}
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
      <div className="text-body-lg text-text-muted">{options.message}</div>
      {prompt && (
        <div>
          {options.label && <label htmlFor="dialog-prompt" className="mb-1.5 block text-label text-text">{options.label}</label>}
          <textarea
            id="dialog-prompt"
            autoFocus
            rows={3}
            value={value}
            placeholder={options.placeholder}
            onChange={(e) => setValue(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) close(value);
            }}
            className="field"
          />
        </div>
      )}
      {options.warning && (
        <div
          role="note"
          className={`flex gap-2 rounded-md border-s-3 p-3 text-body text-text ${danger ? 'border-error bg-error/8' : 'border-warning bg-warning/12'}`}
        >
          <AlertTriangle className={`mt-0.5 h-4 w-4 shrink-0 ${danger ? 'text-error' : 'text-warning-text'}`} aria-hidden="true" />
          <div>{options.warning}</div>
        </div>
      )}
    </Modal>
  );
}

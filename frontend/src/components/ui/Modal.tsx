import { useEffect, useId, type ReactNode } from 'react';
import { X } from 'lucide-react';

export function Modal({
  title,
  icon,
  onClose,
  children,
  footer,
  size = 'md',
}: {
  title: string;
  icon?: ReactNode;
  onClose: () => void;
  children: ReactNode;
  footer: ReactNode;
  size?: 'md' | 'lg';
}) {
  const titleId = useId();

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [onClose]);

  return (
    <div
      className="fixed inset-0 z-(--z-modal) flex items-center justify-center p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
    >
      <div className="absolute inset-0 bg-scrim" onClick={onClose} aria-hidden="true" />
      <div
        className={`relative flex max-h-[calc(100dvh-2rem)] w-full flex-col overflow-hidden rounded-xl border border-border bg-surface shadow-lg ${
          size === 'lg' ? 'max-w-2xl' : 'max-w-md'
        }`}
      >
        <div className="flex items-center justify-between gap-3 border-b border-border px-6 py-4">
          <h2 id={titleId} className="flex items-center gap-2 text-h3 text-text">
            {icon}
            {title}
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="-me-3 inline-flex h-11 w-11 items-center justify-center rounded-md text-text-subtle hover:bg-surface-muted hover:text-text"
            aria-label="بستن"
          >
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>
        <div className="space-y-4 overflow-y-auto px-6 py-5">{children}</div>
        <div className="flex gap-2 border-t border-border px-6 py-4">{footer}</div>
      </div>
    </div>
  );
}

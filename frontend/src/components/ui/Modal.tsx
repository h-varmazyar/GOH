import { useEffect, type ReactNode } from 'react';
import { X } from 'lucide-react';

export function Modal({
  title,
  icon,
  onClose,
  children,
  footer,
}: {
  title: string;
  icon?: ReactNode;
  onClose: () => void;
  children: ReactNode;
  footer: ReactNode;
}) {
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4" role="dialog" aria-modal="true">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative w-full max-w-md rounded-xl border border-surface-200 bg-white shadow-xl dark:border-surface-700 dark:bg-surface-900">
        <div className="flex items-center justify-between border-b border-surface-200 p-4 dark:border-surface-700">
          <h3 className="flex items-center gap-2 text-lg font-semibold text-surface-900 dark:text-white">
            {icon}
            {title}
          </h3>
          <button onClick={onClose} className="rounded p-1 hover:bg-surface-100 dark:hover:bg-surface-800">
            <X className="h-5 w-5 text-surface-500" />
          </button>
        </div>
        <div className="space-y-4 p-4">{children}</div>
        <div className="flex gap-2 border-t border-surface-200 p-4 dark:border-surface-700">{footer}</div>
      </div>
    </div>
  );
}

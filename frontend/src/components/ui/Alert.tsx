import type { ReactNode } from 'react';
import { AlertTriangle, CheckCircle2, Info, X, XCircle } from 'lucide-react';

type Tone = 'success' | 'info' | 'warning' | 'error';

interface AlertProps {
  tone: Tone;
  children: ReactNode;
  title?: ReactNode;
  /** Trailing content such as action buttons. */
  actions?: ReactNode;
  onDismiss?: () => void;
  className?: string;
}

/* §7 Alerts: body text stays `text`; the semantic color goes to the icon and a 3px start border. */
const tones: Record<Tone, { box: string; icon: string; Icon: typeof Info }> = {
  success: { box: 'border-success bg-success/8', icon: 'text-success', Icon: CheckCircle2 },
  info: { box: 'border-info bg-info/8', icon: 'text-info', Icon: Info },
  warning: { box: 'border-warning bg-warning/12', icon: 'text-warning-text', Icon: AlertTriangle },
  error: { box: 'border-error bg-error/8', icon: 'text-error', Icon: XCircle },
};

export function Alert({ tone, children, title, actions, onDismiss, className = '' }: AlertProps) {
  const { box, icon, Icon } = tones[tone];
  // Errors and urgent changes interrupt; the rest is announced politely.
  const role = tone === 'error' ? 'alert' : 'status';

  return (
    <div role={role} className={`flex items-start gap-3 rounded-md border-s-3 px-4 py-3 text-body text-text ${box} ${className}`}>
      <Icon className={`mt-0.5 h-5 w-5 shrink-0 ${icon}`} aria-hidden="true" />
      <div className="min-w-0 flex-1 space-y-1">
        {title && <p className="text-label">{title}</p>}
        <div className="break-words">{children}</div>
      </div>
      {actions && <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div>}
      {onDismiss && (
        <button
          type="button"
          onClick={onDismiss}
          className="-m-2 inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-sm text-text-subtle hover:bg-surface-muted hover:text-text"
          aria-label="بستن پیام"
        >
          <X className="h-4 w-4" aria-hidden="true" />
        </button>
      )}
    </div>
  );
}

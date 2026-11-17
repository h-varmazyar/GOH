import { useId } from 'react';

interface ToggleProps {
  enabled: boolean;
  onChange: (enabled: boolean) => void;
  label?: string;
  disabled?: boolean;
  /** Accessible name when there is no visible label. */
  'aria-label'?: string;
}

export function Toggle({ enabled, onChange, label, disabled = false, 'aria-label': ariaLabel }: ToggleProps) {
  const labelId = useId();

  return (
    <span className="inline-flex items-center gap-3">
      <button
        type="button"
        role="switch"
        aria-checked={enabled}
        aria-labelledby={label ? labelId : undefined}
        aria-label={label ? undefined : ariaLabel}
        disabled={disabled}
        onClick={() => onChange(!enabled)}
        className={`relative inline-flex h-6 w-11 shrink-0 items-center rounded-pill border transition-colors duration-(--duration-base)
          disabled:cursor-not-allowed disabled:opacity-40
          ${enabled ? 'border-action bg-action' : 'border-border-strong bg-surface-muted'}`}
      >
        <span
          aria-hidden="true"
          className={`absolute top-1/2 h-4 w-4 -translate-y-1/2 rounded-pill shadow-sm transition-[inset-inline-start] duration-(--duration-base) ease-standard
            ${enabled ? 'start-6 bg-on-action' : 'start-0.5 bg-border-strong'}`}
        />
      </button>
      {label && (
        <span
          id={labelId}
          className={`select-none text-body text-text ${disabled ? '' : 'cursor-pointer'}`}
          onClick={() => !disabled && onChange(!enabled)}
        >
          {label}
        </span>
      )}
    </span>
  );
}

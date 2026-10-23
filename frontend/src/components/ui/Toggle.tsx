interface ToggleProps {
  enabled: boolean;
  onChange: (enabled: boolean) => void;
  label?: string;
  disabled?: boolean;
}

export function Toggle({ enabled, onChange, label, disabled = false }: ToggleProps) {
  return (
    <label className="inline-flex items-center gap-3 cursor-pointer">
      <button
        type="button"
        role="switch"
        aria-checked={enabled}
        disabled={disabled}
        onClick={() => onChange(!enabled)}
        dir="ltr"
        className={`relative inline-flex h-6 w-11 shrink-0 rounded-full transition-colors
          focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 dark:focus:ring-offset-surface-900
          disabled:opacity-50 disabled:cursor-not-allowed
          ${enabled ? 'bg-primary-600' : 'bg-surface-300 dark:bg-surface-600'}`}
      >
        <span
          aria-hidden="true"
          className={`absolute top-1 h-4 w-4 rounded-full bg-white shadow-sm transition-all duration-200
            ${enabled ? 'left-6' : 'left-1'}`}
        />
      </button>
      {label && <span className="text-sm text-surface-700 dark:text-surface-300 select-none">{label}</span>}
    </label>
  );
}

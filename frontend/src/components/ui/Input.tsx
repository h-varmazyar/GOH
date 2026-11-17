import { forwardRef, useId, type InputHTMLAttributes, type ReactNode } from 'react';
import { AlertCircle } from 'lucide-react';

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  hint?: string;
  /** Decorative icon at the inline start of the field. */
  leadingIcon?: ReactNode;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ label, error, hint, leadingIcon, className = '', id, ...props }, ref) => {
    const generatedId = useId();
    const inputId = id || generatedId;
    const messageId = `${inputId}-message`;
    const message = error || hint;

    return (
      <div className="w-full">
        {label && (
          <label htmlFor={inputId} className="mb-1.5 block text-label text-text">
            {label}
          </label>
        )}
        {/* Shares the field direction so the icon and its padding stay on the same side. */}
        <div className="relative" dir={props.dir}>
          {leadingIcon && (
            <span
              className="pointer-events-none absolute inset-y-0 start-3 flex items-center text-text-subtle [&_svg]:h-4 [&_svg]:w-4"
              aria-hidden="true"
            >
              {leadingIcon}
            </span>
          )}
          <input
            ref={ref}
            id={inputId}
            aria-invalid={error ? true : undefined}
            aria-describedby={message ? messageId : undefined}
            className={`field ${leadingIcon ? 'ps-10' : ''} ${className}`}
            {...props}
          />
        </div>
        {error ? (
          <p id={messageId} className="mt-1.5 flex items-center gap-1.5 text-caption text-error">
            <AlertCircle className="h-4 w-4 shrink-0" aria-hidden="true" />
            {error}
          </p>
        ) : (
          hint && (
            <p id={messageId} className="mt-1.5 text-caption text-text-subtle">
              {hint}
            </p>
          )
        )}
      </div>
    );
  }
);

Input.displayName = 'Input';

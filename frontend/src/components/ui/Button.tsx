import type { ButtonHTMLAttributes, ReactNode } from 'react';
import { Loader2 } from 'lucide-react';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  /** danger is the destructive variant: its label must state the consequence. */
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
  /** sm: 40px dense desktop control · md: 44px on touch, 40px from lg · lg: 44px always. */
  size?: 'sm' | 'md' | 'lg';
  isLoading?: boolean;
  /** Read by screen readers while isLoading is set. */
  loadingLabel?: string;
  children: ReactNode;
}

const variants = {
  primary: 'bg-action text-on-action enabled:hover:bg-action-hover enabled:active:bg-action-pressed',
  secondary:
    'border border-border-strong text-brand-primary dark:text-text enabled:hover:bg-surface-muted enabled:active:bg-border',
  ghost: 'text-brand-primary dark:text-text enabled:hover:bg-surface-muted enabled:active:bg-border',
  danger: 'bg-destructive text-white enabled:hover:bg-error-hover enabled:active:bg-error-hover',
};

const sizes = {
  sm: 'h-10 px-4 text-body',
  md: 'h-11 px-5 text-body lg:h-10 lg:px-4',
  lg: 'h-11 px-5 text-body-lg',
};

export function Button({
  variant = 'primary',
  size = 'md',
  isLoading = false,
  loadingLabel = 'در حال انجام…',
  children,
  className = '',
  disabled,
  ...props
}: ButtonProps) {
  return (
    <button
      className={`relative inline-flex shrink-0 items-center justify-center gap-2 rounded-md font-semibold whitespace-nowrap
        transition-colors duration-(--duration-fast)
        disabled:cursor-not-allowed disabled:opacity-40
        ${variants[variant]} ${sizes[size]} ${className}`}
      disabled={disabled || isLoading}
      aria-busy={isLoading || undefined}
      {...props}
    >
      {/* The label keeps its space while loading so the button width stays fixed. */}
      <span className={`inline-flex items-center gap-2 ${isLoading ? 'invisible' : ''}`}>{children}</span>
      {isLoading && (
        <span className="absolute inset-0 flex items-center justify-center">
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          <span className="sr-only">{loadingLabel}</span>
        </span>
      )}
    </button>
  );
}

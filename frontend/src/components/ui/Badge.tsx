import type { ReactNode } from 'react';

interface BadgeProps {
  children: ReactNode;
  /** Semantic variants only for real status; default is the neutral badge. */
  variant?: 'default' | 'success' | 'danger' | 'warning' | 'primary' | 'info';
  size?: 'sm' | 'md';
}

const variants = {
  default: 'bg-surface-muted text-text-muted',
  success: 'bg-success/12 text-success',
  danger: 'bg-error/12 text-error',
  warning: 'bg-warning/20 text-warning-text',
  info: 'bg-info/12 text-info',
  primary: 'bg-selected/15 text-link',
};

const sizes = {
  sm: 'h-6 px-2.5',
  md: 'h-7 px-3',
};

export function Badge({ children, variant = 'default', size = 'sm' }: BadgeProps) {
  return (
    <span
      className={`inline-flex shrink-0 items-center gap-1 rounded-pill text-caption font-semibold whitespace-nowrap ${variants[variant]} ${sizes[size]}`}
    >
      {children}
    </span>
  );
}

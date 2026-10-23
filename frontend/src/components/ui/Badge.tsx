import type { ReactNode } from 'react';

interface BadgeProps {
  children: ReactNode;
  variant?: 'default' | 'success' | 'danger' | 'warning' | 'primary';
  size?: 'sm' | 'md';
}

const variants = {
  default: 'bg-surface-100 text-surface-700 dark:bg-surface-700 dark:text-surface-300',
  success: 'bg-success-400/20 text-success-600 dark:text-success-400',
  danger: 'bg-danger-400/20 text-danger-600 dark:text-danger-400',
  warning: 'bg-warning-400/20 text-warning-600 dark:text-warning-400',
  primary: 'bg-primary-400/20 text-primary-700 dark:text-primary-300',
};

const sizes = {
  sm: 'px-2 py-0.5 text-xs',
  md: 'px-2.5 py-1 text-sm',
};

export function Badge({ children, variant = 'default', size = 'sm' }: BadgeProps) {
  return (
    <span className={`inline-flex items-center rounded-full font-medium ${variants[variant]} ${sizes[size]}`}>
      {children}
    </span>
  );
}

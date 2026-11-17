import type { ButtonHTMLAttributes, ReactNode } from 'react';

interface IconButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'> {
  /** Accessible name, also shown as the tooltip. */
  title: string;
  icon: ReactNode;
  tone?: 'default' | 'danger';
}

const toneClasses = {
  default: 'text-text-muted hover:text-selected',
  danger: 'text-error hover:text-error-hover',
};

/** A bare icon action: no background, only the icon color changes on hover. */
export function IconButton({ title, icon, tone = 'default', className = '', type = 'button', ...props }: IconButtonProps) {
  return (
    <button
      type={type}
      title={title}
      aria-label={title}
      className={`inline-flex shrink-0 items-center justify-center rounded-sm p-0.5 transition-colors duration-(--duration-base)
        focus-visible:outline-2 focus-visible:outline-focus-ring disabled:cursor-not-allowed disabled:opacity-40
        [&_svg]:h-4 [&_svg]:w-4 ${toneClasses[tone]} ${className}`}
      {...props}
    >
      {icon}
    </button>
  );
}

import { useId } from 'react';

interface LogoProps {
  size?: number;
  className?: string;
}

/** Brand mark: deep forest green → mint gradient (gradient.brand), white gate and rising line. */
export function Logo({ size = 32, className = '' }: LogoProps) {
  const id = useId();
  const bg = `${id}-bg`;

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 64 64"
      width={size}
      height={size}
      className={className}
      role="img"
      aria-label="Gate Trading"
    >
      <defs>
        <linearGradient id={bg} x1="0" y1="0" x2="64" y2="64" gradientUnits="userSpaceOnUse">
          <stop offset="0" style={{ stopColor: 'var(--color-brand-primary)' }} />
          <stop offset="1" style={{ stopColor: 'var(--color-brand-mint)' }} />
        </linearGradient>
      </defs>

      <rect x="2" y="2" width="60" height="60" rx="16" fill={`url(#${bg})`} />

      {/* Gate uprights */}
      <rect x="14" y="16" width="3" height="32" rx="1.5" fill="#FFFFFF" fillOpacity="0.92" />
      <rect x="47" y="16" width="3" height="32" rx="1.5" fill="#FFFFFF" fillOpacity="0.92" />

      {/* Top arch */}
      <path d="M14 18 Q32 8 50 18" stroke="#FFFFFF" strokeOpacity="0.92" strokeWidth="3" strokeLinecap="round" fill="none" />

      {/* Rising chart line */}
      <path
        d="M22 44 L30 36 L36 40 L46 28"
        stroke="#FFFFFF"
        strokeWidth="3.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        fill="none"
      />
      <circle cx="46" cy="28" r="2.75" style={{ fill: 'var(--color-neutral-50)' }} />
    </svg>
  );
}

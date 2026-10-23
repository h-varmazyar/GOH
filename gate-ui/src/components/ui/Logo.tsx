interface LogoProps {
  size?: number;
  className?: string;
}

export function Logo({ size = 32, className = '' }: LogoProps) {
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
        <linearGradient id="logo-bg" x1="0" y1="0" x2="64" y2="64" gradientUnits="userSpaceOnUse">
          <stop offset="0" stopColor="#2FA084" />
          <stop offset="1" stopColor="#12372A" />
        </linearGradient>
        <linearGradient id="logo-arrow" x1="32" y1="14" x2="32" y2="50" gradientUnits="userSpaceOnUse">
          <stop offset="0" stopColor="#8BDFDD" />
          <stop offset="1" stopColor="#FFFFFF" />
        </linearGradient>
      </defs>

      <rect x="2" y="2" width="60" height="60" rx="14" fill="url(#logo-bg)" />
      <rect
        x="2"
        y="2"
        width="60"
        height="60"
        rx="14"
        fill="none"
        stroke="#FFFFFF"
        strokeOpacity="0.08"
        strokeWidth="1"
      />

      {/* Gate uprights */}
      <rect x="14" y="16" width="3" height="32" rx="1.5" fill="#FFFFFF" fillOpacity="0.9" />
      <rect x="47" y="16" width="3" height="32" rx="1.5" fill="#FFFFFF" fillOpacity="0.9" />

      {/* Top arch */}
      <path
        d="M14 18 Q32 8 50 18"
        stroke="#FFFFFF"
        strokeOpacity="0.9"
        strokeWidth="3"
        strokeLinecap="round"
        fill="none"
      />

      {/* Upward chart line */}
      <path
        d="M22 44 L30 36 L36 40 L46 28"
        stroke="url(#logo-arrow)"
        strokeWidth="3.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        fill="none"
      />

      <circle cx="46" cy="28" r="2.5" fill="#8BDFDD" />
    </svg>
  );
}

import { useMemo } from 'react';
import { encodeQr } from '../../utils/qrcode';

interface QrCodeProps {
  value: string;
  /** Rendered width and height in pixels. */
  size?: number;
  /** Accessible description of what the code contains. */
  label?: string;
  className?: string;
}

/** Quiet zone required around the symbol, in modules. */
const QUIET_ZONE = 4;

/**
 * Renders a QR code as crisp SVG. Always dark-on-white, also in dark mode,
 * because phone scanners need that contrast.
 */
export function QrCode({ value, size = 208, label = 'QR code', className = '' }: QrCodeProps) {
  const { path, dimension } = useMemo(() => {
    const modules = encodeQr(value);
    let d = '';
    modules.forEach((row, y) => {
      row.forEach((dark, x) => {
        if (dark) d += `M${x + QUIET_ZONE} ${y + QUIET_ZONE}h1v1h-1z`;
      });
    });
    return { path: d, dimension: modules.length + QUIET_ZONE * 2 };
  }, [value]);

  return (
    <svg
      role="img"
      aria-label={label}
      width={size}
      height={size}
      viewBox={`0 0 ${dimension} ${dimension}`}
      shapeRendering="crispEdges"
      className={`rounded-md ${className}`}
    >
      <rect width={dimension} height={dimension} fill="#ffffff" />
      <path d={path} fill="#000000" />
    </svg>
  );
}

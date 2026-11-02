export const formatDate = (value: string | null) =>
  value ? new Date(value).toLocaleDateString('fa-IR', { timeZone: 'UTC' }) : '—';

export const formatDateTime = (value: string | null) => (value ? new Date(value).toLocaleString('fa-IR') : '—');

export const formatNumber = (value: number | null, digits = 0) =>
  value === null || value === undefined ? '—' : value.toLocaleString('fa-IR', { maximumFractionDigits: digits });

export const errorMessage = (e: unknown, fallback: string) => (e instanceof Error ? e.message : fallback);

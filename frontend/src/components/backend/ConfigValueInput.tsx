import type { BackendConfig, ConfigValue } from '../../services/backendService';

interface ConfigValueInputProps {
  config: BackendConfig;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  ariaLabel?: string;
}

const normalizeType = (type?: string) => (type || 'string').trim().toLowerCase().replaceAll('_', '-');
const booleanTypes = new Set(['boolean', 'bool']);
const structuredTypes = new Set(['json', 'object', 'array', 'list', 'map', 'slice']);
const dateTypes: Record<string, string> = {
  date: 'date',
  time: 'time',
  datetime: 'datetime-local',
  'datetime-local': 'datetime-local',
  timestamp: 'datetime-local',
  month: 'month',
  week: 'week',
};
const nativeTextTypes = new Set(['email', 'url', 'tel', 'password', 'color']);

const isNumericType = (type: string) =>
  ['number', 'integer', 'decimal', 'float', 'double'].includes(type)
  || /^(u?int)(8|16|32|64)?$/.test(type)
  || /^float(32|64)$/.test(type);

export const serializeConfigValue = (value: ConfigValue, type?: string): string => {
  const normalizedType = normalizeType(type);
  if (value === null || value === undefined) return '';
  if (structuredTypes.has(normalizedType)) {
    return typeof value === 'string' ? value : JSON.stringify(value, null, 2);
  }
  if ((normalizedType === 'datetime' || normalizedType === 'timestamp') && typeof value === 'string') {
    return value.replace(/Z$/, '').slice(0, 16);
  }
  return String(value);
};

export const parseConfigValue = (value: string, type?: string): ConfigValue => {
  const normalizedType = normalizeType(type);
  if (booleanTypes.has(normalizedType)) return value === 'true';
  if (isNumericType(normalizedType)) {
    const number = Number(value);
    if (value.trim() === '' || Number.isNaN(number)) throw new Error('مقدار عددی معتبر نیست');
    return number;
  }
  if (structuredTypes.has(normalizedType)) {
    try {
      return JSON.parse(value) as ConfigValue;
    } catch {
      throw new Error('ساختار JSON معتبر نیست');
    }
  }
  return value;
};

export function ConfigValueInput({ config, value, onChange, disabled, ariaLabel }: ConfigValueInputProps) {
  const normalizedType = normalizeType(config.type);
  const baseClass = `w-full px-3 py-2 rounded-lg border bg-white dark:bg-surface-800
    border-surface-300 dark:border-surface-600 text-surface-900 dark:text-surface-100
    focus:outline-none focus:ring-2 focus:ring-primary-500 disabled:opacity-50 disabled:cursor-not-allowed`;

  if (booleanTypes.has(normalizedType)) {
    return (
      <label className="inline-flex items-center gap-3 py-2 cursor-pointer">
        <input
          type="checkbox"
          aria-label={ariaLabel}
          checked={value === 'true'}
          onChange={(event) => onChange(String(event.target.checked))}
          disabled={disabled}
          className="w-5 h-5 rounded border-surface-300 text-primary-600 focus:ring-primary-500"
        />
        <span className="text-sm text-surface-700 dark:text-surface-300">
          {value === 'true' ? 'فعال' : 'غیرفعال'}
        </span>
      </label>
    );
  }

  if (structuredTypes.has(normalizedType)) {
    return (
      <textarea
        aria-label={ariaLabel}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        disabled={disabled}
        rows={5}
        dir="ltr"
        className={`${baseClass} font-mono text-sm`}
      />
    );
  }

  const inputType = isNumericType(normalizedType)
    ? 'number'
    : dateTypes[normalizedType] ?? (nativeTextTypes.has(normalizedType) ? normalizedType : 'text');

  return (
    <input
      type={inputType}
      aria-label={ariaLabel}
      value={value}
      onChange={(event) => onChange(event.target.value)}
      disabled={disabled}
      step={isNumericType(normalizedType) ? 'any' : undefined}
      dir={inputType === 'text' ? 'auto' : 'ltr'}
      className={baseClass}
    />
  );
}
import { Input } from '../ui/Input';
import { Toggle } from '../ui/Toggle';
import type { Strategy } from '../../types';

interface Props {
  strategy: Strategy;
  onChange: (updates: Partial<Strategy>) => void;
}

export function StepFilters({ strategy, onChange }: Props) {
  const { filters } = strategy;

  const updateMarketFilter = (key: string, value: number | boolean) => {
    onChange({
      filters: {
        ...filters,
        market: { ...filters.market, [key]: value },
      },
    });
  };

  const updateVolatility = (key: string, value: number) => {
    onChange({
      filters: {
        ...filters,
        volatility: { atr: { ...filters.volatility.atr, [key]: value } },
      },
    });
  };

  const updateCooldown = (key: string, value: number) => {
    onChange({
      filters: {
        ...filters,
        cooldown: { ...filters.cooldown, [key]: value },
      },
    });
  };

  const updateCorrelation = (key: string, value: boolean | number) => {
    onChange({
      filters: {
        ...filters,
        correlation: { ...filters.correlation, [key]: value },
      },
    });
  };

  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold text-surface-900 dark:text-white">فیلترها</h2>

      {/* Market Filters */}
      <div>
        <h3 className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-3">فیلترهای مارکت</h3>
        <div className="grid sm:grid-cols-2 gap-4">
          <Input
            label="حداقل حجم روزانه (USDT)"
            type="number"
            value={filters.market.min_daily_volume_usdt}
            onChange={(e) => updateMarketFilter('min_daily_volume_usdt', Number(e.target.value))}
          />
          <Input
            label="حداکثر اسپرد (%)"
            type="number"
            value={filters.market.max_spread_percent}
            onChange={(e) => updateMarketFilter('max_spread_percent', Number(e.target.value))}
            step="0.01"
          />
          <Input
            label="حداقل قیمت"
            type="number"
            value={filters.market.min_price}
            onChange={(e) => updateMarketFilter('min_price', Number(e.target.value))}
            step="0.0001"
          />
          <Input
            label="حداکثر قیمت"
            type="number"
            value={filters.market.max_price}
            onChange={(e) => updateMarketFilter('max_price', Number(e.target.value))}
          />
        </div>
        <div className="mt-3">
          <Toggle
            enabled={filters.market.blacklist_low_liquidity}
            onChange={(v) => updateMarketFilter('blacklist_low_liquidity', v)}
            label="حذف نمادهای با نقدینگی پایین"
          />
        </div>
      </div>

      {/* Volatility */}
      <div>
        <h3 className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-3">نوسان (ATR)</h3>
        <div className="grid sm:grid-cols-2 gap-4">
          <Input
            label="حداقل پرسنتایل"
            type="number"
            value={filters.volatility.atr.min_percentile}
            onChange={(e) => updateVolatility('min_percentile', Number(e.target.value))}
            min={0}
            max={100}
          />
          <Input
            label="حداکثر پرسنتایل"
            type="number"
            value={filters.volatility.atr.max_percentile}
            onChange={(e) => updateVolatility('max_percentile', Number(e.target.value))}
            min={0}
            max={100}
          />
        </div>
      </div>

      {/* Cooldown */}
      <div>
        <h3 className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-3">زمان انتظار (Cooldown)</h3>
        <div className="grid sm:grid-cols-2 gap-4">
          <Input
            label="بعد از ورود (ثانیه)"
            type="number"
            value={filters.cooldown.after_entry_seconds}
            onChange={(e) => updateCooldown('after_entry_seconds', Number(e.target.value))}
            min={0}
          />
          <Input
            label="بعد از استاپ‌لاس (ثانیه)"
            type="number"
            value={filters.cooldown.after_stoploss_seconds}
            onChange={(e) => updateCooldown('after_stoploss_seconds', Number(e.target.value))}
            min={0}
          />
        </div>
      </div>

      {/* Correlation */}
      <div>
        <h3 className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-3">همبستگی (Correlation)</h3>
        <div className="space-y-3">
          <Toggle
            enabled={filters.correlation.enabled}
            onChange={(v) => updateCorrelation('enabled', v)}
            label="فعال بودن فیلتر همبستگی"
          />
          {filters.correlation.enabled && (
            <Input
              label="حداکثر پوزیشن هم‌جهت"
              type="number"
              value={filters.correlation.max_same_direction_positions}
              onChange={(e) => updateCorrelation('max_same_direction_positions', Number(e.target.value))}
              min={1}
              max={50}
            />
          )}
        </div>
      </div>
    </div>
  );
}

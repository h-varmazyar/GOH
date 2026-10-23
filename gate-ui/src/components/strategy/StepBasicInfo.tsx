import { Input } from '../ui/Input';
import { Select } from '../ui/Select';
import { Toggle } from '../ui/Toggle';
import { mockProviders, mockTags } from '../../data/mock';
import type { Strategy, AssetClass } from '../../types';

interface Props {
  strategy: Strategy;
  onChange: (updates: Partial<Strategy>) => void;
}

export function StepBasicInfo({ strategy, onChange }: Props) {
  const filteredProviders = mockProviders.filter((provider) => provider.assetClass === strategy.assetClass);
  const handleAssetClass = (assetClass: AssetClass) => {
    const firstProvider = mockProviders.find((provider) => provider.assetClass === assetClass);
    onChange({ assetClass, provider: firstProvider?.id || '', providerName: firstProvider?.name || '', market: { ...strategy.market, symbols: [], excluded_symbols: [], sessions: assetClass === 'stock' ? ['tse_market'] : ['24x7'] } });
  };
  const handleTagToggle = (tag: string) => {
    const tags = strategy.tags.includes(tag)
      ? strategy.tags.filter((t) => t !== tag)
      : [...strategy.tags, tag];
    onChange({ tags });
  };

  return (
    <div className="space-y-5">
      <h2 className="text-lg font-semibold text-surface-900 dark:text-white">اطلاعات پایه استراتژی</h2>

      <Input
        label="نام استراتژی"
        value={strategy.name}
        onChange={(e) => onChange({ name: e.target.value })}
        placeholder="مثال: stochastic_rsi_trend"
        hint="نام یکتا برای شناسایی استراتژی"
      />

      <div className="grid sm:grid-cols-2 gap-4">
        <Select
          label="کلاس دارایی"
          value={strategy.assetClass}
          onChange={(e) => handleAssetClass(e.target.value as AssetClass)}
          options={[{ value: 'crypto', label: 'رمزارز' }, { value: 'stock', label: 'سهام تهران' }]}
        />
        <Select
          label="ارائه‌دهنده داده"
          value={strategy.provider}
          onChange={(e) => {
            const provider = filteredProviders.find((item) => item.id === e.target.value);
            onChange({ provider: e.target.value, providerName: provider?.name || e.target.value, market: { ...strategy.market, symbols: [], excluded_symbols: [] } });
          }}
          options={filteredProviders.map((p) => ({ value: p.id, label: p.name }))}
          placeholder="انتخاب ارائه‌دهنده..."
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-surface-700 dark:text-surface-300 mb-1.5">
          توضیحات
        </label>
        <textarea
          value={strategy.description}
          onChange={(e) => onChange({ description: e.target.value })}
          placeholder="توضیح مختصری درباره استراتژی..."
          rows={3}
          className="w-full px-3 py-2 rounded-lg border transition-colors
            bg-white dark:bg-surface-800
            border-surface-300 dark:border-surface-600
            text-surface-900 dark:text-surface-100
            placeholder:text-surface-400 dark:placeholder:text-surface-500
            focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500
            resize-none"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-surface-700 dark:text-surface-300 mb-2">
          برچسب‌ها (Tags)
        </label>
        <div className="flex flex-wrap gap-2">
          {mockTags.map((tag) => (
            <button
              key={tag}
              type="button"
              onClick={() => handleTagToggle(tag)}
              className={`px-3 py-1.5 rounded-full text-sm font-medium transition-colors border
                ${strategy.tags.includes(tag)
                  ? 'bg-primary-600 text-white border-primary-600'
                  : 'bg-white dark:bg-surface-800 text-surface-600 dark:text-surface-400 border-surface-300 dark:border-surface-600 hover:border-primary-400'
                }`}
            >
              {tag}
            </button>
          ))}
        </div>
      </div>

      <Toggle
        enabled={strategy.enabled}
        onChange={(enabled) => onChange({ enabled })}
        label="فعال بودن استراتژی"
      />
    </div>
  );
}

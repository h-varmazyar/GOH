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
      <h2 className="text-h3 font-semibold text-text">اطلاعات پایه استراتژی</h2>

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
        <label className="block text-body font-medium text-text-muted mb-1.5">
          توضیحات
        </label>
        <textarea
          value={strategy.description}
          onChange={(e) => onChange({ description: e.target.value })}
          placeholder="توضیح مختصری درباره استراتژی..."
          rows={3}
          className="field resize-none"
        />
      </div>

      <div>
        <label className="block text-body font-medium text-text-muted mb-2">
          برچسب‌ها (Tags)
        </label>
        <div className="flex flex-wrap gap-2">
          {mockTags.map((tag) => (
            <button
              key={tag}
              type="button"
              onClick={() => handleTagToggle(tag)}
              className={`px-3 py-1.5 rounded-pill text-body font-medium transition-colors border
                ${strategy.tags.includes(tag)
                  ? 'bg-action text-on-action border-selected'
                  : 'bg-surface text-text-muted border-border-strong hover:border-selected'
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

import { useState } from 'react';
import { Search } from 'lucide-react';
import { Select } from '../ui/Select';
import { Badge } from '../ui/Badge';
import { Toggle } from '../ui/Toggle';
import { mockMarkets, mockTimeframes, mockSessions } from '../../data/mock';
import type { Strategy } from '../../types';
import { Input } from '../ui/Input';

interface Props {
  strategy: Strategy;
  onChange: (updates: Partial<Strategy>) => void;
}

export function StepMarket({ strategy, onChange }: Props) {
  const [searchSymbol, setSearchSymbol] = useState('');
  const [selectAll, setSelectAll] = useState(strategy.market.symbols.includes('*'));

  const availableMarkets = mockMarkets.filter((market) => market.assetClass === strategy.assetClass && market.providerIds.includes(strategy.provider));
  const filteredMarkets = availableMarkets.filter(
    (m) =>
      m.symbol.toLowerCase().includes(searchSymbol.toLowerCase()) ||
      m.name.toLowerCase().includes(searchSymbol.toLowerCase())
  );

  const handleSelectAll = (checked: boolean) => {
    setSelectAll(checked);
    if (checked) {
      onChange({ market: { ...strategy.market, symbols: ['*'] } });
    } else {
      onChange({ market: { ...strategy.market, symbols: [] } });
    }
  };

  const handleSymbolToggle = (symbol: string) => {
    if (selectAll) return;
    const symbols = strategy.market.symbols.includes(symbol)
      ? strategy.market.symbols.filter((s) => s !== symbol)
      : [...strategy.market.symbols, symbol];
    onChange({ market: { ...strategy.market, symbols } });
  };

  const handleExcludeToggle = (symbol: string) => {
    const excluded = strategy.market.excluded_symbols.includes(symbol)
      ? strategy.market.excluded_symbols.filter((s) => s !== symbol)
      : [...strategy.market.excluded_symbols, symbol];
    onChange({ market: { ...strategy.market, excluded_symbols: excluded } });
  };

  const handleTimeframeToggle = (tf: string) => {
    const timeframes = strategy.market.timeframes.includes(tf)
      ? strategy.market.timeframes.filter((t) => t !== tf)
      : [...strategy.market.timeframes, tf];
    onChange({ market: { ...strategy.market, timeframes } });
  };

  const handleSessionChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value;
    if (value && !strategy.market.sessions.includes(value)) {
      onChange({ market: { ...strategy.market, sessions: [...strategy.market.sessions, value] } });
    }
  };

  const removeSession = (session: string) => {
    onChange({ market: { ...strategy.market, sessions: strategy.market.sessions.filter((s) => s !== session) } });
  };

  return (
    <div className="space-y-6">
      <h2 className="text-h3 font-semibold text-text">تنظیمات مارکت</h2>

      {/* Symbols */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <label className="text-body font-medium text-text-muted">نمادها (Symbols)</label>
          <Toggle enabled={selectAll} onChange={handleSelectAll} label="همه نمادها" />
        </div>

        {!selectAll && (
          <>
            <div className="mb-3"><Input leadingIcon={<Search />} aria-label="جستجو" value={searchSymbol} onChange={(e) => setSearchSymbol(e.target.value)} placeholder="جستجوی نماد..." /></div>
            <div className="max-h-48 overflow-y-auto border border-border rounded-md p-2 space-y-1">
              {filteredMarkets.map((market) => (
                <label
                  key={market.symbol}
                  className="flex items-center gap-3 p-2 rounded-sm hover:bg-surface-muted cursor-pointer"
                >
                  <input
                    type="checkbox"
                    checked={strategy.market.symbols.includes(market.symbol)}
                    onChange={() => handleSymbolToggle(market.symbol)}
                    className="w-4 h-4 rounded-sm border-border-strong text-link"
                  />
                  <span className="text-body text-text font-medium">{market.symbol}</span>
                  <span className="text-caption text-text-subtle">{market.name}</span>
                </label>
              ))}
            </div>
          </>
        )}

        {/* Selected symbols */}
        {strategy.market.symbols.length > 0 && !selectAll && (
          <div className="flex flex-wrap gap-1 mt-2">
            {strategy.market.symbols.map((s) => (
              <Badge key={s} variant="primary" size="sm">{s}</Badge>
            ))}
          </div>
        )}
      </div>

      {/* Excluded Symbols */}
      <div>
        <label className="block text-body font-medium text-text-muted mb-2">
          نمادهای حذف شده
        </label>
        <div className="max-h-36 overflow-y-auto border border-border rounded-md p-2 space-y-1">
          {availableMarkets.map((market) => (
            <label
              key={market.symbol}
              className="flex items-center gap-3 p-2 rounded-sm hover:bg-surface-muted cursor-pointer"
            >
              <input
                type="checkbox"
                checked={strategy.market.excluded_symbols.includes(market.symbol)}
                onChange={() => handleExcludeToggle(market.symbol)}
                className="w-4 h-4 rounded-sm border-border-strong text-error"
              />
              <span className="text-body text-text">{market.symbol}</span>
            </label>
          ))}
        </div>
        {strategy.market.excluded_symbols.length > 0 && (
          <div className="flex flex-wrap gap-1 mt-2">
            {strategy.market.excluded_symbols.map((s) => (
              <Badge key={s} variant="danger" size="sm">{s}</Badge>
            ))}
          </div>
        )}
      </div>

      {/* Timeframes */}
      <div>
        <label className="block text-body font-medium text-text-muted mb-2">
          تایم‌فریم‌ها
        </label>
        <div className="flex flex-wrap gap-2">
          {mockTimeframes.map((tf) => (
            <button
              key={tf.value}
              type="button"
              onClick={() => handleTimeframeToggle(tf.value)}
              className={`px-3 py-1.5 rounded-md text-body font-medium transition-colors border
                ${strategy.market.timeframes.includes(tf.value)
                  ? 'bg-action text-on-action border-selected'
                  : 'bg-surface text-text-muted border-border-strong hover:border-selected'
                }`}
            >
              {tf.label}
            </button>
          ))}
        </div>
      </div>

      {/* Sessions */}
      <div>
        <Select
          label="جلسات معاملاتی"
          value=""
          onChange={handleSessionChange}
          options={mockSessions.map((s) => ({ value: s, label: s }))}
          placeholder="افزودن جلسه..."
        />
        <div className="flex flex-wrap gap-1 mt-2">
          {strategy.market.sessions.map((s) => (
            <button
              key={s}
              onClick={() => removeSession(s)}
              className="inline-flex items-center gap-1 px-2 py-1 rounded-pill text-caption font-medium bg-selected/15 text-link hover:bg-selected/20 transition-colors"
            >
              {s} ×
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}

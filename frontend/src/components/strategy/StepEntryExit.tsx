import { useState, useMemo } from 'react';
import { Plus, Trash2, Info } from 'lucide-react';
import { Input } from '../ui/Input';
import { Button } from '../ui/Button';
import { Autocomplete } from '../ui/Autocomplete';
import { Badge } from '../ui/Badge';
import { mockIndicators } from '../../data/mock';
import type { Strategy } from '../../types';
import { Alert } from '../ui/Alert';

interface Props {
  strategy: Strategy;
  onChange: (updates: Partial<Strategy>) => void;
}

// Common helper functions and properties
const HELPERS = ['crosses_over', 'crosses_under', 'and', 'or', 'not'];
const COMMON_FIELDS = ['close', 'open', 'high', 'low', 'volume', 'trailing_stop'];

export function StepEntryExit({ strategy, onChange }: Props) {
  const [newLongEntry, setNewLongEntry] = useState('');
  const [newShortEntry, setNewShortEntry] = useState('');
  const [newLongExit, setNewLongExit] = useState('');
  const [newShortExit, setNewShortExit] = useState('');

  // Build suggestions from defined indicators
  const suggestions = useMemo(() => {
    const list: string[] = [];
    Object.entries(strategy.indicators).forEach(([name, ind]) => {
      // Find indicator type definition for known properties
      const def = mockIndicators.find((i) => i.type === ind.type);
      list.push(name);
      // Common properties per type
      const propertiesByType: Record<string, string[]> = {
        ema: ['ema'],
        sma: ['sma'],
        rsi: ['rsi'],
        macd: ['macd', 'signal', 'histogram'],
        stochastic: ['k', 'd'],
        adx: ['adx', 'plus_di', 'minus_di'],
        bollinger: ['upper', 'middle', 'lower'],
        atr: ['atr', 'percentile'],
        vwap: ['value'],
        volume_sma: ['sma20', 'current'],
      };
      const props = propertiesByType[ind.type] || [];
      props.forEach((p) => list.push(`${name}.${p}`));

      // Also expose configs as accessible
      if (def) {
        def.configs.forEach((c) => {
          list.push(`${name}.${c.key}`);
        });
      }
    });
    return [...new Set([...list, ...HELPERS, ...COMMON_FIELDS])];
  }, [strategy.indicators]);

  const indicatorNames = Object.keys(strategy.indicators);

  // Entry Long
  const addLongEntry = () => {
    if (!newLongEntry.trim()) return;
    onChange({
      entry: {
        ...strategy.entry,
        long: { ...strategy.entry.long, all: [...strategy.entry.long.all, newLongEntry.trim()] },
      },
    });
    setNewLongEntry('');
  };
  const removeLongEntry = (i: number) => onChange({
    entry: { ...strategy.entry, long: { ...strategy.entry.long, all: strategy.entry.long.all.filter((_, idx) => idx !== i) } },
  });

  // Entry Short
  const addShortEntry = () => {
    if (!newShortEntry.trim()) return;
    onChange({
      entry: {
        ...strategy.entry,
        short: { ...strategy.entry.short, all: [...strategy.entry.short.all, newShortEntry.trim()] },
      },
    });
    setNewShortEntry('');
  };
  const removeShortEntry = (i: number) => onChange({
    entry: { ...strategy.entry, short: { ...strategy.entry.short, all: strategy.entry.short.all.filter((_, idx) => idx !== i) } },
  });

  // Exit Long
  const addLongExit = () => {
    if (!newLongExit.trim()) return;
    onChange({
      exit: { ...strategy.exit, long: { any: [...strategy.exit.long.any, newLongExit.trim()] } },
    });
    setNewLongExit('');
  };
  const removeLongExit = (i: number) => onChange({
    exit: { ...strategy.exit, long: { any: strategy.exit.long.any.filter((_, idx) => idx !== i) } },
  });

  // Exit Short
  const addShortExit = () => {
    if (!newShortExit.trim()) return;
    onChange({
      exit: { ...strategy.exit, short: { any: [...strategy.exit.short.any, newShortExit.trim()] } },
    });
    setNewShortExit('');
  };
  const removeShortExit = (i: number) => onChange({
    exit: { ...strategy.exit, short: { any: strategy.exit.short.any.filter((_, idx) => idx !== i) } },
  });

  // Score weights
  const updateLongScore = (key: string, value: number) => {
    onChange({
      entry: {
        ...strategy.entry,
        long: {
          ...strategy.entry.long,
          score: {
            ...strategy.entry.long.score,
            ...(key === 'minimum' ? { minimum: value } : { weights: { ...strategy.entry.long.score.weights, [key]: value } }),
          },
        },
      },
    });
  };

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-h3 font-semibold text-text">شرایط ورود و خروج</h2>
        <p className="text-body text-text-subtle mt-1">
          شرط‌ها با نام اندیکاتور شروع می‌شوند و سپس با نقطه (.) به ویژگی‌های آن دسترسی پیدا می‌کنید.
        </p>
      </div>

      {/* Available indicators reminder */}
      {indicatorNames.length > 0 ? (
        <div className="p-3 rounded-md bg-selected/10 border border-selected/40">
          <div className="flex items-start gap-2">
            <Info className="w-4 h-4 text-link mt-0.5 flex-shrink-0" />
            <div className="flex-1">
              <p className="text-caption font-medium text-link mb-2">
                اندیکاتورهای موجود (کلیک کنید برای کپی):
              </p>
              <div className="flex flex-wrap gap-1.5">
                {indicatorNames.map((n) => (
                  <button
                    key={n}
                    type="button"
                    onClick={() => navigator.clipboard?.writeText(n)}
                    className="px-2 py-0.5 rounded-sm text-caption font-mono bg-surface border border-selected/50 text-link hover:bg-selected/15 transition-colors"
                    dir="ltr"
                  >
                    {n}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>
      ) : (
        <Alert tone="warning">ابتدا در مرحله «اندیکاتورها» حداقل یک اندیکاتور اضافه کنید.</Alert>
      )}

      {/* Entry Long */}
      <ConditionSection
        title="شرایط ورود Long"
        subtitle="همه شروط باید برقرار باشند"
        conditions={strategy.entry.long.all}
        newValue={newLongEntry}
        onNewValueChange={setNewLongEntry}
        onAdd={addLongEntry}
        onRemove={removeLongEntry}
        suggestions={suggestions}
        placeholder='ema.ema50 > ema.ema200'
        color="success"
      />

      {/* Score Weights for Long */}
      <div className="p-4 rounded-md bg-surface-muted border border-border">
        <h4 className="text-body font-medium text-text-muted mb-3">امتیازدهی ورود Long</h4>
        <div className="grid grid-cols-2 sm:grid-cols-5 gap-3">
          <Input label="حداقل" type="number" value={strategy.entry.long.score.minimum}
            onChange={(e) => updateLongScore('minimum', Number(e.target.value))}
            step="0.05" min={0} max={1} />
          <Input label="ترند" type="number" value={strategy.entry.long.score.weights.trend}
            onChange={(e) => updateLongScore('trend', Number(e.target.value))}
            step="0.05" min={0} max={1} />
          <Input label="مومنتوم" type="number" value={strategy.entry.long.score.weights.momentum}
            onChange={(e) => updateLongScore('momentum', Number(e.target.value))}
            step="0.05" min={0} max={1} />
          <Input label="نوسان" type="number" value={strategy.entry.long.score.weights.volatility}
            onChange={(e) => updateLongScore('volatility', Number(e.target.value))}
            step="0.05" min={0} max={1} />
          <Input label="حجم" type="number" value={strategy.entry.long.score.weights.volume}
            onChange={(e) => updateLongScore('volume', Number(e.target.value))}
            step="0.05" min={0} max={1} />
        </div>
      </div>

      {/* Entry Short */}
      <ConditionSection
        title="شرایط ورود Short"
        subtitle="همه شروط باید برقرار باشند"
        conditions={strategy.entry.short.all}
        newValue={newShortEntry}
        onNewValueChange={setNewShortEntry}
        onAdd={addShortEntry}
        onRemove={removeShortEntry}
        suggestions={suggestions}
        placeholder='ema.ema50 < ema.ema200'
        color="danger"
      />

      {/* Exit Long */}
      <ConditionSection
        title="شرایط خروج Long"
        subtitle="هر شرط که برقرار شود کافی است"
        conditions={strategy.exit.long.any}
        newValue={newLongExit}
        onNewValueChange={setNewLongExit}
        onAdd={addLongExit}
        onRemove={removeLongExit}
        suggestions={suggestions}
        placeholder='rsi.rsi > 75'
        color="warning"
      />

      {/* Exit Short */}
      <ConditionSection
        title="شرایط خروج Short"
        subtitle="هر شرط که برقرار شود کافی است"
        conditions={strategy.exit.short.any}
        newValue={newShortExit}
        onNewValueChange={setNewShortExit}
        onAdd={addShortExit}
        onRemove={removeShortExit}
        suggestions={suggestions}
        placeholder='rsi.rsi < 25'
        color="warning"
      />
    </div>
  );
}

function ConditionSection({
  title, subtitle, conditions, newValue, onNewValueChange, onAdd, onRemove,
  suggestions, placeholder, color,
}: {
  title: string;
  subtitle: string;
  conditions: string[];
  newValue: string;
  onNewValueChange: (v: string) => void;
  onAdd: () => void;
  onRemove: (i: number) => void;
  suggestions: string[];
  placeholder: string;
  color: 'success' | 'danger' | 'warning';
}) {
  const accent = {
    success: 'border-success/40',
    danger: 'border-error/40',
    warning: 'border-warning/50',
  };

  return (
    <div className={`p-4 rounded-md border ${accent[color]} bg-surface-muted `}>
      <div className="flex items-center justify-between mb-3">
        <div>
          <h3 className="text-body font-medium text-text">{title}</h3>
          <p className="text-caption text-text-subtle mt-0.5">{subtitle}</p>
        </div>
        <Badge size="sm" variant={color === 'warning' ? 'warning' : color === 'danger' ? 'danger' : 'success'}>
          {conditions.length} شرط
        </Badge>
      </div>

      {/* Add new */}
      <div className="flex gap-2 mb-3">
        <Autocomplete
          value={newValue}
          onChange={onNewValueChange}
          onSubmit={onAdd}
          suggestions={suggestions}
          placeholder={placeholder}
          className="flex-1"
        />
        <Button size="sm" onClick={onAdd} disabled={!newValue.trim()}>
          <Plus className="w-4 h-4" />
        </Button>
      </div>

      {/* List */}
      {conditions.length === 0 ? (
        <p className="text-caption text-text-subtle">هنوز شرطی اضافه نشده</p>
      ) : (
        <div className="space-y-1.5">
          {conditions.map((condition, i) => (
            <div key={i} className="flex items-center gap-2 group">
              <span className="flex-1 text-caption font-mono p-2 rounded-sm bg-surface text-text-muted border border-border" dir="ltr">
                {condition}
              </span>
              <button
                type="button"
                aria-label="حذف شرط"
                onClick={() => onRemove(i)}
                className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-md text-error opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100 hover:bg-error/8"
              >
                <Trash2 className="h-4 w-4" aria-hidden="true" />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

import { useState } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import { Input } from '../ui/Input';
import { Select } from '../ui/Select';
import { Button } from '../ui/Button';
import { Card } from '../ui/Card';
import { Badge } from '../ui/Badge';
import { mockIndicators, mockTimeframes } from '../../data/mock';
import type { Strategy, Indicator } from '../../types';

interface Props {
  strategy: Strategy;
  onChange: (updates: Partial<Strategy>) => void;
}

export function StepIndicators({ strategy, onChange }: Props) {
  const [showAdd, setShowAdd] = useState(false);
  const [newName, setNewName] = useState('');
  const [nameError, setNameError] = useState('');
  const [newType, setNewType] = useState('');
  const [newTimeframe, setNewTimeframe] = useState('5m');
  const [newConfigs, setNewConfigs] = useState<Record<string, string | number>>({});

  const selectedIndicatorDef = mockIndicators.find((i) => i.type === newType);

  const handleTypeChange = (type: string) => {
    setNewType(type);
    const def = mockIndicators.find((i) => i.type === type);
    if (def) {
      const configs: Record<string, string | number> = {};
      def.configs.forEach((c) => {
        configs[c.key] = c.default;
      });
      setNewConfigs(configs);
    }
  };

  const handleAdd = () => {
    if (!newName || !newType) return;

    // Validate name: only English letters, digits, underscore (no spaces)
    if (!/^[a-zA-Z][a-zA-Z0-9_]*$/.test(newName)) {
      setNameError('نام باید فقط شامل حروف انگلیسی، عدد و خط زیر باشد و با حرف شروع شود');
      return;
    }

    if (strategy.indicators[newName]) {
      setNameError('این نام قبلاً استفاده شده است');
      return;
    }

    const indicator: Indicator = {
      id: null,
      name: newName,
      type: newType,
      timeframe: newTimeframe,
      configs: { ...newConfigs },
    };

    onChange({
      indicators: { ...strategy.indicators, [newName]: indicator },
    });

    // Reset
    setNewName('');
    setNewType('');
    setNewTimeframe('5m');
    setNewConfigs({});
    setNameError('');
    setShowAdd(false);
  };

  const handleRemove = (key: string) => {
    const updated = { ...strategy.indicators };
    delete updated[key];
    onChange({ indicators: updated });
  };

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-surface-900 dark:text-white">اندیکاتورها</h2>
        <Button variant="secondary" size="sm" onClick={() => setShowAdd(!showAdd)}>
          <Plus className="w-4 h-4" />
          افزودن
        </Button>
      </div>

      {/* Add Form */}
      {showAdd && (
        <Card className="border-primary-200 dark:border-primary-800 bg-primary-50/50 dark:bg-primary-950/20">
          <div className="space-y-4">
            <div className="grid sm:grid-cols-3 gap-3">
              <Input
                label="نام (کلید)"
                value={newName}
                onChange={(e) => {
                  // Only allow English letters, digits, underscore (no spaces)
                  const v = e.target.value.replace(/[^a-zA-Z0-9_]/g, '');
                  setNewName(v);
                  setNameError('');
                }}
                placeholder="fast_ema"
                error={nameError}
                hint="فقط حروف انگلیسی، عدد و _"
              />
              <Select
                label="نوع اندیکاتور"
                value={newType}
                onChange={(e) => handleTypeChange(e.target.value)}
                options={mockIndicators.map((i) => ({ value: i.type, label: i.name }))}
                placeholder="انتخاب..."
              />
              <Select
                label="تایم‌فریم"
                value={newTimeframe}
                onChange={(e) => setNewTimeframe(e.target.value)}
                options={mockTimeframes.map((t) => ({ value: t.value, label: t.label }))}
              />
            </div>

            {/* Dynamic configs */}
            {selectedIndicatorDef && selectedIndicatorDef.configs.length > 0 && (
              <div>
                <p className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-2">تنظیمات</p>
                <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-3">
                  {selectedIndicatorDef.configs.map((config) => (
                    <div key={config.key}>
                      {config.type === 'select' ? (
                        <Select
                          label={config.label}
                          value={String(newConfigs[config.key] || config.default)}
                          onChange={(e) => setNewConfigs({ ...newConfigs, [config.key]: e.target.value })}
                          options={(config.options || []).map((o) => ({ value: o, label: o }))}
                        />
                      ) : (
                        <Input
                          label={config.label}
                          type="number"
                          value={String(newConfigs[config.key] ?? config.default)}
                          onChange={(e) => setNewConfigs({ ...newConfigs, [config.key]: Number(e.target.value) })}
                          min={config.min}
                          max={config.max}
                        />
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}

            <div className="flex gap-2">
              <Button size="sm" onClick={handleAdd} disabled={!newName || !newType}>
                افزودن اندیکاتور
              </Button>
              <Button variant="ghost" size="sm" onClick={() => setShowAdd(false)}>
                انصراف
              </Button>
            </div>
          </div>
        </Card>
      )}

      {/* Indicators List */}
      {Object.keys(strategy.indicators).length === 0 ? (
        <p className="text-sm text-surface-500 dark:text-surface-400 text-center py-6">
          هنوز اندیکاتوری اضافه نشده. روی "افزودن" کلیک کنید.
        </p>
      ) : (
        <div className="grid sm:grid-cols-2 gap-3">
          {Object.entries(strategy.indicators).map(([key, ind]) => (
            <div
              key={key}
              className="p-3 rounded-lg border border-surface-200 dark:border-surface-700 bg-surface-50 dark:bg-surface-800/50"
            >
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-surface-900 dark:text-white">{key}</span>
                  <Badge size="sm">{ind.type}</Badge>
                </div>
                <button
                  onClick={() => handleRemove(key)}
                  className="p-1 rounded hover:bg-danger-400/10 text-danger-500 transition-colors"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
              <p className="text-xs text-surface-500 dark:text-surface-400 mb-1">تایم‌فریم: {ind.timeframe}</p>
              <div className="flex flex-wrap gap-1">
                {Object.entries(ind.configs).map(([ck, cv]) => (
                  <span key={ck} className="text-xs bg-surface-200 dark:bg-surface-700 px-1.5 py-0.5 rounded">
                    {ck}: {String(cv)}
                  </span>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

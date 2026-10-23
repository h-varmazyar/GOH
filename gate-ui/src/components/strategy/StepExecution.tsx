import { Input } from '../ui/Input';
import { Select } from '../ui/Select';
import { Toggle } from '../ui/Toggle';
import { mockOrderTypes, mockTimeframes, mockMetrics } from '../../data/mock';
import type { Strategy } from '../../types';

interface Props {
  strategy: Strategy;
  onChange: (updates: Partial<Strategy>) => void;
}

export function StepExecution({ strategy, onChange }: Props) {
  const { execution, scoring, multi_timeframe, telemetry } = strategy;

  const updateExecution = (updates: Partial<typeof execution>) => {
    onChange({ execution: { ...execution, ...updates } });
  };

  const updateMultiTf = (updates: Partial<typeof multi_timeframe>) => {
    onChange({ multi_timeframe: { ...multi_timeframe, ...updates } });
  };

  const toggleMetric = (metric: string) => {
    const metrics = telemetry.metrics.includes(metric)
      ? telemetry.metrics.filter((m) => m !== metric)
      : [...telemetry.metrics, metric];
    onChange({ telemetry: { ...telemetry, metrics } });
  };

  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold text-surface-900 dark:text-white">اجرا و تنظیمات پیشرفته</h2>

      {/* Execution */}
      <div>
        <h3 className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-3">اجرای سفارش</h3>
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <Select
            label="نوع سفارش"
            value={execution.order_type}
            onChange={(e) => updateExecution({ order_type: e.target.value })}
            options={mockOrderTypes.map((t) => ({ value: t, label: t }))}
          />
          <Input
            label="حداکثر اسلیپیج (%)"
            type="number"
            value={execution.slippage.max_percent}
            onChange={(e) => updateExecution({ slippage: { max_percent: Number(e.target.value) } })}
            step="0.01"
            min={0}
          />
          <Input
            label="تایم‌اوت (ms)"
            type="number"
            value={execution.timeout_ms}
            onChange={(e) => updateExecution({ timeout_ms: Number(e.target.value) })}
            min={100}
          />
        </div>
        <div className="mt-3 space-y-3">
          <Toggle
            enabled={execution.retry.enabled}
            onChange={(v) => updateExecution({ retry: { ...execution.retry, enabled: v } })}
            label="تلاش مجدد"
          />
          {execution.retry.enabled && (
            <div className="grid sm:grid-cols-2 gap-4">
              <Input
                label="تعداد تلاش"
                type="number"
                value={execution.retry.attempts}
                onChange={(e) => updateExecution({ retry: { ...execution.retry, attempts: Number(e.target.value) } })}
                min={1}
                max={10}
              />
              <Input
                label="تأخیر (ms)"
                type="number"
                value={execution.retry.delay_ms}
                onChange={(e) => updateExecution({ retry: { ...execution.retry, delay_ms: Number(e.target.value) } })}
                min={50}
              />
            </div>
          )}
        </div>
      </div>

      {/* Scoring */}
      <div>
        <h3 className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-3">امتیاز اطمینان</h3>
        <Input
          label="حداقل امتیاز اطمینان"
          type="number"
          value={scoring.confidence.minimum}
          onChange={(e) => onChange({ scoring: { ...scoring, confidence: { minimum: Number(e.target.value) } } })}
          step="0.05"
          min={0}
          max={1}
        />
      </div>

      {/* Multi Timeframe */}
      <div>
        <h3 className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-3">چند تایم‌فریمی</h3>
        <div className="space-y-3">
          <Toggle
            enabled={multi_timeframe.enabled}
            onChange={(v) => updateMultiTf({ enabled: v })}
            label="فعال"
          />
          {multi_timeframe.enabled && (
            <Select
              label="تایم‌فریم بالاتر"
              value={multi_timeframe.confirmation.higher_timeframe}
              onChange={(e) => updateMultiTf({ confirmation: { ...multi_timeframe.confirmation, higher_timeframe: e.target.value } })}
              options={mockTimeframes.map((t) => ({ value: t.value, label: t.label }))}
            />
          )}
        </div>
      </div>

      {/* Telemetry */}
      <div>
        <h3 className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-3">تلمتری و لاگ</h3>
        <div className="space-y-3">
          <Toggle
            enabled={telemetry.enabled}
            onChange={(v) => onChange({ telemetry: { ...telemetry, enabled: v } })}
            label="فعال بودن تلمتری"
          />
          {telemetry.enabled && (
            <>
              <div className="flex flex-wrap gap-3">
                <Toggle
                  enabled={telemetry.log_signals}
                  onChange={(v) => onChange({ telemetry: { ...telemetry, log_signals: v } })}
                  label="لاگ سیگنال‌ها"
                />
                <Toggle
                  enabled={telemetry.log_rejections}
                  onChange={(v) => onChange({ telemetry: { ...telemetry, log_rejections: v } })}
                  label="لاگ ردها"
                />
                <Toggle
                  enabled={telemetry.log_scores}
                  onChange={(v) => onChange({ telemetry: { ...telemetry, log_scores: v } })}
                  label="لاگ امتیازها"
                />
              </div>
              <div>
                <p className="text-xs text-surface-500 dark:text-surface-400 mb-2">متریک‌ها</p>
                <div className="flex flex-wrap gap-2">
                  {mockMetrics.map((metric) => (
                    <button
                      key={metric}
                      type="button"
                      onClick={() => toggleMetric(metric)}
                      className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-colors border
                        ${telemetry.metrics.includes(metric)
                          ? 'bg-primary-600 text-white border-primary-600'
                          : 'bg-white dark:bg-surface-800 text-surface-600 dark:text-surface-400 border-surface-300 dark:border-surface-600 hover:border-primary-400'
                        }`}
                    >
                      {metric}
                    </button>
                  ))}
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

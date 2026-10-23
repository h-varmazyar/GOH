import { Input } from '../ui/Input';
import { Select } from '../ui/Select';
import { Toggle } from '../ui/Toggle';
import { mockStoplossTypes, mockTakeprofitTypes } from '../../data/mock';
import type { Strategy } from '../../types';

interface Props {
  strategy: Strategy;
  onChange: (updates: Partial<Strategy>) => void;
}

export function StepRisk({ strategy, onChange }: Props) {
  const { risk } = strategy;

  const updatePosition = (key: string, value: number) => {
    onChange({ risk: { ...risk, position: { ...risk.position, [key]: value } } });
  };

  const updateStoploss = (updates: Partial<typeof risk.stoploss>) => {
    onChange({ risk: { ...risk, stoploss: { ...risk.stoploss, ...updates } } });
  };

  const updateTrailing = (updates: Partial<typeof risk.trailing_stop>) => {
    onChange({ risk: { ...risk, trailing_stop: { ...risk.trailing_stop, ...updates } } });
  };

  const updateTakeprofit = (updates: Partial<typeof risk.takeprofit>) => {
    onChange({ risk: { ...risk, takeprofit: { ...risk.takeprofit, ...updates } } });
  };

  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold text-surface-900 dark:text-white">مدیریت ریسک</h2>

      {/* Position */}
      <div>
        <h3 className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-3">پوزیشن</h3>
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <Input
            label="حداکثر پوزیشن باز"
            type="number"
            value={risk.position.max_open_positions}
            onChange={(e) => updatePosition('max_open_positions', Number(e.target.value))}
            min={1}
          />
          <Input
            label="حداکثر پوزیشن هر نماد"
            type="number"
            value={risk.position.max_position_per_symbol}
            onChange={(e) => updatePosition('max_position_per_symbol', Number(e.target.value))}
            min={1}
          />
          <Input
            label="ریسک هر معامله (%)"
            type="number"
            value={risk.position.risk_per_trade_percent}
            onChange={(e) => updatePosition('risk_per_trade_percent', Number(e.target.value))}
            step="0.1"
            min={0.1}
          />
          <Input
            label="حداکثر ضرر روزانه (%)"
            type="number"
            value={risk.position.max_daily_loss_percent}
            onChange={(e) => updatePosition('max_daily_loss_percent', Number(e.target.value))}
            step="0.5"
            min={0.5}
          />
          <Input
            label="حداکثر افت سرمایه (%)"
            type="number"
            value={risk.position.max_drawdown_percent}
            onChange={(e) => updatePosition('max_drawdown_percent', Number(e.target.value))}
            step="0.5"
            min={1}
          />
        </div>
      </div>

      {/* Stoploss */}
      <div>
        <h3 className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-3">استاپ‌لاس</h3>
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <Select
            label="نوع"
            value={risk.stoploss.type}
            onChange={(e) => updateStoploss({ type: e.target.value })}
            options={mockStoplossTypes.map((t) => ({ value: t, label: t }))}
          />
          {risk.stoploss.type === 'atr' && (
            <>
              <Input
                label="ضریب ATR"
                type="number"
                value={risk.stoploss.atr.multiplier}
                onChange={(e) => updateStoploss({ atr: { ...risk.stoploss.atr, multiplier: Number(e.target.value) } })}
                step="0.1"
                min={0.1}
              />
              <Input
                label="حداقل (%)"
                type="number"
                value={risk.stoploss.atr.minimum_percent}
                onChange={(e) => updateStoploss({ atr: { ...risk.stoploss.atr, minimum_percent: Number(e.target.value) } })}
                step="0.1"
              />
              <Input
                label="حداکثر (%)"
                type="number"
                value={risk.stoploss.atr.maximum_percent}
                onChange={(e) => updateStoploss({ atr: { ...risk.stoploss.atr, maximum_percent: Number(e.target.value) } })}
                step="0.1"
              />
            </>
          )}
        </div>
      </div>

      {/* Trailing Stop */}
      <div>
        <h3 className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-3">تریلینگ استاپ</h3>
        <div className="space-y-3">
          <Toggle
            enabled={risk.trailing_stop.enabled}
            onChange={(v) => updateTrailing({ enabled: v })}
            label="فعال"
          />
          {risk.trailing_stop.enabled && (
            <div className="grid sm:grid-cols-2 gap-4">
              <Input
                label="فعال‌سازی در R:R"
                type="number"
                value={risk.trailing_stop.activation_rr}
                onChange={(e) => updateTrailing({ activation_rr: Number(e.target.value) })}
                step="0.1"
                min={0.1}
              />
              <Input
                label="ضریب ATR"
                type="number"
                value={risk.trailing_stop.atr_multiplier}
                onChange={(e) => updateTrailing({ atr_multiplier: Number(e.target.value) })}
                step="0.1"
                min={0.1}
              />
            </div>
          )}
        </div>
      </div>

      {/* Take Profit */}
      <div>
        <h3 className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-3">تیک‌پرافیت</h3>
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <Select
            label="نوع"
            value={risk.takeprofit.type}
            onChange={(e) => updateTakeprofit({ type: e.target.value })}
            options={mockTakeprofitTypes.map((t) => ({ value: t, label: t }))}
          />
          {risk.takeprofit.type === 'risk_reward' && (
            <>
              <Input
                label="حداقل R:R"
                type="number"
                value={risk.takeprofit.risk_reward.minimum}
                onChange={(e) => updateTakeprofit({ risk_reward: { ...risk.takeprofit.risk_reward, minimum: Number(e.target.value) } })}
                step="0.1"
                min={0.5}
              />
              <Input
                label="هدف R:R"
                type="number"
                value={risk.takeprofit.risk_reward.target}
                onChange={(e) => updateTakeprofit({ risk_reward: { ...risk.takeprofit.risk_reward, target: Number(e.target.value) } })}
                step="0.1"
                min={0.5}
              />
              <Input
                label="بستن جزئی در R:R"
                type="number"
                value={risk.takeprofit.risk_reward.partial_close_rr}
                onChange={(e) => updateTakeprofit({ risk_reward: { ...risk.takeprofit.risk_reward, partial_close_rr: Number(e.target.value) } })}
                step="0.1"
              />
              <Input
                label="درصد بستن جزئی"
                type="number"
                value={risk.takeprofit.risk_reward.partial_close_percent}
                onChange={(e) => updateTakeprofit({ risk_reward: { ...risk.takeprofit.risk_reward, partial_close_percent: Number(e.target.value) } })}
                min={1}
                max={100}
              />
            </>
          )}
        </div>
      </div>
    </div>
  );
}

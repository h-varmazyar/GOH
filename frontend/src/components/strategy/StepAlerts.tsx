import { Bell, Clock3 } from 'lucide-react';
import { Input } from '../ui/Input';
import { Toggle } from '../ui/Toggle';
import type { AlertChannelType, AlertEvent, Strategy } from '../../types';

interface Props { strategy: Strategy; onChange: (updates: Partial<Strategy>) => void; }

const channelLabels: Record<AlertChannelType, string> = { telegram: 'تلگرام', sms: 'پیامک', push: 'Push', email: 'ایمیل' };
const eventLabels: Record<AlertEvent, string> = { signal: 'سیگنال', entry: 'ورود', exit: 'خروج', stop_loss: 'حد ضرر', take_profit: 'حد سود', error: 'خطا' };

export function StepAlerts({ strategy, onChange }: Props) {
  const updateAlerts = (updates: Partial<Strategy['alerts']>) => onChange({ alerts: { ...strategy.alerts, ...updates } });
  const toggleEvent = (event: AlertEvent) => updateAlerts({ events: strategy.alerts.events.includes(event) ? strategy.alerts.events.filter((item) => item !== event) : [...strategy.alerts.events, event] });
  const updateChannel = (channel: AlertChannelType, updates: Partial<Strategy['alerts']['channels'][AlertChannelType]>) => updateAlerts({ channels: { ...strategy.alerts.channels, [channel]: { ...strategy.alerts.channels[channel], ...updates } } });

  return <div className="space-y-6">
    <div className="flex items-start gap-3"><Bell className="w-5 h-5 text-link mt-1" /><div><h2 className="text-h3 font-semibold text-text">هشدارها</h2><p className="text-body text-text-subtle">کانال‌ها و رویدادهای اطلاع‌رسانی را تنظیم کنید.</p></div></div>
    <Toggle enabled={strategy.alerts.enabled} onChange={(enabled) => updateAlerts({ enabled })} label="فعال بودن هشدارها" />
    <div className={strategy.alerts.enabled ? 'space-y-6' : 'space-y-6 opacity-50 pointer-events-none'}>
      <div><h3 className="text-body font-medium text-text mb-3">کانال‌ها</h3><div className="grid sm:grid-cols-2 gap-3">
        {(Object.keys(channelLabels) as AlertChannelType[]).map((channel) => <div key={channel} className="p-3 rounded-md border border-border space-y-3">
          <Toggle enabled={strategy.alerts.channels[channel].enabled} onChange={(enabled) => updateChannel(channel, { enabled })} label={channelLabels[channel]} />
          <Input value={strategy.alerts.channels[channel].destination} onChange={(event) => updateChannel(channel, { destination: event.target.value })} placeholder={channel === 'telegram' ? '@channel یا Chat ID' : channel === 'sms' ? '09xxxxxxxxx' : channel === 'email' ? 'name@example.com' : 'شناسه دستگاه'} disabled={!strategy.alerts.channels[channel].enabled} dir="ltr" />
        </div>)}
      </div></div>
      <div><h3 className="text-body font-medium text-text mb-3">رویدادها</h3><div className="flex flex-wrap gap-2">{(Object.keys(eventLabels) as AlertEvent[]).map((event) => <button type="button" key={event} onClick={() => toggleEvent(event)} className={`px-3 py-2 rounded-md border text-body ${strategy.alerts.events.includes(event) ? 'bg-action border-selected text-on-action' : 'border-border-strong text-text-muted'}`}>{eventLabels[event]}</button>)}</div></div>
      <Input label="حداقل اطمینان هشدار (%)" type="number" min={0} max={100} value={strategy.alerts.minimumConfidence} onChange={(event) => updateAlerts({ minimumConfidence: Number(event.target.value) })} />
      <div className="p-4 rounded-md bg-surface-muted border border-border space-y-3"><div className="flex items-center gap-2"><Clock3 className="w-4 h-4 text-text-subtle" /><Toggle enabled={strategy.alerts.quietHours.enabled} onChange={(enabled) => updateAlerts({ quietHours: { ...strategy.alerts.quietHours, enabled } })} label="ساعات سکوت" /></div>{strategy.alerts.quietHours.enabled && <div className="grid grid-cols-2 gap-3"><Input label="از ساعت" type="time" value={strategy.alerts.quietHours.start} onChange={(event) => updateAlerts({ quietHours: { ...strategy.alerts.quietHours, start: event.target.value } })} /><Input label="تا ساعت" type="time" value={strategy.alerts.quietHours.end} onChange={(event) => updateAlerts({ quietHours: { ...strategy.alerts.quietHours, end: event.target.value } })} /></div>}</div>
    </div>
  </div>;
}

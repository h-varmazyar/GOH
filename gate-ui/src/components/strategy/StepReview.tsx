import { Badge } from '../ui/Badge';
import type { Strategy } from '../../types';

interface Props {
  strategy: Strategy;
}

export function StepReview({ strategy }: Props) {
  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold text-surface-900 dark:text-white">بررسی نهایی</h2>
      <p className="text-sm text-surface-500 dark:text-surface-400">
        لطفاً اطلاعات استراتژی را بررسی کنید. در صورت نیاز به تغییر، به مراحل قبل بازگردید.
      </p>

      {/* Basic Info */}
      <Section title="اطلاعات پایه">
        <InfoRow label="نام" value={strategy.name || '—'} />
        <InfoRow label="نوع دارایی" value={strategy.assetClass === 'stock' ? 'سهام تهران' : 'رمزارز'} />
        <InfoRow label="ارائه‌دهنده" value={strategy.providerName || strategy.provider || '—'} />
        <InfoRow label="توضیحات" value={strategy.description || '—'} />
        <InfoRow label="وضعیت" value={strategy.enabled ? 'فعال' : 'غیرفعال'} />
        <div className="flex items-center gap-2">
          <span className="text-xs text-surface-500 dark:text-surface-400 w-24">برچسب‌ها:</span>
          <div className="flex flex-wrap gap-1">
            {strategy.tags.length > 0
              ? strategy.tags.map((t) => <Badge key={t} size="sm">{t}</Badge>)
              : <span className="text-xs text-surface-400">—</span>}
          </div>
        </div>
      </Section>

      {/* Indicators */}
      <Section title={`اندیکاتورها (${Object.keys(strategy.indicators).length})`}>
        {Object.keys(strategy.indicators).length === 0 ? (
          <p className="text-xs text-surface-400">بدون اندیکاتور</p>
        ) : (
          <div className="space-y-1">
            {Object.entries(strategy.indicators).map(([key, ind]) => (
              <div key={key} className="flex items-center gap-2 text-xs">
                <Badge size="sm" variant="primary">{ind.type}</Badge>
                <span className="text-surface-700 dark:text-surface-300 font-medium">{key}</span>
                <span className="text-surface-400">({ind.timeframe})</span>
              </div>
            ))}
          </div>
        )}
      </Section>

      {/* Market */}
      <Section title="مارکت">
        <InfoRow label="نمادها" value={strategy.market.symbols.join(', ') || '—'} />
        <InfoRow label="حذف شده" value={strategy.market.excluded_symbols.join(', ') || '—'} />
        <InfoRow label="تایم‌فریم" value={strategy.market.timeframes.join(', ') || '—'} />
        <InfoRow label="جلسات" value={strategy.market.sessions.join(', ') || '—'} />
      </Section>

      {/* Alerts */}
      <Section title="هشدارها">
        <InfoRow label="وضعیت" value={strategy.alerts.enabled ? 'فعال' : 'غیرفعال'} />
        <InfoRow label="کانال فعال" value={Object.entries(strategy.alerts.channels).filter(([, channel]) => channel.enabled).map(([name]) => name).join('، ') || '—'} />
        <InfoRow label="رویدادها" value={strategy.alerts.events.join('، ') || '—'} />
        <InfoRow label="حداقل اطمینان" value={`${strategy.alerts.minimumConfidence}%`} />
      </Section>

      {/* Risk */}
      <Section title="ریسک">
        <InfoRow label="حداکثر پوزیشن" value={String(strategy.risk.position.max_open_positions)} />
        <InfoRow label="ریسک هر معامله" value={`${strategy.risk.position.risk_per_trade_percent}%`} />
        <InfoRow label="استاپ‌لاس" value={strategy.risk.stoploss.type} />
        <InfoRow label="تریلینگ" value={strategy.risk.trailing_stop.enabled ? 'فعال' : 'غیرفعال'} />
        <InfoRow label="تیک‌پرافیت" value={`${strategy.risk.takeprofit.type} (R:R ${strategy.risk.takeprofit.risk_reward.target})`} />
      </Section>

      {/* Entry/Exit Summary */}
      <Section title="ورود/خروج">
        <InfoRow label="شرایط ورود Long" value={`${strategy.entry.long.all.length} شرط`} />
        <InfoRow label="شرایط ورود Short" value={`${strategy.entry.short.all.length} شرط`} />
        <InfoRow label="شرایط خروج Long" value={`${strategy.exit.long.any.length} شرط`} />
        <InfoRow label="شرایط خروج Short" value={`${strategy.exit.short.any.length} شرط`} />
      </Section>

      {/* Execution */}
      <Section title="اجرا">
        <InfoRow label="نوع سفارش" value={strategy.execution.order_type} />
        <InfoRow label="اسلیپیج" value={`${strategy.execution.slippage.max_percent}%`} />
        <InfoRow label="تلاش مجدد" value={strategy.execution.retry.enabled ? `${strategy.execution.retry.attempts} بار` : 'غیرفعال'} />
      </Section>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="p-4 rounded-lg bg-surface-50 dark:bg-surface-800/50 border border-surface-200 dark:border-surface-700">
      <h3 className="text-sm font-semibold text-surface-800 dark:text-surface-200 mb-3">{title}</h3>
      <div className="space-y-2">{children}</div>
    </div>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-start gap-2 text-xs">
      <span className="text-surface-500 dark:text-surface-400 w-28 flex-shrink-0">{label}:</span>
      <span className="text-surface-800 dark:text-surface-200 font-medium">{value}</span>
    </div>
  );
}

import { useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { ArrowRight, Settings, BarChart3, Shield, Zap, Activity, Radio, Bell } from 'lucide-react';
import { Card } from '../components/ui/Card';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Toggle } from '../components/ui/Toggle';
import { useStrategyStore } from '../store/strategyStore';

export function StrategyDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { strategies, fetchStrategies, toggleStrategy, isLoading, hasSeeded } = useStrategyStore();

  useEffect(() => {
    if (!hasSeeded) void fetchStrategies();
  }, [hasSeeded, fetchStrategies]);

  const strategy = strategies.find((s) => s.id === id);

  if (!hasSeeded || isLoading) {
    return (
      <div className="flex justify-center py-12">
        <div className="w-8 h-8 border-2 border-primary-500 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  if (!strategy) {
    return (
      <div className="text-center py-12">
        <p className="text-surface-500 dark:text-surface-400">استراتژی یافت نشد</p>
        <Link to="/strategies" className="text-primary-600 dark:text-primary-400 hover:underline text-sm mt-2 inline-block">
          بازگشت به لیست
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Link to="/strategies">
          <Button variant="ghost" size="sm">
            <ArrowRight className="w-4 h-4" />
          </Button>
        </Link>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-surface-900 dark:text-white">{strategy.name}</h1>
            <Toggle enabled={strategy.enabled} onChange={() => toggleStrategy(strategy.id!)} />
          </div>
          <p className="text-sm text-surface-500 dark:text-surface-400 mt-1">{strategy.description}</p>
        </div>
      </div>

      {/* Tags, asset & Provider */}
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant="primary" size="md">{strategy.assetClass === 'stock' ? 'سهام تهران' : 'رمزارز'}</Badge>
        <Badge size="md">{strategy.providerName || strategy.provider}</Badge>
        {strategy.tags.map((tag) => (
          <Badge key={tag} size="md">{tag}</Badge>
        ))}
        <Badge variant={strategy.enabled ? 'success' : 'default'} size="md">
          {strategy.enabled ? 'فعال' : 'غیرفعال'}
        </Badge>
      </div>

      {/* Indicators */}
      <Card>
        <h2 className="text-base font-semibold text-surface-900 dark:text-white mb-4 flex items-center gap-2">
          <Activity className="w-5 h-5 text-primary-500" />
          اندیکاتورها
        </h2>
        {Object.keys(strategy.indicators).length === 0 ? <EmptyState text="اندیکاتوری تعریف نشده است." /> : <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {Object.entries(strategy.indicators).map(([key, ind]) => (
            <div key={key} className="p-3 rounded-lg bg-surface-50 dark:bg-surface-800/50 border border-surface-100 dark:border-surface-700">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-medium text-surface-900 dark:text-white">{key}</span>
                <Badge size="sm">{ind.type}</Badge>
              </div>
              <p className="text-xs text-surface-500 dark:text-surface-400">تایم‌فریم: {ind.timeframe}</p>
              <div className="mt-2 space-y-1">
                {Object.entries(ind.configs).map(([ck, cv]) => (
                  <div key={ck} className="flex justify-between text-xs">
                    <span className="text-surface-500 dark:text-surface-400">{ck}:</span>
                    <span className="text-surface-700 dark:text-surface-300 font-medium">{String(cv)}</span>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>}
      </Card>

      {/* Market */}
      <Card>
        <h2 className="text-base font-semibold text-surface-900 dark:text-white mb-4 flex items-center gap-2">
          <BarChart3 className="w-5 h-5 text-primary-500" />
          مارکت
        </h2>
        <div className="grid sm:grid-cols-2 gap-4">
          <div>
            <p className="text-sm text-surface-500 dark:text-surface-400 mb-2">نمادها</p>
            <div className="flex flex-wrap gap-1">
              {strategy.market.symbols.length ? strategy.market.symbols.map((s) => (
                <Badge key={s}>{s === '*' ? 'همه نمادها' : s}</Badge>
              )) : <EmptyState text="نمادی انتخاب نشده است." />}
            </div>
          </div>
          <div>
            <p className="text-sm text-surface-500 dark:text-surface-400 mb-2">نمادهای حذف شده</p>
            <div className="flex flex-wrap gap-1">
              {strategy.market.excluded_symbols.length ? strategy.market.excluded_symbols.map((s) => (
                <Badge key={s} variant="danger">{s}</Badge>
              )) : <EmptyState text="موردی وجود ندارد." />}
            </div>
          </div>
          <div>
            <p className="text-sm text-surface-500 dark:text-surface-400 mb-2">تایم‌فریم</p>
            <div className="flex flex-wrap gap-1">
              {strategy.market.timeframes.length ? strategy.market.timeframes.map((t) => (
                <Badge key={t} variant="primary">{t}</Badge>
              )) : <EmptyState text="تایم‌فریمی انتخاب نشده است." />}
            </div>
          </div>
          <div>
            <p className="text-sm text-surface-500 dark:text-surface-400 mb-2">جلسات</p>
            <div className="flex flex-wrap gap-1">
              {strategy.market.sessions.length ? strategy.market.sessions.map((s) => (
                <Badge key={s}>{s}</Badge>
              )) : <EmptyState text="جلسه‌ای انتخاب نشده است." />}
            </div>
          </div>
        </div>
      </Card>

      {/* Alerts */}
      <Card>
        <h2 className="text-base font-semibold text-surface-900 dark:text-white mb-4 flex items-center gap-2">
          <Bell className="w-5 h-5 text-primary-500" />هشدارها
        </h2>
        <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <InfoItem label="وضعیت" value={strategy.alerts.enabled ? 'فعال' : 'غیرفعال'} />
          <InfoItem label="حداقل اطمینان" value={`${strategy.alerts.minimumConfidence}%`} />
          <InfoItem label="رویدادها" value={strategy.alerts.events.join('، ') || 'بدون رویداد'} />
          <InfoItem label="ساعات سکوت" value={strategy.alerts.quietHours.enabled ? `${strategy.alerts.quietHours.start} تا ${strategy.alerts.quietHours.end}` : 'غیرفعال'} />
        </div>
        <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-3 mt-4">
          {Object.entries(strategy.alerts.channels).map(([name, channel]) => <div key={name} className="p-3 rounded-lg bg-surface-50 dark:bg-surface-900/30"><div className="flex justify-between"><span className="text-sm font-medium">{name}</span><Badge variant={channel.enabled ? 'success' : 'default'}>{channel.enabled ? 'فعال' : 'خاموش'}</Badge></div><p className="text-xs text-surface-500 mt-2 break-all" dir="ltr">{channel.destination || '—'}</p></div>)}
        </div>
      </Card>

      {/* Risk Management */}
      <Card>
        <h2 className="text-base font-semibold text-surface-900 dark:text-white mb-4 flex items-center gap-2">
          <Shield className="w-5 h-5 text-danger-500" />
          مدیریت ریسک
        </h2>
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <InfoItem label="حداکثر پوزیشن باز" value={strategy.risk.position.max_open_positions} />
          <InfoItem label="ریسک هر معامله" value={`${strategy.risk.position.risk_per_trade_percent}%`} />
          <InfoItem label="حداکثر ضرر روزانه" value={`${strategy.risk.position.max_daily_loss_percent}%`} />
          <InfoItem label="حداکثر افت سرمایه" value={`${strategy.risk.position.max_drawdown_percent}%`} />
          <InfoItem label="نوع استاپ‌لاس" value={strategy.risk.stoploss.type} />
          <InfoItem label="ضریب ATR" value={strategy.risk.stoploss.atr.multiplier} />
          <InfoItem label="تریلینگ استاپ" value={strategy.risk.trailing_stop.enabled ? 'فعال' : 'غیرفعال'} />
          <InfoItem label="نوع تیک‌پرافیت" value={strategy.risk.takeprofit.type} />
          <InfoItem label="هدف R:R" value={strategy.risk.takeprofit.risk_reward.target} />
        </div>
      </Card>

      {/* Execution */}
      <Card>
        <h2 className="text-base font-semibold text-surface-900 dark:text-white mb-4 flex items-center gap-2">
          <Zap className="w-5 h-5 text-warning-500" />
          اجرا
        </h2>
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <InfoItem label="نوع سفارش" value={strategy.execution.order_type} />
          <InfoItem label="حداکثر اسلیپیج" value={`${strategy.execution.slippage.max_percent}%`} />
          <InfoItem label="تلاش مجدد" value={strategy.execution.retry.enabled ? `${strategy.execution.retry.attempts} بار` : 'غیرفعال'} />
          <InfoItem label="تایم‌اوت" value={`${strategy.execution.timeout_ms}ms`} />
        </div>
      </Card>

      {/* Entry/Exit Conditions */}
      <div className="grid lg:grid-cols-2 gap-4">
        <Card>
          <h2 className="text-base font-semibold text-surface-900 dark:text-white mb-4 flex items-center gap-2">
            <Radio className="w-5 h-5 text-success-500" />
            شرایط ورود (Long)
          </h2>
          <div className="space-y-2">
            {strategy.entry.long.all.length ? strategy.entry.long.all.map((condition, i) => (
              <div key={i} className="text-xs font-mono p-2 rounded bg-surface-50 dark:bg-surface-800/50 text-surface-700 dark:text-surface-300 border border-surface-100 dark:border-surface-700">
                {condition}
              </div>
            )) : <EmptyState text="شرط ورودی تعریف نشده است." />}
          </div>
          <div className="mt-3 pt-3 border-t border-surface-100 dark:border-surface-700">
            <p className="text-xs text-surface-500 dark:text-surface-400">
              حداقل امتیاز: {strategy.entry.long.score.minimum} | وزن‌ها: ترند {strategy.entry.long.score.weights.trend}, مومنتوم {strategy.entry.long.score.weights.momentum}
            </p>
          </div>
        </Card>

        <Card>
          <h2 className="text-base font-semibold text-surface-900 dark:text-white mb-4 flex items-center gap-2">
            <Radio className="w-5 h-5 text-danger-500" />
            شرایط خروج (Long)
          </h2>
          <div className="space-y-2">
            {strategy.exit.long.any.length ? strategy.exit.long.any.map((condition, i) => (
              <div key={i} className="text-xs font-mono p-2 rounded bg-surface-50 dark:bg-surface-800/50 text-surface-700 dark:text-surface-300 border border-surface-100 dark:border-surface-700">
                {condition}
              </div>
            )) : <EmptyState text="شرط خروجی تعریف نشده است." />}
          </div>
        </Card>
      </div>

      {/* Scoring */}
      <Card>
        <h2 className="text-base font-semibold text-surface-900 dark:text-white mb-4 flex items-center gap-2">
          <Settings className="w-5 h-5 text-surface-500" />
          امتیازدهی
        </h2>
        <div className="grid sm:grid-cols-2 gap-6">
          <div>
            <p className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-2">جریمه‌ها</p>
            <div className="space-y-1">
              {Object.entries(strategy.scoring.penalties).map(([key, val]) => (
                <div key={key} className="flex justify-between text-sm">
                  <span className="text-surface-500 dark:text-surface-400">{key}</span>
                  <span className="text-danger-500 font-medium">{val}</span>
                </div>
              ))}
            </div>
          </div>
          <div>
            <p className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-2">پاداش‌ها</p>
            <div className="space-y-1">
              {Object.entries(strategy.scoring.bonuses).map(([key, val]) => (
                <div key={key} className="flex justify-between text-sm">
                  <span className="text-surface-500 dark:text-surface-400">{key}</span>
                  <span className="text-success-500 font-medium">+{val}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </Card>
    </div>
  );
}

function EmptyState({ text }: { text: string }) {
  return <p className="text-xs text-surface-400 py-1">{text}</p>;
}

function InfoItem({ label, value }: { label: string; value: string | number }) {
  return (
    <div>
      <p className="text-xs text-surface-500 dark:text-surface-400">{label}</p>
      <p className="text-sm font-medium text-surface-900 dark:text-white mt-0.5">{String(value)}</p>
    </div>
  );
}

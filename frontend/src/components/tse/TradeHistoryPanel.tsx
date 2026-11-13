import { useEffect, useState } from 'react';
import { AlertTriangle, CheckCircle2, Download, History, RefreshCw } from 'lucide-react';
import { Badge } from '../ui/Badge';
import { Button } from '../ui/Button';
import { Card } from '../ui/Card';
import { confirmDialog } from '../ui/dialogs';
import { Input } from '../ui/Input';
import { Pagination } from '../ui/Pagination';
import { Select } from '../ui/Select';
import { TradeHistoryJobs } from './TradeHistoryJobs';
import { errorMessage, formatDate, formatDateTime } from './format';
import type { Pagination as PaginationInfo } from '../../services/airdropService';
import {
  tradeDayStatusOptions,
  tradeHistoryService,
  type TradeDay,
  type TradeDayStatus,
  type TradeHistorySummary,
} from '../../services/tradeHistoryService';

const PAGE_SIZE = 30;
const POLL_INTERVAL = 10_000;

const fa = (value: number | null | undefined) => (value === null || value === undefined ? '—' : value.toLocaleString('fa-IR'));

const dayStatusStyle: Record<TradeDay['status'], { variant: 'success' | 'warning' | 'danger' | 'default'; label: string }> = {
  FETCHED: { variant: 'success', label: 'دریافت‌شده' },
  EMPTY: { variant: 'warning', label: 'پاسخ خالی' },
  FAILED: { variant: 'danger', label: 'خطا' },
  MISSING: { variant: 'default', label: 'دریافت‌نشده' },
};

const weekdayName = (date: string) => new Date(date).toLocaleDateString('fa-IR', { weekday: 'long', timeZone: 'UTC' });

function Stat({ label, value, tone, active, onClick }: { label: string; value: number; tone?: string; active?: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded-lg border p-3 text-right transition-colors ${
        active ? 'border-primary-400 bg-primary-50 dark:bg-primary-950/40' : 'border-surface-200 hover:bg-surface-50 dark:border-surface-700 dark:hover:bg-surface-800'
      }`}
    >
      <div className="text-xs text-surface-500">{label}</div>
      <div className={`mt-1 text-xl font-semibold ${tone || 'text-surface-900 dark:text-white'}`}>{fa(value)}</div>
    </button>
  );
}

/** A trade history figure next to its closing price figure; differing ones are marked. */
function Compare({ history, closing, fetched }: { history: number; closing: number; fetched: boolean }) {
  if (!fetched) return <span className="text-surface-400">— / {fa(closing)}</span>;
  const differs = history !== Math.round(closing);
  return (
    <span className={differs ? 'font-medium text-danger-600' : 'text-success-700 dark:text-success-400'}>
      {fa(history)} / {fa(closing)}
    </span>
  );
}

interface Filters {
  status: TradeDayStatus | '';
  from: string;
  to: string;
}

/**
 * The trade history of one TSE market. Its trading days are its daily closing
 * prices; each day shows the totals of its fetched trades next to the closing
 * price figures, so the two workers can be checked against each other. Days can
 * be fetched one by one right away, or queued for the worker.
 */
export function TradeHistoryPanel({ tseMarketId }: { tseMarketId: number }) {
  const [summary, setSummary] = useState<TradeHistorySummary | null>(null);
  const [days, setDays] = useState<TradeDay[]>([]);
  const [pagination, setPagination] = useState<PaginationInfo | null>(null);
  const [filters, setFilters] = useState<Filters>({ status: '', from: '', to: '' });
  const [page, setPage] = useState(1);
  const [selected, setSelected] = useState<string[]>([]);
  const [includeEmpty, setIncludeEmpty] = useState(false);
  const [fetching, setFetching] = useState<string[]>([]);
  const [queueing, setQueueing] = useState(false);
  const [reload, setReload] = useState(0);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  const hasActiveJobs = (summary?.active_jobs.length ?? 0) > 0;

  useEffect(() => {
    let cancelled = false;
    Promise.all([tradeHistoryService.summary(tseMarketId), tradeHistoryService.days(tseMarketId, filters, page, PAGE_SIZE)])
      .then(([summaryRes, daysRes]) => {
        if (cancelled) return;
        setSummary(summaryRes);
        setDays(daysRes.data || []);
        setPagination(daysRes.pagination);
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(errorMessage(e, 'خطا در دریافت سابقه معاملات'));
      });
    return () => {
      cancelled = true;
    };
  }, [tseMarketId, filters, page, reload]);

  // While a job of the market runs, its progress and the days are refreshed.
  useEffect(() => {
    if (!hasActiveJobs) return;
    const timer = setInterval(() => setReload((value) => value + 1), POLL_INTERVAL);
    return () => clearInterval(timer);
  }, [hasActiveJobs]);

  const changeFilter = (next: Partial<Filters>) => {
    setFilters((current) => ({ ...current, ...next }));
    setPage(1);
  };

  const fetchDay = async (date: string) => {
    setFetching((current) => [...current, date]);
    setError('');
    try {
      const result = await tradeHistoryService.fetchDay(tseMarketId, date);
      const label = formatDate(date);
      if (result.status === 'FETCHED')
        setNotice(`${fa(result.trade_count)} معامله روز ${label} دریافت شد (${fa(result.canceled_count)} لغوشده، حجم خالص ${fa(result.net_volume)}).`);
      else if (result.status === 'EMPTY') setNotice(`پاسخ سابقه معاملات روز ${label} خالی بود.`);
      else setError(`دریافت روز ${label} ناموفق بود: ${result.error}`);
      setReload((value) => value + 1);
    } catch (e) {
      setError(errorMessage(e, 'خطا در دریافت روز'));
    } finally {
      setFetching((current) => current.filter((item) => item !== date));
    }
  };

  const queue = async (scope: 'gaps' | 'selected' | 'mismatched' | 'empty') => {
    if (!summary) return;
    const statuses: TradeDayStatus[] =
      scope === 'mismatched' ? ['MISMATCHED'] : scope === 'empty' ? ['EMPTY'] : includeEmpty ? ['MISSING', 'FAILED', 'EMPTY'] : ['MISSING', 'FAILED'];
    const countOf: Record<string, number> = {
      MISSING: summary.missing_days,
      FAILED: summary.failed_days,
      EMPTY: summary.empty_days,
      MISMATCHED: summary.mismatched_days,
    };
    const count = scope === 'selected' ? selected.length : statuses.reduce((total, status) => total + countOf[status], 0);
    const confirmed = await confirmDialog({
      title: 'دریافت سابقه معاملات',
      message: `دریافت ${fa(count)} روز از سابقه معاملات ${summary.symbol} در صف قرار بگیرد؟`,
      warning: count > 500 ? 'هر روز یک درخواست جداگانه به TSETMC است و این کار ممکن است چند ساعت طول بکشد.' : undefined,
      confirmLabel: 'افزودن به صف',
    });
    if (!confirmed) return;
    setQueueing(true);
    setError('');
    try {
      const job = await tradeHistoryService.createJob(
        scope === 'selected' ? { tse_market_ids: [tseMarketId], dates: selected } : { tse_market_ids: [tseMarketId], day_statuses: statuses },
      );
      setNotice(`کار «${job.title}» در صف قرار گرفت؛ ورکر tse-trade-history آن را اجرا می‌کند.`);
      setSelected([]);
      setReload((value) => value + 1);
    } catch (e) {
      setError(errorMessage(e, 'خطا در افزودن کار به صف'));
    } finally {
      setQueueing(false);
    }
  };

  const incomplete = summary ? summary.missing_days + summary.failed_days : 0;
  const neverFetched = summary !== null && summary.total_days > 0 && summary.missing_days === summary.total_days;
  const covered = summary && summary.total_days > 0 ? Math.round((100 * (summary.fetched_days + summary.empty_days)) / summary.total_days) : 0;
  const pageDates = days.map((day) => day.date);
  const allPageSelected = pageDates.length > 0 && pageDates.every((date) => selected.includes(date));

  return (
    <Card padding="sm" className="space-y-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="flex items-center gap-2 font-semibold text-surface-900 dark:text-white">
            <History className="h-5 w-5" />
            سابقه معاملات
          </h2>
          <p className="text-xs text-surface-500">
            {summary && summary.total_days > 0
              ? `${fa(summary.total_days)} روز معاملاتی از ${formatDate(summary.first_date)} تا ${formatDate(summary.last_date)} · پوشش ${fa(covered)}٪`
              : 'روزهای معاملاتی همان روزهای قیمت پایانی نماد هستند.'}
          </p>
        </div>
        <Button size="sm" variant="ghost" onClick={() => setReload((value) => value + 1)}>
          <RefreshCw className="h-4 w-4" />
          به‌روزرسانی
        </Button>
      </div>

      {summary && summary.total_days === 0 && (
        <div className="flex items-center gap-2 rounded-lg bg-warning-400/15 p-3 text-sm text-warning-700 dark:text-warning-400">
          <AlertTriangle className="h-4 w-4 shrink-0" />
          قیمت پایانی روزانه این نماد هنوز دریافت نشده است. روزهای معاملاتی را ورکر tse-market-data-sync ثبت می‌کند و پس از آن سابقه معاملات قابل دریافت است.
        </div>
      )}

      {summary && incomplete > 0 && (
        <div className="flex flex-wrap items-center gap-3 rounded-lg bg-warning-400/15 p-3 text-sm text-warning-700 dark:text-warning-400">
          <AlertTriangle className="h-4 w-4 shrink-0" />
          <span className="flex-1">
            {neverFetched
              ? `سابقه معاملات این نماد هنوز دریافت نشده است (${fa(summary.total_days)} روز معاملاتی).`
              : `${fa(incomplete)} روز سابقه معاملات دریافت نشده است: ${fa(summary.missing_days)} روز دریافت‌نشده و ${fa(summary.failed_days)} روز خطادار.`}
          </span>
          <label className="flex items-center gap-1.5 text-xs">
            <input type="checkbox" className="h-4 w-4 accent-primary-600" checked={includeEmpty} onChange={(e) => setIncludeEmpty(e.target.checked)} />
            پاسخ‌های خالی هم دوباره دریافت شوند
          </label>
          <Button size="sm" isLoading={queueing} disabled={hasActiveJobs} onClick={() => queue('gaps')}>
            {!queueing && <Download className="h-4 w-4" />}
            {neverFetched ? 'دریافت سابقه کامل معاملات' : `دریافت ${fa(incomplete)} روز ناقص`}
          </Button>
        </div>
      )}

      {summary && summary.total_days > 0 && incomplete === 0 && (
        <div className="flex flex-wrap items-center gap-2 rounded-lg bg-success-50 p-3 text-sm text-success-700 dark:bg-success-950/40">
          <CheckCircle2 className="h-4 w-4" />
          سابقه معاملات همه روزهای معاملاتی دریافت شده است.
          {summary.empty_days > 0 && (
            <Button size="sm" variant="secondary" className="mr-auto" isLoading={queueing} disabled={hasActiveJobs} onClick={() => queue('empty')}>
              دریافت مجدد {fa(summary.empty_days)} روز با پاسخ خالی
            </Button>
          )}
        </div>
      )}

      {summary && summary.mismatched_days > 0 && (
        <div className="flex flex-wrap items-center gap-2 rounded-lg bg-danger-50 p-3 text-sm text-danger-600 dark:bg-danger-950/40">
          <AlertTriangle className="h-4 w-4 shrink-0" />
          <span className="flex-1">
            در {fa(summary.mismatched_days)} روز تعداد یا حجم معاملات دریافتی (بدون معاملات لغوشده) با قیمت پایانی همان روز یکی نیست.
          </span>
          <Button size="sm" variant="ghost" onClick={() => changeFilter({ status: 'MISMATCHED' })}>نمایش</Button>
          <Button size="sm" variant="secondary" isLoading={queueing} disabled={hasActiveJobs} onClick={() => queue('mismatched')}>
            دریافت مجدد این روزها
          </Button>
        </div>
      )}

      {error && <div className="rounded-lg bg-danger-50 p-3 text-sm text-danger-600 dark:bg-danger-950/40">{error}</div>}
      {notice && <div className="rounded-lg bg-success-50 p-3 text-sm text-success-700 dark:bg-success-950/40">{notice}</div>}

      {summary && summary.total_days > 0 && (
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-6">
          <Stat label="روز معاملاتی" value={summary.total_days} active={!filters.status} onClick={() => changeFilter({ status: '' })} />
          <Stat label="دریافت‌شده" value={summary.fetched_days} tone="text-success-600" active={filters.status === 'FETCHED'} onClick={() => changeFilter({ status: 'FETCHED' })} />
          <Stat label="پاسخ خالی" value={summary.empty_days} tone="text-warning-600" active={filters.status === 'EMPTY'} onClick={() => changeFilter({ status: 'EMPTY' })} />
          <Stat label="خطا" value={summary.failed_days} tone="text-danger-600" active={filters.status === 'FAILED'} onClick={() => changeFilter({ status: 'FAILED' })} />
          <Stat label="دریافت‌نشده" value={summary.missing_days} active={filters.status === 'MISSING'} onClick={() => changeFilter({ status: 'MISSING' })} />
          <Stat label="مغایرت" value={summary.mismatched_days} tone="text-danger-600" active={filters.status === 'MISMATCHED'} onClick={() => changeFilter({ status: 'MISMATCHED' })} />
        </div>
      )}

      {summary && summary.active_jobs.length > 0 && (
        <div className="rounded-lg border border-primary-200 p-3 dark:border-primary-900">
          <h3 className="mb-1 text-sm font-semibold text-surface-900 dark:text-white">کارهای در جریان این نماد</h3>
          <TradeHistoryJobs jobs={summary.active_jobs} onChanged={() => setReload((value) => value + 1)} onError={setError} />
        </div>
      )}

      {summary && summary.total_days > 0 && (
        <>
          <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
            <Select
              placeholder="همه وضعیت‌ها"
              value={filters.status}
              onChange={(e) => changeFilter({ status: e.target.value as TradeDayStatus | '' })}
              options={tradeDayStatusOptions}
            />
            <Input type="date" aria-label="از تاریخ" value={filters.from} onChange={(e) => changeFilter({ from: e.target.value })} />
            <Input type="date" aria-label="تا تاریخ" value={filters.to} onChange={(e) => changeFilter({ to: e.target.value })} />
            <div className="flex items-center gap-2">
              {selected.length > 0 && (
                <>
                  <Button size="sm" isLoading={queueing} onClick={() => queue('selected')}>
                    دریافت {fa(selected.length)} روز انتخابی
                  </Button>
                  <Button size="sm" variant="ghost" onClick={() => setSelected([])}>لغو انتخاب</Button>
                </>
              )}
            </div>
          </div>

          <p className="text-[11px] text-surface-500">
            ستون‌های مقایسه: «سابقه معاملات بدون لغوشده‌ها / قیمت پایانی». عدد قرمز یعنی دو منبع یکی نیستند.
          </p>

          <div className="overflow-x-auto rounded-lg border border-surface-200 dark:border-surface-700">
            <table className="w-full min-w-[960px] text-sm">
              <thead className="bg-surface-50 text-xs text-surface-500 dark:bg-surface-900">
                <tr>
                  <th className="p-2">
                    <input
                      type="checkbox"
                      aria-label="انتخاب روزهای این صفحه"
                      className="h-4 w-4 accent-primary-600"
                      checked={allPageSelected}
                      onChange={() =>
                        setSelected((current) =>
                          allPageSelected ? current.filter((date) => !pageDates.includes(date)) : [...new Set([...current, ...pageDates])],
                        )
                      }
                    />
                  </th>
                  <th className="p-2 text-right">تاریخ</th>
                  <th className="p-2 text-right">وضعیت</th>
                  <th className="p-2 text-right">تعداد معاملات</th>
                  <th className="p-2 text-right">حجم</th>
                  <th className="p-2 text-right">لغوشده (تعداد / حجم)</th>
                  <th className="p-2 text-right">تلاش</th>
                  <th className="p-2 text-right">آخرین دریافت</th>
                  <th className="p-2" />
                </tr>
              </thead>
              <tbody className="divide-y divide-surface-100 dark:divide-surface-800">
                {days.map((day) => {
                  const style = dayStatusStyle[day.status];
                  const fetched = day.status === 'FETCHED' || day.status === 'EMPTY';
                  const busy = fetching.includes(day.date);
                  return (
                    <tr key={day.date} className={selected.includes(day.date) ? 'bg-primary-50/60 dark:bg-primary-950/30' : ''}>
                      <td className="p-2 text-center">
                        <input
                          type="checkbox"
                          aria-label={`انتخاب ${day.date}`}
                          className="h-4 w-4 accent-primary-600"
                          checked={selected.includes(day.date)}
                          onChange={() =>
                            setSelected((current) => (current.includes(day.date) ? current.filter((date) => date !== day.date) : [...current, day.date]))
                          }
                        />
                      </td>
                      <td className="p-2">
                        <div className="font-medium text-surface-900 dark:text-white">{formatDate(day.date)}</div>
                        <div className="text-[11px] text-surface-400">
                          {weekdayName(day.date)} · <span dir="ltr">{day.date}</span>
                        </div>
                      </td>
                      <td className="p-2">
                        <div className="flex flex-wrap items-center gap-1">
                          <Badge variant={style.variant}>{style.label}</Badge>
                          {day.mismatched && <Badge variant="danger">مغایرت</Badge>}
                        </div>
                        {day.error && (
                          <p className="mt-1 max-w-xs truncate text-[11px] text-danger-600" title={day.error} dir="ltr">
                            {day.error}
                          </p>
                        )}
                      </td>
                      <td className="p-2">
                        <Compare history={day.history_trade_count - day.history_canceled_count} closing={day.trade_count} fetched={fetched} />
                      </td>
                      <td className="p-2">
                        <Compare history={day.history_net_volume} closing={day.volume} fetched={fetched} />
                      </td>
                      <td className="p-2 text-surface-600 dark:text-surface-300">
                        {fetched ? `${fa(day.history_canceled_count)} / ${fa(day.history_volume - day.history_net_volume)}` : '—'}
                      </td>
                      <td className="p-2">{day.attempts > 0 ? fa(day.attempts) : '—'}</td>
                      <td className="p-2 text-xs text-surface-500">{formatDateTime(day.fetched_at)}</td>
                      <td className="p-2 text-left">
                        <Button size="sm" variant={day.status === 'FETCHED' && !day.mismatched ? 'ghost' : 'secondary'} isLoading={busy} onClick={() => fetchDay(day.date)}>
                          {!busy && <Download className="h-3.5 w-3.5" />}
                          {day.status === 'MISSING' ? 'دریافت' : 'دریافت مجدد'}
                        </Button>
                      </td>
                    </tr>
                  );
                })}
                {days.length === 0 && (
                  <tr>
                    <td colSpan={9} className="p-6 text-center text-sm text-surface-500">روزی با این فیلتر یافت نشد.</td>
                  </tr>
                )}
              </tbody>
            </table>
            {pagination && (
              <Pagination page={page} totalPages={pagination.total_pages} totalItems={pagination.total_items} onChange={setPage} />
            )}
          </div>
        </>
      )}
    </Card>
  );
}

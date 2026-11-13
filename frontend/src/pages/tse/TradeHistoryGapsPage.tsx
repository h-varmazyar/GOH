import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { AlertTriangle, CheckCircle2, Download, History, RefreshCw, Search } from 'lucide-react';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import { confirmDialog } from '../../components/ui/dialogs';
import { Input } from '../../components/ui/Input';
import { Pagination } from '../../components/ui/Pagination';
import { Select } from '../../components/ui/Select';
import { TradeHistoryJobs } from '../../components/tse/TradeHistoryJobs';
import { errorMessage, formatDate, formatDateTime } from '../../components/tse/format';
import type { Pagination as PaginationInfo } from '../../services/airdropService';
import {
  gapProblemOptions,
  tradeHistoryService,
  type CreateTradeJob,
  type TradeDayStatus,
  type TradeGap,
  type TradeGapFilter,
  type TradeGapProblem,
  type TradeGapTotals,
  type TradeHistoryJob,
} from '../../services/tradeHistoryService';

const PAGE_SIZE = 20;
const POLL_INTERVAL = 10_000;

const fa = (value: number | null | undefined) => (value === null || value === undefined ? '—' : value.toLocaleString('fa-IR'));

const boolOptions = (yes: string, no: string) => [
  { value: 'true', label: yes },
  { value: 'false', label: no },
];

interface Filters {
  q: string;
  problem: TradeGapProblem;
  dataCollection: string;
  delisted: string;
}

const toApiFilter = (filters: Filters): TradeGapFilter => ({
  q: filters.q.trim() || undefined,
  problem: filters.problem || undefined,
  data_collection: filters.dataCollection === '' ? undefined : filters.dataCollection === 'true',
  delisted: filters.delisted === '' ? undefined : filters.delisted === 'true',
});

function Tile({ label, value, tone, hint }: { label: string; value: string; tone?: string; hint?: string }) {
  return (
    <div className="rounded-xl border border-surface-200 bg-white p-4 dark:border-surface-700 dark:bg-surface-900">
      <div className="text-sm text-surface-500">{label}</div>
      <div className={`mt-1 text-2xl font-semibold ${tone || 'text-surface-900 dark:text-white'}`}>{value}</div>
      {hint && <div className="mt-0.5 text-[11px] text-surface-400">{hint}</div>}
    </div>
  );
}

function Coverage({ gap }: { gap: TradeGap }) {
  const total = Math.max(gap.total_days, 1);
  const width = (value: number) => `${(100 * value) / total}%`;
  const percent = Math.round((100 * (gap.fetched_days + gap.empty_days)) / total);
  return (
    <div className="min-w-[120px]">
      <div className="flex h-2 overflow-hidden rounded-full bg-surface-100 dark:bg-surface-700" title="دریافت‌شده / خالی / خطا / دریافت‌نشده">
        <div className="bg-success-500" style={{ width: width(gap.fetched_days) }} />
        <div className="bg-warning-400" style={{ width: width(gap.empty_days) }} />
        <div className="bg-danger-500" style={{ width: width(gap.failed_days) }} />
      </div>
      <div className="mt-0.5 text-[11px] text-surface-400">{fa(percent)}٪ از {fa(gap.total_days)} روز</div>
    </div>
  );
}

/**
 * Markets whose trade history is incomplete. Missing, failed or empty days of
 * the selected markets, or of every market matching the filter, are queued as
 * one job for the tse-trade-history worker.
 */
export function TradeHistoryGapsPage() {
  const [filters, setFilters] = useState<Filters>({ q: '', problem: '', dataCollection: 'true', delisted: '' });
  const [query, setQuery] = useState('');
  const [page, setPage] = useState(1);
  const [gaps, setGaps] = useState<TradeGap[]>([]);
  const [totals, setTotals] = useState<TradeGapTotals | null>(null);
  const [pagination, setPagination] = useState<PaginationInfo | null>(null);
  const [jobs, setJobs] = useState<TradeHistoryJob[]>([]);
  const [selected, setSelected] = useState<number[]>([]);
  const [includeEmpty, setIncludeEmpty] = useState(false);
  const [queueing, setQueueing] = useState(false);
  const [reload, setReload] = useState(0);
  const [loadedKey, setLoadedKey] = useState('');
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  useEffect(() => {
    const timer = setTimeout(() => {
      setFilters((current) => (current.q === query ? current : { ...current, q: query }));
      setPage(1);
    }, 400);
    return () => clearTimeout(timer);
  }, [query]);

  const requestKey = JSON.stringify({ filters, page, reload });
  const isLoading = loadedKey !== requestKey;

  useEffect(() => {
    let cancelled = false;
    Promise.all([tradeHistoryService.gaps(toApiFilter(filters), page, PAGE_SIZE), tradeHistoryService.jobs({}, 1, 10)])
      .then(([gapRes, jobRes]) => {
        if (cancelled) return;
        setGaps(gapRes.data || []);
        setTotals(gapRes.totals);
        setPagination(gapRes.pagination);
        setJobs(jobRes.data || []);
        setError('');
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(errorMessage(e, 'خطا در دریافت وضعیت سابقه معاملات'));
      })
      .finally(() => {
        if (!cancelled) setLoadedKey(requestKey);
      });
    return () => {
      cancelled = true;
    };
  }, [filters, page, requestKey]);

  const hasActiveJobs = jobs.some((job) => job.status === 'PENDING' || job.status === 'RUNNING');
  useEffect(() => {
    if (!hasActiveJobs) return;
    const timer = setInterval(() => setReload((value) => value + 1), POLL_INTERVAL);
    return () => clearInterval(timer);
  }, [hasActiveJobs]);

  const changeFilter = (key: keyof Filters, value: string) => {
    setFilters((current) => ({ ...current, [key]: value }));
    setPage(1);
    setSelected([]);
  };

  const dayStatuses = (): TradeDayStatus[] => (includeEmpty ? ['MISSING', 'FAILED', 'EMPTY', 'MISMATCHED'] : ['MISSING', 'FAILED']);

  const queue = async (target: Pick<CreateTradeJob, 'tse_market_ids' | 'gaps' | 'all_markets'>, description: string) => {
    const confirmed = await confirmDialog({
      title: 'دریافت سابقه معاملات',
      message: `دریافت روزهای ${includeEmpty ? 'دریافت‌نشده، خطادار، خالی و دارای مغایرت' : 'دریافت‌نشده و خطادار'} ${description} در صف قرار بگیرد؟`,
      warning: 'هر روز یک درخواست جداگانه به TSETMC است؛ کارهای بزرگ ممکن است چند روز طول بکشند. روزهای دریافت‌شده دوباره دریافت نمی‌شوند.',
      confirmLabel: 'افزودن به صف',
    });
    if (!confirmed) return;
    setQueueing(true);
    setError('');
    try {
      const job = await tradeHistoryService.createJob({ ...target, day_statuses: dayStatuses() });
      setNotice(`کار «${job.title}» در صف قرار گرفت؛ ورکر tse-trade-history آن را اجرا می‌کند.`);
      setSelected([]);
      setReload((value) => value + 1);
    } catch (e) {
      setError(errorMessage(e, 'خطا در افزودن کار به صف'));
    } finally {
      setQueueing(false);
    }
  };

  const totalItems = pagination?.total_items ?? 0;
  const pageIds = gaps.map((gap) => gap.tse_market_id);
  const allPageSelected = pageIds.length > 0 && pageIds.every((id) => selected.includes(id));
  const coverage = totals && totals.total_days > 0 ? (100 * (totals.fetched_days + totals.empty_days)) / totals.total_days : 0;

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="flex items-center gap-2 text-xl font-semibold text-surface-900 dark:text-white">
            <History className="h-6 w-6" />
            پایش سابقه معاملات
          </h1>
          <p className="mt-1 text-sm text-surface-500">
            نمادهایی که سابقه معاملات بعضی روزهایشان دریافت نشده، خطا یا پاسخ خالی داشته یا با قیمت پایانی همان روز مغایرت دارد. روزهای معاملاتی همان روزهای قیمت پایانی هستند.
          </p>
        </div>
        <Button size="sm" variant="ghost" onClick={() => setReload((value) => value + 1)}>
          <RefreshCw className="h-4 w-4" />
          به‌روزرسانی
        </Button>
      </div>

      {totals && (
        <div className="grid gap-3 sm:grid-cols-3 lg:grid-cols-6">
          <Tile label="نمادهای نتیجه فیلتر" value={fa(totals.markets)} />
          <Tile label="پوشش روزها" value={`${coverage.toLocaleString('fa-IR', { maximumFractionDigits: 1 })}٪`} hint={`${fa(totals.total_days)} روز معاملاتی`} />
          <Tile label="روز دریافت‌نشده" value={fa(totals.missing_days)} />
          <Tile label="روز خطادار" value={fa(totals.failed_days)} tone="text-danger-600" />
          <Tile label="روز با پاسخ خالی" value={fa(totals.empty_days)} tone="text-warning-600" />
          <Tile label="روز دارای مغایرت" value={fa(totals.mismatched_days)} tone="text-danger-600" hint="با قیمت پایانی" />
        </div>
      )}

      {totals && totals.without_days > 0 && (
        <div className="flex items-center gap-2 rounded-lg bg-warning-400/15 p-3 text-sm text-warning-700 dark:text-warning-400">
          <AlertTriangle className="h-4 w-4 shrink-0" />
          {fa(totals.without_days)} نماد هنوز قیمت پایانی روزانه ندارند و در این فهرست نیستند؛ روزهای معاملاتی‌شان را ورکر tse-market-data-sync ثبت می‌کند.
        </div>
      )}

      <Card padding="sm">
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <div className="relative">
            <Search className="absolute right-3 top-2.5 h-5 w-5 text-surface-400" />
            <Input className="pr-10" placeholder="نماد، نام یا insCode..." value={query} onChange={(e) => setQuery(e.target.value)} />
          </div>
          <Select value={filters.problem} onChange={(e) => changeFilter('problem', e.target.value)} options={gapProblemOptions} />
          <Select placeholder="دریافت داده: همه" value={filters.dataCollection} onChange={(e) => changeFilter('dataCollection', e.target.value)} options={boolOptions('دریافت داده فعال', 'دریافت داده غیرفعال')} />
          <Select placeholder="پذیرش: همه" value={filters.delisted} onChange={(e) => changeFilter('delisted', e.target.value)} options={boolOptions('لغو پذیرش‌شده', 'فعال در بازار')} />
        </div>
      </Card>

      <Card padding="sm">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <label className="flex items-center gap-2 text-sm text-surface-600 dark:text-surface-300">
            <input
              type="checkbox"
              className="h-4 w-4 accent-primary-600"
              checked={allPageSelected}
              onChange={() =>
                setSelected((current) => (allPageSelected ? current.filter((id) => !pageIds.includes(id)) : [...new Set([...current, ...pageIds])]))
              }
            />
            {selected.length > 0 ? `${fa(selected.length)} نماد انتخاب شده` : `${fa(totalItems)} نماد`}
          </label>
          <div className="flex flex-wrap items-center gap-2">
            <label className="flex items-center gap-1.5 text-xs text-surface-600 dark:text-surface-300">
              <input type="checkbox" className="h-4 w-4 accent-primary-600" checked={includeEmpty} onChange={(e) => setIncludeEmpty(e.target.checked)} />
              روزهای خالی و دارای مغایرت هم دوباره دریافت شوند
            </label>
            {selected.length > 0 ? (
              <>
                <Button size="sm" isLoading={queueing} onClick={() => queue({ tse_market_ids: selected }, `${fa(selected.length)} نماد انتخاب‌شده`)}>
                  {!queueing && <Download className="h-4 w-4" />}
                  دریافت برای انتخاب‌شده‌ها
                </Button>
                <Button size="sm" variant="ghost" onClick={() => setSelected([])}>لغو انتخاب</Button>
              </>
            ) : (
              <Button
                size="sm"
                variant="secondary"
                isLoading={queueing}
                disabled={totalItems === 0}
                onClick={() => queue({ gaps: toApiFilter(filters) }, `همه ${fa(totalItems)} نماد نتیجه فیلتر`)}
              >
                {!queueing && <Download className="h-4 w-4" />}
                دریافت برای همه نتایج فیلتر
              </Button>
            )}
          </div>
        </div>
      </Card>

      {error && <div className="rounded-lg bg-danger-50 p-3 text-sm text-danger-600 dark:bg-danger-950/40">{error}</div>}
      {notice && <div className="rounded-lg bg-success-50 p-3 text-sm text-success-700 dark:bg-success-950/40">{notice}</div>}

      <Card padding="none" className="overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[980px] text-sm">
            <thead className="bg-surface-50 text-xs text-surface-500 dark:bg-surface-900">
              <tr>
                <th className="w-10 p-2" />
                <th className="p-2 text-right">نماد</th>
                <th className="p-2 text-right">بازه</th>
                <th className="p-2 text-right">پوشش</th>
                <th className="p-2 text-right">دریافت‌نشده</th>
                <th className="p-2 text-right">خطا</th>
                <th className="p-2 text-right">خالی</th>
                <th className="p-2 text-right">مغایرت</th>
                <th className="p-2 text-right">آخرین دریافت</th>
                <th className="p-2" />
              </tr>
            </thead>
            <tbody className="divide-y divide-surface-100 dark:divide-surface-800">
              {gaps.map((gap) => (
                <tr key={gap.tse_market_id} className={selected.includes(gap.tse_market_id) ? 'bg-primary-50/60 dark:bg-primary-950/30' : ''}>
                  <td className="p-2 text-center">
                    <input
                      type="checkbox"
                      aria-label={`انتخاب ${gap.symbol}`}
                      className="h-4 w-4 accent-primary-600"
                      checked={selected.includes(gap.tse_market_id)}
                      onChange={() =>
                        setSelected((current) =>
                          current.includes(gap.tse_market_id) ? current.filter((id) => id !== gap.tse_market_id) : [...current, gap.tse_market_id],
                        )
                      }
                    />
                  </td>
                  <td className="p-2">
                    <div className="flex flex-wrap items-center gap-1.5">
                      <Link to={`/tse/markets/${gap.tse_market_id}`} className="font-bold text-primary-700 hover:underline dark:text-primary-300">
                        {gap.symbol}
                      </Link>
                      {gap.delisted && <Badge variant="danger">لغو پذیرش</Badge>}
                      {!gap.data_collection_enabled && <Badge>دریافت خاموش</Badge>}
                    </div>
                    <div className="max-w-[220px] truncate text-[11px] text-surface-400">{gap.name} · <span dir="ltr">{gap.ins_code}</span></div>
                  </td>
                  <td className="p-2 text-xs text-surface-600 dark:text-surface-300">
                    {formatDate(gap.first_date)} تا {formatDate(gap.last_date)}
                  </td>
                  <td className="p-2"><Coverage gap={gap} /></td>
                  <td className="p-2 font-medium">{fa(gap.missing_days)}</td>
                  <td className="p-2">
                    <span className={gap.failed_days > 0 ? 'font-medium text-danger-600' : ''}>{fa(gap.failed_days)}</span>
                  </td>
                  <td className="p-2 text-warning-600">{fa(gap.empty_days)}</td>
                  <td className="p-2">
                    <span className={gap.mismatched_days > 0 ? 'font-medium text-danger-600' : ''}>{fa(gap.mismatched_days)}</span>
                  </td>
                  <td className="p-2 text-xs text-surface-500">{gap.last_fetched_at ? formatDateTime(gap.last_fetched_at) : 'هرگز'}</td>
                  <td className="p-2 text-left">
                    <Button size="sm" variant="secondary" disabled={queueing} onClick={() => queue({ tse_market_ids: [gap.tse_market_id] }, `نماد ${gap.symbol}`)}>
                      <Download className="h-3.5 w-3.5" />
                      دریافت
                    </Button>
                  </td>
                </tr>
              ))}
              {!isLoading && gaps.length === 0 && (
                <tr>
                  <td colSpan={10} className="p-6">
                    <p className="flex items-center justify-center gap-2 text-sm text-surface-500">
                      <CheckCircle2 className="h-5 w-5 text-success-500" />
                      نمادی با این فیلتر یافت نشد
                    </p>
                  </td>
                </tr>
              )}
              {isLoading && gaps.length === 0 && (
                <tr>
                  <td colSpan={10} className="p-6 text-center text-sm text-surface-500">در حال دریافت...</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        {pagination && <Pagination page={page} totalPages={pagination.total_pages} totalItems={totalItems} onChange={setPage} />}
      </Card>

      <Card padding="sm">
        <h2 className="mb-2 font-semibold text-surface-900 dark:text-white">کارهای دریافت سابقه معاملات</h2>
        <p className="mb-2 text-xs text-surface-500">ورکر tse-trade-history کارها را به ترتیب اجرا می‌کند. کاری که با توقف ورکر نیمه‌کاره بماند دوباره در صف قرار می‌گیرد و از روزهای باقی‌مانده ادامه می‌یابد.</p>
        <TradeHistoryJobs jobs={jobs} onChanged={() => setReload((value) => value + 1)} onError={setError} emptyText="هنوز کاری ثبت نشده است." />
      </Card>
    </div>
  );
}

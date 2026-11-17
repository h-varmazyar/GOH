import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { CheckCircle2, Download, History, RefreshCw, Search } from 'lucide-react';
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
import { Alert } from '../../components/ui/Alert';

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
    <div className="rounded-lg border border-border bg-surface p-4">
      <div className="text-body text-text-subtle">{label}</div>
      <div className={`mt-1 text-h2 font-semibold ${tone || 'text-text'}`}>{value}</div>
      {hint && <div className="mt-0.5 text-[11px] text-text-subtle">{hint}</div>}
    </div>
  );
}

function Coverage({ gap }: { gap: TradeGap }) {
  const total = Math.max(gap.total_days, 1);
  const width = (value: number) => `${(100 * value) / total}%`;
  const percent = Math.round((100 * (gap.fetched_days + gap.empty_days)) / total);
  return (
    <div className="min-w-[120px]">
      <div className="flex h-2 overflow-hidden rounded-pill bg-surface-muted" title="دریافت‌شده / خالی / خطا / دریافت‌نشده">
        <div className="bg-success" style={{ width: width(gap.fetched_days) }} />
        <div className="bg-warning" style={{ width: width(gap.empty_days) }} />
        <div className="bg-error" style={{ width: width(gap.failed_days) }} />
      </div>
      <div className="mt-0.5 text-[11px] text-text-subtle">{fa(percent)}٪ از {fa(gap.total_days)} روز</div>
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
          <h1 className="flex items-center gap-2 text-h3 font-semibold text-text">
            <History className="h-6 w-6" />
            پایش سابقه معاملات
          </h1>
          <p className="mt-1 text-body text-text-subtle">
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
          <Tile label="روز خطادار" value={fa(totals.failed_days)} tone="text-error" />
          <Tile label="روز با پاسخ خالی" value={fa(totals.empty_days)} tone="text-warning-text" />
          <Tile label="روز دارای مغایرت" value={fa(totals.mismatched_days)} tone="text-error" hint="با قیمت پایانی" />
        </div>
      )}

      {totals && totals.without_days > 0 && (
        <Alert tone="warning">
          {fa(totals.without_days)} نماد هنوز قیمت پایانی روزانه ندارند و در این فهرست نیستند؛ روزهای معاملاتی‌شان را ورکر tse-market-data-sync ثبت می‌کند.
        </Alert>
      )}

      <Card padding="sm">
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <Input leadingIcon={<Search />} aria-label="جستجو" placeholder="نماد، نام یا insCode..." value={query} onChange={(e) => setQuery(e.target.value)} />
          <Select value={filters.problem} onChange={(e) => changeFilter('problem', e.target.value)} options={gapProblemOptions} />
          <Select placeholder="دریافت داده: همه" value={filters.dataCollection} onChange={(e) => changeFilter('dataCollection', e.target.value)} options={boolOptions('دریافت داده فعال', 'دریافت داده غیرفعال')} />
          <Select placeholder="پذیرش: همه" value={filters.delisted} onChange={(e) => changeFilter('delisted', e.target.value)} options={boolOptions('لغو پذیرش‌شده', 'فعال در بازار')} />
        </div>
      </Card>

      <Card padding="sm">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <label className="flex items-center gap-2 text-body text-text-muted">
            <input
              type="checkbox"
              className="h-4 w-4 accent-action"
              checked={allPageSelected}
              onChange={() =>
                setSelected((current) => (allPageSelected ? current.filter((id) => !pageIds.includes(id)) : [...new Set([...current, ...pageIds])]))
              }
            />
            {selected.length > 0 ? `${fa(selected.length)} نماد انتخاب شده` : `${fa(totalItems)} نماد`}
          </label>
          <div className="flex flex-wrap items-center gap-2">
            <label className="flex items-center gap-1.5 text-caption text-text-muted">
              <input type="checkbox" className="h-4 w-4 accent-action" checked={includeEmpty} onChange={(e) => setIncludeEmpty(e.target.checked)} />
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

      {error && <Alert tone="error">{error}</Alert>}
      {notice && <Alert tone="success">{notice}</Alert>}

      <Card padding="none">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[980px] text-body">
            <thead className="bg-surface-muted text-label text-text-muted">
              <tr>
                <th className="w-10 p-2" />
                <th className="p-2 text-start">نماد</th>
                <th className="p-2 text-start">بازه</th>
                <th className="p-2 text-start">پوشش</th>
                <th className="p-2 text-start">دریافت‌نشده</th>
                <th className="p-2 text-start">خطا</th>
                <th className="p-2 text-start">خالی</th>
                <th className="p-2 text-start">مغایرت</th>
                <th className="p-2 text-start">آخرین دریافت</th>
                <th className="p-2" />
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {gaps.map((gap) => (
                <tr key={gap.tse_market_id} className={selected.includes(gap.tse_market_id) ? 'bg-selected/8' : ''}>
                  <td className="p-2 text-center">
                    <input
                      type="checkbox"
                      aria-label={`انتخاب ${gap.symbol}`}
                      className="h-4 w-4 accent-action"
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
                      <Link to={`/tse/markets/${gap.tse_market_id}`} className="font-bold text-link hover:underline">
                        {gap.symbol}
                      </Link>
                      {gap.delisted && <Badge variant="danger">لغو پذیرش</Badge>}
                      {!gap.data_collection_enabled && <Badge>دریافت خاموش</Badge>}
                    </div>
                    <div className="max-w-[220px] truncate text-[11px] text-text-subtle">{gap.name} · <span dir="ltr">{gap.ins_code}</span></div>
                  </td>
                  <td className="p-2 text-caption text-text-muted">
                    {formatDate(gap.first_date)} تا {formatDate(gap.last_date)}
                  </td>
                  <td className="p-2"><Coverage gap={gap} /></td>
                  <td className="p-2 font-medium">{fa(gap.missing_days)}</td>
                  <td className="p-2">
                    <span className={gap.failed_days > 0 ? 'font-medium text-error' : ''}>{fa(gap.failed_days)}</span>
                  </td>
                  <td className="p-2 text-warning-text">{fa(gap.empty_days)}</td>
                  <td className="p-2">
                    <span className={gap.mismatched_days > 0 ? 'font-medium text-error' : ''}>{fa(gap.mismatched_days)}</span>
                  </td>
                  <td className="p-2 text-caption text-text-subtle">{gap.last_fetched_at ? formatDateTime(gap.last_fetched_at) : 'هرگز'}</td>
                  <td className="p-2 text-end">
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
                    <p className="flex items-center justify-center gap-2 text-body text-text-subtle">
                      <CheckCircle2 className="h-5 w-5 text-success" />
                      نمادی با این فیلتر یافت نشد
                    </p>
                  </td>
                </tr>
              )}
              {isLoading && gaps.length === 0 && (
                <tr>
                  <td colSpan={10} className="p-6 text-center text-body text-text-subtle">در حال دریافت...</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        {pagination && <Pagination page={page} totalPages={pagination.total_pages} totalItems={totalItems} onChange={setPage} />}
      </Card>

      <Card padding="sm">
        <h2 className="mb-2 font-semibold text-text">کارهای دریافت سابقه معاملات</h2>
        <p className="mb-2 text-caption text-text-subtle">ورکر tse-trade-history کارها را به ترتیب اجرا می‌کند. کاری که با توقف ورکر نیمه‌کاره بماند دوباره در صف قرار می‌گیرد و از روزهای باقی‌مانده ادامه می‌یابد.</p>
        <TradeHistoryJobs jobs={jobs} onChanged={() => setReload((value) => value + 1)} onError={setError} emptyText="هنوز کاری ثبت نشده است." />
      </Card>
    </div>
  );
}

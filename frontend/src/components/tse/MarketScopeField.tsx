import { useEffect, useRef, useState } from 'react';
import { Loader2, Search, X } from 'lucide-react';
import { Badge } from '../ui/Badge';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { Select } from '../ui/Select';
import { errorMessage } from './format';
import { marketTypeOptions, tseMarketService, type IndustryGroup } from '../../services/tseMarketService';
import {
  indicatorService,
  scopeLabel,
  type IndicatorMarket,
  type IndicatorMarketFilter,
  type IndicatorScope,
  type MarketRef,
} from '../../services/indicatorService';
import type { MarketScopeValue } from './marketScope';

const PAGE_SIZE = 30;
/** Every result of the filter can be selected at once up to this many markets. */
const SELECT_ALL_LIMIT = 1000;
const fa = (value: number) => value.toLocaleString('fa-IR');

const scopeHints: Record<IndicatorScope, string> = {
  ALL: 'برای همه نمادهای بورس تهران محاسبه می‌شود.',
  GROUP: 'برای نمادهای انتخاب‌شده محاسبه می‌شود.',
  MARKET: 'فقط برای یک نماد محاسبه می‌شود.',
};

/** Chooses every market, a group of markets or one market. */
export function MarketScopeField({
  value,
  onChange,
  error,
}: {
  value: MarketScopeValue;
  onChange: (value: MarketScopeValue) => void;
  error?: string;
}) {
  const setScope = (scope: IndicatorScope) =>
    // One market keeps a single selection only.
    onChange({ scope, markets: scope === 'MARKET' && value.markets.length > 1 ? [] : value.markets });

  const selectedIds = value.markets.map((market) => market.id);
  const toggle = (markets: MarketRef[], checked: boolean) => {
    if (value.scope === 'MARKET') {
      onChange({ ...value, markets: checked ? markets.slice(0, 1) : [] });
      return;
    }
    const ids = new Set(markets.map((market) => market.id));
    const rest = value.markets.filter((market) => !ids.has(market.id));
    onChange({ ...value, markets: checked ? [...rest, ...markets] : rest });
  };

  return (
    <fieldset>
      <legend className="text-body font-semibold text-text">نمادها</legend>
      <div className="mt-3 grid gap-2 sm:grid-cols-3">
        {(['ALL', 'GROUP', 'MARKET'] as const).map((scope) => (
          <label
            key={scope}
            className={`flex cursor-pointer gap-3 rounded-md border p-3 transition-colors has-focus-visible:ring-2 has-focus-visible:ring-focus-ring ${
              value.scope === scope ? 'border-selected bg-selected/10' : 'border-border-strong hover:bg-surface-muted'
            }`}
          >
            <input type="radio" name="indicator-scope" className="mt-1 h-4 w-4 accent-action" checked={value.scope === scope} onChange={() => setScope(scope)} />
            <span>
              <span className="block text-body font-medium text-text">{scopeLabel[scope]}</span>
              <span className="block text-caption text-text-subtle">{scopeHints[scope]}</span>
            </span>
          </label>
        ))}
      </div>

      {value.scope !== 'ALL' && (
        <div className="mt-4 space-y-3">
          <SelectedMarkets
            markets={value.markets}
            onRemove={(market) => toggle([market], false)}
            onClear={() => onChange({ ...value, markets: [] })}
            single={value.scope === 'MARKET'}
          />
          <MarketTable multiple={value.scope === 'GROUP'} selectedIds={selectedIds} onToggle={toggle} />
          {error && <p className="text-caption text-error">{error}</p>}
        </div>
      )}
    </fieldset>
  );
}

const SHOWN_CHIPS = 40;

function SelectedMarkets({
  markets,
  onRemove,
  onClear,
  single,
}: {
  markets: MarketRef[];
  onRemove: (market: MarketRef) => void;
  onClear: () => void;
  single: boolean;
}) {
  if (markets.length === 0) {
    return <p className="text-caption text-text-subtle">{single ? 'نمادی انتخاب نشده است.' : 'هنوز نمادی به گروه افزوده نشده است.'}</p>;
  }
  return (
    <div className="rounded-md border border-border bg-surface-muted/50 p-3">
      <div className="mb-2 flex items-center justify-between gap-2">
        <span className="text-label text-text">{single ? 'نماد انتخاب‌شده' : `${fa(markets.length)} نماد انتخاب‌شده`}</span>
        {!single && (
          <Button size="sm" variant="ghost" onClick={onClear}>
            حذف همه
          </Button>
        )}
      </div>
      <ul className="flex flex-wrap gap-1.5">
        {markets.slice(0, SHOWN_CHIPS).map((market) => (
          <li key={market.id} className="inline-flex items-center gap-1 rounded-pill border border-border bg-surface ps-2.5 text-caption text-text">
            {market.label || `#${market.id}`}
            <button
              type="button"
              onClick={() => onRemove(market)}
              className="inline-flex h-7 w-7 items-center justify-center rounded-pill text-text-subtle hover:bg-surface-muted hover:text-text"
              aria-label={`حذف ${market.label}`}
            >
              <X className="h-3.5 w-3.5" aria-hidden="true" />
            </button>
          </li>
        ))}
        {markets.length > SHOWN_CHIPS && (
          <li className="inline-flex items-center px-2 text-caption text-text-subtle">و {fa(markets.length - SHOWN_CHIPS)} نماد دیگر</li>
        )}
      </ul>
    </div>
  );
}

interface LoadedPages {
  key: string;
  rows: IndicatorMarket[];
  page: number;
  totalPages: number;
  total: number;
}

const marketTypeLabel = (value: string) => marketTypeOptions.find((option) => option.value === value)?.label ?? (value || '—');

/** Lists TSE markets page by page as the table scrolls. */
function MarketTable({
  multiple,
  selectedIds,
  onToggle,
}: {
  multiple: boolean;
  selectedIds: number[];
  onToggle: (markets: MarketRef[], checked: boolean) => void;
}) {
  const [query, setQuery] = useState('');
  const [q, setQ] = useState('');
  const [group, setGroup] = useState('');
  const [includeDelisted, setIncludeDelisted] = useState(false);
  const [groups, setGroups] = useState<IndustryGroup[]>([]);
  const [loaded, setLoaded] = useState<LoadedPages | null>(null);
  const [pendingPage, setPendingPage] = useState('');
  const [failure, setFailure] = useState<{ key: string; message: string } | null>(null);
  const [selectingAll, setSelectingAll] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const loadMoreRef = useRef<() => void>(() => {});

  const filter: IndicatorMarketFilter = { q: q.trim() || undefined, group: group || undefined, include_delisted: includeDelisted || undefined };
  const key = JSON.stringify(filter);
  const current = loaded?.key === key ? loaded : null;
  const error = failure?.key === key ? failure.message : '';

  useEffect(() => {
    tseMarketService.industryGroups().then((res) => setGroups(res.data || [])).catch(() => setGroups([]));
  }, []);

  // Debounce the search box.
  useEffect(() => {
    const timer = setTimeout(() => setQ(query), 300);
    return () => clearTimeout(timer);
  }, [query]);

  // A new filter starts again from its first page.
  useEffect(() => {
    let cancelled = false;
    const requestFilter = JSON.parse(key) as IndicatorMarketFilter;
    indicatorService
      .markets(requestFilter, 1, PAGE_SIZE)
      .then((res) => {
        if (cancelled) return;
        setLoaded({ key, rows: res.data || [], page: 1, totalPages: res.pagination.total_pages, total: res.pagination.total_items });
        setFailure(null);
      })
      .catch((e: unknown) => {
        if (!cancelled) setFailure({ key, message: errorMessage(e, 'خطا در دریافت نمادها') });
      });
    scrollRef.current?.scrollTo({ top: 0 });
    return () => {
      cancelled = true;
    };
  }, [key]);

  const loadMore = () => {
    if (!current || current.page >= current.totalPages) return;
    const next = current.page + 1;
    const request = `${key}:${next}`;
    if (pendingPage === request) return;
    setPendingPage(request);
    indicatorService
      .markets(filter, next, PAGE_SIZE)
      .then((res) =>
        setLoaded((previous) =>
          previous && previous.key === key && previous.page === next - 1
            ? { ...previous, rows: [...previous.rows, ...(res.data || [])], page: next, totalPages: res.pagination.total_pages, total: res.pagination.total_items }
            : previous,
        ),
      )
      .catch((e: unknown) => setFailure({ key, message: errorMessage(e, 'خطا در دریافت نمادها') }))
      .finally(() => setPendingPage((value) => (value === request ? '' : value)));
  };

  useEffect(() => {
    loadMoreRef.current = loadMore;
  });

  // A new observer reports at once whether the end of the table is in view,
  // so pages load until the table fills its box.
  const rowCount = current?.rows.length ?? 0;
  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel || rowCount === 0) return;
    const observer = new IntersectionObserver((entries) => entries.some((entry) => entry.isIntersecting) && loadMoreRef.current(), {
      root: scrollRef.current,
      rootMargin: '120px',
    });
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [rowCount, key]);

  const rows = current?.rows ?? [];
  const toRef = (market: IndicatorMarket): MarketRef => ({ id: market.id, label: market.label });
  const allLoadedSelected = rows.length > 0 && rows.every((market) => selectedIds.includes(market.id));
  const total = current?.total ?? 0;
  const hasMore = current ? current.page < current.totalPages : false;

  // Selects every market of the filter, such as a whole industry group.
  const selectAllResults = async () => {
    setSelectingAll(true);
    try {
      const markets: IndicatorMarket[] = [];
      for (let page = 1; ; page++) {
        const res = await indicatorService.markets(filter, page, 100);
        markets.push(...(res.data || []));
        if (page >= res.pagination.total_pages) break;
      }
      onToggle(markets.map(toRef), true);
    } catch (e) {
      setFailure({ key, message: errorMessage(e, 'خطا در دریافت نمادها') });
    } finally {
      setSelectingAll(false);
    }
  };

  return (
    <div className="overflow-hidden rounded-md border border-border">
      <div className="grid gap-2 border-b border-border p-3 sm:grid-cols-[minmax(0,1fr)_minmax(0,14rem)]">
        <Input leadingIcon={<Search />} aria-label="جستجوی نماد" placeholder="نماد، نام شرکت یا insCode..." value={query} onChange={(e) => setQuery(e.target.value)} />
        <Select aria-label="گروه صنعت" placeholder="همه گروه‌های صنعت" value={group} onChange={(e) => setGroup(e.target.value)} options={groups.map((item) => ({ value: item.code, label: item.name || item.code }))} />
        <div className="flex flex-wrap items-center justify-between gap-2 sm:col-span-2">
          <label className="inline-flex cursor-pointer items-center gap-2 text-caption text-text-muted">
            <input type="checkbox" className="h-4 w-4 accent-action" checked={includeDelisted} onChange={(e) => setIncludeDelisted(e.target.checked)} />
            نمایش نمادهای لغو پذیرش‌شده
          </label>
          <span className="flex items-center gap-2 text-caption text-text-subtle">
            {current ? `${fa(total)} نماد` : ''}
            {multiple && total > 0 && total <= SELECT_ALL_LIMIT && (
              <Button size="sm" variant="secondary" isLoading={selectingAll} loadingLabel="در حال انتخاب..." onClick={selectAllResults}>
                انتخاب همه {fa(total)} نتیجه
              </Button>
            )}
          </span>
        </div>
      </div>

      <div ref={scrollRef} className="max-h-96 overflow-y-auto">
        <table className="w-full min-w-[560px] text-start">
          <thead className="sticky top-0 z-[1] bg-surface-muted">
            <tr className="border-b border-border">
              <th className="w-12 px-3 py-2.5">
                {multiple && (
                  <input
                    type="checkbox"
                    className="h-4 w-4 accent-action"
                    aria-label="انتخاب همه ردیف‌های نمایش‌داده‌شده"
                    checked={allLoadedSelected}
                    disabled={rows.length === 0}
                    onChange={() => onToggle(rows.map(toRef), !allLoadedSelected)}
                  />
                )}
              </th>
              {['نماد', 'نام', 'گروه صنعت', 'بازار'].map((title) => (
                <th key={title} className="px-3 py-2.5 text-start text-label text-text-muted">{title}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((market) => {
              const checked = selectedIds.includes(market.id);
              return (
                <tr
                  key={market.id}
                  onClick={(e) => {
                    if (!(e.target as HTMLElement).closest('input')) onToggle([toRef(market)], multiple ? !checked : true);
                  }}
                  className={`cursor-pointer border-b border-border last:border-b-0 ${checked ? 'bg-selected/10' : 'hover:bg-surface-muted'}`}
                >
                  <td className="px-3 py-2.5">
                    <input
                      type={multiple ? 'checkbox' : 'radio'}
                      name={multiple ? undefined : 'indicator-market'}
                      className="h-4 w-4 accent-action"
                      aria-label={`انتخاب ${market.label}`}
                      checked={checked}
                      onChange={() => onToggle([toRef(market)], multiple ? !checked : true)}
                    />
                  </td>
                  <td className="px-3 py-2.5 text-body font-medium text-text">
                    {market.label}
                    {market.delisted && <span className="ms-2"><Badge variant="warning">لغو پذیرش</Badge></span>}
                  </td>
                  <td className="max-w-[16rem] truncate px-3 py-2.5 text-body text-text-muted" title={market.name}>{market.name || '—'}</td>
                  <td className="px-3 py-2.5 text-caption text-text-subtle">{market.industry_group_name || '—'}</td>
                  <td className="px-3 py-2.5 text-caption text-text-subtle">{marketTypeLabel(market.market_type)}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
        {!current && !error && <p className="px-3 py-6 text-center text-body text-text-subtle">در حال دریافت...</p>}
        {current && rows.length === 0 && <p className="px-3 py-6 text-center text-body text-text-subtle">نمادی پیدا نشد.</p>}
        {error && (
          <div className="flex items-center justify-center gap-3 px-3 py-4">
            <p className="text-caption text-error">{error}</p>
            {current && <Button size="sm" variant="ghost" onClick={loadMore}>تلاش دوباره</Button>}
          </div>
        )}
        <div ref={sentinelRef} aria-hidden="true" className="h-px" />
        {hasMore && pendingPage === `${key}:${(current?.page ?? 0) + 1}` && (
          <p className="flex items-center justify-center gap-2 px-3 py-3 text-caption text-text-subtle">
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
            در حال دریافت ادامه فهرست...
          </p>
        )}
      </div>
    </div>
  );
}

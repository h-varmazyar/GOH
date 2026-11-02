import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { AlertTriangle, Ban, CheckSquare, Layers, Link2, Plus, RefreshCw, RotateCcw, Search, Trash2, X } from 'lucide-react';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import { Input } from '../../components/ui/Input';
import { Pagination } from '../../components/ui/Pagination';
import { Select } from '../../components/ui/Select';
import { Toggle } from '../../components/ui/Toggle';
import { AddMarketModal, LinkMarketModal } from '../../components/tse/MarketModals';
import { confirmDelisted, deleteMarket } from '../../components/tse/marketActions';
import { confirmDialog } from '../../components/ui/dialogs';
import { errorMessage, formatDate } from '../../components/tse/format';
import type { Pagination as PaginationInfo } from '../../services/airdropService';
import {
  kindOptions,
  marketTypeOptions,
  openIssueCount,
  tseMarketIssueService,
  tseMarketService,
  type IndustryGroup,
  type TseMarket,
  type TseMarketFilter,
  type TseMarketFlags,
} from '../../services/tseMarketService';

const PAGE_SIZE = 50;

const booleanOptions = (on: string, off: string) => [
  { value: 'true', label: on },
  { value: 'false', label: off },
];

const toBoolean = (value: string) => (value === '' ? undefined : value === 'true');

interface Filters {
  q: string;
  /** Lists the insCodes of one market. */
  marketId?: number;
  marketType: string;
  kind: string;
  group: string;
  delisted: string;
  dataCollection: string;
  evaluation: string;
}

const emptyFilters: Filters = { q: '', marketType: '', kind: '', group: '', delisted: '', dataCollection: '', evaluation: '' };

const toApiFilter = (filters: Filters): TseMarketFilter => ({
  q: filters.q.trim() || undefined,
  market_id: filters.marketId,
  market_type: filters.marketType || undefined,
  kind: filters.kind || undefined,
  group: filters.group || undefined,
  delisted: toBoolean(filters.delisted),
  data_collection: toBoolean(filters.dataCollection),
  evaluation: toBoolean(filters.evaluation),
});

export function MarketsPage() {
  const [filters, setFilters] = useState<Filters>(emptyFilters);
  const [query, setQuery] = useState('');
  const [page, setPage] = useState(1);
  const [markets, setMarkets] = useState<TseMarket[]>([]);
  const [pagination, setPagination] = useState<PaginationInfo | null>(null);
  const [groups, setGroups] = useState<IndustryGroup[]>([]);
  const [selected, setSelected] = useState<number[]>([]);
  const [reload, setReload] = useState(0);
  const [loadedKey, setLoadedKey] = useState('');
  const [isUpdating, setIsUpdating] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [showAdd, setShowAdd] = useState(false);
  const [linking, setLinking] = useState<TseMarket | null>(null);
  const [busyId, setBusyId] = useState<number | null>(null);
  const [marketLabel, setMarketLabel] = useState('');
  const [openIssues, setOpenIssues] = useState(0);

  // Debounce the search box.
  useEffect(() => {
    const timer = setTimeout(() => {
      setFilters((current) => (current.q === query ? current : { ...current, q: query }));
      setPage(1);
    }, 400);
    return () => clearTimeout(timer);
  }, [query]);

  const requestKey = JSON.stringify({ filters, page, reload });
  const isLoading = loadedKey !== requestKey;
  const load = () => setReload((value) => value + 1);

  useEffect(() => {
    let cancelled = false;
    tseMarketService
      .list(toApiFilter(filters), page, PAGE_SIZE)
      .then((res) => {
        if (cancelled) return;
        setMarkets(res.data || []);
        setPagination(res.pagination);
        setError('');
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(errorMessage(e, 'خطا در دریافت فهرست نمادها'));
      })
      .finally(() => {
        if (!cancelled) setLoadedKey(requestKey);
      });
    return () => {
      cancelled = true;
    };
  }, [filters, page, requestKey]);

  useEffect(() => {
    tseMarketService.industryGroups().then((res) => setGroups(res.data || [])).catch(() => setGroups([]));
  }, []);

  useEffect(() => {
    tseMarketIssueService.summary().then((summary) => setOpenIssues(openIssueCount(summary))).catch(() => setOpenIssues(0));
  }, [reload]);

  const changeFilter = (key: keyof Filters, value: string) => {
    setFilters((current) => ({ ...current, [key]: value }));
    setPage(1);
    setSelected([]);
  };

  const resetFilters = () => {
    setQuery('');
    setFilters(emptyFilters);
    setPage(1);
    setSelected([]);
  };

  const showMarket = (market: TseMarket) => {
    setQuery('');
    setFilters({ ...emptyFilters, marketId: market.market_id });
    setMarketLabel(market.market_label || market.symbol);
    setPage(1);
    setSelected([]);
  };

  const replaceRow = (updated: TseMarket) =>
    setMarkets((current) => current.map((row) => (row.id === updated.id ? updated : row)));

  const updateRow = async (market: TseMarket, flags: TseMarketFlags) => {
    setError('');
    try {
      replaceRow(await tseMarketService.setFlags(market.id, flags));
    } catch (e) {
      setError(errorMessage(e, 'خطا در تغییر وضعیت نماد'));
    }
  };

  const runRowAction = async (market: TseMarket, action: () => Promise<TseMarket>, done: string, failed: string) => {
    setBusyId(market.id);
    setError('');
    setNotice('');
    try {
      replaceRow(await action());
      setNotice(done);
    } catch (e) {
      setError(errorMessage(e, failed));
    } finally {
      setBusyId(null);
    }
  };

  const refreshRow = (market: TseMarket) =>
    runRowAction(market, () => tseMarketService.refresh(market.id), `اطلاعات ${market.symbol} (${market.ins_code}) از منابع به‌روز شد.`, 'خطا در به‌روزرسانی نماد');

  const toggleDelisted = async (market: TseMarket) => {
    if (!(await confirmDelisted(market))) return;
    runRowAction(market, () => tseMarketService.setDelisted(market.id, !market.delisted), `وضعیت لغو پذیرش ${market.symbol} تغییر کرد.`, 'خطا در تغییر وضعیت لغو پذیرش');
  };

  const removeRow = async (market: TseMarket) => {
    setBusyId(market.id);
    setError('');
    setNotice('');
    try {
      const message = await deleteMarket(market, false);
      if (message) {
        setNotice(message);
        load();
      }
    } catch (e) {
      setError(errorMessage(e, 'خطا در حذف نماد'));
    } finally {
      setBusyId(null);
    }
  };

  const bulkUpdate = async (flags: TseMarketFlags, scope: 'selected' | 'filtered') => {
    if (scope === 'filtered') {
      const total = pagination?.total_items ?? 0;
      const confirmed = await confirmDialog({
        title: 'تغییر گروهی',
        message: `این تغییر روی همه ${total.toLocaleString('fa-IR')} نماد نتیجه فیلتر اعمال شود؟`,
        confirmLabel: 'اعمال',
      });
      if (!confirmed) return;
    }
    setIsUpdating(true);
    setError('');
    setNotice('');
    try {
      const target = scope === 'selected' ? { ids: selected } : { filter: toApiFilter(filters) };
      const res = await tseMarketService.bulkSetFlags(flags, target);
      setNotice(`وضعیت ${res.updated.toLocaleString('fa-IR')} نماد تغییر کرد.`);
      setSelected([]);
      load();
    } catch (e) {
      setError(errorMessage(e, 'خطا در تغییر گروهی وضعیت'));
    } finally {
      setIsUpdating(false);
    }
  };

  const visibleIds = markets.map((item) => item.id);
  const allVisibleSelected = visibleIds.length > 0 && visibleIds.every((id) => selected.includes(id));
  const toggleSelection = (id: number) =>
    setSelected((current) => (current.includes(id) ? current.filter((item) => item !== id) : [...current, id]));
  const togglePage = () =>
    setSelected((current) =>
      allVisibleSelected ? current.filter((id) => !visibleIds.includes(id)) : [...new Set([...current, ...visibleIds])],
    );

  const totalItems = pagination?.total_items ?? 0;

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold text-surface-900 dark:text-white">مدیریت نمادها</h1>
          <p className="mt-1 text-sm text-surface-500">تعیین نمادهای فعال برای دریافت داده و ارزیابی</p>
        </div>
        <div className="flex items-center gap-2">
          {openIssues > 0 && (
            <Link
              to="/tse/markets/issues"
              className="inline-flex items-center gap-1 rounded-full bg-warning-400/20 px-2.5 py-1 text-xs font-medium text-warning-700 hover:bg-warning-400/30 dark:text-warning-400"
            >
              <AlertTriangle className="h-3.5 w-3.5" />
              {openIssues.toLocaleString('fa-IR')} مورد نیازمند بررسی
            </Link>
          )}
          <Badge variant="primary">{totalItems.toLocaleString('fa-IR')} نماد</Badge>
          <Button size="sm" onClick={() => setShowAdd(true)}>
            <Plus className="h-4 w-4" />
            افزودن نماد
          </Button>
        </div>
      </div>

      <Card padding="sm">
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <div className="relative md:col-span-2">
            <Search className="absolute right-3 top-2.5 h-5 w-5 text-surface-400" />
            <Input className="pr-10" placeholder="نماد، نام، insCode یا شناسه IR..." value={query} onChange={(e) => setQuery(e.target.value)} />
          </div>
          {filters.marketId !== undefined && (
            <div className="flex items-center gap-2 md:col-span-2 xl:col-span-4">
              <Badge variant="primary">سوابق مارکت {marketLabel}</Badge>
              <Button size="sm" variant="ghost" onClick={resetFilters}>
                <X className="h-4 w-4" />
                نمایش همه
              </Button>
            </div>
          )}
          <Select placeholder="همه بازارها" value={filters.marketType} onChange={(e) => changeFilter('marketType', e.target.value)} options={marketTypeOptions} />
          <Select placeholder="همه انواع" value={filters.kind} onChange={(e) => changeFilter('kind', e.target.value)} options={kindOptions} />
          <Select placeholder="همه گروه‌های صنعت" value={filters.group} onChange={(e) => changeFilter('group', e.target.value)} options={groups.map((group) => ({ value: group.code, label: group.name || group.code }))} />
          <Select placeholder="فعال و لغو پذیرش‌شده" value={filters.delisted} onChange={(e) => changeFilter('delisted', e.target.value)} options={booleanOptions('لغو پذیرش‌شده', 'فعال')} />
          <Select placeholder="دریافت داده: همه" value={filters.dataCollection} onChange={(e) => changeFilter('dataCollection', e.target.value)} options={booleanOptions('دریافت داده روشن', 'دریافت داده خاموش')} />
          <div className="flex gap-2">
            <Select className="flex-1" placeholder="ارزیابی: همه" value={filters.evaluation} onChange={(e) => changeFilter('evaluation', e.target.value)} options={booleanOptions('ارزیابی روشن', 'ارزیابی خاموش')} />
            <Button size="sm" variant="ghost" onClick={resetFilters}>پاک کردن</Button>
          </div>
        </div>
      </Card>

      <Card padding="sm" className={selected.length > 0 ? 'border-primary-200 bg-primary-50 dark:border-primary-800 dark:bg-primary-950/40' : ''}>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <span className="flex items-center gap-2 text-sm font-medium text-primary-700 dark:text-primary-300">
            <CheckSquare className="h-4 w-4" />
            {selected.length > 0
              ? `${selected.length.toLocaleString('fa-IR')} نماد انتخاب شده`
              : `تغییر گروهی روی همه ${totalItems.toLocaleString('fa-IR')} نتیجه فیلتر`}
          </span>
          <div className="flex flex-wrap gap-2">
            {(() => {
              const scope = selected.length > 0 ? 'selected' : 'filtered';
              const disabled = isUpdating || (scope === 'filtered' && totalItems === 0);
              return (
                <>
                  <Button size="sm" disabled={disabled} onClick={() => bulkUpdate({ data_collection_enabled: true }, scope)}>دریافت داده: روشن</Button>
                  <Button size="sm" variant="secondary" disabled={disabled} onClick={() => bulkUpdate({ data_collection_enabled: false }, scope)}>دریافت داده: خاموش</Button>
                  <Button size="sm" disabled={disabled} onClick={() => bulkUpdate({ evaluation_enabled: true }, scope)}>ارزیابی: روشن</Button>
                  <Button size="sm" variant="secondary" disabled={disabled} onClick={() => bulkUpdate({ evaluation_enabled: false }, scope)}>ارزیابی: خاموش</Button>
                  {selected.length > 0 && <Button size="sm" variant="ghost" onClick={() => setSelected([])}>لغو انتخاب</Button>}
                </>
              );
            })()}
          </div>
        </div>
      </Card>

      {error && <div className="rounded-lg bg-danger-50 p-3 text-sm text-danger-600 dark:bg-danger-950/40">{error}</div>}
      {notice && <div className="rounded-lg bg-success-50 p-3 text-sm text-success-700 dark:bg-success-950/40">{notice}</div>}

      <Card padding="none">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[1100px] text-right">
            <thead>
              <tr className="border-b border-surface-200 bg-surface-50 dark:border-surface-700 dark:bg-surface-800/50">
                <th className="px-3 py-3">
                  <input type="checkbox" aria-label="انتخاب همه صفحه" checked={allVisibleSelected} onChange={togglePage} className="h-4 w-4 accent-primary-600" />
                </th>
                {['نماد', 'نام', 'نوع', 'بازار', 'گروه صنعت', 'تاریخ درج', 'تاریخ عرضه', 'تاریخ پذیرش', 'دریافت داده', 'ارزیابی', 'عملیات'].map((title) => (
                  <th key={title} className="px-3 py-3 text-xs font-medium text-surface-500">{title}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {isLoading && markets.length === 0 && (
                <tr><td colSpan={12} className="px-3 py-8 text-center text-sm text-surface-500">در حال دریافت...</td></tr>
              )}
              {!isLoading && markets.length === 0 && (
                <tr><td colSpan={12} className="px-3 py-8 text-center text-sm text-surface-500">نمادی یافت نشد</td></tr>
              )}
              {markets.map((item) => (
                <tr key={item.id} className={`border-b border-surface-100 dark:border-surface-700 ${selected.includes(item.id) ? 'bg-primary-50/70 dark:bg-primary-950/30' : ''}`}>
                  <td className="px-3 py-2">
                    <input type="checkbox" aria-label={`انتخاب ${item.symbol}`} checked={selected.includes(item.id)} onChange={() => toggleSelection(item.id)} className="h-4 w-4 accent-primary-600" />
                  </td>
                  <td className="px-3 py-2">
                    <div className="flex items-center gap-2">
                      <Link to={`/tse/markets/${item.id}`} className="font-bold text-primary-700 hover:underline dark:text-primary-300">{item.symbol}</Link>
                      {item.delisted && <Badge variant="danger">لغو پذیرش</Badge>}
                      {item.manually_added && <Badge variant="warning">دستی</Badge>}
                    </div>
                    <div className="mt-0.5 text-[11px] text-surface-400" dir="ltr">{item.instrument_id || '—'} · {item.ins_code}</div>
                    <button
                      type="button"
                      onClick={() => showMarket(item)}
                      title="نمایش همه insCodeهای این مارکت"
                      className="mt-0.5 flex items-center gap-1 text-[11px] text-surface-500 hover:text-primary-600"
                    >
                      <Layers className="h-3 w-3" />
                      مارکت {item.market_label || '—'}
                      {item.market_members > 1 && ` · ${item.market_members.toLocaleString('fa-IR')} insCode`}
                    </button>
                  </td>
                  <td className="px-3 py-2 text-sm">{item.name}</td>
                  <td className="px-3 py-2"><Badge>{item.kind_label || '—'}</Badge></td>
                  <td className="px-3 py-2 text-xs text-surface-600 dark:text-surface-300">{item.market_type_label || '—'}</td>
                  <td className="px-3 py-2 text-xs text-surface-600 dark:text-surface-300">{item.industry_group_name || item.industry_group_code || '—'}</td>
                  <td className="px-3 py-2 text-xs">{formatDate(item.insert_date)}</td>
                  <td className="px-3 py-2 text-xs" title={item.release_date_source === 'price_history' ? 'از قدیمی‌ترین داده معاملاتی' : undefined}>
                    {formatDate(item.release_date)}
                    {item.release_date && item.release_date_source === 'price_history' && <span className="mr-1 text-surface-400">*</span>}
                  </td>
                  <td className="px-3 py-2 text-xs">{formatDate(item.acceptance_date)}</td>
                  <td className="px-3 py-2">
                    <Toggle enabled={item.data_collection_enabled} onChange={(value) => updateRow(item, { data_collection_enabled: value })} />
                  </td>
                  <td className="px-3 py-2">
                    <Toggle enabled={item.evaluation_enabled} onChange={(value) => updateRow(item, { evaluation_enabled: value })} />
                  </td>
                  <td className="px-3 py-2">
                    <div className="flex items-center gap-1">
                      <Button size="sm" variant="ghost" title="خواندن دوباره از منابع" disabled={busyId !== null} isLoading={busyId === item.id} onClick={() => refreshRow(item)}>
                        {busyId !== item.id && <RefreshCw className="h-4 w-4" />}
                      </Button>
                      <Button size="sm" variant="ghost" title={item.delisted ? 'برداشتن علامت لغو پذیرش' : 'علامت لغو پذیرش'} disabled={busyId !== null} onClick={() => toggleDelisted(item)}>
                        {item.delisted ? <RotateCcw className="h-4 w-4" /> : <Ban className="h-4 w-4" />}
                      </Button>
                      <Button size="sm" variant="ghost" title="انتقال به مارکت دیگر" disabled={busyId !== null} onClick={() => setLinking(item)}>
                        <Link2 className="h-4 w-4" />
                      </Button>
                      <Button size="sm" variant="ghost" title="حذف insCode" disabled={busyId !== null} onClick={() => removeRow(item)}>
                        <Trash2 className="h-4 w-4 text-danger-500" />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <Pagination page={page} totalPages={pagination?.total_pages || 1} totalItems={totalItems} onChange={setPage} />
      </Card>
      <p className="text-xs text-surface-500">* تاریخ عرضه از قدیمی‌ترین داده معاملاتی نماد برداشته شده است، چون منابع پذیرش آن را نداشتند.</p>

      {linking && (
        <LinkMarketModal
          market={linking}
          initialLabel={`${linking.symbol}-${linking.ins_code.slice(-4)}`}
          onClose={() => setLinking(null)}
          onLinked={(market) => {
            setLinking(null);
            setNotice(`insCode ${market.ins_code} به مارکت ${market.market_label} منتقل شد.`);
            load();
          }}
        />
      )}

      {showAdd && (
        <AddMarketModal
          onClose={() => setShowAdd(false)}
          onCreated={(market) => {
            setShowAdd(false);
            setNotice(`نماد ${market.symbol} (${market.market_type_label || market.market_type}) به مارکت ${market.market_label} افزوده شد.`);
            load();
          }}
        />
      )}
    </div>
  );
}

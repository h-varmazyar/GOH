import { useEffect, useState } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { Activity, Edit3, Plus, Search, Trash2 } from 'lucide-react';
import { Alert } from '../../components/ui/Alert';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import { IconButton } from '../../components/ui/IconButton';
import { Input } from '../../components/ui/Input';
import { Pagination } from '../../components/ui/Pagination';
import { Select } from '../../components/ui/Select';
import { Toggle } from '../../components/ui/Toggle';
import { confirmDialog } from '../../components/ui/dialogs';
import { errorMessage, formatDateTime } from '../../components/tse/format';
import type { Pagination as PaginationInfo } from '../../services/airdropService';
import {
  formatParams,
  indicatorService,
  scopeLabel,
  timeframeLabel,
  type IndicatorCatalog,
  type IndicatorFilter,
  type IndicatorScope,
  type TseIndicator,
} from '../../services/indicatorService';

const PAGE_SIZE = 20;

interface Filters {
  q: string;
  category: string;
  timeframe: string;
  scope: '' | IndicatorScope;
  enabled: string;
}

const emptyFilters: Filters = { q: '', category: '', timeframe: '', scope: '', enabled: '' };

const toApiFilter = (filters: Filters): IndicatorFilter => ({
  q: filters.q.trim() || undefined,
  category: filters.category || undefined,
  timeframe_id: filters.timeframe ? Number(filters.timeframe) : undefined,
  scope: filters.scope || undefined,
  enabled: filters.enabled === '' ? undefined : filters.enabled === 'true',
});

export function IndicatorsPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const [catalog, setCatalog] = useState<IndicatorCatalog | null>(null);
  const [filters, setFilters] = useState<Filters>(emptyFilters);
  const [query, setQuery] = useState('');
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<TseIndicator[]>([]);
  const [pagination, setPagination] = useState<PaginationInfo | null>(null);
  const [reload, setReload] = useState(0);
  const [loadedKey, setLoadedKey] = useState('');
  const [error, setError] = useState('');
  const [notice, setNotice] = useState<string>((location.state as { notice?: string } | null)?.notice ?? '');
  const [busyId, setBusyId] = useState<number | null>(null);

  // The notice of the add page is shown once.
  useEffect(() => {
    if (location.state) navigate(location.pathname, { replace: true, state: null });
  }, [location, navigate]);

  useEffect(() => {
    indicatorService.catalog().then(setCatalog).catch(() => setCatalog(null));
  }, []);

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
    indicatorService
      .list(toApiFilter(filters), page, PAGE_SIZE)
      .then((res) => {
        if (cancelled) return;
        setItems(res.data || []);
        setPagination(res.pagination);
        setError('');
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(errorMessage(e, 'خطا در دریافت فهرست اندیکاتورها'));
      })
      .finally(() => {
        if (!cancelled) setLoadedKey(requestKey);
      });
    return () => {
      cancelled = true;
    };
  }, [filters, page, requestKey]);

  const changeFilter = (key: keyof Filters, value: string) => {
    setFilters((current) => ({ ...current, [key]: value }));
    setPage(1);
  };

  const resetFilters = () => {
    setQuery('');
    setFilters(emptyFilters);
    setPage(1);
  };

  const definitionOf = (item: TseIndicator) => catalog?.indicators.find((definition) => definition.type === item.type);
  const categoryLabel = (key: string) => catalog?.categories.find((category) => category.key === key)?.label ?? '—';
  const replaceRow = (updated: TseIndicator) => setItems((current) => current.map((row) => (row.id === updated.id ? updated : row)));

  const toggleEnabled = async (item: TseIndicator, enabled: boolean) => {
    setBusyId(item.id);
    setError('');
    try {
      replaceRow(await indicatorService.setEnabled(item.id, enabled));
    } catch (e) {
      setError(errorMessage(e, 'خطا در تغییر وضعیت اندیکاتور'));
    } finally {
      setBusyId(null);
    }
  };

  const remove = async (item: TseIndicator) => {
    const confirmed = await confirmDialog({
      title: 'حذف اندیکاتور',
      message: `${item.title} (${formatParams(item.params, definitionOf(item))}) در بازه ${timeframeLabel(item.timeframe)} حذف شود؟`,
      warning: 'برای توقف موقت محاسبه، به جای حذف می‌توانید اندیکاتور را غیرفعال کنید.',
      confirmLabel: 'حذف',
      tone: 'danger',
    });
    if (!confirmed) return;
    setBusyId(item.id);
    setError('');
    setNotice('');
    try {
      await indicatorService.remove(item.id);
      setNotice(`${item.title} حذف شد.`);
      if (items.length === 1 && page > 1) setPage(page - 1);
      else load();
    } catch (e) {
      setError(errorMessage(e, 'خطا در حذف اندیکاتور'));
    } finally {
      setBusyId(null);
    }
  };

  const totalItems = pagination?.total_items ?? 0;
  const hasFilters = query !== '' || JSON.stringify(filters) !== JSON.stringify(emptyFilters);

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-h3 font-semibold text-text">اندیکاتورهای بازار</h1>
          <p className="mt-1 text-body text-text-subtle">اندیکاتورهایی که روی کندل‌های نمادهای بورس تهران محاسبه می‌شوند</p>
        </div>
        <div className="flex items-center gap-2">
          <Badge variant="primary">{totalItems.toLocaleString('fa-IR')} اندیکاتور</Badge>
          <Link
            to="/tse/indicators/new"
            className="inline-flex h-10 items-center gap-2 rounded-md bg-action px-4 text-body font-semibold text-on-action hover:bg-action-hover active:bg-action-pressed"
          >
            <Plus className="h-4 w-4" aria-hidden="true" />
            افزودن اندیکاتور
          </Link>
        </div>
      </div>

      <Card padding="sm">
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-[minmax(0,2fr)_repeat(4,minmax(0,1fr))_auto]">
          <div className="md:col-span-2 xl:col-span-1">
            <Input leadingIcon={<Search />} aria-label="جستجو" placeholder="نام یا کد اندیکاتور..." value={query} onChange={(e) => setQuery(e.target.value)} />
          </div>
          <Select aria-label="دسته" placeholder="همه دسته‌ها" value={filters.category} onChange={(e) => changeFilter('category', e.target.value)} options={(catalog?.categories ?? []).map((category) => ({ value: category.key, label: category.label }))} />
          <Select aria-label="بازه زمانی" placeholder="همه بازه‌ها" value={filters.timeframe} onChange={(e) => changeFilter('timeframe', e.target.value)} options={(catalog?.timeframes ?? []).map((timeframe) => ({ value: String(timeframe.id), label: timeframeLabel(timeframe.label) }))} />
          <Select aria-label="نمادها" placeholder="همه دامنه‌ها" value={filters.scope} onChange={(e) => changeFilter('scope', e.target.value)} options={(['ALL', 'GROUP', 'MARKET'] as const).map((scope) => ({ value: scope, label: scopeLabel[scope] }))} />
          <Select aria-label="وضعیت" placeholder="فعال و غیرفعال" value={filters.enabled} onChange={(e) => changeFilter('enabled', e.target.value)} options={[{ value: 'true', label: 'فعال' }, { value: 'false', label: 'غیرفعال' }]} />
          <Button size="sm" variant="ghost" disabled={!hasFilters} onClick={resetFilters}>پاک کردن</Button>
        </div>
      </Card>

      {error && <Alert tone="error">{error}</Alert>}
      {notice && <Alert tone="success" onDismiss={() => setNotice('')}>{notice}</Alert>}

      <Card padding="none">
        {!isLoading && items.length === 0 ? (
          <div className="flex flex-col items-center gap-3 px-4 py-12 text-center">
            <Activity className="h-8 w-8 text-text-subtle" aria-hidden="true" />
            {hasFilters ? (
              <>
                <p className="text-body text-text-muted">اندیکاتوری با این فیلترها پیدا نشد.</p>
                <Button size="sm" variant="secondary" onClick={resetFilters}>پاک کردن فیلترها</Button>
              </>
            ) : (
              <>
                <p className="text-body text-text-muted">هنوز اندیکاتوری افزوده نشده است.</p>
                <Button size="sm" onClick={() => navigate('/tse/indicators/new')}>
                  <Plus className="h-4 w-4" aria-hidden="true" />
                  افزودن اولین اندیکاتور
                </Button>
              </>
            )}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[900px] text-center [&_td]:align-middle">
              <thead>
                <tr className="border-b border-border bg-surface-muted">
                  {['اندیکاتور', 'دسته', 'بازه زمانی', 'نمادها', 'پارامترها', 'آخرین تغییر', 'فعال', 'عملیات'].map((title) => (
                    <th key={title} className="px-3 py-3 text-center text-label text-text-muted">{title}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {isLoading && items.length === 0 && (
                  <tr><td colSpan={8} className="px-3 py-8 text-center text-body text-text-subtle">در حال دریافت...</td></tr>
                )}
                {items.map((item) => (
                  <tr key={item.id} className={`border-b border-border ${isLoading ? 'opacity-60' : ''}`}>
                    <td className="px-3 py-3 text-start">
                      <p className="text-body font-medium text-text">{item.title}</p>
                      <code className="text-caption text-text-subtle" dir="ltr">{item.name}</code>
                    </td>
                    <td className="px-3 py-3"><Badge>{categoryLabel(item.category)}</Badge></td>
                    <td className="px-3 py-3 text-body text-text">{item.timeframe ? timeframeLabel(item.timeframe) : '—'}</td>
                    <td className="px-3 py-3 text-body text-text">
                      <MarketsCell item={item} />
                    </td>
                    <td className="max-w-[260px] px-3 py-3 text-caption text-text-subtle">{formatParams(item.params, definitionOf(item))}</td>
                    <td className="px-3 py-3 text-caption text-text-subtle">{formatDateTime(item.updated_at)}</td>
                    <td className="px-3 py-3">
                      <div className="flex justify-center">
                        <Toggle
                          aria-label={`فعال بودن ${item.title} در بازه ${timeframeLabel(item.timeframe)}`}
                          enabled={item.enabled}
                          disabled={busyId === item.id}
                          onChange={(enabled) => toggleEnabled(item, enabled)}
                        />
                      </div>
                    </td>
                    <td className="px-3 py-3">
                      <div className="flex justify-center gap-1">
                        <IconButton
                          title="ویرایش"
                          icon={<Edit3 className="h-4 w-4" />}
                          disabled={busyId === item.id || !definitionOf(item)}
                          onClick={() => navigate(`/tse/indicators/${item.id}/edit`)}
                        />
                        <IconButton title="حذف" tone="danger" icon={<Trash2 className="h-4 w-4" />} disabled={busyId === item.id} onClick={() => remove(item)} />
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        {pagination && pagination.total_pages > 1 && (
          <Pagination page={page} totalPages={pagination.total_pages} totalItems={pagination.total_items} onChange={setPage} />
        )}
      </Card>

    </div>
  );
}

/** The markets of an indicator: every market, one market, or the first few of a group. */
function MarketsCell({ item }: { item: TseIndicator }) {
  if (item.scope === 'ALL') return <span className="text-text-muted">همه نمادها</span>;
  const labels = item.markets.map((market) => market.label || `#${market.id}`);
  if (item.scope === 'MARKET') return <>{labels[0]}</>;
  const shown = labels.slice(0, 3).join('، ');
  return (
    <span title={labels.join('، ')}>
      <Badge variant="info">گروه {item.markets.length.toLocaleString('fa-IR')} نماد</Badge>
      <span className="mt-1 block text-caption text-text-subtle">
        {shown}
        {labels.length > 3 && ' …'}
      </span>
    </span>
  );
}

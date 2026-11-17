import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { AlertOctagon, AlertTriangle, CheckCircle2, GitMerge, Info, Plus, RotateCcw, ScanSearch, Search } from 'lucide-react';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import { confirmDialog, promptDialog } from '../../components/ui/dialogs';
import { Input } from '../../components/ui/Input';
import { Pagination } from '../../components/ui/Pagination';
import { Select } from '../../components/ui/Select';
import { InstrumentTable } from '../../components/tse/InstrumentTable';
import { AddMarketModal } from '../../components/tse/MarketModals';
import { errorMessage, formatDateTime } from '../../components/tse/format';
import type { Pagination as PaginationInfo } from '../../services/airdropService';
import {
  issueSeverityOptions,
  issueStatusOptions,
  openIssueCount,
  tseMarketIssueService,
  tseMarketService,
  type TseIssueFilter,
  type TseIssueSeverity,
  type TseIssueSummary,
  type TseMarket,
  type TseMarketIssue,
} from '../../services/tseMarketService';
import { Alert } from '../../components/ui/Alert';

const PAGE_SIZE = 10;

const severityStyle: Record<TseIssueSeverity, { variant: 'danger' | 'warning' | 'default'; label: string; icon: typeof Info }> = {
  ERROR: { variant: 'danger', label: 'خطا', icon: AlertOctagon },
  WARNING: { variant: 'warning', label: 'هشدار', icon: AlertTriangle },
  INFO: { variant: 'default', label: 'اطلاع', icon: Info },
};

const statusLabel = Object.fromEntries(issueStatusOptions.map((option) => [option.value, option.label]));

interface Filters {
  q: string;
  type: string;
  severity: string;
  status: string;
}

const toApiFilter = (filters: Filters): TseIssueFilter => ({
  q: filters.q.trim() || undefined,
  type: filters.type || undefined,
  severity: filters.severity || undefined,
  status: filters.status || undefined,
});

/** What each issue type asks the user to decide. */
const typeHelp: Record<string, string> = {
  NAME_MISMATCH: 'insCodeهای یک مارکت با نام متفاوت: اگر شرکت دیگری‌اند منتقلشان کنید، وگرنه تأیید کنید.',
  SAME_NAME: 'مارکت‌های با نام شرکت یکسان: اگر یک شرکت‌اند ادغامشان کنید، وگرنه تأیید کنید.',
  MULTIPLE_ACTIVE: 'چند insCode فعال در یک مارکت: insCode بی‌معامله را لغو پذیرش کنید یا دریافت داده‌اش را خاموش کنید.',
  PRICE_OVERLAP: 'قیمت هم‌زمان چند insCode در یک روز: دریافت داده insCode نادرست را خاموش کنید.',
  SKIPPED: 'insCode از TSETMC خوانده نشد: دوباره اضافه کنید یا تأیید کنید.',
  IFB_BOARD_UNMATCHED: 'نماد فرابورس بدون insCode فعال: با insCode درست اضافه کنید یا تأیید کنید.',
};

export function MarketIssuesPage() {
  const [searchParams] = useSearchParams();
  const [filters, setFilters] = useState<Filters>({ q: searchParams.get('q') || '', type: '', severity: '', status: 'OPEN' });
  const [query, setQuery] = useState(filters.q);
  const [page, setPage] = useState(1);
  const [issues, setIssues] = useState<TseMarketIssue[]>([]);
  const [pagination, setPagination] = useState<PaginationInfo | null>(null);
  const [summary, setSummary] = useState<TseIssueSummary | null>(null);
  const [selected, setSelected] = useState<number[]>([]);
  const [reload, setReload] = useState(0);
  const [loadedKey, setLoadedKey] = useState('');
  const [isDetecting, setIsDetecting] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [adding, setAdding] = useState<{ symbol: string; insCode: string } | null>(null);

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
    Promise.all([tseMarketIssueService.list(toApiFilter(filters), page, PAGE_SIZE), tseMarketIssueService.summary()])
      .then(([res, counts]) => {
        if (cancelled) return;
        setIssues(res.data || []);
        setPagination(res.pagination);
        setSummary(counts);
        setError('');
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(errorMessage(e, 'خطا در دریافت موارد بررسی'));
      })
      .finally(() => {
        if (!cancelled) setLoadedKey(requestKey);
      });
    return () => {
      cancelled = true;
    };
  }, [filters, page, requestKey]);

  /** Checks the markets again after a change, so that settled issues are resolved. */
  const detectAndReload = async (message?: string) => {
    setIsDetecting(true);
    try {
      const result = await tseMarketIssueService.detect();
      if (message === undefined) {
        setNotice(`بررسی انجام شد: ${result.found.toLocaleString('fa-IR')} مورد یافت شد و ${result.resolved.toLocaleString('fa-IR')} مورد رفع شد.`);
      }
    } catch (e) {
      setError(errorMessage(e, 'خطا در بررسی دوباره نمادها'));
    } finally {
      setIsDetecting(false);
      setReload((value) => value + 1);
    }
  };

  const changed = (message: string) => {
    setError('');
    setNotice(message);
    detectAndReload(message);
  };

  const changeFilter = (key: keyof Filters, value: string) => {
    setFilters((current) => ({ ...current, [key]: value }));
    setPage(1);
    setSelected([]);
  };

  const setStatus = async (status: 'OPEN' | 'IGNORED', target: { ids: number[] } | { filter: TseIssueFilter }, note?: string) => {
    setError('');
    try {
      const res = await tseMarketIssueService.setStatus(status, target, note);
      setNotice(`${res.updated.toLocaleString('fa-IR')} مورد ${status === 'IGNORED' ? 'تأیید شد' : 'دوباره باز شد'}.`);
      setSelected([]);
      setReload((value) => value + 1);
    } catch (e) {
      setError(errorMessage(e, 'خطا در تغییر وضعیت مورد'));
    }
  };

  const ignore = async (issue: TseMarketIssue) => {
    const note = await promptDialog({
      title: 'تأیید مورد',
      message: `«${issue.type_label}» برای ${issue.symbol || issue.name} تأیید شود؟ تا وقتی دوباره بازش نکنید در بررسی‌های بعدی هم تأییدشده می‌ماند.`,
      label: 'یادداشت تصمیم (اختیاری)',
      initialValue: issue.note || '',
      confirmLabel: 'تأیید',
    });
    if (note === null) return;
    setStatus('IGNORED', { ids: [issue.id] }, note);
  };

  const bulkIgnore = async (scope: 'selected' | 'filtered') => {
    if (scope === 'filtered') {
      const total = pagination?.total_items ?? 0;
      const confirmed = await confirmDialog({
        title: 'تأیید گروهی',
        message: `همه ${total.toLocaleString('fa-IR')} مورد نتیجه فیلتر تأیید شوند؟`,
        confirmLabel: 'تأیید همه',
      });
      if (!confirmed) return;
    }
    setStatus('IGNORED', scope === 'selected' ? { ids: selected } : { filter: toApiFilter(filters) });
  };

  const merge = async (issue: TseMarketIssue, target: TseMarket) => {
    const movers = issue.instruments.filter((item) => item.market_id !== target.market_id);
    if (movers.length === 0) return;
    const names = movers.map((item) => `${item.symbol} (${item.ins_code})`).join('، ');
    const confirmed = await confirmDialog({
      title: 'ادغام مارکت‌ها',
      message: `insCodeهای ${names} به مارکت ${target.market_label} منتقل شوند؟`,
      warning: 'مارکت‌هایی که خالی می‌شوند حذف می‌شوند. insCodeهایی که قیمتشان ذخیره شده منتقل نمی‌شوند.',
      confirmLabel: 'ادغام',
    });
    if (!confirmed) return;
    setError('');
    const failed: string[] = [];
    for (const item of movers) {
      try {
        await tseMarketService.link(item.id, target.market_label);
      } catch (e) {
        failed.push(`${item.symbol} (${item.ins_code}): ${errorMessage(e, 'خطا')}`);
      }
    }
    if (failed.length > 0) setError(`انتقال این insCodeها انجام نشد: ${failed.join(' | ')}`);
    changed(`${(movers.length - failed.length).toLocaleString('fa-IR')} insCode در مارکت ${target.market_label} ادغام شد.`);
  };

  const totalItems = pagination?.total_items ?? 0;
  const visibleIds = issues.filter((issue) => issue.status !== 'RESOLVED').map((issue) => issue.id);
  const allVisibleSelected = visibleIds.length > 0 && visibleIds.every((id) => selected.includes(id));
  const countBy = (severity: TseIssueSeverity) =>
    (summary?.counts || []).filter((count) => count.status === 'OPEN' && count.severity === severity).reduce((total, count) => total + count.count, 0);
  const typeOptions = (summary?.types || []).map((item) => {
    const open = openIssueCount(summary, item.type);
    return { value: item.type, label: open > 0 ? `${item.label} (${open.toLocaleString('fa-IR')})` : item.label };
  });

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-h3 font-semibold text-text">بررسی نمادها</h1>
          <p className="mt-1 text-body text-text-subtle">
            تداخل‌ها و ناهمخوانی‌های فهرست نمادها که تصمیمشان با شماست. تصمیم «تأیید» در بررسی‌های بعدی حفظ می‌شود.
          </p>
        </div>
        <Button size="sm" isLoading={isDetecting} onClick={() => detectAndReload()}>
          {!isDetecting && <ScanSearch className="h-4 w-4" />}
          بررسی دوباره
        </Button>
      </div>

      <div className="grid gap-3 sm:grid-cols-3">
        {(['ERROR', 'WARNING', 'INFO'] as TseIssueSeverity[]).map((severity) => {
          const style = severityStyle[severity];
          const Icon = style.icon;
          const active = filters.severity === severity;
          return (
            <button
              key={severity}
              type="button"
              onClick={() => changeFilter('severity', active ? '' : severity)}
              className={`flex items-center justify-between rounded-lg border p-4 text-start transition-colors ${
                active ? 'border-selected/70 bg-selected/10' : 'border-border bg-surface hover:bg-surface-muted'
              }`}
            >
              <span className="flex items-center gap-2 text-body text-text-muted">
                <Icon className="h-5 w-5" />
                {style.label}‌های باز
              </span>
              <span className="text-h2 font-semibold text-text">{countBy(severity).toLocaleString('fa-IR')}</span>
            </button>
          );
        })}
      </div>

      <Card padding="sm">
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <Input leadingIcon={<Search />} aria-label="جستجو" placeholder="نماد، نام یا insCode..." value={query} onChange={(e) => setQuery(e.target.value)} />
          <Select placeholder="همه انواع مورد" value={filters.type} onChange={(e) => changeFilter('type', e.target.value)} options={typeOptions} />
          <Select placeholder="همه سطوح" value={filters.severity} onChange={(e) => changeFilter('severity', e.target.value)} options={issueSeverityOptions} />
          <Select placeholder="همه وضعیت‌ها" value={filters.status} onChange={(e) => changeFilter('status', e.target.value)} options={issueStatusOptions} />
        </div>
        {filters.type && typeHelp[filters.type] && <p className="mt-3 text-caption text-text-subtle">{typeHelp[filters.type]}</p>}
      </Card>

      <Card padding="sm">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <label className="flex items-center gap-2 text-body text-text-muted">
            <input
              type="checkbox"
              className="h-4 w-4 accent-action"
              checked={allVisibleSelected}
              onChange={() =>
                setSelected((current) =>
                  allVisibleSelected ? current.filter((id) => !visibleIds.includes(id)) : [...new Set([...current, ...visibleIds])],
                )
              }
            />
            {selected.length > 0 ? `${selected.length.toLocaleString('fa-IR')} مورد انتخاب شده` : `${totalItems.toLocaleString('fa-IR')} مورد`}
          </label>
          <div className="flex flex-wrap gap-2">
            {selected.length > 0 ? (
              <>
                <Button size="sm" onClick={() => bulkIgnore('selected')}>تأیید انتخاب‌شده‌ها</Button>
                <Button size="sm" variant="secondary" onClick={() => setStatus('OPEN', { ids: selected })}>بازگشایی انتخاب‌شده‌ها</Button>
                <Button size="sm" variant="ghost" onClick={() => setSelected([])}>لغو انتخاب</Button>
              </>
            ) : (
              <Button size="sm" variant="secondary" disabled={totalItems === 0 || filters.status === 'RESOLVED'} onClick={() => bulkIgnore('filtered')}>
                تأیید همه نتایج فیلتر
              </Button>
            )}
          </div>
        </div>
      </Card>

      {error && <Alert tone="error">{error}</Alert>}
      {notice && <Alert tone="success">{notice}</Alert>}

      {isLoading && issues.length === 0 && <Card><p className="text-center text-body text-text-subtle">در حال دریافت...</p></Card>}
      {!isLoading && issues.length === 0 && (
        <Card>
          <p className="flex items-center justify-center gap-2 text-body text-text-subtle">
            <CheckCircle2 className="h-5 w-5 text-success" />
            موردی یافت نشد
          </p>
        </Card>
      )}

      {issues.map((issue) => {
        const style = severityStyle[issue.severity] || severityStyle.INFO;
        const markets = [...new Map(issue.instruments.map((item) => [item.market_id, item])).values()];
        const storedInsCodes = issue.instruments.map((item) => item.ins_code);
        const missing = issue.ins_codes.filter((insCode) => !storedInsCodes.includes(insCode));
        return (
          <Card key={issue.id} padding="sm" className={selected.includes(issue.id) ? 'border-selected/50' : ''}>
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div className="flex items-start gap-3">
                {issue.status !== 'RESOLVED' && (
                  <input
                    type="checkbox"
                    aria-label={`انتخاب مورد ${issue.symbol}`}
                    className="mt-1 h-4 w-4 accent-action"
                    checked={selected.includes(issue.id)}
                    onChange={() =>
                      setSelected((current) => (current.includes(issue.id) ? current.filter((id) => id !== issue.id) : [...current, issue.id]))
                    }
                  />
                )}
                <div>
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge variant={style.variant}>{style.label}</Badge>
                    <span className="font-semibold text-text">{issue.type_label}</span>
                    <span className="font-bold text-link">{issue.symbol}</span>
                    <Badge variant={issue.status === 'OPEN' ? 'primary' : issue.status === 'IGNORED' ? 'success' : 'default'}>
                      {statusLabel[issue.status] || issue.status}
                    </Badge>
                  </div>
                  <p className="mt-2 text-body leading-7 text-text-muted">{issue.detail}</p>
                  {issue.note && <p className="mt-1 text-caption text-text-subtle">یادداشت: {issue.note}</p>}
                  <p className="mt-1 text-[11px] text-text-subtle">
                    آخرین شناسایی: {formatDateTime(issue.detected_at)}
                    {issue.resolved_at && ` · رفع: ${formatDateTime(issue.resolved_at)}`}
                  </p>
                </div>
              </div>
              <div className="flex flex-wrap gap-2">
                {issue.type === 'SAME_NAME' && issue.status !== 'RESOLVED' &&
                  markets.map((target) => (
                    <Button key={target.market_id} size="sm" variant="secondary" onClick={() => merge(issue, target)}>
                      <GitMerge className="h-4 w-4" />
                      ادغام در {target.market_label}
                    </Button>
                  ))}
                {(issue.type === 'SKIPPED' || issue.type === 'IFB_BOARD_UNMATCHED') && issue.status !== 'RESOLVED' && (
                  <Button size="sm" variant="secondary" onClick={() => setAdding({ symbol: issue.symbol, insCode: missing[0] || '' })}>
                    <Plus className="h-4 w-4" />
                    افزودن نماد
                  </Button>
                )}
                {issue.status === 'OPEN' && <Button size="sm" onClick={() => ignore(issue)}>تأیید</Button>}
                {issue.status === 'IGNORED' && (
                  <Button size="sm" variant="ghost" onClick={() => setStatus('OPEN', { ids: [issue.id] })}>
                    <RotateCcw className="h-4 w-4" />
                    بازگشایی
                  </Button>
                )}
              </div>
            </div>
            {issue.instruments.length > 0 && (
              <div className="mt-3 border-t border-border pt-3">
                <InstrumentTable
                  instruments={issue.instruments}
                  highlight={issue.ins_codes}
                  onChanged={(_, message) => changed(message)}
                  onError={setError}
                />
              </div>
            )}
            {missing.length > 0 && (
              <p className="mt-2 text-caption text-text-subtle" dir="rtl">
                insCode ذخیره‌نشده: <span dir="ltr">{missing.join('، ')}</span>
              </p>
            )}
          </Card>
        );
      })}

      {totalItems > 0 && (
        <Card padding="none">
          <Pagination page={page} totalPages={pagination?.total_pages || 1} totalItems={totalItems} onChange={setPage} />
        </Card>
      )}

      {adding && (
        <AddMarketModal
          initialSymbol={adding.symbol}
          initialInsCode={adding.insCode}
          onClose={() => setAdding(null)}
          onCreated={(market) => {
            setAdding(null);
            changed(`نماد ${market.symbol} به مارکت ${market.market_label} افزوده شد.`);
          }}
        />
      )}
    </div>
  );
}

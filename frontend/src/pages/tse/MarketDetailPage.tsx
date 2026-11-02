import { useEffect, useState, type ReactNode } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { AlertTriangle, ArrowRight, Trash2 } from 'lucide-react';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import { InstrumentTable } from '../../components/tse/InstrumentTable';
import { deleteMarket } from '../../components/tse/marketActions';
import { errorMessage, formatDate, formatDateTime, formatNumber } from '../../components/tse/format';
import {
  issueStatusOptions,
  tseMarketIssueService,
  tseMarketService,
  type TseMarket,
  type TseMarketIssue,
} from '../../services/tseMarketService';

const statusLabel = Object.fromEntries(issueStatusOptions.map((option) => [option.value, option.label]));

const releaseSource: Record<string, string> = {
  listing: 'منابع پذیرش',
  price_history: 'قدیمی‌ترین داده معاملاتی',
};

function Field({ label, children, ltr }: { label: string; children: ReactNode; ltr?: boolean }) {
  return (
    <div className="flex items-start justify-between gap-3 border-b border-surface-100 py-2 last:border-0 dark:border-surface-800">
      <span className="text-xs text-surface-500">{label}</span>
      <span className="text-left text-sm text-surface-900 dark:text-surface-100" dir={ltr ? 'ltr' : undefined}>
        {children === '' || children === null || children === undefined ? '—' : children}
      </span>
    </div>
  );
}

export function MarketDetailPage() {
  const { marketId } = useParams<{ marketId: string }>();
  const id = Number(marketId);
  const navigate = useNavigate();
  const [deleting, setDeleting] = useState(false);
  const [market, setMarket] = useState<TseMarket | null>(null);
  const [members, setMembers] = useState<TseMarket[]>([]);
  const [issues, setIssues] = useState<TseMarketIssue[]>([]);
  const [reload, setReload] = useState(0);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  useEffect(() => {
    let cancelled = false;
    tseMarketService
      .get(id)
      .then(async (item) => {
        const [memberRes, issueRes] = await Promise.all([
          tseMarketService.list({ market_id: item.market_id }, 1, 100),
          tseMarketIssueService.list({ ins_code: item.ins_code, market_id: item.market_id }, 1, 50),
        ]);
        if (cancelled) return;
        setMarket(item);
        setMembers(memberRes.data || []);
        setIssues(issueRes.data || []);
        setError('');
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(errorMessage(e, 'خطا در دریافت نماد'));
      });
    return () => {
      cancelled = true;
    };
  }, [id, reload]);

  const changed = (_: TseMarket, message: string) => {
    setError('');
    setNotice(message);
    setReload((value) => value + 1);
  };

  // After the shown insCode is deleted, another insCode of its market is shown.
  const deleted = (item: TseMarket, message: string) => {
    const next = members.find((member) => member.id !== item.id);
    if (item.id !== id) return changed(item, message);
    if (next) navigate(`/tse/markets/${next.id}`);
    else navigate('/tse/markets');
    setNotice(message);
  };

  const removeMarket = async (item: TseMarket) => {
    setDeleting(true);
    try {
      if (await deleteMarket(item, true)) navigate('/tse/markets');
    } catch (e) {
      setError(errorMessage(e, 'خطا در حذف نماد'));
    } finally {
      setDeleting(false);
    }
  };

  if (!market) {
    return (
      <div className="space-y-4">
        <Link to="/tse/markets" className="inline-flex items-center gap-1 text-sm text-primary-600 hover:underline">
          <ArrowRight className="h-4 w-4" />
          بازگشت به فهرست نمادها
        </Link>
        <Card>
          <p className="text-center text-sm text-surface-500">{error || 'در حال دریافت...'}</p>
        </Card>
      </div>
    );
  }

  const openIssues = issues.filter((issue) => issue.status === 'OPEN');

  return (
    <div className="space-y-4">
      <Link to="/tse/markets" className="inline-flex items-center gap-1 text-sm text-primary-600 hover:underline">
        <ArrowRight className="h-4 w-4" />
        بازگشت به فهرست نمادها
      </Link>

      <div className="flex flex-wrap items-center gap-2">
        <h1 className="text-2xl font-bold text-surface-900 dark:text-white">{market.symbol}</h1>
        <span className="text-surface-600 dark:text-surface-300">{market.name}</span>
        {market.kind_label && <Badge>{market.kind_label}</Badge>}
        {market.market_type_label && <Badge variant="primary">{market.market_type_label}</Badge>}
        {market.delisted && <Badge variant="danger">لغو پذیرش</Badge>}
        {market.manually_added && <Badge variant="warning">دستی</Badge>}
        <Button size="sm" variant="danger" className="mr-auto" isLoading={deleting} onClick={() => removeMarket(market)}>
          <Trash2 className="h-4 w-4" />
          حذف کامل نماد
        </Button>
      </div>

      {error && <div className="rounded-lg bg-danger-50 p-3 text-sm text-danger-600 dark:bg-danger-950/40">{error}</div>}
      {notice && <div className="rounded-lg bg-success-50 p-3 text-sm text-success-700 dark:bg-success-950/40">{notice}</div>}

      {openIssues.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 rounded-lg bg-warning-400/15 p-3 text-sm text-warning-700 dark:text-warning-400">
          <AlertTriangle className="h-4 w-4" />
          {openIssues.length.toLocaleString('fa-IR')} مورد نیازمند تصمیم درباره این نماد یا مارکت آن هست.
          <Link to={`/tse/markets/issues?q=${encodeURIComponent(market.ins_code)}`} className="font-medium underline">
            مشاهده در صفحه بررسی
          </Link>
        </div>
      )}

      <div className="grid gap-4 lg:grid-cols-3">
        <Card padding="sm">
          <h2 className="mb-2 font-semibold text-surface-900 dark:text-white">شناسه‌ها</h2>
          <Field label="insCode" ltr>{market.ins_code}</Field>
          <Field label="شناسه نماد (IR)" ltr>{market.instrument_id}</Field>
          <Field label="شناسه شرکت" ltr>{market.company_isin}</Field>
          <Field label="نام انگلیسی" ltr>{market.english_name}</Field>
          <Field label="کد نوع TSETMC" ltr>{market.instrument_type || ''}</Field>
          <Field label="کد تابلو" ltr>{market.board_code}</Field>
        </Card>
        <Card padding="sm">
          <h2 className="mb-2 font-semibold text-surface-900 dark:text-white">طبقه‌بندی و تاریخ‌ها</h2>
          <Field label="گروه صنعت">{market.industry_group_name || market.industry_group_code}</Field>
          <Field label="زیرگروه" ltr>{market.industry_sub_code}</Field>
          <Field label="تاریخ درج">{formatDate(market.insert_date)}</Field>
          <Field label="تاریخ عرضه">
            {formatDate(market.release_date)}
            {market.release_date && market.release_date_source && (
              <span className="mr-1 text-xs text-surface-400">({releaseSource[market.release_date_source] || market.release_date_source})</span>
            )}
          </Field>
          <Field label="تاریخ پذیرش">{formatDate(market.acceptance_date)}</Field>
        </Card>
        <Card padding="sm">
          <h2 className="mb-2 font-semibold text-surface-900 dark:text-white">اطلاعات TSETMC</h2>
          <Field label="تعداد سهام">{formatNumber(market.total_shares)}</Field>
          <Field label="حجم مبنا">{formatNumber(market.base_volume)}</Field>
          <Field label="NAV">{formatNumber(market.nav)}</Field>
          <Field label="EPS">{formatNumber(market.eps)}</Field>
          <Field label="P/E گروه">{formatNumber(market.sector_pe, 2)}</Field>
          <Field label="PSR">{formatNumber(market.psr, 2)}</Field>
          <Field label="به‌روزرسانی">{formatDateTime(market.info_updated_at)}</Field>
        </Card>
      </div>

      <Card padding="sm">
        <div className="mb-3">
          <h2 className="font-semibold text-surface-900 dark:text-white">سوابق مارکت {market.market_label}</h2>
          <p className="text-xs text-surface-500">
            همه insCodeهایی که در این مارکت‌اند، مثلاً insCodeهای قبلی نماد پیش از انتقال بین بازارها. هر insCode قیمت‌های جداگانه خودش را دارد.
          </p>
        </div>
        <InstrumentTable instruments={members} highlight={[market.ins_code]} onChanged={changed} onDeleted={deleted} onError={setError} />
      </Card>

      <Card padding="sm">
        <h2 className="mb-3 font-semibold text-surface-900 dark:text-white">موارد بررسی</h2>
        {issues.length === 0 ? (
          <p className="text-sm text-surface-500">موردی برای این نماد یا مارکت آن ثبت نشده است.</p>
        ) : (
          <div className="space-y-2">
            {issues.map((issue) => (
              <div key={issue.id} className="rounded-lg border border-surface-100 p-3 dark:border-surface-800">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant={issue.severity === 'ERROR' ? 'danger' : issue.severity === 'WARNING' ? 'warning' : 'default'}>{issue.type_label}</Badge>
                  <Badge variant={issue.status === 'OPEN' ? 'primary' : issue.status === 'IGNORED' ? 'success' : 'default'}>
                    {statusLabel[issue.status] || issue.status}
                  </Badge>
                  <span className="text-[11px] text-surface-400">{formatDateTime(issue.detected_at)}</span>
                </div>
                <p className="mt-1 text-sm leading-7 text-surface-700 dark:text-surface-300">{issue.detail}</p>
                {issue.note && <p className="text-xs text-surface-500">یادداشت: {issue.note}</p>}
              </div>
            ))}
            <Link to={`/tse/markets/issues?q=${encodeURIComponent(market.ins_code)}`} className="text-sm text-primary-600 hover:underline">
              تصمیم‌گیری در صفحه بررسی نمادها
            </Link>
          </div>
        )}
      </Card>
    </div>
  );
}

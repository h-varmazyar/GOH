import { Fragment, useEffect, useState, type ReactNode } from 'react';
import { Eye, ChevronLeft, ChevronRight, ChevronDown, X, RefreshCw } from 'lucide-react';
import { Card } from '../../components/ui/Card';
import { Button } from '../../components/ui/Button';
import { Badge } from '../../components/ui/Badge';
import { Input } from '../../components/ui/Input';
import { airdropService, type AirdropGift, type AirdropGiftAllocation, type Pagination } from '../../services/airdropService';
import { Alert } from '../../components/ui/Alert';

const PAGE_SIZE = 20;

/** Shamsi date and time, as fa-IR formats it. */
const formatDateTime = (value: string) => (value ? new Date(value).toLocaleString('fa-IR') : '—');

export function AirdropGiftsList() {
  const [gifts, setGifts] = useState<AirdropGift[]>([]);
  const [pagination, setPagination] = useState<Pagination | null>(null);
  const [page, setPage] = useState(1);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [selectedGift, setSelectedGift] = useState<AirdropGift | null>(null);
  const [retryGiftId, setRetryGiftId] = useState<number | null>(null);
  const [manualGiftCode, setManualGiftCode] = useState<string>('');
  const [isRetrying, setIsRetrying] = useState(false);
  const [retryError, setRetryError] = useState<string>('');
  const [retrySuccess, setRetrySuccess] = useState<string>('');
  const [expandedGiftIds, setExpandedGiftIds] = useState<Set<number>>(new Set());
  const [retryingAllocationId, setRetryingAllocationId] = useState<number | null>(null);
  const [allocationError, setAllocationError] = useState('');

  useEffect(() => {
    fetchGifts();
  }, [page]);

  const fetchGifts = async () => {
    setIsLoading(true);
    setError('');
    try {
      const res = await airdropService.getGifts(page, PAGE_SIZE);
      setGifts(res.data || []);
      setPagination(res.pagination);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'خطا در دریافت لیست');
    } finally {
      setIsLoading(false);
    }
  };

  const handleRetry = async (gift: AirdropGift) => {
    if (isRetrying) return;
    
    setIsRetrying(true);
    setRetryError('');
    setRetrySuccess('');
    
    try {
      const giftCode = manualGiftCode.trim() || undefined;
      const res = await airdropService.retryGift(gift.ID, giftCode);
      
      setRetrySuccess('درخواست تست مجدد ارسال شد');
      
      // آپدیت gift در لیست
      setGifts(prev => prev.map(g => (g.ID === gift.ID ? res : g)));
      
      // آپدیت selected gift اگر باز است
      if (selectedGift?.ID === gift.ID) {
        setSelectedGift(res);
      }
      
      // پاک کردن input
      setManualGiftCode('');
      
      // بستن modal retry بعد از 2 ثانیه
      setTimeout(() => {
        setRetryGiftId(null);
      }, 2000);
      
    } catch (e: unknown) {
      setRetryError(e instanceof Error ? e.message : 'خطا در ارسال درخواست تست مجدد');
    } finally {
      setIsRetrying(false);
    }
  };

  const handleRetryAllocation = async (allocation: AirdropGiftAllocation) => {
    if (retryingAllocationId !== null) return;

    setRetryingAllocationId(allocation.ID);
    setAllocationError('');
    try {
      const res = await airdropService.retryAllocation(allocation.ID);
      const updated: AirdropGiftAllocation = { ...allocation, Currency: res.Currency, Status: res.Status, ErrorMessage: res.ErrorMessage || '' };
      setGifts(prev => prev.map(g => (g.ID === allocation.GiftID ? replaceAllocation(g, updated) : g)));
      if (res.Status !== 'received') {
        setAllocationError(res.ErrorMessage || 'تلاش مجدد ناموفق بود');
      } else {
        // وضعیت gift ممکن است در سرور به «دریافت شده» تغییر کرده باشد
        const gift = await airdropService.getGiftDetail(allocation.GiftID);
        setGifts(prev => prev.map(g => (g.ID === gift.ID ? gift : g)));
      }
    } catch (e: unknown) {
      setAllocationError(e instanceof Error ? e.message : 'خطا در ارسال درخواست تلاش مجدد');
    } finally {
      setRetryingAllocationId(null);
    }
  };

  const toggleExpanded = (giftId: number) => {
    setExpandedGiftIds(prev => {
      const next = new Set(prev);
      if (next.has(giftId)) next.delete(giftId);
      else next.add(giftId);
      return next;
    });
  };

  const rowOffset = ((pagination?.page ?? page) - 1) * (pagination?.page_size ?? PAGE_SIZE);

  const getStatusBadge = (status: string) => {
    switch (status.toLowerCase()) {
      case 'success':
      case 'completed':
      case 'claimed':
      case 'received':
        return <Badge variant="success">موفق</Badge>;
      case 'partialy_received':
        return <Badge variant="warning">دریافت ناقص</Badge>;
      case 'initiated':
        return <Badge variant="warning">در حال پردازش</Badge>;
      case 'failed':
      case 'error':
        return <Badge variant="danger">ناموفق</Badge>;
      case 'pending':
      case 'processing':
        return <Badge variant="warning">در حال پردازش</Badge>;
      case 'route_not_found':
        return <Badge variant="danger">مسیر یافت نشد</Badge>;
      default:
        return <Badge>{status}</Badge>;
    }
  };

  // تابع کمکی برای بررسی وضعیت موفق
  const isSuccessfulStatus = (status: string) => {
    const successStatuses = ['success', 'completed', 'claimed', 'received'];
    return successStatuses.includes(status.toLowerCase());
  };

  return (
    <>
      <Card>
        {error && (
          <Alert tone="error" className="mb-4">{error}</Alert>
        )}

        {allocationError && (
          <Alert tone="error" className="mb-4" onDismiss={() => setAllocationError('')}>
            {allocationError}
          </Alert>
        )}

        {isLoading ? (
          <div className="text-center py-8">
            <p className="text-body text-text-subtle">در حال بارگذاری...</p>
          </div>
        ) : gifts.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-body text-text-subtle">هیچ ایردراپی یافت نشد</p>
          </div>
        ) : (
          <>
            {/* Table */}
            <div className="overflow-x-auto">
              <table className="w-full text-body text-center">
                <thead>
                  <tr className="border-b border-border">
                    {['ردیف', 'عبارت جستجو', 'کد هدیه', 'وضعیت', 'خطا', 'تاریخ', 'تخصیص‌ها', 'عملیات'].map((title) => (
                      <th key={title} className="text-center py-3 px-2 text-label text-text-muted">
                        {title}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {gifts.map((gift, index) => {
                    const allocations = gift.Allocations ?? [];
                    const isExpanded = expandedGiftIds.has(gift.ID);
                    return (
                      <Fragment key={gift.ID}>
                        <tr className="border-b border-border hover:bg-surface-muted">
                          <td className="py-3 px-2 text-text-subtle">
                            {(rowOffset + index + 1).toLocaleString('fa-IR')}
                          </td>
                          <td className="py-3 px-2 text-text">
                            {gift.SearchPhrase || '—'}
                          </td>
                          <td className="py-3 px-2 text-text font-mono text-caption" dir="ltr">
                            {gift.GiftCode || '—'}
                          </td>
                          <td className="py-3 px-2">{getStatusBadge(gift.Status)}</td>
                          <td className="py-3 px-2 text-text-subtle text-caption max-w-[150px] truncate" title={gift.ErrorMessage}>
                            {gift.ErrorMessage || '—'}
                          </td>
                          <td className="py-3 px-2 text-text-muted text-caption whitespace-nowrap">
                            {formatDateTime(gift.CreatedAt)}
                          </td>
                          <td className="py-3 px-2">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => toggleExpanded(gift.ID)}
                              disabled={allocations.length === 0}
                              title={isExpanded ? 'بستن تخصیص‌ها' : 'نمایش تخصیص‌ها'}
                            >
                              {allocations.length.toLocaleString('fa-IR')}
                              {allocations.length > 0 && (
                                <ChevronDown className={`w-4 h-4 transition-transform ${isExpanded ? 'rotate-180' : ''}`} />
                              )}
                            </Button>
                          </td>
                          <td className="py-3 px-2">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => setSelectedGift(gift)}
                              title="مشاهده جزئیات"
                            >
                              <Eye className="w-4 h-4" />
                            </Button>
                          </td>
                        </tr>
                        {isExpanded && allocations.length > 0 && (
                          <tr className="border-b border-border bg-surface-muted">
                            <td colSpan={8} className="py-2 px-4">
                              <AllocationsTable
                                allocations={allocations}
                                retryingAllocationId={retryingAllocationId}
                                onRetry={handleRetryAllocation}
                                getStatusBadge={getStatusBadge}
                                isSuccessfulStatus={isSuccessfulStatus}
                              />
                            </td>
                          </tr>
                        )}
                      </Fragment>
                    );
                  })}
                </tbody>
              </table>
            </div>

            {/* Pagination */}
            {pagination && pagination.total_pages > 1 && (
              <div className="flex items-center justify-between mt-4 pt-4 border-t border-border">
                <p className="text-caption text-text-subtle">
                  صفحه {pagination.page} از {pagination.total_pages} (مجموع: {pagination.total_items})
                </p>
                <div className="flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={page <= 1}
                  >
                    <ChevronRight className="w-4 h-4" />
                  </Button>
                  <span className="px-3 py-1 text-body text-text-muted">
                    {pagination.page}
                  </span>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setPage((p) => Math.min(pagination.total_pages, p + 1))}
                    disabled={page >= pagination.total_pages}
                  >
                    <ChevronLeft className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            )}
          </>
        )}
      </Card>

      {/* Detail Modal */}
      {selectedGift && (
        <div role="dialog" aria-modal="true" className="fixed inset-0 z-(--z-modal) flex items-center justify-center p-4">
          <div
            className="absolute inset-0 bg-scrim"
            onClick={() => setSelectedGift(null)}
          />
          <div className="relative bg-surface rounded-xl shadow-lg w-full max-w-lg max-h-[80vh] overflow-y-auto border border-border">
            <div className="flex items-center justify-between p-4 border-b border-border">
              <h3 className="text-h3 font-semibold text-text">جزئیات ایردراپ</h3>
              <button
                type="button"
                aria-label="بستن"
                onClick={() => setSelectedGift(null)}
                className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-md text-text-subtle hover:text-text hover:bg-surface-muted"
              >
                <X className="h-5 w-5" aria-hidden="true" />
              </button>
            </div>
            <div className="p-4 space-y-4">
              <DetailRow label="عبارت جستجو" value={selectedGift.SearchPhrase} />
              <DetailRow label="کد هدیه" value={selectedGift.GiftCode} dir="ltr" />
              <DetailRow label="مسیر" value={selectedGift.Route} dir="ltr" />
              <div>
                <p className="text-caption font-medium text-text-subtle mb-1">وضعیت</p>
                {getStatusBadge(selectedGift.Status)}
              </div>
              <DetailRow label="پیام خطا" value={selectedGift.ErrorMessage} />
              <div>
                <p className="text-caption font-medium text-text-subtle mb-1">متن پیام</p>
                <div className="p-3 rounded-md bg-surface-muted border border-border text-body text-text whitespace-pre-wrap max-h-48 overflow-y-auto" dir="ltr">
                  {selectedGift.RawMessage || '—'}
                </div>
              </div>
              <DetailRow label="تاریخ ایجاد" value={formatDateTime(selectedGift.CreatedAt)} />
              {(selectedGift.Allocations ?? []).length > 0 && (
                <div>
                  <p className="text-caption font-medium text-text-subtle mb-1">تخصیص‌ها</p>
                  <AllocationsTable
                    allocations={selectedGift.Allocations ?? []}
                    retryingAllocationId={retryingAllocationId}
                    onRetry={handleRetryAllocation}
                    getStatusBadge={getStatusBadge}
                    isSuccessfulStatus={isSuccessfulStatus}
                  />
                </div>
              )}
            </div>
            <div className="p-4 border-t border-border flex gap-3">
              <Button variant="secondary" className="flex-1" onClick={() => setSelectedGift(null)}>
                بستن
              </Button>
              {!isSuccessfulStatus(selectedGift.Status) && (
                <Button
                  variant="primary"
                  className="flex-1"
                  onClick={() => {
                    setRetryGiftId(selectedGift.ID);
                    setManualGiftCode(selectedGift.GiftCode || '');
                  }}
                >
                  <RefreshCw className="w-4 h-4 me-2" />
                  تلاش مجدد
                </Button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Retry Modal */}
      {retryGiftId && (
        <div role="dialog" aria-modal="true" className="fixed inset-0 z-(--z-toast) flex items-center justify-center p-4">
          <div
            className="absolute inset-0 bg-scrim"
            onClick={() => !isRetrying && setRetryGiftId(null)}
          />
          <div className="relative bg-surface rounded-xl shadow-lg w-full max-w-md border border-border">
            <div className="flex items-center justify-between p-4 border-b border-border">
              <h3 className="text-h3 font-semibold text-text">تلاش مجدد برای دریافت جایزه</h3>
              <button
                type="button"
                aria-label="بستن"
                onClick={() => !isRetrying && setRetryGiftId(null)}
                disabled={isRetrying}
                className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-md text-text-subtle hover:text-text hover:bg-surface-muted disabled:opacity-40"
              >
                <X className="h-5 w-5" aria-hidden="true" />
              </button>
            </div>
            
            <div className="p-4 space-y-4">
              {retryError && (
                <Alert tone="error">{retryError}</Alert>
              )}
              
              {retrySuccess && (
                <Alert tone="success">{retrySuccess}</Alert>
              )}
              
              <div>
                <label className="block text-body font-medium text-text-muted mb-2">
                  کد هدیه (اختیاری)
                </label>
                <Input
                  type="text"
                  value={manualGiftCode}
                  onChange={(e) => setManualGiftCode(e.target.value)}
                  placeholder="کد هدیه را وارد کنید"
                  dir="ltr"
                  className="font-mono"
                  disabled={isRetrying}
                />
                <p className="text-caption text-text-subtle mt-1">
                  در صورت خالی گذاشتن، از کد موجود در سیستم استفاده می‌شود
                </p>
              </div>
              
              <div className="text-body text-text-muted">
                <p>این عملیات درخواست تست مجدد برای دریافت جایزه ارسال می‌کند.</p>
              </div>
            </div>
            
            <div className="p-4 border-t border-border flex gap-3">
              <Button
                variant="secondary"
                className="flex-1"
                onClick={() => !isRetrying && setRetryGiftId(null)}
                disabled={isRetrying}
              >
                انصراف
              </Button>
              <Button
                variant="primary"
                className="flex-1"
                onClick={() => handleRetry(gifts.find(g => g.ID === retryGiftId)!)}
                disabled={isRetrying}
              >
                {isRetrying ? (
                  <>
                    <RefreshCw className="w-4 h-4 me-2 animate-spin" />
                    در حال ارسال...
                  </>
                ) : (
                  <>
                    <RefreshCw className="w-4 h-4 me-2" />
                    ارسال درخواست
                  </>
                )}
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

function replaceAllocation(gift: AirdropGift, allocation: AirdropGiftAllocation): AirdropGift {
  return { ...gift, Allocations: (gift.Allocations ?? []).map(a => (a.ID === allocation.ID ? allocation : a)) };
}

function AllocationsTable({
  allocations,
  retryingAllocationId,
  onRetry,
  getStatusBadge,
  isSuccessfulStatus,
}: {
  allocations: AirdropGiftAllocation[];
  retryingAllocationId: number | null;
  onRetry: (allocation: AirdropGiftAllocation) => void;
  getStatusBadge: (status: string) => ReactNode;
  isSuccessfulStatus: (status: string) => boolean;
}) {
  return (
    <table className="w-full text-caption text-center">
      <thead>
        <tr className="text-text-subtle">
          {['ردیف', 'ارز', 'وضعیت', 'خطا', 'تاریخ', 'عملیات'].map((title) => (
            <th key={title} className="text-center py-2 px-2 text-label text-text-muted">{title}</th>
          ))}
        </tr>
      </thead>
      <tbody>
        {allocations.map((allocation, index) => {
          const isRetrying = retryingAllocationId === allocation.ID;
          return (
            <tr key={allocation.ID} className="border-t border-border">
              <td className="py-2 px-2 text-text-subtle">{(index + 1).toLocaleString('fa-IR')}</td>
              <td className="py-2 px-2 font-mono text-text" dir="ltr">{allocation.Currency || '—'}</td>
              <td className="py-2 px-2">{getStatusBadge(allocation.Status)}</td>
              <td className="py-2 px-2 text-text-subtle max-w-[200px] truncate" title={allocation.ErrorMessage}>
                {allocation.ErrorMessage || '—'}
              </td>
              <td className="py-2 px-2 text-text-muted whitespace-nowrap">{formatDateTime(allocation.CreatedAt)}</td>
              <td className="py-2 px-2">
                {isSuccessfulStatus(allocation.Status) ? (
                  '—'
                ) : (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onRetry(allocation)}
                    disabled={retryingAllocationId !== null}
                    title="تلاش مجدد"
                  >
                    <RefreshCw className={`w-4 h-4 ${isRetrying ? 'animate-spin' : ''}`} />
                  </Button>
                )}
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

function DetailRow({ label, value, dir }: { label: string; value?: string; dir?: string }) {
  return (
    <div>
      <p className="text-caption font-medium text-text-subtle mb-1">{label}</p>
      <p className="text-body text-text" dir={dir}>
        {value || '—'}
      </p>
    </div>
  );
}

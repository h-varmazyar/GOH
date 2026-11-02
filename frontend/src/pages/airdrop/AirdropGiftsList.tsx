import { Fragment, useEffect, useState, type ReactNode } from 'react';
import { Eye, ChevronLeft, ChevronRight, ChevronDown, X, RefreshCw } from 'lucide-react';
import { Card } from '../../components/ui/Card';
import { Button } from '../../components/ui/Button';
import { Badge } from '../../components/ui/Badge';
import { Input } from '../../components/ui/Input';
import { airdropService, type AirdropGift, type AirdropGiftAllocation, type Pagination } from '../../services/airdropService';

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
          <div className="mb-4 p-3 rounded-lg bg-danger-50 dark:bg-danger-950/20 border border-danger-200 dark:border-danger-800">
            <p className="text-sm text-danger-700 dark:text-danger-300">{error}</p>
          </div>
        )}

        {allocationError && (
          <div className="mb-4 p-3 rounded-lg bg-danger-50 dark:bg-danger-950/20 border border-danger-200 dark:border-danger-800 flex items-start justify-between gap-2">
            <p className="text-sm text-danger-700 dark:text-danger-300">{allocationError}</p>
            <button onClick={() => setAllocationError('')} className="p-0.5 rounded hover:bg-danger-100 dark:hover:bg-danger-900/40">
              <X className="w-4 h-4 text-danger-500" />
            </button>
          </div>
        )}

        {isLoading ? (
          <div className="text-center py-8">
            <p className="text-sm text-surface-500 dark:text-surface-400">در حال بارگذاری...</p>
          </div>
        ) : gifts.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-sm text-surface-500 dark:text-surface-400">هیچ ایردراپی یافت نشد</p>
          </div>
        ) : (
          <>
            {/* Table */}
            <div className="overflow-x-auto">
              <table className="w-full text-sm text-center">
                <thead>
                  <tr className="border-b border-surface-200 dark:border-surface-700">
                    {['ردیف', 'عبارت جستجو', 'کد هدیه', 'وضعیت', 'خطا', 'تاریخ', 'تخصیص‌ها', 'عملیات'].map((title) => (
                      <th key={title} className="text-center py-3 px-2 font-medium text-surface-600 dark:text-surface-400">
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
                        <tr className="border-b border-surface-100 dark:border-surface-800 hover:bg-surface-50 dark:hover:bg-surface-800/50">
                          <td className="py-3 px-2 text-surface-500 dark:text-surface-400">
                            {(rowOffset + index + 1).toLocaleString('fa-IR')}
                          </td>
                          <td className="py-3 px-2 text-surface-900 dark:text-surface-100">
                            {gift.SearchPhrase || '—'}
                          </td>
                          <td className="py-3 px-2 text-surface-900 dark:text-surface-100 font-mono text-xs" dir="ltr">
                            {gift.GiftCode || '—'}
                          </td>
                          <td className="py-3 px-2">{getStatusBadge(gift.Status)}</td>
                          <td className="py-3 px-2 text-surface-500 dark:text-surface-400 text-xs max-w-[150px] truncate" title={gift.ErrorMessage}>
                            {gift.ErrorMessage || '—'}
                          </td>
                          <td className="py-3 px-2 text-surface-700 dark:text-surface-300 text-xs whitespace-nowrap">
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
                          <tr className="border-b border-surface-100 dark:border-surface-800 bg-surface-50/60 dark:bg-surface-800/30">
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
              <div className="flex items-center justify-between mt-4 pt-4 border-t border-surface-200 dark:border-surface-700">
                <p className="text-xs text-surface-500 dark:text-surface-400">
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
                  <span className="px-3 py-1 text-sm text-surface-700 dark:text-surface-300">
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
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div
            className="absolute inset-0 bg-black/50"
            onClick={() => setSelectedGift(null)}
          />
          <div className="relative bg-white dark:bg-surface-900 rounded-xl shadow-xl w-full max-w-lg max-h-[80vh] overflow-y-auto border border-surface-200 dark:border-surface-700">
            <div className="flex items-center justify-between p-4 border-b border-surface-200 dark:border-surface-700">
              <h3 className="text-lg font-semibold text-surface-900 dark:text-white">جزئیات ایردراپ</h3>
              <button
                onClick={() => setSelectedGift(null)}
                className="p-1 rounded hover:bg-surface-100 dark:hover:bg-surface-800"
              >
                <X className="w-5 h-5 text-surface-500" />
              </button>
            </div>
            <div className="p-4 space-y-4">
              <DetailRow label="عبارت جستجو" value={selectedGift.SearchPhrase} />
              <DetailRow label="کد هدیه" value={selectedGift.GiftCode} dir="ltr" />
              <DetailRow label="مسیر" value={selectedGift.Route} dir="ltr" />
              <div>
                <p className="text-xs font-medium text-surface-500 dark:text-surface-400 mb-1">وضعیت</p>
                {getStatusBadge(selectedGift.Status)}
              </div>
              <DetailRow label="پیام خطا" value={selectedGift.ErrorMessage} />
              <div>
                <p className="text-xs font-medium text-surface-500 dark:text-surface-400 mb-1">متن پیام</p>
                <div className="p-3 rounded-lg bg-surface-50 dark:bg-surface-800 border border-surface-200 dark:border-surface-700 text-sm text-surface-800 dark:text-surface-200 whitespace-pre-wrap max-h-48 overflow-y-auto" dir="ltr">
                  {selectedGift.RawMessage || '—'}
                </div>
              </div>
              <DetailRow label="تاریخ ایجاد" value={formatDateTime(selectedGift.CreatedAt)} />
              {(selectedGift.Allocations ?? []).length > 0 && (
                <div>
                  <p className="text-xs font-medium text-surface-500 dark:text-surface-400 mb-1">تخصیص‌ها</p>
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
            <div className="p-4 border-t border-surface-200 dark:border-surface-700 flex gap-3">
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
                  <RefreshCw className="w-4 h-4 ml-2" />
                  تلاش مجدد
                </Button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Retry Modal */}
      {retryGiftId && (
        <div className="fixed inset-0 z-[60] flex items-center justify-center p-4">
          <div
            className="absolute inset-0 bg-black/50"
            onClick={() => !isRetrying && setRetryGiftId(null)}
          />
          <div className="relative bg-white dark:bg-surface-900 rounded-xl shadow-xl w-full max-w-md border border-surface-200 dark:border-surface-700">
            <div className="flex items-center justify-between p-4 border-b border-surface-200 dark:border-surface-700">
              <h3 className="text-lg font-semibold text-surface-900 dark:text-white">تلاش مجدد برای دریافت جایزه</h3>
              <button
                onClick={() => !isRetrying && setRetryGiftId(null)}
                disabled={isRetrying}
                className="p-1 rounded hover:bg-surface-100 dark:hover:bg-surface-800 disabled:opacity-50"
              >
                <X className="w-5 h-5 text-surface-500" />
              </button>
            </div>
            
            <div className="p-4 space-y-4">
              {retryError && (
                <div className="p-3 rounded-lg bg-danger-50 dark:bg-danger-950/20 border border-danger-200 dark:border-danger-800">
                  <p className="text-sm text-danger-700 dark:text-danger-300">{retryError}</p>
                </div>
              )}
              
              {retrySuccess && (
                <div className="p-3 rounded-lg bg-success-50 dark:bg-success-950/20 border border-success-200 dark:border-success-800">
                  <p className="text-sm text-success-700 dark:text-success-300">{retrySuccess}</p>
                </div>
              )}
              
              <div>
                <label className="block text-sm font-medium text-surface-700 dark:text-surface-300 mb-2">
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
                <p className="text-xs text-surface-500 dark:text-surface-400 mt-1">
                  در صورت خالی گذاشتن، از کد موجود در سیستم استفاده می‌شود
                </p>
              </div>
              
              <div className="text-sm text-surface-600 dark:text-surface-400">
                <p>این عملیات درخواست تست مجدد برای دریافت جایزه ارسال می‌کند.</p>
              </div>
            </div>
            
            <div className="p-4 border-t border-surface-200 dark:border-surface-700 flex gap-3">
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
                    <RefreshCw className="w-4 h-4 ml-2 animate-spin" />
                    در حال ارسال...
                  </>
                ) : (
                  <>
                    <RefreshCw className="w-4 h-4 ml-2" />
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
    <table className="w-full text-xs text-center">
      <thead>
        <tr className="text-surface-500 dark:text-surface-400">
          {['ردیف', 'ارز', 'وضعیت', 'خطا', 'تاریخ', 'عملیات'].map((title) => (
            <th key={title} className="text-center py-2 px-2 font-medium">{title}</th>
          ))}
        </tr>
      </thead>
      <tbody>
        {allocations.map((allocation, index) => {
          const isRetrying = retryingAllocationId === allocation.ID;
          return (
            <tr key={allocation.ID} className="border-t border-surface-200/70 dark:border-surface-700/70">
              <td className="py-2 px-2 text-surface-500 dark:text-surface-400">{(index + 1).toLocaleString('fa-IR')}</td>
              <td className="py-2 px-2 font-mono text-surface-900 dark:text-surface-100" dir="ltr">{allocation.Currency || '—'}</td>
              <td className="py-2 px-2">{getStatusBadge(allocation.Status)}</td>
              <td className="py-2 px-2 text-surface-500 dark:text-surface-400 max-w-[200px] truncate" title={allocation.ErrorMessage}>
                {allocation.ErrorMessage || '—'}
              </td>
              <td className="py-2 px-2 text-surface-700 dark:text-surface-300 whitespace-nowrap">{formatDateTime(allocation.CreatedAt)}</td>
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
      <p className="text-xs font-medium text-surface-500 dark:text-surface-400 mb-1">{label}</p>
      <p className="text-sm text-surface-900 dark:text-surface-100" dir={dir}>
        {value || '—'}
      </p>
    </div>
  );
}

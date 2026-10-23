import { useEffect, useState } from 'react';
import { Plus, ChevronLeft, ChevronRight, Trash2 } from 'lucide-react';
import { Card } from '../../components/ui/Card';
import { Button } from '../../components/ui/Button';
import { Input } from '../../components/ui/Input';
import { Select } from '../../components/ui/Select';
import {
  subscriptionFilterService,
  telegramService,
  type SubscriptionFilter,
  type TelegramAccountListItem,
  type CreateSubscriptionFilterPayload,
} from '../../services/telegramService';
import type { Pagination } from '../../services/airdropService';
import { X } from 'lucide-react';

export function SubscriptionFiltersList() {
  const [filters, setFilters] = useState<SubscriptionFilter[]>([]);
  const [pagination, setPagination] = useState<Pagination | null>(null);
  const [page, setPage] = useState(1);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  // Modal state
  const [showModal, setShowModal] = useState(false);
  const [accounts, setAccounts] = useState<TelegramAccountListItem[]>([]);
  const [formData, setFormData] = useState<CreateSubscriptionFilterPayload>({
    account_id: '',
    chat_id: 0,
  });
  const [chatIdInput, setChatIdInput] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [isDeleting, setIsDeleting] = useState<string | null>(null);

  useEffect(() => {
    fetchFilters();
  }, [page]);

  const fetchFilters = async () => {
    setIsLoading(true);
    setError('');
    try {
      const res = await subscriptionFilterService.getFilters(page);
      setFilters(res.data || []);
      setPagination(res.pagination);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'خطا در دریافت لیست فیلترها');
    } finally {
      setIsLoading(false);
    }
  };

  const fetchAccounts = async () => {
    try {
      const res = await telegramService.getAccounts(1, 100);
      setAccounts(res.data || []);
    } catch {
      // Silently fail – the select will just be empty
    }
  };

  const handleOpenCreate = () => {
    setFormData({ account_id: '', chat_id: 0 });
    setChatIdInput('');
    setShowModal(true);
    fetchAccounts();
  };

  const handleSave = async () => {
    const chatId = Number(chatIdInput);
    if (!formData.account_id || !chatId) return;

    setIsSaving(true);
    setError('');
    try {
      await subscriptionFilterService.create({
        account_id: formData.account_id,
        chat_id: chatId,
      });
      setShowModal(false);
      fetchFilters();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'خطا در ذخیره فیلتر');
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async (filterId: string) => {
    if (!confirm('آیا از حذف این فیلتر اطمینان دارید؟')) return;
    setIsDeleting(filterId);
    try {
      await subscriptionFilterService.delete(filterId);
      fetchFilters();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'خطا در حذف فیلتر');
    } finally {
      setIsDeleting(null);
    }
  };

  return (
    <>
      <Card>
        {/* Header */}
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-medium text-surface-700 dark:text-surface-300">
            لیست تخصیص اکانت‌ها به کانال‌ها
          </h3>
          <Button size="sm" onClick={handleOpenCreate}>
            <Plus className="w-4 h-4" />
            افزودن فیلتر
          </Button>
        </div>

        {error && (
          <div className="mb-4 p-3 rounded-lg bg-danger-50 dark:bg-danger-950/20 border border-danger-200 dark:border-danger-800">
            <p className="text-sm text-danger-700 dark:text-danger-300">{error}</p>
          </div>
        )}

        {isLoading ? (
          <div className="text-center py-8">
            <p className="text-sm text-surface-500 dark:text-surface-400">در حال بارگذاری...</p>
          </div>
        ) : filters.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-sm text-surface-500 dark:text-surface-400">
              هیچ فیلتری تعریف نشده است
            </p>
          </div>
        ) : (
          <>
            {/* Table */}
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-surface-200 dark:border-surface-700">
                    <th className="text-right py-3 px-2 font-medium text-surface-600 dark:text-surface-400">
                      اکانت
                    </th>
                    <th className="text-center py-3 px-2 font-medium text-surface-600 dark:text-surface-400">
                      شناسه چت
                    </th>
                    <th className="text-center py-3 px-2 font-medium text-surface-600 dark:text-surface-400">
                      تاریخ ایجاد
                    </th>
                    <th className="text-center py-3 px-2 font-medium text-surface-600 dark:text-surface-400">
                      عملیات
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {filters.map((filter) => (
                    <tr
                      key={filter.ID}
                      className="border-b border-surface-100 dark:border-surface-800 hover:bg-surface-50 dark:hover:bg-surface-800/50"
                    >
                      <td className="py-3 px-2 text-surface-900 dark:text-surface-100">
                        {filter.AccountUsername
                          ? `@${filter.AccountUsername}`
                          : filter.AccountPhone || filter.AccountID}
                      </td>
                      <td className="py-3 px-2 text-center text-surface-900 dark:text-surface-100 font-mono text-xs" dir="ltr">
                        {filter.ChatID}
                      </td>
                      <td className="py-3 px-2 text-center text-surface-500 dark:text-surface-400 text-xs" dir="ltr">
                        {new Date(filter.CreatedAt).toLocaleDateString('fa-IR')}
                      </td>
                      <td className="py-3 px-2 text-center">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleDelete(filter.ID)}
                          disabled={isDeleting === filter.ID}
                          className="text-danger-500 hover:text-danger-700"
                        >
                          <Trash2 className="w-4 h-4" />
                        </Button>
                      </td>
                    </tr>
                  ))}
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

      {/* Create Modal */}
      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/50" onClick={() => setShowModal(false)} />
          <div className="relative bg-white dark:bg-surface-900 rounded-xl shadow-xl w-full max-w-md border border-surface-200 dark:border-surface-700">
            <div className="flex items-center justify-between p-4 border-b border-surface-200 dark:border-surface-700">
              <h3 className="text-lg font-semibold text-surface-900 dark:text-white">
                افزودن فیلتر اشتراک
              </h3>
              <button
                onClick={() => setShowModal(false)}
                className="p-1 rounded hover:bg-surface-100 dark:hover:bg-surface-800"
              >
                <X className="w-5 h-5 text-surface-500" />
              </button>
            </div>
            <div className="p-4 space-y-4">
              <Select
                label="اکانت تلگرام"
                value={formData.account_id}
                onChange={(e) =>
                  setFormData({ ...formData, account_id: e.target.value })
                }
                options={[
                  { value: '', label: 'انتخاب اکانت...' },
                  ...accounts.map((acc) => ({
                    value: acc.ID,
                    label: acc.Username
                      ? `@${acc.Username} (${acc.Phone || acc.AccountType})`
                      : acc.Phone || acc.ID,
                  })),
                ]}
              />
              <Input
                label="شناسه چت (Chat ID)"
                placeholder="-1003944858901"
                value={chatIdInput}
                onChange={(e) => setChatIdInput(e.target.value)}
                dir="ltr"
              />
              <p className="text-xs text-surface-500 dark:text-surface-400">
                شناسه چت کانال یا گروهی که پیام‌ها از آن خوانده می‌شود را وارد کنید.
              </p>
            </div>
            <div className="flex gap-2 p-4 border-t border-surface-200 dark:border-surface-700">
              <Button variant="secondary" className="flex-1" onClick={() => setShowModal(false)}>
                انصراف
              </Button>
              <Button
                className="flex-1"
                onClick={handleSave}
                disabled={!formData.account_id || !chatIdInput.trim()}
                isLoading={isSaving}
              >
                ذخیره
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

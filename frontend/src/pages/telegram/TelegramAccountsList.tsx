import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Settings, CheckCircle, AlertCircle, XCircle, Loader2 } from 'lucide-react';
import { Card } from '../../components/ui/Card';
import { Button } from '../../components/ui/Button';
import { Badge } from '../../components/ui/Badge';
import { telegramService, type TelegramAccountListItem } from '../../services/telegramService';
import type { Pagination } from '../../services/airdropService';

export function TelegramAccountsList() {
  const navigate = useNavigate();
  const [accounts, setAccounts] = useState<TelegramAccountListItem[]>([]);
  const [pagination, setPagination] = useState<Pagination | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);

  useEffect(() => {
    fetchAccounts(page);
  }, [page]);

  const fetchAccounts = async (p: number) => {
    setLoading(true);
    setError(null);
    try {
      const res = await telegramService.getAccounts(p);
      setAccounts(res.data);
      setPagination(res.pagination);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'خطا در دریافت لیست اکانت‌ها';
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'ready':
        return <Badge variant="success" size="sm">آماده</Badge>;
      case 'needs_auth':
        return (
          <button
            onClick={() => navigate(`/telegram/auth/2`)}
            className="cursor-pointer"
          >
            <Badge variant="warning" size="sm">نیاز به احراز هویت</Badge>
          </button>
        );
      case 'disconnected':
        return <Badge variant="danger" size="sm">قطع اتصال</Badge>;
      default:
        return <Badge variant="default" size="sm">{status}</Badge>;
    }
  };

  const getRoleLabel = (role: string) => {
    switch (role) {
      case 'bot':
        return 'ربات';
      case 'client':
        return 'حساب شخصی';
      default:
        return role;
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'ready':
        return <CheckCircle className="w-5 h-5 text-success-500" />;
      case 'needs_auth':
        return <AlertCircle className="w-5 h-5 text-warning-500" />;
      case 'disconnected':
        return <XCircle className="w-5 h-5 text-danger-500" />;
      default:
        return null;
    }
  };

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-surface-900 dark:text-white">لیست اکانت‌ها</h2>
          <p className="text-sm text-surface-500 dark:text-surface-400 mt-1">مدیریت اکانت‌های تلگرام متصل</p>
        </div>
        <Button onClick={() => navigate('/telegram/add')} size="sm">
          افزودن اکانت جدید
        </Button>
      </div>

      {/* Error */}
      {error && (
        <Card>
          <p className="text-sm text-danger-500">{error}</p>
        </Card>
      )}

      {/* Accounts Table */}
      <Card padding="none">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-surface-200 dark:border-surface-700 bg-surface-50 dark:bg-surface-800/50">
                <th className="text-center px-4 py-3 text-xs font-medium text-surface-500 dark:text-surface-400">وضعیت</th>
                <th className="text-right px-4 py-3 text-xs font-medium text-surface-500 dark:text-surface-400">نام کاربری</th>
                <th className="text-right px-4 py-3 text-xs font-medium text-surface-500 dark:text-surface-400">نام اکانت</th>
                <th className="text-center px-4 py-3 text-xs font-medium text-surface-500 dark:text-surface-400">نقش</th>
                <th className="text-center px-4 py-3 text-xs font-medium text-surface-500 dark:text-surface-400">شماره موبایل</th>
                <th className="text-center px-4 py-3 text-xs font-medium text-surface-500 dark:text-surface-400">تنظیمات</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center">
                    <Loader2 className="w-5 h-5 animate-spin mx-auto text-surface-400" />
                  </td>
                </tr>
              ) : accounts.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-sm text-surface-500 dark:text-surface-400">
                    هیچ اکانتی وجود ندارد
                  </td>
                </tr>
              ) : (
                accounts.map((account) => (
                  <tr key={account.ID} className="border-b border-surface-100 dark:border-surface-700 hover:bg-surface-50 dark:hover:bg-surface-800/50 transition-colors">
                    <td className="px-4 py-3 text-center">
                      <div className="flex items-center justify-center gap-2">
                        {getStatusIcon(account.Status)}
                        {getStatusBadge(account.Status)}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-sm font-medium text-surface-900 dark:text-white">
                        {account.Username ? `@${account.Username}` : '—'}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-sm text-surface-700 dark:text-surface-300">
                        {account.AccountName || '—'}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-center">
                      <Badge variant="primary" size="sm">{getRoleLabel(account.Role)}</Badge>
                    </td>
                    <td className="px-4 py-3 text-center">
                      <span className="text-sm text-surface-600 dark:text-surface-400" dir="ltr">
                        {account.Mobile || '—'}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-center">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => navigate(`/telegram/settings/${account.ID}`)}
                      >
                        <Settings className="w-4 h-4" />
                        تنظیمات
                      </Button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {pagination && pagination.total_pages > 1 && (
          <div className="flex items-center justify-center gap-2 px-4 py-3 border-t border-surface-200 dark:border-surface-700">
            <Button
              variant="ghost"
              size="sm"
              disabled={page <= 1}
              onClick={() => setPage((p) => p - 1)}
            >
              قبلی
            </Button>
            <span className="text-sm text-surface-600 dark:text-surface-400">
              صفحه {pagination.page} از {pagination.total_pages}
            </span>
            <Button
              variant="ghost"
              size="sm"
              disabled={page >= pagination.total_pages}
              onClick={() => setPage((p) => p + 1)}
            >
              بعدی
            </Button>
          </div>
        )}
      </Card>
    </div>
  );
}

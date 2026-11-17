import { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Settings, CheckCircle, AlertCircle, XCircle, Loader2, PlusCircle } from 'lucide-react';
import { Card } from '../../components/ui/Card';
import { Button } from '../../components/ui/Button';
import { Badge } from '../../components/ui/Badge';
import { telegramService, type TelegramAccountListItem } from '../../services/telegramService';
import type { Pagination } from '../../services/airdropService';
import { TelegramAddAccountDialog } from './TelegramAddAccountDialog';

export function TelegramAccountsList() {
  const navigate = useNavigate();
  const [accounts, setAccounts] = useState<TelegramAccountListItem[]>([]);
  const [pagination, setPagination] = useState<Pagination | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  // ?add=1 opens the add-account dialog (used by the sidebar and overview links)
  const [searchParams, setSearchParams] = useSearchParams();
  const showAddDialog = searchParams.get('add') === '1';

  const setShowAddDialog = (show: boolean) =>
    setSearchParams(
      (params) => {
        if (show) params.set('add', '1');
        else params.delete('add');
        return params;
      },
      { replace: !show }
    );

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

  useEffect(() => {
    fetchAccounts(page);
  }, [page]);

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
        return <CheckCircle className="w-5 h-5 text-success" />;
      case 'needs_auth':
        return <AlertCircle className="w-5 h-5 text-warning-text" />;
      case 'disconnected':
        return <XCircle className="w-5 h-5 text-error" />;
      default:
        return null;
    }
  };

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-h3 font-semibold text-text">لیست اکانت‌ها</h2>
          <p className="text-body text-text-subtle mt-1">مدیریت اکانت‌های تلگرام متصل</p>
        </div>
        <Button onClick={() => setShowAddDialog(true)} size="sm">
          <PlusCircle className="w-4 h-4" />
          افزودن اکانت جدید
        </Button>
      </div>

      {/* Error */}
      {error && (
        <Card>
          <p className="text-body text-error">{error}</p>
        </Card>
      )}

      {/* Accounts Table */}
      <Card padding="none">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-surface-muted">
                <th className="text-center px-4 py-3 text-label text-text-muted">وضعیت</th>
                <th className="text-start px-4 py-3 text-label text-text-muted">نام کاربری</th>
                <th className="text-start px-4 py-3 text-label text-text-muted">نام اکانت</th>
                <th className="text-center px-4 py-3 text-label text-text-muted">نقش</th>
                <th className="text-center px-4 py-3 text-label text-text-muted">شماره موبایل</th>
                <th className="text-center px-4 py-3 text-label text-text-muted">تنظیمات</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center">
                    <Loader2 className="w-5 h-5 animate-spin mx-auto text-text-subtle" />
                  </td>
                </tr>
              ) : accounts.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-body text-text-subtle">
                    هیچ اکانتی وجود ندارد
                  </td>
                </tr>
              ) : (
                accounts.map((account) => (
                  <tr key={account.ID} className="border-b border-border hover:bg-surface-muted transition-colors">
                    <td className="px-4 py-3 text-center">
                      <div className="flex items-center justify-center gap-2">
                        {getStatusIcon(account.Status)}
                        {getStatusBadge(account.Status)}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-body font-medium text-text">
                        {account.Username ? `@${account.Username}` : '—'}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-body text-text-muted">
                        {account.AccountName || '—'}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-center">
                      <Badge variant="primary" size="sm">{getRoleLabel(account.Role)}</Badge>
                    </td>
                    <td className="px-4 py-3 text-center">
                      <span className="text-body text-text-muted" dir="ltr">
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
          <div className="flex items-center justify-center gap-2 px-4 py-3 border-t border-border">
            <Button
              variant="ghost"
              size="sm"
              disabled={page <= 1}
              onClick={() => setPage((p) => p - 1)}
            >
              قبلی
            </Button>
            <span className="text-body text-text-muted">
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

      {showAddDialog && (
        <TelegramAddAccountDialog
          onClose={() => {
            setShowAddDialog(false);
            // A QR login is persisted just after it reports ready, so refresh once more on close
            fetchAccounts(page);
          }}
          onAdded={() => fetchAccounts(page)}
        />
      )}
    </div>
  );
}

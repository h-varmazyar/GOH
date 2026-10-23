import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Send, Users, PlusCircle, CheckCircle, AlertCircle, XCircle } from 'lucide-react';
import { Card } from '../../components/ui/Card';
import { Button } from '../../components/ui/Button';
import { Badge } from '../../components/ui/Badge';
import type { TelegramAccount } from '../../types';

// Mock data — replace with real API later
const mockAccounts: TelegramAccount[] = [
  { id: '1', username: 'mybot', type: 'bot', accountId: '123456789', status: 'connected', createdAt: '2024-01-15' },
  { id: '2', username: 'john_doe', type: 'personal', accountId: '987654321', status: 'needs_auth', phone: '+989123456789', createdAt: '2024-01-20' },
];

export function TelegramOverview() {
  const navigate = useNavigate();
  const [accounts] = useState<TelegramAccount[]>(mockAccounts);

  const connectedCount = accounts.filter((a) => a.status === 'connected').length;
  const needsAuthCount = accounts.filter((a) => a.status === 'needs_auth').length;

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h1 className="text-2xl font-bold text-surface-900 dark:text-white">تلگرام</h1>
        <p className="text-sm text-surface-500 dark:text-surface-400 mt-1">نمای کلی وضعیت اکانت‌های تلگرام</p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <Card padding="sm" className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-primary-400/10">
            <Users className="w-5 h-5 text-primary-500" />
          </div>
          <div>
            <p className="text-xs text-surface-500 dark:text-surface-400">کل اکانت‌ها</p>
            <p className="text-lg font-bold text-surface-900 dark:text-white">{accounts.length}</p>
          </div>
        </Card>
        <Card padding="sm" className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-success-400/10">
            <CheckCircle className="w-5 h-5 text-success-500" />
          </div>
          <div>
            <p className="text-xs text-surface-500 dark:text-surface-400">متصل</p>
            <p className="text-lg font-bold text-surface-900 dark:text-white">{connectedCount}</p>
          </div>
        </Card>
        <Card padding="sm" className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-warning-400/10">
            <AlertCircle className="w-5 h-5 text-warning-500" />
          </div>
          <div>
            <p className="text-xs text-surface-500 dark:text-surface-400">نیاز به احراز هویت</p>
            <p className="text-lg font-bold text-surface-900 dark:text-white">{needsAuthCount}</p>
          </div>
        </Card>
      </div>

      {/* Quick Actions */}
      <Card>
        <h2 className="text-lg font-semibold text-surface-900 dark:text-white mb-4">دسترسی سریع</h2>
        <div className="flex flex-wrap gap-3">
          <Button onClick={() => navigate('/telegram/add')}>
            <PlusCircle className="w-4 h-4" />
            افزودن اکانت جدید
          </Button>
          <Button variant="secondary" onClick={() => navigate('/telegram/accounts')}>
            <Users className="w-4 h-4" />
            مشاهده لیست اکانت‌ها
          </Button>
        </div>
      </Card>

      {/* Recent accounts */}
      {accounts.length > 0 && (
        <Card>
          <h2 className="text-lg font-semibold text-surface-900 dark:text-white mb-4">آخرین اکانت‌ها</h2>
          <div className="space-y-3">
            {accounts.slice(0, 5).map((account) => (
              <div
                key={account.id}
                className="flex items-center justify-between p-3 rounded-lg bg-surface-50 dark:bg-surface-800/50 border border-surface-100 dark:border-surface-700"
              >
                <div className="flex items-center gap-3">
                  {account.status === 'connected' && <CheckCircle className="w-4 h-4 text-success-500" />}
                  {account.status === 'needs_auth' && <AlertCircle className="w-4 h-4 text-warning-500" />}
                  {account.status === 'disconnected' && <XCircle className="w-4 h-4 text-danger-500" />}
                  <div>
                    <p className="text-sm font-medium text-surface-900 dark:text-white">@{account.username}</p>
                    <p className="text-xs text-surface-500 dark:text-surface-400">
                      {account.type === 'bot' ? 'ربات' : 'حساب شخصی'}
                    </p>
                  </div>
                </div>
                <Badge variant={account.status === 'connected' ? 'success' : account.status === 'needs_auth' ? 'warning' : 'danger'} size="sm">
                  {account.status === 'connected' ? 'متصل' : account.status === 'needs_auth' ? 'نیاز به تایید' : 'قطع'}
                </Badge>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
}

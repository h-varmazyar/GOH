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
        <h1 className="text-h1 font-bold text-text">تلگرام</h1>
        <p className="text-body text-text-subtle mt-1">نمای کلی وضعیت اکانت‌های تلگرام</p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <Card padding="sm" className="flex items-center gap-3">
          <div className="p-2 rounded-md bg-selected/10">
            <Users className="w-5 h-5 text-link" />
          </div>
          <div>
            <p className="text-caption text-text-subtle">کل اکانت‌ها</p>
            <p className="text-h3 font-bold text-text">{accounts.length}</p>
          </div>
        </Card>
        <Card padding="sm" className="flex items-center gap-3">
          <div className="p-2 rounded-md bg-success/8">
            <CheckCircle className="w-5 h-5 text-success" />
          </div>
          <div>
            <p className="text-caption text-text-subtle">متصل</p>
            <p className="text-h3 font-bold text-text">{connectedCount}</p>
          </div>
        </Card>
        <Card padding="sm" className="flex items-center gap-3">
          <div className="p-2 rounded-md bg-warning/12">
            <AlertCircle className="w-5 h-5 text-warning-text" />
          </div>
          <div>
            <p className="text-caption text-text-subtle">نیاز به احراز هویت</p>
            <p className="text-h3 font-bold text-text">{needsAuthCount}</p>
          </div>
        </Card>
      </div>

      {/* Quick Actions */}
      <Card>
        <h2 className="text-h3 font-semibold text-text mb-4">دسترسی سریع</h2>
        <div className="flex flex-wrap gap-3">
          <Button onClick={() => navigate('/telegram/accounts?add=1')}>
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
          <h2 className="text-h3 font-semibold text-text mb-4">آخرین اکانت‌ها</h2>
          <div className="space-y-3">
            {accounts.slice(0, 5).map((account) => (
              <div
                key={account.id}
                className="flex items-center justify-between p-3 rounded-md bg-surface-muted border border-border"
              >
                <div className="flex items-center gap-3">
                  {account.status === 'connected' && <CheckCircle className="w-4 h-4 text-success" />}
                  {account.status === 'needs_auth' && <AlertCircle className="w-4 h-4 text-warning-text" />}
                  {account.status === 'disconnected' && <XCircle className="w-4 h-4 text-error" />}
                  <div>
                    <p className="text-body font-medium text-text">@{account.username}</p>
                    <p className="text-caption text-text-subtle">
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

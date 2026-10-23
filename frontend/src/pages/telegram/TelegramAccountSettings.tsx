import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import { Settings, Trash2, Bell, BellOff, Save } from 'lucide-react';
import { Card } from '../../components/ui/Card';
import { Button } from '../../components/ui/Button';
import { Badge } from '../../components/ui/Badge';
import { Toggle } from '../../components/ui/Toggle';

export function TelegramAccountSettings() {
  const { accountId } = useParams<{ accountId: string }>();
  const navigate = useNavigate();
  
  // Mock account data
  const [account] = useState({
    id: accountId || '1',
    username: 'mybot',
    type: 'bot' as const,
    accountId: '123456789',
    status: 'connected' as const,
    phone: '+989123456789',
  });

  const [settings, setSettings] = useState({
    notifications: true,
    autoReconnect: true,
    logMessages: false,
  });

  const handleSave = () => {
    // Simulate API call
    console.log('Saving settings:', settings);
  };

  const handleDisconnect = () => {
    // Simulate disconnect
    console.log('Disconnecting account:', account.id);
  };

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="sm" onClick={() => navigate('/telegram')}>
          <ArrowLeft className="w-4 h-4" />
        </Button>
        <div>
          <h2 className="text-lg font-semibold text-surface-900 dark:text-white">تنظیمات اکانت</h2>
          <p className="text-sm text-surface-500 dark:text-surface-400 mt-1">
            مدیریت تنظیمات @{account.username}
          </p>
        </div>
      </div>

      {/* Account Info Card */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 rounded-full bg-primary-100 dark:bg-primary-900 flex items-center justify-center">
              <Settings className="w-6 h-6 text-primary-600" />
            </div>
            <div>
              <h3 className="font-semibold text-surface-900 dark:text-white">@{account.username}</h3>
              <p className="text-sm text-surface-500 dark:text-surface-400" dir="ltr">{account.accountId}</p>
            </div>
          </div>
          <Badge variant={account.status === 'connected' ? 'success' : 'warning'} size="md">
            {account.status === 'connected' ? 'متصل' : 'نیاز به احراز هویت'}
          </Badge>
        </div>

        <div className="grid grid-cols-2 gap-4 pt-4 border-t border-surface-200 dark:border-surface-700">
          <div>
            <p className="text-xs text-surface-500 dark:text-surface-400">نوع اکانت</p>
            <p className="text-sm font-medium text-surface-900 dark:text-white">
              {account.type === 'bot' ? 'ربات' : 'حساب شخصی'}
            </p>
          </div>
          <div>
            <p className="text-xs text-surface-500 dark:text-surface-400">شماره تلفن</p>
            <p className="text-sm font-medium text-surface-900 dark:text-white" dir="ltr">{account.phone}</p>
          </div>
        </div>
      </Card>

      {/* Settings Form */}
      <Card>
        <h3 className="text-base font-semibold text-surface-900 dark:text-white mb-4">تنظیمات</h3>
        
        <div className="space-y-4">
          <div className="flex items-center justify-between p-3 rounded-lg bg-surface-50 dark:bg-surface-800/50">
            <div className="flex items-center gap-3">
              {settings.notifications ? (
                <Bell className="w-5 h-5 text-primary-500" />
              ) : (
                <BellOff className="w-5 h-5 text-surface-400" />
              )}
              <div>
                <p className="text-sm font-medium text-surface-900 dark:text-white">اعلان‌ها</p>
                <p className="text-xs text-surface-500 dark:text-surface-400">دریافت اعلان برای رویدادها</p>
              </div>
            </div>
            <Toggle
              enabled={settings.notifications}
              onChange={(checked) => setSettings({ ...settings, notifications: checked })}
            />
          </div>

          <div className="flex items-center justify-between p-3 rounded-lg bg-surface-50 dark:bg-surface-800/50">
            <div className="flex items-center gap-3">
              <Settings className="w-5 h-5 text-primary-500" />
              <div>
                <p className="text-sm font-medium text-surface-900 dark:text-white">اتصال خودکار</p>
                <p className="text-xs text-surface-500 dark:text-surface-400">اتصال مجدد خودکار در صورت قطع</p>
              </div>
            </div>
            <Toggle
              enabled={settings.autoReconnect}
              onChange={(checked) => setSettings({ ...settings, autoReconnect: checked })}
            />
          </div>

          <div className="flex items-center justify-between p-3 rounded-lg bg-surface-50 dark:bg-surface-800/50">
            <div className="flex items-center gap-3">
              <Save className="w-5 h-5 text-primary-500" />
              <div>
                <p className="text-sm font-medium text-surface-900 dark:text-white">ذخیره لاگ پیام‌ها</p>
                <p className="text-xs text-surface-500 dark:text-surface-400">ثبت تاریخچه پیام‌ها</p>
              </div>
            </div>
            <Toggle
              enabled={settings.logMessages}
              onChange={(checked) => setSettings({ ...settings, logMessages: checked })}
            />
          </div>
        </div>

        <div className="mt-6">
          <Button className="w-full" onClick={handleSave}>
            ذخیره تنظیمات
          </Button>
        </div>
      </Card>

      {/* Danger Zone */}
      <Card className="border-danger-200 dark:border-danger-800">
        <h3 className="text-base font-semibold text-danger-600 mb-4">منطقه خطر</h3>
        
        <div className="flex items-center justify-between p-3 rounded-lg bg-danger-50 dark:bg-danger-950/20">
          <div>
            <p className="text-sm font-medium text-danger-900 dark:text-danger-100">قطع اتصال اکانت</p>
            <p className="text-xs text-danger-700 dark:text-danger-300 mt-0.5">
              این اکانت از سیستم حذف خواهد شد
            </p>
          </div>
          <Button variant="danger" size="sm" onClick={handleDisconnect}>
            <Trash2 className="w-4 h-4" />
            قطع اتصال
          </Button>
        </div>
      </Card>
    </div>
  );
}

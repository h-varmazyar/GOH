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
          <h2 className="text-h3 font-semibold text-text">تنظیمات اکانت</h2>
          <p className="text-body text-text-subtle mt-1">
            مدیریت تنظیمات @{account.username}
          </p>
        </div>
      </div>

      {/* Account Info Card */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 rounded-pill bg-selected/15 flex items-center justify-center">
              <Settings className="w-6 h-6 text-link" />
            </div>
            <div>
              <h3 className="font-semibold text-text">@{account.username}</h3>
              <p className="text-body text-text-subtle" dir="ltr">{account.accountId}</p>
            </div>
          </div>
          <Badge variant={account.status === 'connected' ? 'success' : 'warning'} size="md">
            {account.status === 'connected' ? 'متصل' : 'نیاز به احراز هویت'}
          </Badge>
        </div>

        <div className="grid grid-cols-2 gap-4 pt-4 border-t border-border">
          <div>
            <p className="text-caption text-text-subtle">نوع اکانت</p>
            <p className="text-body font-medium text-text">
              {account.type === 'bot' ? 'ربات' : 'حساب شخصی'}
            </p>
          </div>
          <div>
            <p className="text-caption text-text-subtle">شماره تلفن</p>
            <p className="text-body font-medium text-text" dir="ltr">{account.phone}</p>
          </div>
        </div>
      </Card>

      {/* Settings Form */}
      <Card>
        <h3 className="text-body-lg font-semibold text-text mb-4">تنظیمات</h3>
        
        <div className="space-y-4">
          <div className="flex items-center justify-between p-3 rounded-md bg-surface-muted">
            <div className="flex items-center gap-3">
              {settings.notifications ? (
                <Bell className="w-5 h-5 text-link" />
              ) : (
                <BellOff className="w-5 h-5 text-text-subtle" />
              )}
              <div>
                <p className="text-body font-medium text-text">اعلان‌ها</p>
                <p className="text-caption text-text-subtle">دریافت اعلان برای رویدادها</p>
              </div>
            </div>
            <Toggle
              aria-label="اعلان‌ها"
              enabled={settings.notifications}
              onChange={(checked) => setSettings({ ...settings, notifications: checked })}
            />
          </div>

          <div className="flex items-center justify-between p-3 rounded-md bg-surface-muted">
            <div className="flex items-center gap-3">
              <Settings className="w-5 h-5 text-link" />
              <div>
                <p className="text-body font-medium text-text">اتصال خودکار</p>
                <p className="text-caption text-text-subtle">اتصال مجدد خودکار در صورت قطع</p>
              </div>
            </div>
            <Toggle
              aria-label="اتصال خودکار"
              enabled={settings.autoReconnect}
              onChange={(checked) => setSettings({ ...settings, autoReconnect: checked })}
            />
          </div>

          <div className="flex items-center justify-between p-3 rounded-md bg-surface-muted">
            <div className="flex items-center gap-3">
              <Save className="w-5 h-5 text-link" />
              <div>
                <p className="text-body font-medium text-text">ذخیره لاگ پیام‌ها</p>
                <p className="text-caption text-text-subtle">ثبت تاریخچه پیام‌ها</p>
              </div>
            </div>
            <Toggle
              aria-label="ذخیره لاگ پیام‌ها"
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
      <Card className="border-error/30">
        <h3 className="text-body-lg font-semibold text-error mb-4">منطقه خطر</h3>
        
        <div className="flex items-center justify-between p-3 rounded-md bg-error/8">
          <div>
            <p className="text-body font-medium text-error">قطع اتصال اکانت</p>
            <p className="text-caption text-error mt-0.5">
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

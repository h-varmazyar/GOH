import { useState } from 'react';
import { useAuthStore } from '../store/authStore';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Card } from '../components/ui/Card';
import { Badge } from '../components/ui/Badge';
import { User, Lock, Phone, Shield, AtSign } from 'lucide-react';

export function ProfilePage() {
  const { user, updateProfile, changePassword } = useAuthStore();
  const [firstName, setFirstName] = useState(user?.firstName || '');
  const [lastName, setLastName] = useState(user?.lastName || '');
  const [isSaving, setIsSaving] = useState(false);
  const [saveMessage, setSaveMessage] = useState('');

  // Password change
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [passwordError, setPasswordError] = useState('');
  const [passwordSuccess, setPasswordSuccess] = useState('');
  const [isChangingPassword, setIsChangingPassword] = useState(false);

  const handleSaveProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSaving(true);
    setSaveMessage('');
    await new Promise((resolve) => setTimeout(resolve, 400));
    updateProfile({ firstName, lastName });
    setSaveMessage('اطلاعات با موفقیت ذخیره شد');
    setIsSaving(false);
    setTimeout(() => setSaveMessage(''), 3000);
  };

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setPasswordError('');
    setPasswordSuccess('');

    if (newPassword.length < 8) {
      setPasswordError('رمز عبور جدید باید حداقل ۸ کاراکتر باشد');
      return;
    }
    if (newPassword !== confirmPassword) {
      setPasswordError('رمز عبور جدید و تکرار آن مطابقت ندارند');
      return;
    }

    setIsChangingPassword(true);
    const result = await changePassword(oldPassword, newPassword);
    setIsChangingPassword(false);

    if (result.success) {
      setPasswordSuccess('رمز عبور با موفقیت تغییر کرد');
      setOldPassword('');
      setNewPassword('');
      setConfirmPassword('');
      setTimeout(() => setPasswordSuccess(''), 3000);
    } else {
      setPasswordError(result.error || 'تغییر رمز عبور ناموفق');
    }
  };

  const roleLabels: Record<string, string> = {
    admin: 'مدیر سیستم',
    user: 'کاربر عادی',
    viewer: 'مشاهده‌کننده',
  };

  const fullName = `${user?.firstName || ''} ${user?.lastName || ''}`.trim() || user?.username || '';

  return (
    <div className="space-y-6 max-w-2xl">
      {/* Page Header */}
      <div>
        <h1 className="text-h1 font-bold text-text">پروفایل کاربری</h1>
        <p className="text-body text-text-subtle mt-1">مدیریت اطلاعات حساب کاربری</p>
      </div>

      {/* User Info Card */}
      <Card>
        <div className="flex items-center gap-4 mb-6">
          <div className="w-16 h-16 rounded-pill bg-selected/15 flex items-center justify-center">
            <User className="w-8 h-8 text-link" />
          </div>
          <div>
            <h2 className="text-h3 font-semibold text-text">{fullName}</h2>
            <div className="flex items-center gap-2 mt-1 flex-wrap">
              <Badge variant={user?.role === 'admin' ? 'primary' : 'default'}>
                <Shield className="w-3 h-3 me-1" />
                {roleLabels[user?.role || 'user']}
              </Badge>
              <Badge variant={user?.isVerified ? 'success' : 'warning'}>
                {user?.isVerified ? 'تأیید شده' : 'تأیید نشده'}
              </Badge>
            </div>
          </div>
        </div>

        <div className="space-y-2">
          <div className="flex items-center gap-2 p-3 rounded-md bg-surface-muted border border-border">
            <AtSign className="w-4 h-4 text-text-subtle" />
            <span className="text-body text-text-muted" dir="ltr">{user?.username}</span>
          </div>
          <div className="flex items-center gap-2 p-3 rounded-md bg-surface-muted border border-border">
            <Phone className="w-4 h-4 text-text-subtle" />
            <span className="text-body text-text-muted" dir="ltr">{user?.phone}</span>
          </div>
        </div>
      </Card>

      {/* Edit Profile */}
      <Card>
        <h3 className="text-body-lg font-semibold text-text mb-4 flex items-center gap-2">
          <User className="w-5 h-5" />
          ویرایش اطلاعات
        </h3>
        <form onSubmit={handleSaveProfile} className="space-y-4">
          <div className="grid sm:grid-cols-2 gap-4">
            <Input
              label="نام"
              value={firstName}
              onChange={(e) => setFirstName(e.target.value)}
              placeholder="نام"
            />
            <Input
              label="نام خانوادگی"
              value={lastName}
              onChange={(e) => setLastName(e.target.value)}
              placeholder="نام خانوادگی"
            />
          </div>
          {saveMessage && (
            <p className="text-body text-success">{saveMessage}</p>
          )}
          <Button type="submit" isLoading={isSaving}>
            ذخیره تغییرات
          </Button>
        </form>
      </Card>

      {/* Change Password */}
      <Card>
        <h3 className="text-body-lg font-semibold text-text mb-4 flex items-center gap-2">
          <Lock className="w-5 h-5" />
          تغییر رمز عبور
        </h3>
        <form onSubmit={handleChangePassword} className="space-y-4">
          <Input
            label="رمز عبور فعلی"
            type="password"
            value={oldPassword}
            onChange={(e) => setOldPassword(e.target.value)}
            placeholder="••••••••"
            dir="ltr"
          />
          <Input
            label="رمز عبور جدید"
            type="password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            placeholder="••••••••"
            dir="ltr"
            hint="حداقل ۸ کاراکتر"
          />
          <Input
            label="تکرار رمز عبور جدید"
            type="password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            placeholder="••••••••"
            dir="ltr"
          />
          {passwordError && (
            <p className="text-body text-error">{passwordError}</p>
          )}
          {passwordSuccess && (
            <p className="text-body text-success">{passwordSuccess}</p>
          )}
          <Button type="submit" isLoading={isChangingPassword}>
            تغییر رمز عبور
          </Button>
        </form>
      </Card>
    </div>
  );
}

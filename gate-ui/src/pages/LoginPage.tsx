import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { LineChart, Phone, Lock } from 'lucide-react';
import { useAuthStore } from '../store/authStore';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Card } from '../components/ui/Card';

export function LoginPage() {
  const [mobile, setMobile] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const { login, isLoading } = useAuthStore();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    const cleanMobile = mobile.replace(/\s/g, '');
    if (!/^09\d{9}$/.test(cleanMobile)) {
      setError('شماره موبایل باید ۱۱ رقمی و با ۰۹ شروع شود');
      return;
    }
    if (!password) {
      setError('رمز عبور را وارد کنید');
      return;
    }

    const result = await login(cleanMobile, password);
    if (result.success) {
      navigate('/dashboard');
    } else {
      setError(result.error || 'ورود ناموفق');
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4 bg-gradient-to-br from-surface-50 to-surface-100 dark:from-surface-950 dark:to-surface-900">
      <Card className="w-full max-w-sm" padding="lg">
        {/* Logo */}
        <div className="flex flex-col items-center mb-8">
          <div className="w-14 h-14 rounded-2xl bg-primary-600 flex items-center justify-center mb-4 shadow-lg shadow-primary-600/30">
            <LineChart className="w-8 h-8 text-white" />
          </div>
          <h1 className="text-2xl font-bold text-surface-900 dark:text-white">Gate Trading</h1>
          <p className="text-sm text-surface-500 dark:text-surface-400 mt-1">ورود به سیستم معاملات خودکار</p>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="relative">
            <Phone className="absolute right-3 top-9 w-4 h-4 text-surface-400 pointer-events-none" />
            <Input
              label="شماره موبایل"
              type="tel"
              value={mobile}
              onChange={(e) => setMobile(e.target.value)}
              placeholder="09123456789"
              dir="ltr"
              maxLength={11}
              className="pr-10"
              autoFocus
            />
          </div>

          <div className="relative">
            <Lock className="absolute right-3 top-9 w-4 h-4 text-surface-400 pointer-events-none" />
            <Input
              label="رمز عبور"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              dir="ltr"
              className="pr-10"
            />
          </div>

          {error && (
            <p className="text-sm text-danger-500 text-center">{error}</p>
          )}

          <div className="flex justify-end">
            <Link
              to="/forgot-password"
              className="text-xs text-primary-600 dark:text-primary-300 hover:underline"
            >
              فراموشی رمز عبور؟
            </Link>
          </div>

          <Button type="submit" isLoading={isLoading} className="w-full" size="lg">
            ورود
          </Button>
        </form>

        <div className="mt-6 pt-6 border-t border-surface-200 dark:border-surface-700 text-center">
          <p className="text-sm text-surface-500 dark:text-surface-400">
            حساب کاربری ندارید؟{' '}
            <Link to="/register" className="text-primary-600 dark:text-primary-300 font-medium hover:underline">
              ثبت‌نام کنید
            </Link>
          </p>
        </div>
      </Card>
    </div>
  );
}

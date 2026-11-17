import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { Phone, Lock } from 'lucide-react';
import { useAuthStore } from '../store/authStore';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Alert } from '../components/ui/Alert';
import { AuthLayout } from '../components/layout/AuthLayout';

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
    <AuthLayout title="ورود به Gate" subtitle="ورود به سیستم معاملات خودکار">
      <form onSubmit={handleSubmit} className="space-y-4">
        <Input
          leadingIcon={<Phone />}
          label="شماره موبایل"
          type="tel"
          value={mobile}
          onChange={(e) => setMobile(e.target.value)}
          placeholder="09123456789"
          dir="ltr"
          maxLength={11}
          autoFocus
        />

        <Input
          leadingIcon={<Lock />}
          label="رمز عبور"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="••••••••"
          dir="ltr"
        />

        {error && <Alert tone="error">{error}</Alert>}

        <div className="flex justify-end">
          <Link
            to="/forgot-password"
            className="text-caption text-link hover:underline"
          >
            فراموشی رمز عبور؟
          </Link>
        </div>

        <Button type="submit" isLoading={isLoading} className="w-full" size="lg">
          ورود
        </Button>
      </form>

      <div className="mt-6 border-t border-border pt-6 text-center">
        <p className="text-body text-text-subtle">
          حساب کاربری ندارید؟{' '}
          <Link to="/register" className="text-link font-medium hover:underline">
            ثبت‌نام کنید
          </Link>
        </p>
      </div>
    </AuthLayout>
  );
}

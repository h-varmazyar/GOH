import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuthStore } from '../store/authStore';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Alert } from '../components/ui/Alert';
import { AuthLayout } from '../components/layout/AuthLayout';

export function RegisterPage() {
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [username, setUsername] = useState('');
  const [mobile, setMobile] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState('');
  const { register, isLoading, isAuthenticated, user } = useAuthStore();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (!firstName.trim()) return setError('نام را وارد کنید');
    if (!lastName.trim()) return setError('نام خانوادگی را وارد کنید');
    if (username.length < 3) return setError('نام کاربری باید حداقل ۳ کاراکتر باشد');
    if (!/^[a-zA-Z][a-zA-Z0-9_]*$/.test(username)) return setError('نام کاربری فقط شامل حروف انگلیسی، عدد و _ باشد');
    if (!/^09\d{9}$/.test(mobile)) return setError('شماره موبایل باید ۱۱ رقمی و با ۰۹ شروع شود');
    if (password.length < 8) return setError('رمز عبور باید حداقل ۸ کاراکتر باشد');
    if (password !== confirmPassword) return setError('رمز عبور و تکرار آن مطابقت ندارند');

    const result = await register({
      firstName: firstName.trim(),
      lastName: lastName.trim(),
      username: username.trim(),
      mobile,
      password,
    });

    if (result.success) {
      // If server returned a token, user is logged in - go verify phone or dashboard
      if (isAuthenticated || useAuthStore.getState().isAuthenticated) {
        const u = user ?? useAuthStore.getState().user;
        navigate(u && !u.isVerified ? '/verify-phone' : '/dashboard');
      } else {
        // Otherwise redirect to login with a success hint
        navigate('/login');
      }
    } else {
      setError(result.error || 'ثبت‌نام ناموفق');
    }
  };

  return (
    <AuthLayout title="ثبت‌نام در Gate" subtitle="حساب کاربری جدید بسازید" size="md">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="grid gap-3 sm:grid-cols-2">
          <Input
            label="نام"
            value={firstName}
            onChange={(e) => setFirstName(e.target.value)}
            placeholder="نام"
            autoFocus
          />
          <Input
            label="نام خانوادگی"
            value={lastName}
            onChange={(e) => setLastName(e.target.value)}
            placeholder="نام خانوادگی"
          />
        </div>

        <Input
          label="نام کاربری"
          value={username}
          onChange={(e) => setUsername(e.target.value.replace(/[^a-zA-Z0-9_]/g, ''))}
          placeholder="username"
          dir="ltr"
          hint="حروف انگلیسی، عدد و _"
        />

        <Input
          label="شماره موبایل"
          type="tel"
          value={mobile}
          onChange={(e) => setMobile(e.target.value.replace(/\D/g, ''))}
          placeholder="09123456789"
          dir="ltr"
          maxLength={11}
        />

        <Input
          label="رمز عبور"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="••••••••"
          dir="ltr"
          hint="حداقل ۸ کاراکتر"
        />

        <Input
          label="تکرار رمز عبور"
          type="password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          placeholder="••••••••"
          dir="ltr"
        />

        {error && <Alert tone="error">{error}</Alert>}

        <Button type="submit" isLoading={isLoading} className="w-full" size="lg">
          ثبت‌نام
        </Button>
      </form>

      <div className="mt-6 pt-6 border-t border-border text-center">
        <p className="text-body text-text-subtle">
          قبلاً حساب ساخته‌اید؟{' '}
          <Link to="/login" className="text-link font-medium hover:underline">
            ورود
          </Link>
        </p>
      </div>
    </AuthLayout>
  );
}

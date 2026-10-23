import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { LineChart } from 'lucide-react';
import { useAuthStore } from '../store/authStore';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Card } from '../components/ui/Card';

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
    <div className="min-h-screen flex items-center justify-center p-4 bg-gradient-to-br from-surface-50 to-surface-100 dark:from-surface-950 dark:to-surface-900">
      <Card className="w-full max-w-md" padding="lg">
        <div className="flex flex-col items-center mb-6">
          <div className="w-14 h-14 rounded-2xl bg-primary-600 flex items-center justify-center mb-4 shadow-lg shadow-primary-600/30">
            <LineChart className="w-8 h-8 text-white" />
          </div>
          <h1 className="text-xl font-bold text-surface-900 dark:text-white">ثبت‌نام در Gate</h1>
          <p className="text-sm text-surface-500 dark:text-surface-400 mt-1">حساب کاربری جدید بسازید</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
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

          {error && (
            <p className="text-sm text-danger-500 text-center">{error}</p>
          )}

          <Button type="submit" isLoading={isLoading} className="w-full" size="lg">
            ثبت‌نام
          </Button>
        </form>

        <div className="mt-6 pt-6 border-t border-surface-200 dark:border-surface-700 text-center">
          <p className="text-sm text-surface-500 dark:text-surface-400">
            قبلاً حساب ساخته‌اید؟{' '}
            <Link to="/login" className="text-primary-600 dark:text-primary-300 font-medium hover:underline">
              ورود
            </Link>
          </p>
        </div>
      </Card>
    </div>
  );
}

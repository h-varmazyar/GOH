import { useState } from 'react';
import { Link } from 'react-router-dom';
import { LineChart, ArrowRight, CheckCircle2 } from 'lucide-react';
import { useAuthStore } from '../store/authStore';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Card } from '../components/ui/Card';

export function ForgotPasswordPage() {
  const [mobile, setMobile] = useState('');
  const [error, setError] = useState('');
  const [sent, setSent] = useState(false);
  const { forgotPassword, isLoading } = useAuthStore();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (!/^09\d{9}$/.test(mobile)) {
      setError('شماره موبایل باید ۱۱ رقمی و با ۰۹ شروع شود');
      return;
    }

    const result = await forgotPassword(mobile);
    if (result.success) {
      setSent(true);
    } else {
      setError(result.error || 'خطایی رخ داد');
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4 bg-gradient-to-br from-surface-50 to-surface-100 dark:from-surface-950 dark:to-surface-900">
      <Card className="w-full max-w-sm" padding="lg">
        <div className="flex flex-col items-center mb-6">
          <div className="w-14 h-14 rounded-2xl bg-primary-600 flex items-center justify-center mb-4 shadow-lg shadow-primary-600/30">
            <LineChart className="w-8 h-8 text-white" />
          </div>
          <h1 className="text-xl font-bold text-surface-900 dark:text-white">بازیابی رمز عبور</h1>
        </div>

        {sent ? (
          <div className="text-center space-y-4">
            <div className="w-16 h-16 rounded-full bg-success-400/20 mx-auto flex items-center justify-center">
              <CheckCircle2 className="w-8 h-8 text-success-500" />
            </div>
            <p className="text-sm text-surface-700 dark:text-surface-300">
              لینک بازیابی رمز عبور به شماره موبایل شما ارسال شد.
            </p>
            <Link
              to="/login"
              className="inline-flex items-center gap-1 text-sm text-primary-600 dark:text-primary-300 hover:underline"
            >
              <ArrowRight className="w-4 h-4" />
              بازگشت به صفحه ورود
            </Link>
          </div>
        ) : (
          <>
            <p className="text-sm text-surface-500 dark:text-surface-400 mb-5 text-center">
              شماره موبایل خود را وارد کنید تا کد بازیابی برایتان ارسال شود.
            </p>
            <form onSubmit={handleSubmit} className="space-y-4">
              <Input
                label="شماره موبایل"
                type="tel"
                value={mobile}
                onChange={(e) => setMobile(e.target.value.replace(/\D/g, ''))}
                placeholder="09123456789"
                dir="ltr"
                maxLength={11}
                autoFocus
              />

              {error && <p className="text-sm text-danger-500 text-center">{error}</p>}

              <Button type="submit" isLoading={isLoading} className="w-full" size="lg">
                ارسال کد بازیابی
              </Button>
            </form>

            <div className="mt-6 text-center">
              <Link
                to="/login"
                className="inline-flex items-center gap-1 text-sm text-surface-500 hover:text-surface-700 dark:hover:text-surface-300"
              >
                <ArrowRight className="w-4 h-4" />
                بازگشت
              </Link>
            </div>
          </>
        )}
      </Card>
    </div>
  );
}

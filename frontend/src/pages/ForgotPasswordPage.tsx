import { useState } from 'react';
import { Link } from 'react-router-dom';
import { ArrowRight, CheckCircle2 } from 'lucide-react';
import { useAuthStore } from '../store/authStore';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Alert } from '../components/ui/Alert';
import { AuthLayout } from '../components/layout/AuthLayout';

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
    <AuthLayout title="بازیابی رمز عبور">
      {sent ? (
        <div className="text-center space-y-4">
          <div className="w-16 h-16 rounded-pill bg-success/12 mx-auto flex items-center justify-center">
            <CheckCircle2 className="w-8 h-8 text-success" />
          </div>
          <p className="text-body text-text-muted">
            لینک بازیابی رمز عبور به شماره موبایل شما ارسال شد.
          </p>
          <Link
            to="/login"
            className="inline-flex items-center gap-1 text-body text-link hover:underline"
          >
            <ArrowRight className="w-4 h-4" />
            بازگشت به صفحه ورود
          </Link>
        </div>
      ) : (
        <>
          <p className="mb-5 text-body text-text-muted">
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

            {error && <Alert tone="error">{error}</Alert>}

            <Button type="submit" isLoading={isLoading} className="w-full" size="lg">
              ارسال کد بازیابی
            </Button>
          </form>

          <div className="mt-6 text-center">
            <Link
              to="/login"
              className="inline-flex items-center gap-1 text-body text-text-subtle hover:text-text"
            >
              <ArrowRight className="w-4 h-4" />
              بازگشت
            </Link>
          </div>
        </>
      )}
    </AuthLayout>
  );
}

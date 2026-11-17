import { useState, useRef, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowRight } from 'lucide-react';
import { useAuthStore } from '../store/authStore';
import { Button } from '../components/ui/Button';
import { Alert } from '../components/ui/Alert';
import { AuthLayout } from '../components/layout/AuthLayout';

const OTP_LENGTH = 5;

export function VerifyPhonePage() {
  const [otp, setOtp] = useState<string[]>(Array(OTP_LENGTH).fill(''));
  const [error, setError] = useState('');
  const [countdown, setCountdown] = useState(120);
  const inputRefs = useRef<(HTMLInputElement | null)[]>([]);
  const navigate = useNavigate();
  const { verifyPhone, sendPhoneVerification, isLoading, user, logout } = useAuthStore();

  useEffect(() => {
    // Send code on mount
    sendPhoneVerification();
    inputRefs.current[0]?.focus();
  }, [sendPhoneVerification]);

  useEffect(() => {
    if (countdown <= 0) return;
    const t = setInterval(() => setCountdown((c) => c - 1), 1000);
    return () => clearInterval(t);
  }, [countdown]);

  const handleChange = (i: number, v: string) => {
    if (!/^\d*$/.test(v)) return;
    const next = [...otp];
    next[i] = v.slice(-1);
    setOtp(next);
    setError('');
    if (v && i < OTP_LENGTH - 1) inputRefs.current[i + 1]?.focus();
  };

  const handleKeyDown = (i: number, e: React.KeyboardEvent) => {
    if (e.key === 'Backspace' && !otp[i] && i > 0) inputRefs.current[i - 1]?.focus();
  };

  const handlePaste = (e: React.ClipboardEvent) => {
    e.preventDefault();
    const pasted = e.clipboardData.getData('text').replace(/\D/g, '').slice(0, OTP_LENGTH);
    const next = [...otp];
    for (let i = 0; i < pasted.length; i++) next[i] = pasted[i];
    setOtp(next);
    inputRefs.current[Math.min(pasted.length, OTP_LENGTH - 1)]?.focus();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const code = otp.join('');
    if (code.length !== OTP_LENGTH) {
      setError('کد ۵ رقمی را کامل وارد کنید');
      return;
    }
    const result = await verifyPhone(code);
    if (result.success) {
      navigate('/dashboard');
    } else {
      setError(result.error || 'کد وارد شده صحیح نیست');
      setOtp(Array(OTP_LENGTH).fill(''));
      inputRefs.current[0]?.focus();
    }
  };

  const handleResend = async () => {
    if (countdown > 0) return;
    await sendPhoneVerification();
    setCountdown(120);
  };

  const formatTime = (s: number) => {
    const m = Math.floor(s / 60);
    return `${m}:${(s % 60).toString().padStart(2, '0')}`;
  };

  return (
    <AuthLayout
      title="تأیید شماره موبایل"
      subtitle={
        <>
          کد ۵ رقمی ارسال شده به{' '}
          <bdi dir="ltr" className="font-semibold text-text-muted">
            {user?.phone}
          </bdi>{' '}
          را وارد کنید
        </>
      }
    >
      <form onSubmit={handleSubmit} className="space-y-6">
        <div className="flex justify-center gap-2" dir="ltr" onPaste={handlePaste}>
          {otp.map((d, i) => (
            <input
              key={i}
              ref={(el) => { inputRefs.current[i] = el; }}
              type="text"
              inputMode="numeric"
              maxLength={1}
              value={d}
              onChange={(e) => handleChange(i, e.target.value)}
              onKeyDown={(e) => handleKeyDown(i, e)}
              aria-label={`رقم ${(i + 1).toLocaleString('fa-IR')} از ${OTP_LENGTH.toLocaleString('fa-IR')}`}
              aria-invalid={error ? true : undefined}
              className={`h-13 w-11 rounded-md border bg-surface text-center text-h3 text-text tabular-nums transition-colors
                ${error ? 'border-error' : d ? 'border-selected' : 'border-border-strong'}`}
            />
          ))}
        </div>

        {error && <Alert tone="error">{error}</Alert>}

        <Button type="submit" isLoading={isLoading} className="w-full" size="lg">
          تأیید
        </Button>
      </form>

      <div className="mt-6 text-center">
        {countdown > 0 ? (
          <p className="text-body text-text-subtle">
            ارسال مجدد کد تا{' '}
            <span className="font-semibold text-text-muted tabular-nums" dir="ltr">{formatTime(countdown)}</span>
          </p>
        ) : (
          <button
            type="button"
            onClick={handleResend}
            className="min-h-10 px-2 text-body font-semibold text-link hover:underline"
          >
            ارسال مجدد کد
          </button>
        )}
      </div>

      <button
        type="button"
        onClick={() => { logout(); navigate('/login'); }}
        className="mx-auto mt-4 flex min-h-10 items-center gap-1 px-2 text-body text-text-subtle transition-colors hover:text-text"
      >
        <ArrowRight className="h-4 w-4 ltr:rotate-180" aria-hidden="true" />
        خروج
      </button>
    </AuthLayout>
  );
}

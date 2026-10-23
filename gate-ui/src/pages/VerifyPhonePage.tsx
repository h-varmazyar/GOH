import { useState, useRef, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { LineChart, ArrowRight } from 'lucide-react';
import { useAuthStore } from '../store/authStore';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';

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
    <div className="min-h-screen flex items-center justify-center p-4 bg-gradient-to-br from-surface-50 to-surface-100 dark:from-surface-950 dark:to-surface-900">
      <Card className="w-full max-w-sm" padding="lg">
        <div className="flex flex-col items-center mb-8">
          <div className="w-14 h-14 rounded-2xl bg-primary-600 flex items-center justify-center mb-4 shadow-lg shadow-primary-600/30">
            <LineChart className="w-8 h-8 text-white" />
          </div>
          <h1 className="text-xl font-bold text-surface-900 dark:text-white">تأیید شماره موبایل</h1>
          <p className="text-sm text-surface-500 dark:text-surface-400 mt-2 text-center">
            کد ۵ رقمی ارسال شده به{' '}
            <span className="font-medium text-surface-700 dark:text-surface-200" dir="ltr">
              {user?.phone}
            </span>{' '}
            را وارد کنید
          </p>
        </div>

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
                className={`w-11 h-13 text-center text-xl font-bold rounded-lg border transition-all
                  bg-white dark:bg-surface-800 text-surface-900 dark:text-white py-3
                  focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500
                  ${error ? 'border-danger-500' : 'border-surface-300 dark:border-surface-600'}
                  ${d ? 'border-primary-400 dark:border-primary-500' : ''}`}
              />
            ))}
          </div>

          {error && <p className="text-sm text-danger-500 text-center">{error}</p>}

          <Button type="submit" isLoading={isLoading} className="w-full" size="lg">
            تأیید
          </Button>
        </form>

        <div className="mt-6 text-center">
          {countdown > 0 ? (
            <p className="text-sm text-surface-500 dark:text-surface-400">
              ارسال مجدد کد تا{' '}
              <span className="font-medium text-surface-700 dark:text-surface-200">{formatTime(countdown)}</span>
            </p>
          ) : (
            <button
              onClick={handleResend}
              className="text-sm text-primary-600 dark:text-primary-300 hover:underline font-medium"
            >
              ارسال مجدد کد
            </button>
          )}
        </div>

        <button
          type="button"
          onClick={() => { logout(); navigate('/login'); }}
          className="flex items-center gap-1 mx-auto mt-4 text-sm text-surface-500 hover:text-surface-700 dark:hover:text-surface-300 transition-colors"
        >
          <ArrowRight className="w-4 h-4" />
          خروج
        </button>
      </Card>
    </div>
  );
}

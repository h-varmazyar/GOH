import { useEffect, useRef, useState, type FormEvent } from 'react';
import { Bot, Smartphone, CheckCircle, Send, QrCode as QrCodeIcon, ChevronLeft, RefreshCw } from 'lucide-react';
import { Modal } from '../../components/ui/Modal';
import { Button } from '../../components/ui/Button';
import { Input } from '../../components/ui/Input';
import { Alert } from '../../components/ui/Alert';
import { CountryCodeSelect } from '../../components/ui/CountryCodeSelect';
import { QrCode } from '../../components/ui/QrCode';
import { telegramService, type TelegramAccountType } from '../../services/telegramService';
import { defaultCountryCode, type CountryCode } from '../../data/countryCodes';

type FlowStep = 'select-type' | 'phone' | 'bot-token' | 'qr' | 'code' | 'password' | 'done';

const FORM_ID = 'telegram-add-account-form';

/** How often the QR session is polled for a refreshed link or the scan result. */
const QR_POLL_MS = 2000;

const LOGIN_OPTIONS = [
  { type: 'qr', icon: QrCodeIcon, title: 'ورود با QR کد', desc: 'اسکن کد با اپ تلگرام روی گوشی' },
  { type: 'client', icon: Smartphone, title: 'ورود با شماره', desc: 'با شماره موبایل و کد تایید' },
  { type: 'bot', icon: Bot, title: 'ورود با ربات', desc: 'با توکن ربات از @BotFather' },
] as const;

const STEP_LABELS: Record<Exclude<FlowStep, 'done'>, string> = {
  'select-type': 'نوع اکانت',
  phone: 'شماره تلفن',
  'bot-token': 'توکن ربات',
  qr: 'اسکن QR کد',
  code: 'کد تایید',
  password: 'رمز عبور',
};

interface TelegramAddAccountDialogProps {
  onClose: () => void;
  /** Called once the account is connected, so the caller can refresh its list. */
  onAdded?: () => void;
}

export function TelegramAddAccountDialog({ onClose, onAdded }: TelegramAddAccountDialogProps) {
  const [currentStep, setCurrentStep] = useState<FlowStep>('select-type');
  const [sessionType, setSessionType] = useState<TelegramAccountType | null>(null);
  const [sessionId, setSessionId] = useState('');
  const [selectedCountry, setSelectedCountry] = useState<CountryCode>(defaultCountryCode);
  const [formData, setFormData] = useState({
    phone: '',
    botToken: '',
    code: '',
    password: '',
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [qrLink, setQrLink] = useState('');
  /** Set when the QR session ended without a scan (expired or failed). */
  const [qrExpired, setQrExpired] = useState(false);

  // Keep the latest callback for the polling effect without restarting it on every parent render
  const onAddedRef = useRef(onAdded);
  useEffect(() => {
    onAddedRef.current = onAdded;
  });

  // QR login has no request that waits for the scan, so poll the session state
  useEffect(() => {
    if (currentStep !== 'qr' || !sessionId || qrExpired) return;
    let stopped = false;
    const timer = window.setInterval(async () => {
      try {
        const res = await telegramService.getSessionState(sessionId);
        if (stopped) return;
        const step = res.NextStep.toLowerCase();
        if (step === 'wait_qr_scan') {
          if (res.QrLink) setQrLink(res.QrLink);
        } else if (step === 'wait_password') {
          setCurrentStep('password');
        } else if (step === 'ready') {
          setCurrentStep('done');
          onAddedRef.current?.();
        } else {
          setQrExpired(true);
        }
      } catch {
        // The backend drops the session once it expires or fails
        if (!stopped) setQrExpired(true);
      }
    }, QR_POLL_MS);
    return () => {
      stopped = true;
      window.clearInterval(timer);
    };
  }, [currentStep, sessionId, qrExpired]);

  /** Maps the backend's next_step (tgclient.SessionState) to a dialog step. */
  const navigateToNextStep = (nextStep: string | undefined) => {
    const step = (nextStep ?? '').toLowerCase();
    const target: FlowStep | null =
      step === 'wait_bot_token' ? 'bot-token'
      : step === 'wait_phone' ? 'phone'
      : step === 'wait_qr_scan' ? 'qr'
      : step === 'wait_code' ? 'code'
      : step === 'wait_password' ? 'password'
      : step === 'ready' ? 'done'
      : null;

    if (!target) {
      setError(`پاسخ نامشخص از سرور (${nextStep || 'خالی'}). لطفاً دوباره تلاش کنید.`);
      return;
    }
    // Telegram asking again for the same input means the value was not accepted
    if (target === currentStep) {
      setError(`${STEP_LABELS[target as Exclude<FlowStep, 'done'>]} پذیرفته نشد. دوباره وارد کنید.`);
      return;
    }
    setCurrentStep(target);
    if (target === 'done') onAdded?.();
  };

  /** Runs one backend step with shared loading and error handling. */
  const runStep = async (run: () => Promise<void>) => {
    setIsLoading(true);
    setError('');
    try {
      await run();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'خطای ناشناخته');
    } finally {
      setIsLoading(false);
    }
  };

  const handleSelectType = (type: TelegramAccountType) => {
    setSessionType(type);
    setError('');
    // Bot sessions are created on token submit, so a rejected token can simply be retried
    if (type === 'bot') {
      setCurrentStep('bot-token');
      return;
    }
    return runStep(async () => {
      const res = await telegramService.addAccount(type);
      if (res.Failed) {
        setError('خطا در ایجاد نشست. لطفاً دوباره تلاش کنید.');
        return;
      }
      setSessionId(res.SessionId);
      setQrLink(res.QrLink);
      setQrExpired(false);
      navigateToNextStep(res.NextStep);
    });
  };

  const handleSubmitPhone = () =>
    runStep(async () => {
      const fullPhone = selectedCountry.dialCode + formData.phone.replace(/^0+/, '');
      const res = await telegramService.submitPhone(sessionId, fullPhone);
      if (res.Failed) {
        setError('شماره تلفن نامعتبر است یا خطایی رخ داده.');
        return;
      }
      setSessionId(res.SessionId);
      navigateToNextStep(res.NextStep);
    });

  const handleSubmitBotToken = () =>
    runStep(async () => {
      // A failed token closes the backend session, so start a fresh one on every attempt
      const session = await telegramService.addAccount('bot');
      if (session.Failed) {
        setError('خطا در ایجاد نشست. لطفاً دوباره تلاش کنید.');
        return;
      }
      const res = await telegramService.submitBotToken(session.SessionId, formData.botToken.trim());
      if (res.Failed) {
        setError('توکن ربات نامعتبر است.');
        return;
      }
      setSessionId(res.SessionId);
      navigateToNextStep(res.NextStep);
    });

  const handleSubmitCode = () =>
    runStep(async () => {
      const res = await telegramService.submitOtp(sessionId, formData.code);
      if (res.Failed) {
        setError('کد وارد شده نادرست است.');
        return;
      }
      setSessionId(res.SessionId);
      navigateToNextStep(res.NextStep);
    });

  const handleSubmitPassword = () =>
    runStep(async () => {
      const res = await telegramService.submitPassword(sessionId, formData.password);
      if (res.Failed) {
        setError('رمز عبور نادرست است.');
        return;
      }
      setSessionId(res.SessionId);
      navigateToNextStep(res.NextStep);
    });

  const canProceed = () => {
    if (currentStep === 'phone') return formData.phone.length >= 10;
    // Bot tokens look like "<bot_id>:<secret>"
    if (currentStep === 'bot-token') return /^\d+:[\w-]{20,}$/.test(formData.botToken.trim());
    if (currentStep === 'code') return formData.code.length >= 4;
    if (currentStep === 'password') return formData.password.length > 0;
    return false;
  };

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (!canProceed() || isLoading) return;
    switch (currentStep) {
      case 'phone':
        return handleSubmitPhone();
      case 'bot-token':
        return handleSubmitBotToken();
      case 'code':
        return handleSubmitCode();
      case 'password':
        return handleSubmitPassword();
    }
  };

  const handleBack = () => {
    setError('');
    // QR has no phone/code steps, so going back from its password step restarts the flow
    if (
      currentStep === 'phone' ||
      currentStep === 'bot-token' ||
      currentStep === 'qr' ||
      (currentStep === 'password' && sessionType === 'qr')
    ) {
      setCurrentStep('select-type');
      setSessionType(null);
      setSessionId('');
      setQrLink('');
    } else if (currentStep === 'code') {
      setCurrentStep('phone');
    } else if (currentStep === 'password') {
      setCurrentStep('code');
    }
  };

  const stepOrder: FlowStep[] =
    sessionType === 'bot'
      ? ['select-type', 'bot-token']
      : sessionType === 'qr'
        ? ['select-type', 'qr', 'password']
        : ['select-type', 'phone', 'code', 'password'];

  const renderProgress = () => {
    if (currentStep === 'done' || currentStep === 'select-type') return null;
    const index = stepOrder.indexOf(currentStep);
    return (
      <div className="space-y-2">
        <div className="flex gap-1" aria-hidden="true">
          {stepOrder.map((step, i) => (
            <div
              key={step}
              className={`h-1 flex-1 rounded-pill transition-colors ${
                i < index ? 'bg-success' : i === index ? 'bg-action' : 'bg-border'
              }`}
            />
          ))}
        </div>
        <p className="text-caption text-text-subtle">
          مرحله {index + 1} از {stepOrder.length} · <span className="text-text">{STEP_LABELS[currentStep]}</span>
        </p>
      </div>
    );
  };

  const renderFooter = () => {
    if (currentStep === 'done') {
      return (
        <Button className="flex-1" onClick={onClose} autoFocus>
          بستن
        </Button>
      );
    }
    if (currentStep === 'select-type') {
      return (
        <Button variant="ghost" className="ms-auto" onClick={onClose}>
          انصراف
        </Button>
      );
    }
    if (currentStep === 'qr') {
      return (
        <Button variant="ghost" className="ms-auto" onClick={handleBack}>
          بازگشت
        </Button>
      );
    }
    return (
      <>
        <Button
          type="submit"
          form={FORM_ID}
          className="flex-1"
          disabled={!canProceed()}
          isLoading={isLoading}
        >
          {currentStep === 'password' || currentStep === 'bot-token' ? 'افزودن' : 'مرحله بعد'}
        </Button>
        <Button variant="ghost" onClick={handleBack} disabled={isLoading}>
          بازگشت
        </Button>
      </>
    );
  };

  return (
    <Modal
      title="افزودن اکانت تلگرام"
      icon={<Send className="h-5 w-5 text-selected" aria-hidden="true" />}
      onClose={onClose}
      footer={renderFooter()}
    >
      {renderProgress()}

      {error && <Alert tone="error">{error}</Alert>}

      {/* Step: Select Account Type */}
      {currentStep === 'select-type' && (
        <div className="space-y-3">
          <p className="text-body text-text-muted">نوع ورود به اکانت تلگرام را انتخاب کنید</p>
          <div className="space-y-2">
            {LOGIN_OPTIONS.map(({ type, icon: Icon, title, desc }) => (
              <button
                key={type}
                type="button"
                onClick={() => handleSelectType(type)}
                disabled={isLoading}
                aria-busy={isLoading && sessionType === type ? true : undefined}
                className="group flex w-full items-center gap-3 rounded-lg border border-border-strong p-3 text-start transition-colors hover:border-selected hover:bg-selected/5 disabled:opacity-50"
              >
                <div className="shrink-0 rounded-md bg-selected/10 p-2.5 transition-colors group-hover:bg-selected/15">
                  <Icon className="h-5 w-5 text-link" aria-hidden="true" />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-body font-semibold text-text">{title}</p>
                  <p className="text-caption text-text-subtle">{desc}</p>
                </div>
                <ChevronLeft className="h-4 w-4 shrink-0 text-text-subtle ltr:rotate-180" aria-hidden="true" />
              </button>
            ))}
          </div>
          {isLoading && (
            <p className="text-center text-caption text-text-subtle">در حال ایجاد نشست...</p>
          )}
        </div>
      )}

      {/* Step: QR code */}
      {currentStep === 'qr' && (
        <div className="flex flex-col items-center gap-4">
          <div className="relative rounded-lg border border-border p-2">
            {qrLink ? (
              <QrCode
                value={qrLink}
                label="QR کد ورود به تلگرام"
                className={qrExpired ? 'opacity-15 blur-[2px]' : ''}
              />
            ) : (
              <div className="h-52 w-52 animate-pulse rounded-md bg-surface-muted" />
            )}
            {qrExpired && (
              <div className="absolute inset-0 flex items-center justify-center">
                <Button size="sm" onClick={() => handleSelectType('qr')} isLoading={isLoading}>
                  <RefreshCw className="h-4 w-4" aria-hidden="true" />
                  ساخت کد جدید
                </Button>
              </div>
            )}
          </div>
          {qrExpired ? (
            <p className="text-body text-error" role="alert">
              کد منقضی شد یا ورود ناموفق بود.
            </p>
          ) : (
            <ol className="w-full list-decimal space-y-1 ps-5 text-body text-text-muted">
              <li>تلگرام را روی گوشی باز کنید.</li>
              <li>
                به <span className="text-text">Settings ← Devices ← Link Desktop Device</span> بروید.
              </li>
              <li>این کد را اسکن کنید. کد خودکار به‌روز می‌شود.</li>
            </ol>
          )}
        </div>
      )}

      {currentStep !== 'select-type' && currentStep !== 'qr' && currentStep !== 'done' && (
        <form id={FORM_ID} onSubmit={handleSubmit} noValidate>
          {/* Step: Phone Number */}
          {currentStep === 'phone' && (
            <div>
              <label htmlFor="telegram-phone" className="mb-1.5 block text-label text-text">
                شماره تلفن
              </label>
              {/* Dial code + number on one row; LTR so the code sits before the number */}
              <div className="flex items-start gap-2" dir="ltr">
                <div className="w-32 shrink-0">
                  <CountryCodeSelect value={selectedCountry} onChange={setSelectedCountry} />
                </div>
                <div className="min-w-0 flex-1">
                  <input
                    id="telegram-phone"
                    type="tel"
                    inputMode="numeric"
                    autoComplete="tel-national"
                    autoFocus
                    placeholder="9123456789"
                    maxLength={13}
                    value={formData.phone}
                    onChange={(e) =>
                      setFormData({ ...formData, phone: e.target.value.replace(/\D/g, '') })
                    }
                    className="field tabular-nums tracking-wide"
                  />
                </div>
              </div>
              <p className="mt-1.5 text-caption text-text-subtle">
                شماره تلفن خود را بدون صفر ابتدایی وارد کنید
              </p>
            </div>
          )}

          {/* Step: Bot Token */}
          {currentStep === 'bot-token' && (
            <Input
              label="توکن ربات"
              type="text"
              placeholder="123456789:AAHdqTcvCH1vGWJx..."
              value={formData.botToken}
              onChange={(e) => setFormData({ ...formData, botToken: e.target.value })}
              dir="ltr"
              autoComplete="off"
              autoFocus
              hint="توکن دریافت‌شده از @BotFather را وارد کنید"
            />
          )}

          {/* Step: OTP Code */}
          {currentStep === 'code' && (
            <Input
              label="کد تایید"
              type="text"
              inputMode="numeric"
              autoComplete="one-time-code"
              placeholder="12345"
              value={formData.code}
              onChange={(e) => setFormData({ ...formData, code: e.target.value.replace(/\D/g, '') })}
              dir="ltr"
              maxLength={6}
              autoFocus
              className="tabular-nums tracking-[0.3em]"
              hint="کد ارسال شده به تلگرام را وارد کنید"
            />
          )}

          {/* Step: 2FA Password */}
          {currentStep === 'password' && (
            <Input
              label="رمز عبور دو مرحله‌ای"
              type="password"
              placeholder="••••••••"
              value={formData.password}
              onChange={(e) => setFormData({ ...formData, password: e.target.value })}
              dir="ltr"
              autoComplete="current-password"
              autoFocus
              hint="رمز دو مرحله‌ای تلگرام خود را وارد کنید"
            />
          )}
        </form>
      )}

      {/* Success */}
      {currentStep === 'done' && (
        <div className="py-4 text-center">
          <CheckCircle className="mx-auto mb-3 h-12 w-12 text-success" aria-hidden="true" />
          <p className="text-body-lg font-semibold text-text">اکانت با موفقیت اضافه شد</p>
          <p className="mt-1 text-body text-text-subtle">
            اکانت تلگرام شما متصل شد و آماده استفاده است.
          </p>
        </div>
      )}
    </Modal>
  );
}

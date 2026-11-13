import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowLeft, Bot, Smartphone, CheckCircle, ChevronDown } from 'lucide-react';
import { Card } from '../../components/ui/Card';
import { Button } from '../../components/ui/Button';
import { Input } from '../../components/ui/Input';
import { telegramService, type TelegramAccountType } from '../../services/telegramService';
import { countryCodes, defaultCountryCode, type CountryCode } from '../../data/countryCodes';

type FlowStep = 'select-type' | 'phone' | 'bot-token' | 'code' | 'password' | 'done';

export function TelegramAddAccount() {
  const navigate = useNavigate();
  const [currentStep, setCurrentStep] = useState<FlowStep>('select-type');
  const [sessionType, setSessionType] = useState<TelegramAccountType | null>(null);
  const [sessionId, setSessionId] = useState('');
  const [selectedCountry, setSelectedCountry] = useState<CountryCode>(defaultCountryCode);
  const [showCountryDropdown, setShowCountryDropdown] = useState(false);
  const [countrySearch, setCountrySearch] = useState('');
  const [formData, setFormData] = useState({
    phone: '',
    botToken: '',
    code: '',
    password: '',
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSelectType = async (type: TelegramAccountType) => {
    setSessionType(type);
    // Bot sessions are created on token submit, so a rejected token can simply be retried
    if (type === 'bot') {
      setError('');
      setCurrentStep('bot-token');
      return;
    }
    setIsLoading(true);
    setError('');
    try {
      const res = await telegramService.addAccount(type);
      if (res.Failed) {
        setError('خطا در ایجاد نشست. لطفاً دوباره تلاش کنید.');
        setIsLoading(false);
        return;
      }
      setSessionId(res.SessionId);
      // Always goes to phone step after add-account
      setCurrentStep('phone');
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'خطای ناشناخته');
    } finally {
      setIsLoading(false);
    }
  };

  const handleSubmitPhone = async () => {
    setIsLoading(true);
    setError('');
    try {
      // Prepend country dial code to phone number
      const fullPhone = selectedCountry.dialCode + formData.phone.replace(/^0+/, '');
      const res = await telegramService.submitPhone(sessionId, fullPhone);
      if (res.Failed) {
        setError('شماره تلفن نامعتبر است یا خطایی رخ داده.');
        setIsLoading(false);
        return;
      }
      setSessionId(res.SessionId);
      navigateToNextStep(res.NextStep);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'خطای ناشناخته');
    } finally {
      setIsLoading(false);
    }
  };

  const handleSubmitBotToken = async () => {
    setIsLoading(true);
    setError('');
    try {
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
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'خطای ناشناخته');
    } finally {
      setIsLoading(false);
    }
  };

  const handleSubmitCode = async () => {
    setIsLoading(true);
    setError('');
    try {
      const res = await telegramService.submitOtp(sessionId, formData.code);
      if (res.Failed) {
        setError('کد وارد شده نادرست است.');
        setIsLoading(false);
        return;
      }
      setSessionId(res.SessionId);
      navigateToNextStep(res.NextStep);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'خطای ناشناخته');
    } finally {
      setIsLoading(false);
    }
  };

  const handleSubmitPassword = async () => {
    setIsLoading(true);
    setError('');
    try {
      const res = await telegramService.submitPassword(sessionId, formData.password);
      if (res.Failed) {
        setError('رمز عبور نادرست است.');
        setIsLoading(false);
        return;
      }
      setSessionId(res.SessionId);
      navigateToNextStep(res.NextStep);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'خطای ناشناخته');
    } finally {
      setIsLoading(false);
    }
  };

  const navigateToNextStep = (nextStep: string | undefined) => {
    const step = (nextStep ?? '').toLowerCase();
    if (step.includes('bot_token')) {
      setCurrentStep('bot-token');
    } else if (step.includes('otp') || step.includes('code')) {
      setCurrentStep('code');
    } else if (step.includes('password')) {
      setCurrentStep('password');
    } else {
      // "ready" or empty means done
      setCurrentStep('done');
    }
  };

  const handleAction = () => {
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

  const canProceed = () => {
    if (currentStep === 'phone') return formData.phone.length >= 10;
    // Bot tokens look like "<bot_id>:<secret>"
    if (currentStep === 'bot-token') return /^\d+:[\w-]{20,}$/.test(formData.botToken.trim());
    if (currentStep === 'code') return formData.code.length >= 4;
    if (currentStep === 'password') return formData.password.length > 0;
    return false;
  };

  const getStepIndex = (): number => {
    const stepOrder: FlowStep[] =
      sessionType === 'bot'
        ? ['select-type', 'bot-token', 'done']
        : ['select-type', 'phone', 'code', 'password', 'done'];
    return stepOrder.indexOf(currentStep);
  };

  const renderStepIndicator = () => {
    const steps =
      sessionType === 'bot'
        ? [
            { id: 'select-type', label: 'نوع اکانت' },
            { id: 'bot-token', label: 'توکن ربات' },
          ]
        : [
            { id: 'select-type', label: 'نوع اکانت' },
            { id: 'phone', label: 'شماره تلفن' },
            { id: 'code', label: 'کد تایید' },
            { id: 'password', label: 'رمز عبور' },
          ];

    const currentIndex = getStepIndex();

    return (
      <div className="flex items-center justify-center gap-1 mb-6 flex-wrap">
        {steps.map((step, index) => {
          const isActive = step.id === currentStep;
          const isCompleted = index < currentIndex;

          return (
            <div key={step.id} className="flex items-center">
              <div
                className={`flex items-center justify-center w-7 h-7 rounded-full text-xs font-medium transition-colors
                ${
                  isActive
                    ? 'bg-primary-600 text-white'
                    : isCompleted
                      ? 'bg-success-500 text-white'
                      : 'bg-surface-200 dark:bg-surface-700 text-surface-500 dark:text-surface-400'
                }`}
              >
                {isCompleted ? '✓' : index + 1}
              </div>
              <span
                className={`mr-1 text-xs ${isActive ? 'text-primary-600 font-medium' : 'text-surface-500 dark:text-surface-400'}`}
              >
                {step.label}
              </span>
              {index < steps.length - 1 && (
                <div className="w-6 h-0.5 mx-1 bg-surface-200 dark:bg-surface-700" />
              )}
            </div>
          );
        })}
      </div>
    );
  };

  // Success view
  if (currentStep === 'done') {
    return (
      <div className="space-y-4">
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => navigate('/telegram')}>
            <ArrowLeft className="w-4 h-4" />
          </Button>
          <h2 className="text-lg font-semibold text-surface-900 dark:text-white">افزودن اکانت تلگرام</h2>
        </div>

        <Card className="text-center py-8">
          <CheckCircle className="w-16 h-16 text-success-500 mx-auto mb-4" />
          <h3 className="text-lg font-bold text-surface-900 dark:text-white mb-2">
            اکانت با موفقیت اضافه شد
          </h3>
          <p className="text-sm text-surface-500 dark:text-surface-400 mb-6">
            اکانت تلگرام شما با موفقیت متصل شد و آماده استفاده است.
          </p>
          <Button onClick={() => navigate('/telegram')}>بازگشت به لیست اکانت‌ها</Button>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="sm" onClick={() => navigate('/telegram')}>
          <ArrowLeft className="w-4 h-4" />
        </Button>
        <div>
          <h2 className="text-lg font-semibold text-surface-900 dark:text-white">افزودن اکانت تلگرام</h2>
          <p className="text-sm text-surface-500 dark:text-surface-400 mt-1">
            مراحل اتصال اکانت جدید را تکمیل کنید
          </p>
        </div>
      </div>

      <Card>
        {/* Step Indicator */}
        {renderStepIndicator()}

        {/* Error Message */}
        {error && (
          <div className="mb-4 p-3 rounded-lg bg-danger-50 dark:bg-danger-950/20 border border-danger-200 dark:border-danger-800">
            <p className="text-sm text-danger-700 dark:text-danger-300">{error}</p>
          </div>
        )}

        {/* Step: Select Account Type */}
        {currentStep === 'select-type' && (
          <div className="space-y-4">
            <p className="text-sm text-surface-600 dark:text-surface-400 text-center mb-4">
              نوع ورود به اکانت تلگرام را انتخاب کنید
            </p>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <button
                onClick={() => handleSelectType('bot')}
                disabled={isLoading}
                className="flex flex-col items-center gap-3 p-6 rounded-xl border-2 border-surface-200 dark:border-surface-700 hover:border-primary-500 dark:hover:border-primary-500 transition-colors group disabled:opacity-50"
              >
                <div className="p-4 rounded-full bg-primary-50 dark:bg-primary-950 group-hover:bg-primary-100 dark:group-hover:bg-primary-900 transition-colors">
                  <Bot className="w-8 h-8 text-primary-600 dark:text-primary-400" />
                </div>
                <div className="text-center">
                  <p className="font-semibold text-surface-900 dark:text-white">ورود با ربات</p>
                  <p className="text-xs text-surface-500 dark:text-surface-400 mt-1">
                    اتصال از طریق توکن ربات تلگرام
                  </p>
                </div>
              </button>

              <button
                onClick={() => handleSelectType('client')}
                disabled={isLoading}
                className="flex flex-col items-center gap-3 p-6 rounded-xl border-2 border-surface-200 dark:border-surface-700 hover:border-primary-500 dark:hover:border-primary-500 transition-colors group disabled:opacity-50"
              >
                <div className="p-4 rounded-full bg-primary-50 dark:bg-primary-950 group-hover:bg-primary-100 dark:group-hover:bg-primary-900 transition-colors">
                  <Smartphone className="w-8 h-8 text-primary-600 dark:text-primary-400" />
                </div>
                <div className="text-center">
                  <p className="font-semibold text-surface-900 dark:text-white">ورود با شماره موبایل</p>
                  <p className="text-xs text-surface-500 dark:text-surface-400 mt-1">
                    اتصال با شماره تلفن و کد تایید
                  </p>
                </div>
              </button>
            </div>
            {isLoading && (
              <p className="text-center text-sm text-surface-500 dark:text-surface-400 mt-4">
                در حال ایجاد نشست...
              </p>
            )}
          </div>
        )}

        {/* Step: Phone Number */}
        {currentStep === 'phone' && (
          <div className="space-y-4">
            {/* Country Selector */}
            <div>
              <label className="block text-sm font-medium text-surface-700 dark:text-surface-300 mb-1.5">
                کشور
              </label>
              <div className="relative">
                <button
                  type="button"
                  onClick={() => setShowCountryDropdown(!showCountryDropdown)}
                  className="w-full flex items-center justify-between px-3 py-2 rounded-lg border transition-colors
                    bg-white dark:bg-surface-800
                    border-surface-300 dark:border-surface-600
                    text-surface-900 dark:text-surface-100
                    focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                >
                  <span className="flex items-center gap-2">
                    <span className="text-lg">{selectedCountry.flag}</span>
                    <span>{selectedCountry.name}</span>
                    <span className="text-surface-500 dark:text-surface-400" dir="ltr">
                      ({selectedCountry.dialCode})
                    </span>
                  </span>
                  <ChevronDown className="w-4 h-4 text-surface-400" />
                </button>

                {showCountryDropdown && (
                  <div className="absolute z-50 top-full mt-1 w-full bg-white dark:bg-surface-800 border border-surface-300 dark:border-surface-600 rounded-lg shadow-lg max-h-60 overflow-hidden">
                    <div className="p-2 border-b border-surface-200 dark:border-surface-700">
                      <input
                        type="text"
                        placeholder="جستجوی کشور..."
                        value={countrySearch}
                        onChange={(e) => setCountrySearch(e.target.value)}
                        className="w-full px-2 py-1.5 rounded border border-surface-200 dark:border-surface-600 bg-surface-50 dark:bg-surface-900 text-sm text-surface-900 dark:text-surface-100 placeholder:text-surface-400 focus:outline-none"
                        autoFocus
                      />
                    </div>
                    <div className="overflow-y-auto max-h-48">
                      {countryCodes
                        .filter(
                          (c) =>
                            c.name?.includes(countrySearch) ||
                            c.dialCode?.includes(countrySearch) ||
                            (c.code ?? '').toLowerCase().includes((countrySearch ?? '').toLowerCase())
                        )
                        .map((country) => (
                          <button
                            key={country.code}
                            type="button"
                            onClick={() => {
                              setSelectedCountry(country);
                              setShowCountryDropdown(false);
                              setCountrySearch('');
                            }}
                            className={`w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-surface-100 dark:hover:bg-surface-700 transition-colors
                              ${selectedCountry.code === country.code ? 'bg-primary-50 dark:bg-primary-950' : ''}`}
                          >
                            <span className="text-lg">{country.flag}</span>
                            <span className="text-surface-900 dark:text-surface-100">{country.name}</span>
                            <span className="text-surface-500 dark:text-surface-400 mr-auto" dir="ltr">
                              {country.dialCode}
                            </span>
                          </button>
                        ))}
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* Phone Input with country prefix */}
            <div>
              <label className="block text-sm font-medium text-surface-700 dark:text-surface-300 mb-1.5">
                شماره تلفن
              </label>
              <div className="flex items-center gap-2" dir="ltr">
                <span className="flex-shrink-0 px-3 py-2 rounded-lg border border-surface-300 dark:border-surface-600 bg-surface-50 dark:bg-surface-900 text-sm text-surface-700 dark:text-surface-300 font-medium">
                  {selectedCountry.dialCode}
                </span>
                <input
                  type="tel"
                  placeholder="912 345 6789"
                  value={formData.phone}
                  onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
                  className="flex-1 px-3 py-2 rounded-lg border transition-colors
                    bg-white dark:bg-surface-800
                    border-surface-300 dark:border-surface-600
                    text-surface-900 dark:text-surface-100
                    placeholder:text-surface-400 dark:placeholder:text-surface-500
                    focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>
              <p className="mt-2 text-xs text-surface-500 dark:text-surface-400">
                شماره تلفن خود را بدون صفر ابتدایی وارد کنید
              </p>
            </div>
          </div>
        )}

        {/* Step: Bot Token */}
        {currentStep === 'bot-token' && (
          <div className="space-y-4">
            <Input
              label="توکن ربات"
              type="text"
              placeholder="123456789:AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsaw"
              value={formData.botToken}
              onChange={(e) => setFormData({ ...formData, botToken: e.target.value })}
              dir="ltr"
              autoComplete="off"
            />
            <p className="text-xs text-surface-500 dark:text-surface-400">
              توکن دریافت‌شده از @BotFather را وارد کنید
            </p>
          </div>
        )}

        {/* Step: OTP Code */}
        {currentStep === 'code' && (
          <div className="space-y-4">
            <Input
              label="کد تایید"
              type="text"
              placeholder="12345"
              value={formData.code}
              onChange={(e) => setFormData({ ...formData, code: e.target.value })}
              dir="ltr"
              maxLength={6}
            />
            <p className="text-xs text-surface-500 dark:text-surface-400">
              کد ارسال شده به تلگرام را وارد کنید
            </p>
          </div>
        )}

        {/* Step: 2FA Password */}
        {currentStep === 'password' && (
          <div className="space-y-4">
            <Input
              label="رمز عبور دو مرحله‌ای"
              type="password"
              placeholder="••••••••"
              value={formData.password}
              onChange={(e) => setFormData({ ...formData, password: e.target.value })}
              dir="ltr"
            />
            <p className="text-xs text-surface-500 dark:text-surface-400">
              رمز دو مرحله‌ای تلگرام خود را وارد کنید
            </p>
          </div>
        )}

        {/* Actions (for steps after type selection) */}
        {currentStep !== 'select-type' && (
          <div className="flex gap-2 mt-6">
            <Button
              variant="secondary"
              className="flex-1"
              onClick={() => {
                setError('');
                if (currentStep === 'phone' || currentStep === 'bot-token') {
                  setCurrentStep('select-type');
                  setSessionType(null);
                  setSessionId('');
                } else if (currentStep === 'code') {
                  setCurrentStep('phone');
                } else if (currentStep === 'password') {
                  setCurrentStep('code');
                }
              }}
            >
              بازگشت
            </Button>
            <Button
              className="flex-1"
              onClick={handleAction}
              disabled={!canProceed()}
              isLoading={isLoading}
            >
              {currentStep === 'password' || currentStep === 'bot-token' ? 'افزودن' : 'مرحله بعد'}
            </Button>
          </div>
        )}
      </Card>
    </div>
  );
}

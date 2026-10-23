import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import { CheckCircle, AlertCircle } from 'lucide-react';
import { Card } from '../../components/ui/Card';
import { Button } from '../../components/ui/Button';
import { Input } from '../../components/ui/Input';
import { Badge } from '../../components/ui/Badge';

type AuthStep = 'code' | 'password';

export function TelegramAuth() {
  const { accountId } = useParams<{ accountId: string }>();
  const navigate = useNavigate();
  const [currentStep, setCurrentStep] = useState<AuthStep>('code');
  const [formData, setFormData] = useState({
    code: '',
    password: '',
  });
  const [isLoading, setIsLoading] = useState(false);

  // Mock account data
  const account = {
    id: accountId || '2',
    username: 'john_doe',
    type: 'personal' as const,
    phone: '+989123456789',
  };

  const handleNext = () => {
    if (currentStep === 'code' && formData.code) {
      setCurrentStep('password');
    } else if (currentStep === 'password') {
      handleSubmit();
    }
  };

  const handleSubmit = () => {
    setIsLoading(true);
    // Simulate API call
    setTimeout(() => {
      setIsLoading(false);
      navigate('/telegram');
    }, 1500);
  };

  const canProceed = () => {
    if (currentStep === 'code') return formData.code.length >= 4;
    if (currentStep === 'password') return formData.password.length > 0;
    return false;
  };

  const renderStepIndicator = () => {
    const steps = [
      { id: 'code', label: 'کد تایید' },
      { id: 'password', label: 'رمز عبور' },
    ];

    return (
      <div className="flex items-center justify-center gap-2 mb-6">
        {steps.map((step, index) => {
          const isActive = step.id === currentStep;
          const isCompleted = currentStep === 'password' && index === 0;
          
          return (
            <div key={step.id} className="flex items-center">
              <div className={`flex items-center justify-center w-8 h-8 rounded-full text-sm font-medium transition-colors
                ${isActive ? 'bg-primary-600 text-white' : 
                  isCompleted ? 'bg-success-500 text-white' : 
                  'bg-surface-200 dark:bg-surface-700 text-surface-500 dark:text-surface-400'}`}>
                {isCompleted ? '✓' : index + 1}
              </div>
              <span className={`mr-2 text-sm ${isActive ? 'text-primary-600 font-medium' : 'text-surface-500 dark:text-surface-400'}`}>
                {step.label}
              </span>
              {index < steps.length - 1 && (
                <div className="w-8 h-0.5 mx-2 bg-surface-200 dark:bg-surface-700" />
              )}
            </div>
          );
        })}
      </div>
    );
  };

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="sm" onClick={() => navigate('/telegram')}>
          <ArrowLeft className="w-4 h-4" />
        </Button>
        <div>
          <h2 className="text-lg font-semibold text-surface-900 dark:text-white">احراز هویت اکانت</h2>
          <p className="text-sm text-surface-500 dark:text-surface-400 mt-1">
            تکمیل احراز هویت برای @{account.username}
          </p>
        </div>
      </div>

      {/* Account Info */}
      <Card className="bg-primary-50 dark:bg-primary-950/20 border-primary-200 dark:border-primary-800">
        <div className="flex items-center gap-3">
          <AlertCircle className="w-5 h-5 text-primary-600" />
          <div>
            <p className="text-sm font-medium text-primary-900 dark:text-primary-100">
              این اکانت نیاز به احراز هویت دارد
            </p>
            <p className="text-xs text-primary-700 dark:text-primary-300 mt-0.5">
              نوع اکانت: {account.type === 'bot' ? 'ربات' : 'حساب شخصی'}
            </p>
          </div>
        </div>
      </Card>

      <Card>
        {/* Step Indicator */}
        {renderStepIndicator()}

        {/* Form Steps */}
        <div className="space-y-4">
          {currentStep === 'code' && (
            <div>
              <Input
                label="کد تایید"
                type="text"
                placeholder="12345"
                value={formData.code}
                onChange={(e) => setFormData({ ...formData, code: e.target.value })}
                dir="ltr"
                maxLength={5}
              />
              <p className="mt-2 text-xs text-surface-500 dark:text-surface-400">
                کد ارسال شده به تلگرام را وارد کنید
                <br />
                <span className="text-surface-400">شماره: </span>
                <span dir="ltr">{account.phone}</span>
              </p>
            </div>
          )}

          {currentStep === 'password' && (
            <div>
              <Input
                label="رمز عبور دو مرحله‌ای"
                type="password"
                placeholder="••••••••"
                value={formData.password}
                onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                dir="ltr"
              />
              <p className="mt-2 text-xs text-surface-500 dark:text-surface-400">
                رمز دو مرحله‌ای تلگرام خود را وارد کنید
              </p>
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="flex gap-2 mt-6">
          {currentStep !== 'code' && (
            <Button 
              variant="secondary" 
              className="flex-1"
              onClick={() => setCurrentStep('code')}
            >
              بازگشت
            </Button>
          )}
          <Button 
            className="flex-1" 
            onClick={handleNext}
            disabled={!canProceed()}
            isLoading={isLoading}
          >
            {currentStep === 'password' ? 'اتمام احراز هویت' : 'مرحله بعد'}
          </Button>
        </div>
      </Card>
    </div>
  );
}

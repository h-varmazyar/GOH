import { useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { ArrowRight, ArrowLeft, Check, LayoutTemplate } from 'lucide-react';
import { Card } from '../components/ui/Card';
import { Button } from '../components/ui/Button';
import { useStrategyStore } from '../store/strategyStore';
import { StepBasicInfo } from '../components/strategy/StepBasicInfo';
import { StepIndicators } from '../components/strategy/StepIndicators';
import { StepMarket } from '../components/strategy/StepMarket';
import { StepFilters } from '../components/strategy/StepFilters';
import { StepEntryExit } from '../components/strategy/StepEntryExit';
import { StepRisk } from '../components/strategy/StepRisk';
import { StepExecution } from '../components/strategy/StepExecution';
import { StepAlerts } from '../components/strategy/StepAlerts';
import { StepReview } from '../components/strategy/StepReview';
import { strategyTemplates } from '../data/strategyDemo';
import { createDefaultStrategy, deepMerge } from '../utils/strategyDefaults';
import type { Strategy } from '../types';

const STEPS = [
  { id: 'basic', label: 'اطلاعات پایه' }, { id: 'indicators', label: 'اندیکاتورها' },
  { id: 'market', label: 'مارکت' }, { id: 'filters', label: 'فیلترها' },
  { id: 'entry-exit', label: 'ورود/خروج' }, { id: 'risk', label: 'ریسک' },
  { id: 'execution', label: 'اجرا' }, { id: 'alerts', label: 'هشدارها' },
  { id: 'review', label: 'بررسی نهایی' },
];

export function CreateStrategyPage() {
  const [searchParams] = useSearchParams();
  const template = useMemo(() => strategyTemplates.find((item) => item.id === searchParams.get('template')), [searchParams]);
  const [currentStep, setCurrentStep] = useState(0);
  const [maxVisitedStep, setMaxVisitedStep] = useState(0);
  const [error, setError] = useState('');
  const [strategy, setStrategy] = useState<Strategy>(() => template ? deepMerge(createDefaultStrategy(), { ...template.preset, templateId: template.id }) : createDefaultStrategy());
  const addStrategy = useStrategyStore((state) => state.addStrategy);
  const navigate = useNavigate();
  const updateStrategy = (updates: Partial<Strategy>) => setStrategy((previous) => ({ ...previous, ...updates }));

  const validateStep = (step: number) => {
    if (step === 0 && (!strategy.name.trim() || !strategy.provider)) return 'نام استراتژی و ارائه‌دهنده الزامی است.';
    if (step === 2 && (!strategy.market.symbols.length || !strategy.market.timeframes.length)) return 'حداقل یک نماد و یک تایم‌فریم انتخاب کنید.';
    return '';
  };
  const goNext = () => {
    const message = validateStep(currentStep);
    if (message) return setError(message);
    setError('');
    setCurrentStep((step) => Math.min(step + 1, STEPS.length - 1));
    setMaxVisitedStep((step) => Math.max(step, currentStep + 1));
  };
  const submit = () => {
    const basicError = validateStep(0);
    if (basicError) {
      setCurrentStep(0);
      setError(basicError);
      return;
    }
    const marketError = validateStep(2);
    if (marketError) {
      setCurrentStep(2);
      setError(marketError);
      return;
    }
    setError('');
    addStrategy(strategy);
    navigate('/strategies');
  };
  const renderedSteps = [
    <StepBasicInfo strategy={strategy} onChange={updateStrategy} />, <StepIndicators strategy={strategy} onChange={updateStrategy} />,
    <StepMarket strategy={strategy} onChange={updateStrategy} />, <StepFilters strategy={strategy} onChange={updateStrategy} />,
    <StepEntryExit strategy={strategy} onChange={updateStrategy} />, <StepRisk strategy={strategy} onChange={updateStrategy} />,
    <StepExecution strategy={strategy} onChange={updateStrategy} />, <StepAlerts strategy={strategy} onChange={updateStrategy} />,
    <StepReview strategy={strategy} />,
  ];

  return <div className="space-y-6 max-w-5xl mx-auto" dir="rtl">
    <div><h1 className="text-2xl font-bold text-surface-900 dark:text-white">ایجاد استراتژی جدید</h1><p className="text-sm text-surface-500 mt-1">مرحله {currentStep + 1} از {STEPS.length}: {STEPS[currentStep].label}</p></div>
    {template && <div className="flex items-center gap-2 p-3 rounded-lg bg-primary-50 dark:bg-primary-950/30 text-primary-700 dark:text-primary-300 border border-primary-200 dark:border-primary-800"><LayoutTemplate className="w-5 h-5" />قالب «{template.name}» اعمال شده است؛ همه تنظیمات قابل ویرایش‌اند.</div>}
    <div className="overflow-x-auto pb-2"><div className="flex items-center gap-1 min-w-max">{STEPS.map((step, index) => <button key={step.id} disabled={index > maxVisitedStep} onClick={() => index <= maxVisitedStep && setCurrentStep(index)} className={`flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium ${index === currentStep ? 'bg-primary-600 text-white' : index <= maxVisitedStep ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300' : 'bg-surface-100 text-surface-400 dark:bg-surface-800'}`}>{index < currentStep ? <Check className="w-3 h-3" /> : <span className="w-4 h-4 rounded-full border border-current flex items-center justify-center text-[10px]">{index + 1}</span>}<span className="hidden sm:inline">{step.label}</span></button>)}</div></div>
    <Card padding="lg">{renderedSteps[currentStep]}</Card>
    {error && <p className="text-sm text-danger-500 text-center">{error}</p>}
    <div className="flex items-center justify-between"><Button variant="secondary" onClick={() => { setError(''); setCurrentStep((step) => Math.max(0, step - 1)); }} disabled={currentStep === 0}><ArrowRight className="w-4 h-4" />مرحله قبل</Button>{currentStep === STEPS.length - 1 ? <Button onClick={submit}><Check className="w-4 h-4" />ایجاد استراتژی</Button> : <Button onClick={goNext}>مرحله بعد<ArrowLeft className="w-4 h-4" /></Button>}</div>
  </div>;
}

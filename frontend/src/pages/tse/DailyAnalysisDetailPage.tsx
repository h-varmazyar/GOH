import { useNavigate, useParams } from 'react-router-dom';
import { ArrowRight, BarChart3, CircleDollarSign, Info, TrendingDown, TrendingUp, Users } from 'lucide-react';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import { analyses } from './demoData';

export function DailyAnalysisDetailPage() {
  const { analysisId } = useParams<{ analysisId: string }>();
  const navigate = useNavigate();
  const analysis = analyses.find((item) => item.id === analysisId);

  if (!analysis) return <Card><p className="text-sm text-danger-500">تحلیل مورد نظر پیدا نشد.</p><Button className="mt-4" variant="secondary" onClick={() => navigate('/tse/analysis')}>بازگشت به آرشیو</Button></Card>;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3"><Button variant="ghost" size="sm" onClick={() => navigate('/tse/analysis')}><ArrowRight className="h-4 w-4" />بازگشت</Button><div><div className="flex items-center gap-2"><h1 className="text-xl font-bold text-surface-900 dark:text-white">تحلیل بازار {analysis.date}</h1><Badge variant={analysis.mood === 'صعودی' ? 'success' : analysis.mood === 'نزولی' ? 'danger' : 'warning'}>{analysis.mood}</Badge></div><p className="mt-1 text-xs text-surface-500">{analysis.weekday} · داده‌های پایان جلسه معاملاتی</p></div></div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric icon={BarChart3} label="شاخص کل" value={analysis.totalIndex.toLocaleString('fa-IR')} change={analysis.totalIndexChange} />
        <Metric icon={Users} label="شاخص هم‌وزن" value={analysis.indices[1].value.toLocaleString('fa-IR')} change={analysis.equalIndexChange} />
        <Metric icon={CircleDollarSign} label="ارزش معاملات خرد" value={`${analysis.tradeValue.toLocaleString('fa-IR')} همت`} change={18.2} />
        <Metric icon={analysis.netMoney >= 0 ? TrendingUp : TrendingDown} label="خالص پول حقیقی" value={`${Math.abs(analysis.netMoney).toLocaleString('fa-IR')} میلیارد`} change={analysis.netMoney >= 0 ? 12.5 : -12.5} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[1.35fr_.65fr]">
        <div className="space-y-6">
          <Card><div className="mb-3 flex items-center gap-2"><Info className="h-5 w-5 text-primary-600" /><h2 className="font-semibold text-surface-900 dark:text-white">جمع‌بندی بازار</h2></div><p className="text-sm leading-8 text-surface-600 dark:text-surface-300">{analysis.summary}</p><div className="mt-5 grid grid-cols-2 gap-3"><Stat label="نمادهای مثبت" value={analysis.advancing} positive /><Stat label="نمادهای منفی" value={analysis.declining} /></div></Card>
          <Card padding="none"><div className="border-b border-surface-200 p-4 dark:border-surface-700"><h2 className="font-semibold text-surface-900 dark:text-white">وضعیت شاخص‌های منتخب</h2></div><div className="divide-y divide-surface-100 dark:divide-surface-700">{analysis.indices.map((index) => <div key={index.name} className="grid grid-cols-3 items-center px-4 py-3 text-sm"><span className="font-medium">{index.name}</span><span className="text-center" dir="ltr">{index.value.toLocaleString('fa-IR')}</span><span className={`text-left ${index.change >= 0 ? 'text-success-600' : 'text-danger-500'}`} dir="ltr">{index.change >= 0 ? '+' : ''}{index.change.toLocaleString('fa-IR')}٪</span></div>)}</div></Card>
        </div>
        <div className="space-y-6"><Candidates title="۵ گزینه مناسب خرید فردا" items={analysis.buyCandidates} positive /><Candidates title="۵ گزینه نیازمند احتیاط/فروش" items={analysis.sellCandidates} /></div>
      </div>

      <Card><h2 className="font-semibold text-surface-900 dark:text-white">چشم‌انداز جلسه بعد</h2><p className="mt-3 text-sm leading-7 text-surface-600 dark:text-surface-300">با توجه به بهبود ارزش معاملات و برتری تقاضا، شروع متعادل رو به مثبت محتمل است. ورود پله‌ای، رعایت حد ضرر و پرهیز از خرید در صف توصیه می‌شود. این داده‌ها صرفاً دمو و ابزار تصمیم‌یار هستند و پیشنهاد سرمایه‌گذاری محسوب نمی‌شوند.</p></Card>
    </div>
  );
}

function Metric({ icon: Icon, label, value, change }: { icon: typeof BarChart3; label: string; value: string; change: number }) {
  return <Card padding="sm"><div className="flex items-start justify-between"><div><p className="text-xs text-surface-500">{label}</p><p className="mt-2 text-lg font-bold text-surface-900 dark:text-white" dir="ltr">{value}</p><p className={`mt-1 text-xs ${change >= 0 ? 'text-success-600' : 'text-danger-500'}`} dir="ltr">{change >= 0 ? '▲ +' : '▼ '}{change.toLocaleString('fa-IR')}٪</p></div><span className="rounded-lg bg-primary-50 p-2 text-primary-600 dark:bg-primary-950"><Icon className="h-5 w-5" /></span></div></Card>;
}

function Stat({ label, value, positive = false }: { label: string; value: number; positive?: boolean }) {
  return <div className={`rounded-lg p-3 ${positive ? 'bg-success-400/10' : 'bg-danger-400/10'}`}><p className={`text-xl font-bold ${positive ? 'text-success-600' : 'text-danger-500'}`}>{value.toLocaleString('fa-IR')}</p><p className="mt-1 text-xs text-surface-500">{label}</p></div>;
}

function Candidates({ title, items, positive = false }: { title: string; items: typeof analyses[number]['buyCandidates']; positive?: boolean }) {
  return <Card padding="none"><div className={`border-b p-4 ${positive ? 'border-success-400/30 bg-success-400/10' : 'border-danger-400/30 bg-danger-400/10'}`}><h2 className={`font-semibold ${positive ? 'text-success-600' : 'text-danger-500'}`}>{title}</h2></div><div className="divide-y divide-surface-100 dark:divide-surface-700">{items.map((item, index) => <div key={item.symbol} className="flex items-start gap-3 p-3"><span className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-bold ${positive ? 'bg-success-400/20 text-success-600' : 'bg-danger-400/20 text-danger-500'}`}>{index + 1}</span><div className="min-w-0 flex-1"><div className="flex justify-between gap-2"><p className="text-sm font-bold text-surface-900 dark:text-white">{item.symbol}</p><span className="text-xs text-surface-500">{item.score}/۱۰۰</span></div><p className="mt-1 text-xs text-surface-500">{item.reason}</p></div></div>)}</div></Card>;
}

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

  if (!analysis) return <Card><p className="text-body text-error">تحلیل مورد نظر پیدا نشد.</p><Button className="mt-4" variant="secondary" onClick={() => navigate('/tse/analysis')}>بازگشت به آرشیو</Button></Card>;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3"><Button variant="ghost" size="sm" onClick={() => navigate('/tse/analysis')}><ArrowRight className="h-4 w-4" />بازگشت</Button><div><div className="flex items-center gap-2"><h1 className="text-h3 font-bold text-text">تحلیل بازار {analysis.date}</h1><Badge variant={analysis.mood === 'صعودی' ? 'success' : analysis.mood === 'نزولی' ? 'danger' : 'warning'}>{analysis.mood}</Badge></div><p className="mt-1 text-caption text-text-subtle">{analysis.weekday} · داده‌های پایان جلسه معاملاتی</p></div></div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric icon={BarChart3} label="شاخص کل" value={analysis.totalIndex.toLocaleString('fa-IR')} change={analysis.totalIndexChange} />
        <Metric icon={Users} label="شاخص هم‌وزن" value={analysis.indices[1].value.toLocaleString('fa-IR')} change={analysis.equalIndexChange} />
        <Metric icon={CircleDollarSign} label="ارزش معاملات خرد" value={`${analysis.tradeValue.toLocaleString('fa-IR')} همت`} change={18.2} />
        <Metric icon={analysis.netMoney >= 0 ? TrendingUp : TrendingDown} label="خالص پول حقیقی" value={`${Math.abs(analysis.netMoney).toLocaleString('fa-IR')} میلیارد`} change={analysis.netMoney >= 0 ? 12.5 : -12.5} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[1.35fr_.65fr]">
        <div className="space-y-6">
          <Card><div className="mb-3 flex items-center gap-2"><Info className="h-5 w-5 text-link" /><h2 className="font-semibold text-text">جمع‌بندی بازار</h2></div><p className="text-body leading-8 text-text-muted">{analysis.summary}</p><div className="mt-5 grid grid-cols-2 gap-3"><Stat label="نمادهای مثبت" value={analysis.advancing} positive /><Stat label="نمادهای منفی" value={analysis.declining} /></div></Card>
          <Card padding="none"><div className="border-b border-border p-4"><h2 className="font-semibold text-text">وضعیت شاخص‌های منتخب</h2></div><div className="divide-y divide-border">{analysis.indices.map((index) => <div key={index.name} className="grid grid-cols-3 items-center px-4 py-3 text-body"><span className="font-medium">{index.name}</span><span className="text-center" dir="ltr">{index.value.toLocaleString('fa-IR')}</span><span className={`text-start ${index.change >= 0 ? 'text-success' : 'text-error'}`} dir="ltr">{index.change >= 0 ? '+' : ''}{index.change.toLocaleString('fa-IR')}٪</span></div>)}</div></Card>
        </div>
        <div className="space-y-6"><Candidates title="۵ گزینه مناسب خرید فردا" items={analysis.buyCandidates} positive /><Candidates title="۵ گزینه نیازمند احتیاط/فروش" items={analysis.sellCandidates} /></div>
      </div>

      <Card><h2 className="font-semibold text-text">چشم‌انداز جلسه بعد</h2><p className="mt-3 text-body leading-7 text-text-muted">با توجه به بهبود ارزش معاملات و برتری تقاضا، شروع متعادل رو به مثبت محتمل است. ورود پله‌ای، رعایت حد ضرر و پرهیز از خرید در صف توصیه می‌شود. این داده‌ها صرفاً دمو و ابزار تصمیم‌یار هستند و پیشنهاد سرمایه‌گذاری محسوب نمی‌شوند.</p></Card>
    </div>
  );
}

function Metric({ icon: Icon, label, value, change }: { icon: typeof BarChart3; label: string; value: string; change: number }) {
  return <Card padding="sm"><div className="flex items-start justify-between"><div><p className="text-caption text-text-subtle">{label}</p><p className="mt-2 text-h3 font-bold text-text" dir="ltr">{value}</p><p className={`mt-1 text-caption ${change >= 0 ? 'text-success' : 'text-error'}`} dir="ltr">{change >= 0 ? '▲ +' : '▼ '}{change.toLocaleString('fa-IR')}٪</p></div><span className="rounded-md bg-selected/10 p-2 text-link"><Icon className="h-5 w-5" /></span></div></Card>;
}

function Stat({ label, value, positive = false }: { label: string; value: number; positive?: boolean }) {
  return <div className={`rounded-md p-3 ${positive ? 'bg-success/8' : 'bg-error/8'}`}><p className={`text-h3 font-bold ${positive ? 'text-success' : 'text-error'}`}>{value.toLocaleString('fa-IR')}</p><p className="mt-1 text-caption text-text-subtle">{label}</p></div>;
}

function Candidates({ title, items, positive = false }: { title: string; items: typeof analyses[number]['buyCandidates']; positive?: boolean }) {
  return <Card padding="none"><div className={`border-b p-4 ${positive ? 'border-success/40 bg-success/8' : 'border-error/40 bg-error/8'}`}><h2 className={`font-semibold ${positive ? 'text-success' : 'text-error'}`}>{title}</h2></div><div className="divide-y divide-border">{items.map((item, index) => <div key={item.symbol} className="flex items-start gap-3 p-3"><span className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-pill text-caption font-bold ${positive ? 'bg-success/12 text-success' : 'bg-error/12 text-error'}`}>{index + 1}</span><div className="min-w-0 flex-1"><div className="flex justify-between gap-2"><p className="text-body font-bold text-text">{item.symbol}</p><span className="text-caption text-text-subtle">{item.score}/۱۰۰</span></div><p className="mt-1 text-caption text-text-subtle">{item.reason}</p></div></div>)}</div></Card>;
}

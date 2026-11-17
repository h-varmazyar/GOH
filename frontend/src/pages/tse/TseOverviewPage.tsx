import { useNavigate } from 'react-router-dom';
import { Activity, ArrowLeft, BarChart3, CalendarDays, CircleDollarSign, Gauge, TrendingUp } from 'lucide-react';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import { analyses, marketCalendar } from './demoData';

const pulse = [
  { label: 'شاخص کل', value: '۲,۸۴۷,۳۱۰', detail: '۱.۲۴٪', positive: true, icon: BarChart3 },
  { label: 'شاخص هم‌وزن', value: '۸۹۶,۴۲۰', detail: '۰.۷۱٪', positive: true, icon: Activity },
  { label: 'ارزش معاملات خرد', value: '۸.۴ همت', detail: '۲۱٪ بیشتر', positive: true, icon: CircleDollarSign },
  { label: 'ورود پول حقیقی', value: '۳۱۲ میلیارد', detail: 'جریان مثبت', positive: true, icon: TrendingUp },
];

export function TseOverviewPage() {
  const navigate = useNavigate();
  const latest = analyses[0];

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <div className="flex items-center gap-2"><h1 className="text-h1 font-bold text-text">بورس تهران</h1><Badge variant="success">بازار بسته است</Badge></div>
          <p className="mt-1 text-body text-text-subtle">نبض بازار، تحلیل‌ها و ابزارهای تصمیم‌یار در یک نگاه</p>
        </div>
        <span className="text-caption text-text-subtle">آخرین بروزرسانی: امروز، ۱۲:۳۵</span>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {pulse.map((item) => (
          <Card key={item.label} padding="sm">
            <div className="flex items-start justify-between"><div><p className="text-caption text-text-subtle">{item.label}</p><p className="mt-2 text-h3 font-bold text-text" dir="ltr">{item.value}</p></div><span className="rounded-md bg-selected/10 p-2 text-link"><item.icon className="h-5 w-5" /></span></div>
            <p className={`mt-2 text-caption ${item.positive ? 'text-success' : 'text-error'}`} dir="ltr">{item.positive ? '▲' : '▼'} {item.detail}</p>
          </Card>
        ))}
      </div>

      <div className="grid gap-6 xl:grid-cols-[1.25fr_.75fr]">
        <Card padding="none">
          <div className="flex items-center justify-between border-b border-border p-4"><div><h2 className="font-semibold text-text">آخرین تحلیل بازار</h2><p className="mt-1 text-caption text-text-subtle">{latest.date} · {latest.weekday}</p></div><Badge variant="success">{latest.mood}</Badge></div>
          <div className="p-4 sm:p-6"><p className="text-body leading-7 text-text-muted">{latest.summary}</p><div className="mt-5 flex flex-wrap gap-3"><Button size="sm" onClick={() => navigate(`/tse/analysis/${latest.id}`)}>مشاهده تحلیل کامل <ArrowLeft className="h-4 w-4" /></Button><Button size="sm" variant="ghost" onClick={() => navigate('/tse/analysis')}>آرشیو تحلیل‌ها</Button></div></div>
        </Card>
        <Card padding="none">
          <div className="border-b border-border p-4"><h2 className="font-semibold text-text">پهنای بازار</h2></div>
          <div className="p-5"><div className="flex h-3 overflow-hidden rounded-pill bg-surface-muted"><div className="bg-success" style={{ width: '64%' }} /><div className="bg-warning" style={{ width: '8%' }} /><div className="bg-error" style={{ width: '28%' }} /></div><div className="mt-4 grid grid-cols-3 gap-2 text-center"><Breadth label="مثبت" value={latest.advancing} color="text-success" /><Breadth label="متعادل" value={54} color="text-warning-text" /><Breadth label="منفی" value={latest.declining} color="text-error" /></div></div>
        </Card>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card padding="none"><div className="flex items-center justify-between border-b border-border p-4"><h2 className="font-semibold text-text">برترین فرصت‌های فردا</h2><Button variant="ghost" size="sm" onClick={() => navigate(`/tse/analysis/${latest.id}`)}>جزئیات</Button></div><div className="divide-y divide-border">{latest.buyCandidates.slice(0, 4).map((item, index) => <div key={item.symbol} className="flex items-center justify-between px-4 py-3"><div className="flex items-center gap-3"><span className="flex h-7 w-7 items-center justify-center rounded-pill bg-success/12 text-caption font-bold text-success">{index + 1}</span><div><p className="text-body font-bold text-text">{item.symbol}</p><p className="text-caption text-text-subtle">{item.reason}</p></div></div><Badge variant="success">امتیاز {item.score}</Badge></div>)}</div></Card>
        <Card padding="none"><div className="flex items-center justify-between border-b border-border p-4"><h2 className="font-semibold text-text">رویدادهای نزدیک</h2><Button variant="ghost" size="sm" onClick={() => navigate('/tse/calendar')}>تقویم کامل</Button></div><div className="divide-y divide-border">{marketCalendar.slice(0, 4).map((event) => <div key={event.id} className="flex items-center gap-3 px-4 py-3"><span className="rounded-md bg-selected/10 p-2 text-link"><CalendarDays className="h-4 w-4" /></span><div className="min-w-0 flex-1"><p className="truncate text-body font-medium text-text">{event.title}</p><p className="text-caption text-text-subtle">{event.symbol} · {event.date}</p></div><Badge>{event.type}</Badge></div>)}</div></Card>
      </div>

      <Card padding="sm" className="border-warning/50 bg-warning/12"><div className="flex gap-3"><Gauge className="mt-0.5 h-5 w-5 shrink-0 text-warning-text" /><div><p className="text-body font-medium text-text">سطح ریسک بازار: متوسط</p><p className="mt-1 text-caption leading-6 text-text-subtle">اطلاعات این بخش دمو است و توصیه خرید یا فروش محسوب نمی‌شود. پیش از هر تصمیم، شرایط شخصی و مدیریت ریسک را در نظر بگیرید.</p></div></div></Card>
    </div>
  );
}

function Breadth({ label, value, color }: { label: string; value: number; color: string }) {
  return <div><p className={`text-h3 font-bold ${color}`}>{value.toLocaleString('fa-IR')}</p><p className="text-caption text-text-subtle">{label}</p></div>;
}

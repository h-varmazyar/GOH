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
          <div className="flex items-center gap-2"><h1 className="text-2xl font-bold text-surface-900 dark:text-white">بورس تهران</h1><Badge variant="success">بازار بسته است</Badge></div>
          <p className="mt-1 text-sm text-surface-500 dark:text-surface-400">نبض بازار، تحلیل‌ها و ابزارهای تصمیم‌یار در یک نگاه</p>
        </div>
        <span className="text-xs text-surface-500">آخرین بروزرسانی: امروز، ۱۲:۳۵</span>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {pulse.map((item) => (
          <Card key={item.label} padding="sm">
            <div className="flex items-start justify-between"><div><p className="text-xs text-surface-500">{item.label}</p><p className="mt-2 text-xl font-bold text-surface-900 dark:text-white" dir="ltr">{item.value}</p></div><span className="rounded-lg bg-primary-50 p-2 text-primary-600 dark:bg-primary-950"><item.icon className="h-5 w-5" /></span></div>
            <p className={`mt-2 text-xs ${item.positive ? 'text-success-600' : 'text-danger-500'}`} dir="ltr">{item.positive ? '▲' : '▼'} {item.detail}</p>
          </Card>
        ))}
      </div>

      <div className="grid gap-6 xl:grid-cols-[1.25fr_.75fr]">
        <Card padding="none">
          <div className="flex items-center justify-between border-b border-surface-200 p-4 dark:border-surface-700"><div><h2 className="font-semibold text-surface-900 dark:text-white">آخرین تحلیل بازار</h2><p className="mt-1 text-xs text-surface-500">{latest.date} · {latest.weekday}</p></div><Badge variant="success">{latest.mood}</Badge></div>
          <div className="p-4 sm:p-6"><p className="text-sm leading-7 text-surface-600 dark:text-surface-300">{latest.summary}</p><div className="mt-5 flex flex-wrap gap-3"><Button size="sm" onClick={() => navigate(`/tse/analysis/${latest.id}`)}>مشاهده تحلیل کامل <ArrowLeft className="h-4 w-4" /></Button><Button size="sm" variant="ghost" onClick={() => navigate('/tse/analysis')}>آرشیو تحلیل‌ها</Button></div></div>
        </Card>
        <Card padding="none">
          <div className="border-b border-surface-200 p-4 dark:border-surface-700"><h2 className="font-semibold text-surface-900 dark:text-white">پهنای بازار</h2></div>
          <div className="p-5"><div className="flex h-3 overflow-hidden rounded-full bg-surface-100 dark:bg-surface-700"><div className="bg-success-500" style={{ width: '64%' }} /><div className="bg-warning-400" style={{ width: '8%' }} /><div className="bg-danger-500" style={{ width: '28%' }} /></div><div className="mt-4 grid grid-cols-3 gap-2 text-center"><Breadth label="مثبت" value={latest.advancing} color="text-success-600" /><Breadth label="متعادل" value={54} color="text-warning-500" /><Breadth label="منفی" value={latest.declining} color="text-danger-500" /></div></div>
        </Card>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card padding="none"><div className="flex items-center justify-between border-b border-surface-200 p-4 dark:border-surface-700"><h2 className="font-semibold text-surface-900 dark:text-white">برترین فرصت‌های فردا</h2><Button variant="ghost" size="sm" onClick={() => navigate(`/tse/analysis/${latest.id}`)}>جزئیات</Button></div><div className="divide-y divide-surface-100 dark:divide-surface-700">{latest.buyCandidates.slice(0, 4).map((item, index) => <div key={item.symbol} className="flex items-center justify-between px-4 py-3"><div className="flex items-center gap-3"><span className="flex h-7 w-7 items-center justify-center rounded-full bg-success-400/20 text-xs font-bold text-success-600">{index + 1}</span><div><p className="text-sm font-bold text-surface-900 dark:text-white">{item.symbol}</p><p className="text-xs text-surface-500">{item.reason}</p></div></div><Badge variant="success">امتیاز {item.score}</Badge></div>)}</div></Card>
        <Card padding="none"><div className="flex items-center justify-between border-b border-surface-200 p-4 dark:border-surface-700"><h2 className="font-semibold text-surface-900 dark:text-white">رویدادهای نزدیک</h2><Button variant="ghost" size="sm" onClick={() => navigate('/tse/calendar')}>تقویم کامل</Button></div><div className="divide-y divide-surface-100 dark:divide-surface-700">{marketCalendar.slice(0, 4).map((event) => <div key={event.id} className="flex items-center gap-3 px-4 py-3"><span className="rounded-lg bg-primary-50 p-2 text-primary-600 dark:bg-primary-950"><CalendarDays className="h-4 w-4" /></span><div className="min-w-0 flex-1"><p className="truncate text-sm font-medium text-surface-900 dark:text-white">{event.title}</p><p className="text-xs text-surface-500">{event.symbol} · {event.date}</p></div><Badge>{event.type}</Badge></div>)}</div></Card>
      </div>

      <Card padding="sm" className="border-warning-300/60 bg-warning-200/30 dark:bg-warning-400/5"><div className="flex gap-3"><Gauge className="mt-0.5 h-5 w-5 shrink-0 text-warning-600" /><div><p className="text-sm font-medium text-surface-800 dark:text-surface-200">سطح ریسک بازار: متوسط</p><p className="mt-1 text-xs leading-6 text-surface-500">اطلاعات این بخش دمو است و توصیه خرید یا فروش محسوب نمی‌شود. پیش از هر تصمیم، شرایط شخصی و مدیریت ریسک را در نظر بگیرید.</p></div></div></Card>
    </div>
  );
}

function Breadth({ label, value, color }: { label: string; value: number; color: string }) {
  return <div><p className={`text-lg font-bold ${color}`}>{value.toLocaleString('fa-IR')}</p><p className="text-xs text-surface-500">{label}</p></div>;
}

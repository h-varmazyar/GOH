import { useMemo, useState } from 'react';
import { Edit3, Save, Search, X } from 'lucide-react';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import { Input } from '../../components/ui/Input';
import { Pagination } from '../../components/ui/Pagination';
import { Select } from '../../components/ui/Select';
import { Toggle } from '../../components/ui/Toggle';
import { initialIndicators, type Indicator } from './demoData';

const PAGE_SIZE = 6;

export function IndicatorsPage() {
  const [items, setItems] = useState(initialIndicators);
  const [page, setPage] = useState(1);
  const [query, setQuery] = useState('');
  const [category, setCategory] = useState('');
  const [editing, setEditing] = useState<Indicator | null>(null);
  const filtered = useMemo(() => items.filter((item) => (!query || `${item.name} ${item.code}`.toLowerCase().includes(query.toLowerCase())) && (!category || item.category === category)), [items, query, category]);
  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const rows = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);
  const updateFilter = (fn: (value: string) => void, value: string) => { fn(value); setPage(1); };
  const save = () => { if (!editing) return; setItems((current) => current.map((item) => item.id === editing.id ? editing : item)); setEditing(null); };

  return (
    <div className="space-y-4">
      <div><h1 className="text-xl font-semibold text-surface-900 dark:text-white">اندیکاتورهای بازار</h1><p className="mt-1 text-sm text-surface-500">مدیریت محاسبات تکنیکال و پارامترهای اعمال‌شده روی نمادها</p></div>
      <Card padding="sm"><div className="grid gap-3 md:grid-cols-[1fr_220px]"><div className="relative"><Search className="absolute right-3 top-2.5 h-5 w-5 text-surface-400" /><Input className="pr-10" placeholder="جستجوی نام یا کد اندیکاتور..." value={query} onChange={(e) => updateFilter(setQuery, e.target.value)} /></div><Select placeholder="همه دسته‌ها" value={category} onChange={(e) => updateFilter(setCategory, e.target.value)} options={[{ value: 'روند', label: 'روند' }, { value: 'مومنتوم', label: 'مومنتوم' }, { value: 'حجم', label: 'حجم' }, { value: 'نوسان', label: 'نوسان' }]} /></div></Card>
      <Card padding="none"><div className="overflow-x-auto"><table className="w-full min-w-[760px] text-center [&_th]:text-center"><thead><tr className="border-b border-surface-200 bg-surface-50 dark:border-surface-700 dark:bg-surface-800/50">{['اندیکاتور', 'دسته', 'بازه زمانی', 'پارامترها', 'آخرین محاسبه', 'وضعیت', 'تنظیمات'].map((x) => <th key={x} className="px-4 py-3 text-right text-xs font-medium text-surface-500">{x}</th>)}</tr></thead><tbody>{rows.map((item) => <tr key={item.id} className="border-b border-surface-100 dark:border-surface-700"><td className="px-4 py-3"><p className="text-sm font-medium text-surface-900 dark:text-white">{item.name}</p><code className="text-xs text-surface-500">{item.code}</code></td><td className="px-4 py-3"><Badge>{item.category}</Badge></td><td className="px-4 py-3 text-sm">{item.timeframe}</td><td className="px-4 py-3 text-xs text-surface-500" dir="ltr">{Object.entries(item.parameters).map(([key, value]) => `${key}: ${value}`).join(' · ')}</td><td className="px-4 py-3 text-xs text-surface-500">امروز ۱۲:۳۲</td><td className="px-4 py-3"><Toggle enabled={item.enabled} onChange={(enabled) => setItems((current) => current.map((row) => row.id === item.id ? { ...row, enabled } : row))} /></td><td className="px-4 py-3"><Button size="sm" variant="ghost" onClick={() => setEditing({ ...item, parameters: { ...item.parameters } })}><Edit3 className="h-4 w-4" />ویرایش</Button></td></tr>)}</tbody></table></div><Pagination page={page} totalPages={totalPages} totalItems={filtered.length} onChange={setPage} /></Card>

      {editing && <Card className="border-primary-200 dark:border-primary-800"><div className="flex items-center justify-between"><div><h2 className="font-semibold text-surface-900 dark:text-white">تنظیمات {editing.name}</h2><p className="mt-1 text-xs text-surface-500">تغییرات این فرم در دموی محلی ذخیره می‌شود.</p></div><Button variant="ghost" size="sm" onClick={() => setEditing(null)}><X className="h-4 w-4" /></Button></div><div className="mt-5 grid gap-4 md:grid-cols-3"><Select label="بازه زمانی" value={editing.timeframe} onChange={(e) => setEditing({ ...editing, timeframe: e.target.value })} options={[{ value: 'روزانه', label: 'روزانه' }, { value: 'هفتگی', label: 'هفتگی' }, { value: '۶۰ دقیقه', label: '۶۰ دقیقه' }]} />{Object.entries(editing.parameters).map(([key, value]) => <Input key={key} label={key} type="number" value={value} onChange={(e) => setEditing({ ...editing, parameters: { ...editing.parameters, [key]: Number(e.target.value) } })} dir="ltr" />)}</div><div className="mt-5 flex justify-end"><Button onClick={save}><Save className="h-4 w-4" />ذخیره تنظیمات</Button></div></Card>}
    </div>
  );
}

import { useMemo, useState } from 'react';
import { CalendarDays, Search } from 'lucide-react';
import { Badge } from '../../components/ui/Badge';
import { Card } from '../../components/ui/Card';
import { Input } from '../../components/ui/Input';
import { Pagination } from '../../components/ui/Pagination';
import { Select } from '../../components/ui/Select';
import { marketCalendar } from './demoData';

const PAGE_SIZE = 6;

export function MarketCalendarPage() {
  const [page, setPage] = useState(1);
  const [query, setQuery] = useState('');
  const [type, setType] = useState('');
  const filtered = useMemo(() => marketCalendar.filter((item) => (!query || `${item.title} ${item.symbol}`.includes(query)) && (!type || item.type === type)), [query, type]);
  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const rows = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);
  const resetPage = () => setPage(1);

  return <div className="space-y-4">
    <div><h1 className="text-h3 font-semibold text-text">تقویم بازار</h1><p className="mt-1 text-body text-text-subtle">رویدادهای مجامع، عرضه‌های اولیه، سود نقدی و گزارش‌های مهم</p></div>
    <Card padding="sm"><div className="grid gap-3 md:grid-cols-[1fr_220px]"><Input leadingIcon={<Search />} aria-label="جستجو" value={query} onChange={(e) => { setQuery(e.target.value); resetPage(); }} placeholder="جستجوی نماد یا رویداد..." /><Select placeholder="همه رویدادها" value={type} onChange={(e) => { setType(e.target.value); resetPage(); }} options={[{ value: 'مجمع', label: 'مجمع' }, { value: 'عرضه اولیه', label: 'عرضه اولیه' }, { value: 'گزارش', label: 'گزارش' }, { value: 'سود نقدی', label: 'سود نقدی' }]} /></div></Card>
    <Card padding="none"><div className="divide-y divide-border">{rows.map((item) => <div key={item.id} className="grid gap-3 p-4 sm:grid-cols-[120px_1fr_auto] sm:items-center"><div className="flex items-center gap-2 text-body font-medium text-text-muted"><CalendarDays className="h-4 w-4 text-link" />{item.date}</div><div><div className="flex items-center gap-2"><span className="font-bold text-link">{item.symbol}</span><h2 className="text-body font-medium text-text">{item.title}</h2></div><p className="mt-1 text-caption text-text-subtle">{item.description}</p></div><Badge variant={item.type === 'عرضه اولیه' ? 'success' : item.type === 'مجمع' ? 'warning' : 'primary'}>{item.type}</Badge></div>)}</div><Pagination page={page} totalPages={totalPages} totalItems={filtered.length} onChange={setPage} /></Card>
  </div>;
}

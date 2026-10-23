import { useMemo, useState } from 'react';
import { CheckSquare, Filter, Search } from 'lucide-react';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import { Input } from '../../components/ui/Input';
import { Pagination } from '../../components/ui/Pagination';
import { Select } from '../../components/ui/Select';
import { Toggle } from '../../components/ui/Toggle';
import { initialMarkets } from './demoData';

const PAGE_SIZE = 8;

export function MarketsPage() {
  const [markets, setMarkets] = useState(initialMarkets);
  const [selected, setSelected] = useState<string[]>([]);
  const [page, setPage] = useState(1);
  const [query, setQuery] = useState('');
  const [industry, setIndustry] = useState('');
  const [status, setStatus] = useState('');
  const filtered = useMemo(() => markets.filter((item) => (!query || `${item.symbol} ${item.name}`.includes(query)) && (!industry || item.industry === industry) && (!status || (status === 'active') === item.active)), [markets, query, industry, status]);
  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const rows = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);
  const visibleIds = rows.map((item) => item.id);
  const allVisibleSelected = visibleIds.length > 0 && visibleIds.every((id) => selected.includes(id));
  const changeFilter = (setter: (value: string) => void, value: string) => { setter(value); setPage(1); };
  const toggleSelection = (id: string) => setSelected((current) => current.includes(id) ? current.filter((item) => item !== id) : [...current, id]);
  const bulkSet = (active: boolean) => { setMarkets((current) => current.map((item) => selected.includes(item.id) ? { ...item, active } : item)); setSelected([]); };

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3"><div><h1 className="text-xl font-semibold text-surface-900 dark:text-white">مدیریت بازارها</h1><p className="mt-1 text-sm text-surface-500">فیلتر، انتخاب و کنترل نمادهای تحت پردازش</p></div><Badge variant="primary">{markets.filter((item) => item.active).length.toLocaleString('fa-IR')} نماد فعال</Badge></div>
      <Card padding="sm"><div className="grid gap-3 lg:grid-cols-[1fr_220px_180px]"><div className="relative"><Search className="absolute right-3 top-2.5 h-5 w-5 text-surface-400" /><Input className="pr-10" placeholder="جستجوی نماد یا نام شرکت..." value={query} onChange={(e) => changeFilter(setQuery, e.target.value)} /></div><Select placeholder="همه صنایع" value={industry} onChange={(e) => changeFilter(setIndustry, e.target.value)} options={[...new Set(markets.map((item) => item.industry))].map((value) => ({ value, label: value }))} /><Select placeholder="همه وضعیت‌ها" value={status} onChange={(e) => changeFilter(setStatus, e.target.value)} options={[{ value: 'active', label: 'فعال' }, { value: 'inactive', label: 'غیرفعال' }]} /></div></Card>
      {selected.length > 0 && <Card padding="sm" className="border-primary-200 bg-primary-50 dark:border-primary-800 dark:bg-primary-950/40"><div className="flex flex-wrap items-center justify-between gap-3"><span className="flex items-center gap-2 text-sm font-medium text-primary-700 dark:text-primary-300"><CheckSquare className="h-4 w-4" />{selected.length.toLocaleString('fa-IR')} بازار انتخاب شده</span><div className="flex gap-2"><Button size="sm" onClick={() => bulkSet(true)}>فعال‌سازی گروهی</Button><Button size="sm" variant="danger" onClick={() => bulkSet(false)}>غیرفعال‌سازی گروهی</Button><Button size="sm" variant="ghost" onClick={() => setSelected([])}>لغو انتخاب</Button></div></div></Card>}
      <Card padding="none"><div className="overflow-x-auto"><table className="w-full min-w-[900px] text-center [&_th]:text-center"><thead><tr className="border-b border-surface-200 bg-surface-50 dark:border-surface-700 dark:bg-surface-800/50"><th className="px-4 py-3"><input type="checkbox" aria-label="انتخاب همه صفحه" checked={allVisibleSelected} onChange={() => setSelected((current) => allVisibleSelected ? current.filter((id) => !visibleIds.includes(id)) : [...new Set([...current, ...visibleIds])])} className="h-4 w-4 accent-primary-600" /></th>{['نماد', 'نام شرکت', 'صنعت', 'بازار', 'آخرین قیمت', 'تغییر', 'حجم امروز', 'وضعیت'].map((x) => <th key={x} className="px-4 py-3 text-right text-xs font-medium text-surface-500">{x}</th>)}</tr></thead><tbody>{rows.map((item) => <tr key={item.id} className={`border-b border-surface-100 dark:border-surface-700 ${selected.includes(item.id) ? 'bg-primary-50/70 dark:bg-primary-950/30' : ''}`}><td className="px-4 py-3"><input type="checkbox" aria-label={`انتخاب ${item.symbol}`} checked={selected.includes(item.id)} onChange={() => toggleSelection(item.id)} className="h-4 w-4 accent-primary-600" /></td><td className="px-4 py-3 font-bold text-primary-700 dark:text-primary-300">{item.symbol}</td><td className="px-4 py-3 text-sm">{item.name}</td><td className="px-4 py-3"><Badge>{item.industry}</Badge></td><td className="px-4 py-3 text-xs text-surface-500">{item.board}</td><td className="px-4 py-3 text-sm" dir="ltr">{item.price.toLocaleString('fa-IR')}</td><td className={`px-4 py-3 text-sm ${item.change >= 0 ? 'text-success-600' : 'text-danger-500'}`} dir="ltr">{item.change >= 0 ? '+' : ''}{item.change.toLocaleString('fa-IR')}٪</td><td className="px-4 py-3 text-sm">{item.volume.toLocaleString('fa-IR')} م</td><td className="px-4 py-3"><Toggle enabled={item.active} onChange={(active) => setMarkets((current) => current.map((row) => row.id === item.id ? { ...row, active } : row))} /></td></tr>)}</tbody></table></div><Pagination page={page} totalPages={totalPages} totalItems={filtered.length} onChange={setPage} /></Card>
      <p className="flex items-center gap-1 text-xs text-surface-500"><Filter className="h-3 w-3" />وضعیت‌ها و عملیات این صفحه دمو هستند و فقط تا زمان تازه‌سازی صفحه باقی می‌مانند.</p>
    </div>
  );
}

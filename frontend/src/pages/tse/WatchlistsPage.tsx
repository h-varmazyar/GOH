import { useMemo, useState } from 'react';
import { Bell, Eye, Search, Star } from 'lucide-react';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import { Input } from '../../components/ui/Input';
import { Pagination } from '../../components/ui/Pagination';
import { watchlist } from './demoData';

const PAGE_SIZE = 7;

export function WatchlistsPage() {
  const [page, setPage] = useState(1);
  const [query, setQuery] = useState('');
  const [items, setItems] = useState(watchlist);
  const filtered = useMemo(() => items.filter((item) => `${item.symbol} ${item.name} ${item.note}`.includes(query)), [items, query]);
  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const rows = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  return <div className="space-y-4">
    <div><h1 className="text-h3 font-semibold text-text">دیده‌بان بازار</h1><p className="mt-1 text-body text-text-subtle">نمادهای منتخب، هشدارها و تغییرات مهم امروز</p></div>
    <Card padding="sm"><Input leadingIcon={<Search />} aria-label="جستجو" value={query} onChange={(e) => { setQuery(e.target.value); setPage(1); }} placeholder="جستجو در دیده‌بان..." /></Card>
    <Card padding="none"><div className="overflow-x-auto"><table className="w-full min-w-[780px] text-center [&_th]:text-center"><thead><tr className="border-b border-border bg-surface-muted">{['نماد', 'شرکت', 'آخرین قیمت', 'تغییر روز', 'سیگنال', 'یادداشت', 'هشدار'].map((x) => <th key={x} className="px-4 py-3 text-start text-label text-text-muted">{x}</th>)}</tr></thead><tbody>{rows.map((item) => <tr key={item.symbol} className="border-b border-border"><td className="px-4 py-3"><span className="inline-flex items-center gap-2 font-bold text-link"><Star className="h-4 w-4 fill-warning text-warning-text" />{item.symbol}</span></td><td className="px-4 py-3 text-body">{item.name}</td><td className="px-4 py-3 text-body" dir="ltr">{item.price.toLocaleString('fa-IR')}</td><td className={`px-4 py-3 text-body ${item.change >= 0 ? 'text-success' : 'text-error'}`} dir="ltr">{item.change >= 0 ? '+' : ''}{item.change.toLocaleString('fa-IR')}٪</td><td className="px-4 py-3"><Badge variant={item.signal === 'خرید' ? 'success' : item.signal === 'فروش' ? 'danger' : 'warning'}>{item.signal}</Badge></td><td className="max-w-xs px-4 py-3 text-caption text-text-subtle">{item.note}</td><td className="px-4 py-3"><Button size="sm" variant={item.alert ? 'secondary' : 'ghost'} onClick={() => setItems((current) => current.map((row) => row.symbol === item.symbol ? { ...row, alert: !row.alert } : row))}>{item.alert ? <Bell className="h-4 w-4" /> : <Eye className="h-4 w-4" />}{item.alert ? 'فعال' : 'ایجاد'}</Button></td></tr>)}</tbody></table></div><Pagination page={page} totalPages={totalPages} totalItems={filtered.length} onChange={setPage} /></Card>
  </div>;
}

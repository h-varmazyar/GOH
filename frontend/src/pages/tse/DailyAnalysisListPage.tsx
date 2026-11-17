import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowLeft, CalendarSearch, Search, TrendingDown, TrendingUp } from 'lucide-react';
import { Badge } from '../../components/ui/Badge';
import { Card } from '../../components/ui/Card';
import { Input } from '../../components/ui/Input';
import { Pagination } from '../../components/ui/Pagination';
import { Select } from '../../components/ui/Select';
import { analyses } from './demoData';

const PAGE_SIZE = 7;

export function DailyAnalysisListPage() {
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const [query, setQuery] = useState('');
  const [mood, setMood] = useState('');

  const filtered = useMemo(() => analyses.filter((item) =>
    (!query || `${item.date} ${item.summary}`.includes(query)) && (!mood || item.mood === mood)
  ), [query, mood]);
  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const rows = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);
  const changeFilter = (setter: (value: string) => void, value: string) => { setter(value); setPage(1); };

  return (
    <div className="space-y-4">
      <div><h1 className="text-h3 font-semibold text-text">تحلیل روزانه بازار</h1><p className="mt-1 text-body text-text-subtle">آرشیو وضعیت بازار، شاخص‌ها و فرصت‌های پیشنهادی هر روز</p></div>
      <Card padding="sm">
        <div className="grid gap-3 md:grid-cols-[1fr_220px]">
          <Input leadingIcon={<Search />} aria-label="جستجو" value={query} onChange={(e) => changeFilter(setQuery, e.target.value)} placeholder="جستجو در تاریخ یا خلاصه تحلیل..." className="ps-10" />
          <Select value={mood} onChange={(e) => changeFilter(setMood, e.target.value)} placeholder="همه وضعیت‌ها" options={[{ value: 'صعودی', label: 'صعودی' }, { value: 'متعادل', label: 'متعادل' }, { value: 'نزولی', label: 'نزولی' }]} />
        </div>
      </Card>
      <Card padding="none">
        <div className="overflow-x-auto"><table className="w-full min-w-[850px] text-center [&_th]:text-center">
          <thead><tr className="border-b border-border bg-surface-muted">
            {['تاریخ تحلیل', 'وضعیت', 'شاخص کل', 'هم‌وزن', 'ارزش خرد', 'پول حقیقی', 'نمادهای مثبت', ''].map((title) => <th key={title} className="px-4 py-3 text-start text-label text-text-muted">{title}</th>)}
          </tr></thead>
          <tbody>{rows.length ? rows.map((item) => (
            <tr key={item.id} onClick={() => navigate(`/tse/analysis/${item.id}`)} className="cursor-pointer border-b border-border transition-colors hover:bg-selected/8">
              <td className="px-4 py-3"><p className="text-body font-medium text-text">{item.date}</p><p className="text-caption text-text-subtle">{item.weekday}</p></td>
              <td className="px-4 py-3"><Badge variant={item.mood === 'صعودی' ? 'success' : item.mood === 'نزولی' ? 'danger' : 'warning'}>{item.mood}</Badge></td>
              <td className="px-4 py-3"><p className="text-body font-medium" dir="ltr">{item.totalIndex.toLocaleString('fa-IR')}</p><Change value={item.totalIndexChange} /></td>
              <td className="px-4 py-3"><Change value={item.equalIndexChange} /></td>
              <td className="px-4 py-3 text-body">{item.tradeValue.toLocaleString('fa-IR')} همت</td>
              <td className={`px-4 py-3 text-body ${item.netMoney >= 0 ? 'text-success' : 'text-error'}`}>{Math.abs(item.netMoney).toLocaleString('fa-IR')} میلیارد</td>
              <td className="px-4 py-3 text-body">{item.advancing.toLocaleString('fa-IR')} از {(item.advancing + item.declining).toLocaleString('fa-IR')}</td>
              <td className="px-4 py-3 text-link"><ArrowLeft className="h-4 w-4" /></td>
            </tr>
          )) : <tr><td colSpan={8} className="py-12 text-center text-body text-text-subtle"><CalendarSearch className="mx-auto mb-2 h-8 w-8" />تحلیلی با این مشخصات پیدا نشد</td></tr>}</tbody>
        </table></div>
        <Pagination page={page} totalPages={totalPages} totalItems={filtered.length} onChange={setPage} />
      </Card>
    </div>
  );
}

function Change({ value }: { value: number }) {
  const positive = value >= 0;
  return <span className={`inline-flex items-center gap-1 text-caption ${positive ? 'text-success' : 'text-error'}`} dir="ltr">{positive ? <TrendingUp className="h-3 w-3" /> : <TrendingDown className="h-3 w-3" />}{Math.abs(value).toLocaleString('fa-IR')}٪</span>;
}

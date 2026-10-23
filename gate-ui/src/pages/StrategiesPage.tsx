import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { PlusCircle, Eye, Trash2, Power, LayoutTemplate, Search, Bell } from 'lucide-react';
import { Card } from '../components/ui/Card';
import { Button } from '../components/ui/Button';
import { Badge } from '../components/ui/Badge';
import { Select } from '../components/ui/Select';
import { useStrategyStore } from '../store/strategyStore';
import { strategyProviders } from '../data/strategyDemo';

export function StrategiesPage() {
  const { strategies, fetchStrategies, toggleStrategy, deleteStrategy, isLoading } = useStrategyStore();
  const [search, setSearch] = useState('');
  const [asset, setAsset] = useState('all');
  const [status, setStatus] = useState('all');
  const [provider, setProvider] = useState('all');
  useEffect(() => { void fetchStrategies(); }, [fetchStrategies]);
  const filtered = useMemo(() => strategies.filter((item) => {
    const matchesSearch = `${item.name} ${item.description} ${item.market.symbols.join(' ')}`.toLowerCase().includes(search.toLowerCase());
    return matchesSearch && (asset === 'all' || item.assetClass === asset) && (status === 'all' || String(item.enabled) === status) && (provider === 'all' || item.provider === provider);
  }), [strategies, search, asset, status, provider]);

  return <div className="space-y-6" dir="rtl">
    <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3"><div><h1 className="text-2xl font-bold text-surface-900 dark:text-white">استراتژی‌ها</h1><p className="text-sm text-surface-500 mt-1">مدیریت استراتژی‌های رمزارز و سهام</p></div><div className="flex gap-2"><Link to="/strategies/templates"><Button variant="secondary"><LayoutTemplate className="w-4 h-4" />قالب‌ها</Button></Link><Link to="/strategies/new"><Button><PlusCircle className="w-4 h-4" />استراتژی جدید</Button></Link></div></div>
    <Card padding="sm"><div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-3"><div className="relative"><Search className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-surface-400" /><input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="جستجو..." className="w-full pr-9 pl-3 py-2 rounded-lg border bg-white dark:bg-surface-800 border-surface-300 dark:border-surface-600 text-sm" /></div><Select value={asset} onChange={(e) => setAsset(e.target.value)} options={[{ value: 'all', label: 'همه دارایی‌ها' }, { value: 'crypto', label: 'رمزارز' }, { value: 'stock', label: 'سهام تهران' }]} /><Select value={status} onChange={(e) => setStatus(e.target.value)} options={[{ value: 'all', label: 'همه وضعیت‌ها' }, { value: 'true', label: 'فعال' }, { value: 'false', label: 'غیرفعال' }]} /><Select value={provider} onChange={(e) => setProvider(e.target.value)} options={[{ value: 'all', label: 'همه ارائه‌دهنده‌ها' }, ...strategyProviders.map((item) => ({ value: item.id, label: item.name }))]} /></div></Card>
    {isLoading ? <div className="flex justify-center py-12"><div className="w-8 h-8 border-2 border-primary-500 border-t-transparent rounded-full animate-spin" /></div> : filtered.length === 0 ? <Card className="text-center py-12"><p className="text-surface-500 mb-4">استراتژی مطابق فیلترها یافت نشد.</p><Link to="/strategies/new"><Button><PlusCircle className="w-4 h-4" />ایجاد استراتژی</Button></Link></Card> : <div className="grid lg:grid-cols-2 gap-4">{filtered.map((item) => {
      const activeAlerts = Object.values(item.alerts.channels).filter((channel) => channel.enabled).length;
      return <Card key={item.id} padding="none"><div className="p-4 sm:p-5"><div className="flex items-start justify-between gap-3"><div className="min-w-0"><div className="flex items-center gap-2"><span className={`w-2.5 h-2.5 rounded-full ${item.enabled ? 'bg-success-500' : 'bg-surface-400'}`} /><h3 className="font-semibold text-surface-900 dark:text-white truncate">{item.name}</h3></div><p className="text-sm text-surface-500 line-clamp-2 mt-2">{item.description || 'بدون توضیح'}</p></div><div className="flex"><Link to={`/strategies/${item.id}`}><Button variant="ghost" size="sm" title="مشاهده"><Eye className="w-4 h-4" /></Button></Link><Button variant="ghost" size="sm" onClick={() => toggleStrategy(item.id!)} title="تغییر وضعیت"><Power className={`w-4 h-4 ${item.enabled ? 'text-success-500' : ''}`} /></Button><Button variant="ghost" size="sm" onClick={() => confirm('از حذف استراتژی مطمئن هستید؟') && deleteStrategy(item.id!)} title="حذف"><Trash2 className="w-4 h-4 text-danger-500" /></Button></div></div>
      <div className="flex flex-wrap gap-2 mt-4"><Badge variant="primary">{item.assetClass === 'stock' ? 'سهام' : 'رمزارز'}</Badge><Badge>{item.providerName || item.provider}</Badge><Badge>{item.market.symbols.length ? item.market.symbols.join('، ') : 'بدون نماد'}</Badge><Badge variant={activeAlerts ? 'success' : 'default'}><Bell className="w-3 h-3 ml-1" />{activeAlerts} کانال هشدار</Badge></div><div className="flex flex-wrap gap-x-4 gap-y-1 mt-4 pt-3 border-t border-surface-100 dark:border-surface-700 text-xs text-surface-500"><span>تایم‌فریم: {item.market.timeframes.join('، ') || '—'}</span><span>اندیکاتور: {Object.keys(item.indicators).length}</span></div></div></Card>;
    })}</div>}
  </div>;
}

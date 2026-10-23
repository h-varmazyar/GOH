import { Link } from 'react-router-dom';
import { ArrowLeft, Gauge, Layers3 } from 'lucide-react';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { strategyTemplates } from '../data/strategyDemo';

export function StrategyTemplatesPage() {
  return <div className="space-y-6" dir="rtl">
    <div><h1 className="text-2xl font-bold text-surface-900 dark:text-white">استراتژی‌های آماده</h1><p className="text-sm text-surface-500 dark:text-surface-400 mt-1">یک قالب را انتخاب و تنظیمات آن را در ویزارد شخصی‌سازی کنید.</p></div>
    <div className="grid md:grid-cols-2 gap-4">{strategyTemplates.map((template) => <Card key={template.id} className="flex flex-col">
      <div className="flex items-start justify-between gap-3"><div><h2 className="text-lg font-semibold text-surface-900 dark:text-white">{template.name}</h2><p className="text-sm text-surface-500 dark:text-surface-400 mt-2 leading-6">{template.description}</p></div><Badge variant={template.riskLevel === 'کم' ? 'success' : template.riskLevel === 'زیاد' ? 'danger' : 'warning'}>{`ریسک ${template.riskLevel}`}</Badge></div>
      <div className="flex flex-wrap gap-2 mt-4"><Badge variant="primary">{template.assetClass === 'stock' ? 'سهام تهران' : 'رمزارز'}</Badge><Badge>{template.providerName}</Badge><Badge>{template.timeframe}</Badge><Badge>{template.category}</Badge></div>
      <div className="mt-4 p-3 bg-surface-50 dark:bg-surface-900/30 rounded-lg"><p className="text-xs text-surface-500 mb-2 flex items-center gap-1"><Layers3 className="w-4 h-4" />اندیکاتورها</p><div className="flex flex-wrap gap-1">{template.indicators.map((indicator) => <Badge key={indicator}>{indicator}</Badge>)}</div></div>
      <div className="mt-auto pt-5 flex justify-end"><Link to={`/strategies/new?template=${template.id}`}><Button>استفاده از قالب<ArrowLeft className="w-4 h-4" /></Button></Link></div>
    </Card>)}</div>
    <div className="text-xs text-surface-400 flex items-center gap-1"><Gauge className="w-4 h-4" />این قالب‌ها برای دموی محلی هستند و توصیه سرمایه‌گذاری محسوب نمی‌شوند.</div>
  </div>;
}

import { useState, type ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { Ban, Link2, RefreshCw, RotateCcw, Trash2 } from 'lucide-react';
import { Badge } from '../ui/Badge';
import { Button } from '../ui/Button';
import { Toggle } from '../ui/Toggle';
import { tseMarketService, type TseMarket, type TseMarketFlags } from '../../services/tseMarketService';
import { errorMessage, formatDate } from './format';
import { confirmDelisted, deleteMarket } from './marketActions';
import { LinkMarketModal } from './MarketModals';

/**
 * The insCodes of an issue or a market with their flags and actions. Changes are
 * reported through onChanged so the parent can reload.
 */
export function InstrumentTable({
  instruments,
  highlight = [],
  onChanged,
  onDeleted,
  onError,
  extraActions,
}: {
  instruments: TseMarket[];
  /** insCodes to emphasize, such as those an issue is about. */
  highlight?: string[];
  onChanged: (market: TseMarket, message: string) => void;
  /** Called after an insCode is deleted; onChanged is called when it is not given. */
  onDeleted?: (market: TseMarket, message: string) => void;
  onError: (message: string) => void;
  extraActions?: (market: TseMarket) => ReactNode;
}) {
  const [busyId, setBusyId] = useState<number | null>(null);
  const [linking, setLinking] = useState<TseMarket | null>(null);

  const run = async (market: TseMarket, action: () => Promise<TseMarket>, done: string, failed: string) => {
    setBusyId(market.id);
    try {
      onChanged(await action(), done);
    } catch (e) {
      onError(errorMessage(e, failed));
    } finally {
      setBusyId(null);
    }
  };

  const setFlags = (market: TseMarket, flags: TseMarketFlags) =>
    run(market, () => tseMarketService.setFlags(market.id, flags), `وضعیت ${market.symbol} (${market.ins_code}) تغییر کرد.`, 'خطا در تغییر وضعیت نماد');

  const toggleDelisted = async (market: TseMarket) => {
    if (!(await confirmDelisted(market))) return;
    run(market, () => tseMarketService.setDelisted(market.id, !market.delisted), `وضعیت لغو پذیرش ${market.symbol} تغییر کرد.`, 'خطا در تغییر وضعیت لغو پذیرش');
  };

  const remove = async (market: TseMarket) => {
    setBusyId(market.id);
    try {
      const message = await deleteMarket(market, false);
      if (message) (onDeleted || onChanged)(market, message);
    } catch (e) {
      onError(errorMessage(e, 'خطا در حذف نماد'));
    } finally {
      setBusyId(null);
    }
  };

  if (instruments.length === 0) {
    return <p className="text-sm text-surface-500">این insCodeها در دیتابیس ذخیره نشده‌اند.</p>;
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[900px] text-right">
        <thead>
          <tr className="border-b border-surface-200 dark:border-surface-700">
            {['نماد', 'نام', 'مارکت', 'بازار', 'تاریخ عرضه', 'دریافت داده', 'ارزیابی', 'عملیات'].map((title) => (
              <th key={title} className="px-2 py-2 text-xs font-medium text-surface-500">{title}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {instruments.map((item) => (
            <tr
              key={item.id}
              className={`border-b border-surface-100 dark:border-surface-800 ${highlight.includes(item.ins_code) ? 'bg-warning-400/10' : ''}`}
            >
              <td className="px-2 py-2">
                <div className="flex items-center gap-1.5">
                  <Link to={`/tse/markets/${item.id}`} className="font-bold text-primary-700 hover:underline dark:text-primary-300">
                    {item.symbol}
                  </Link>
                  {item.delisted && <Badge variant="danger">لغو پذیرش</Badge>}
                  {item.manually_added && <Badge variant="warning">دستی</Badge>}
                </div>
                <div className="text-[11px] text-surface-400" dir="ltr">{item.instrument_id || '—'} · {item.ins_code}</div>
              </td>
              <td className="px-2 py-2 text-sm">{item.name}</td>
              <td className="px-2 py-2 text-xs">
                {item.market_label}
                {item.market_members > 1 && <span className="text-surface-400"> · {item.market_members.toLocaleString('fa-IR')} insCode</span>}
              </td>
              <td className="px-2 py-2 text-xs text-surface-600 dark:text-surface-300">{item.market_type_label || '—'}</td>
              <td className="px-2 py-2 text-xs">{formatDate(item.release_date)}</td>
              <td className="px-2 py-2">
                <Toggle enabled={item.data_collection_enabled} onChange={(value) => setFlags(item, { data_collection_enabled: value })} />
              </td>
              <td className="px-2 py-2">
                <Toggle enabled={item.evaluation_enabled} onChange={(value) => setFlags(item, { evaluation_enabled: value })} />
              </td>
              <td className="px-2 py-2">
                <div className="flex flex-wrap items-center gap-1">
                  <Button
                    size="sm"
                    variant="ghost"
                    title="خواندن دوباره از منابع"
                    disabled={busyId !== null}
                    isLoading={busyId === item.id}
                    onClick={() => run(item, () => tseMarketService.refresh(item.id), `اطلاعات ${item.symbol} (${item.ins_code}) از منابع به‌روز شد.`, 'خطا در به‌روزرسانی نماد')}
                  >
                    {busyId !== item.id && <RefreshCw className="h-4 w-4" />}
                  </Button>
                  <Button size="sm" variant="ghost" title={item.delisted ? 'برداشتن علامت لغو پذیرش' : 'علامت لغو پذیرش'} disabled={busyId !== null} onClick={() => toggleDelisted(item)}>
                    {item.delisted ? <RotateCcw className="h-4 w-4" /> : <Ban className="h-4 w-4" />}
                  </Button>
                  <Button size="sm" variant="ghost" title="انتقال به مارکت دیگر" disabled={busyId !== null} onClick={() => setLinking(item)}>
                    <Link2 className="h-4 w-4" />
                  </Button>
                  <Button size="sm" variant="ghost" title="حذف insCode" disabled={busyId !== null} onClick={() => remove(item)}>
                    <Trash2 className="h-4 w-4 text-danger-500" />
                  </Button>
                  {extraActions?.(item)}
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {linking && (
        <LinkMarketModal
          market={linking}
          initialLabel={`${linking.symbol}-${linking.ins_code.slice(-4)}`}
          onClose={() => setLinking(null)}
          onLinked={(market) => {
            setLinking(null);
            onChanged(market, `insCode ${market.ins_code} به مارکت ${market.market_label} منتقل شد.`);
          }}
        />
      )}
    </div>
  );
}

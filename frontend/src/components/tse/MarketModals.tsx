import { useState } from 'react';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { Modal } from '../ui/Modal';
import { tseMarketService, type TseMarket } from '../../services/tseMarketService';
import { errorMessage } from './format';

const ErrorBox = ({ error }: { error: string }) =>
  error ? <div className="rounded-lg bg-danger-50 p-3 text-sm text-danger-600 dark:bg-danger-950/40">{error}</div> : null;

export function AddMarketModal({
  initialSymbol = '',
  initialInsCode = '',
  onClose,
  onCreated,
}: {
  initialSymbol?: string;
  initialInsCode?: string;
  onClose: () => void;
  onCreated: (market: TseMarket) => void;
}) {
  const [symbol, setSymbol] = useState(initialSymbol);
  const [insCode, setInsCode] = useState(initialInsCode);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState('');

  const save = async () => {
    setIsSaving(true);
    setError('');
    try {
      onCreated(await tseMarketService.create(symbol.trim(), insCode.trim()));
    } catch (e) {
      setError(errorMessage(e, 'خطا در افزودن نماد'));
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Modal
      title="افزودن نماد"
      onClose={onClose}
      footer={
        <>
          <Button className="flex-1" isLoading={isSaving} disabled={!symbol.trim() || !insCode.trim()} onClick={save}>افزودن</Button>
          <Button variant="ghost" onClick={onClose}>انصراف</Button>
        </>
      }
    >
      <Input label="نماد" placeholder="مثلاً فولاد" value={symbol} onChange={(e) => setSymbol(e.target.value)} />
      <Input label="insCode" placeholder="46348559193224090" dir="ltr" inputMode="numeric" value={insCode} onChange={(e) => setInsCode(e.target.value.replace(/\D/g, ''))} />
      <p className="text-xs text-surface-500">
        سایر اطلاعات (شناسه IR، نوع، بازار، گروه صنعت و تاریخ‌ها) از TSETMC و منابع بورس و فرابورس خوانده می‌شود. نماد باید با نماد ثبت‌شده برای این insCode در TSETMC یکسان باشد.
        اگر نماد قبلاً با insCode دیگری ثبت شده باشد (مثلاً پیش از انتقال بین بازارها)، insCode جدید به همان مارکت اضافه می‌شود.
      </p>
      <ErrorBox error={error} />
    </Modal>
  );
}

export function LinkMarketModal({
  market,
  initialLabel = '',
  onClose,
  onLinked,
}: {
  market: TseMarket;
  initialLabel?: string;
  onClose: () => void;
  onLinked: (market: TseMarket) => void;
}) {
  const [label, setLabel] = useState(initialLabel);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState('');

  const save = async () => {
    setIsSaving(true);
    setError('');
    try {
      onLinked(await tseMarketService.link(market.id, label.trim()));
    } catch (e) {
      setError(errorMessage(e, 'خطا در انتقال به مارکت'));
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Modal
      title="انتقال به مارکت دیگر"
      onClose={onClose}
      footer={
        <>
          <Button className="flex-1" isLoading={isSaving} disabled={!label.trim()} onClick={save}>انتقال</Button>
          <Button variant="ghost" onClick={onClose}>انصراف</Button>
        </>
      }
    >
      <p className="text-sm text-surface-600 dark:text-surface-300">
        {market.symbol} · {market.name} <span dir="ltr">({market.ins_code})</span>
        <br />
        مارکت فعلی: {market.market_label}
      </p>
      <Input label="برچسب مارکت مقصد" placeholder={`مثلاً ${market.symbol}-2`} value={label} onChange={(e) => setLabel(e.target.value)} />
      <p className="text-xs text-surface-500">
        اگر مارکتی با این برچسب باشد insCode به آن می‌پیوندد، وگرنه مارکت تازه‌ای با این برچسب ساخته می‌شود. فقط insCodeهایی که هنوز داده قیمتی برایشان ذخیره نشده منتقل می‌شوند.
      </p>
      <ErrorBox error={error} />
    </Modal>
  );
}

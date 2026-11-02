import { confirmDialog } from '../ui/dialogs';
import { tseMarketService, type TseMarket } from '../../services/tseMarketService';

/**
 * Asks for confirmation and deletes the insCode, or with wholeMarket every insCode
 * of its market. It returns the message to show, or null when the user cancels.
 */
export async function deleteMarket(market: TseMarket, wholeMarket: boolean): Promise<string | null> {
  const lastMember = market.market_members <= 1;
  const confirmed = await confirmDialog({
    title: wholeMarket ? 'حذف کامل نماد' : 'حذف insCode',
    message: wholeMarket
      ? `نماد ${market.market_label} به‌طور کامل حذف شود؟ هر ${market.market_members.toLocaleString('fa-IR')} insCode این مارکت با قیمت‌ها و کندل‌هایشان و خود مارکت حذف می‌شوند.`
      : `insCode ${market.ins_code} نماد ${market.symbol} با قیمت‌ها و کندل‌هایش حذف شود؟${
          lastMember ? ` مارکت ${market.market_label} هم insCode دیگری ندارد و حذف می‌شود.` : ''
        }`,
    warning: 'این کار قابل بازگشت نیست. نمادی که در فهرست بورس باشد با همگام‌سازی بعدی دوباره اضافه می‌شود.',
    confirmLabel: 'حذف',
    tone: 'danger',
  });
  if (!confirmed) return null;
  const result = await tseMarketService.remove(market.id, wholeMarket);
  const markets = result.deleted_markets > 0 ? ` و ${result.deleted_markets.toLocaleString('fa-IR')} مارکت` : '';
  return `${result.deleted_ins_codes.toLocaleString('fa-IR')} insCode${markets} حذف شد.`;
}

/** Asks for confirmation to mark the insCode as delisted, or to clear the mark. */
export function confirmDelisted(market: TseMarket): Promise<boolean> {
  return market.delisted
    ? confirmDialog({
        title: 'برداشتن علامت لغو پذیرش',
        message: `علامت لغو پذیرش ${market.symbol} (${market.ins_code}) برداشته شود؟`,
        warning: 'همگام‌سازی فهرست این علامت را برنمی‌دارد؛ فقط از همین‌جا قابل تغییر است.',
      })
    : confirmDialog({
        title: 'لغو پذیرش',
        message: `نماد ${market.symbol} (${market.ins_code}) لغو پذیرش‌شده علامت بخورد؟`,
        confirmLabel: 'علامت لغو پذیرش',
        tone: 'danger',
      });
}

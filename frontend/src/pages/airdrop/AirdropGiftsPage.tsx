import { AirdropGiftsList } from './AirdropGiftsList';

export function AirdropGiftsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-surface-900 dark:text-white">لیست ایردراپ‌ها</h1>
        <p className="text-sm text-surface-500 dark:text-surface-400 mt-1">لیست هدایا و ایردراپ‌های دریافتی</p>
      </div>
      <AirdropGiftsList />
    </div>
  );
}

import { AirdropGiftsList } from './AirdropGiftsList';

export function AirdropGiftsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-h1 font-bold text-text">لیست ایردراپ‌ها</h1>
        <p className="text-body text-text-subtle mt-1">لیست هدایا و ایردراپ‌های دریافتی</p>
      </div>
      <AirdropGiftsList />
    </div>
  );
}

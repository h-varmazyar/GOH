import { SubscriptionFiltersList } from './SubscriptionFiltersList';

export function SubscriptionFiltersPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-h1 font-bold text-text">فیلترهای اشتراک</h1>
        <p className="text-body text-text-subtle mt-1">
          تخصیص اکانت‌های تلگرام به کانال‌ها برای دریافت ایردراپ
        </p>
      </div>
      <SubscriptionFiltersList />
    </div>
  );
}

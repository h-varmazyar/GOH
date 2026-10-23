import { SubscriptionFiltersList } from './SubscriptionFiltersList';

export function SubscriptionFiltersPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-surface-900 dark:text-white">فیلترهای اشتراک</h1>
        <p className="text-sm text-surface-500 dark:text-surface-400 mt-1">
          تخصیص اکانت‌های تلگرام به کانال‌ها برای دریافت ایردراپ
        </p>
      </div>
      <SubscriptionFiltersList />
    </div>
  );
}

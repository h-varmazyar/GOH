import { KeywordRoutesList } from './KeywordRoutesList';

export function KeywordRoutesPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-surface-900 dark:text-white">کلمات کلیدی تبدیل</h1>
        <p className="text-sm text-surface-500 dark:text-surface-400 mt-1">مدیریت کلمات کلیدی و مسیرهای تبدیل</p>
      </div>
      <KeywordRoutesList />
    </div>
  );
}

import { KeywordRoutesList } from './KeywordRoutesList';

export function KeywordRoutesPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-h1 font-bold text-text">کلمات کلیدی تبدیل</h1>
        <p className="text-body text-text-subtle mt-1">مدیریت کلمات کلیدی و مسیرهای تبدیل</p>
      </div>
      <KeywordRoutesList />
    </div>
  );
}

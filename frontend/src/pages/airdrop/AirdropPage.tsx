import { useState } from 'react';
import { Gift, Replace, Filter } from 'lucide-react';
import { AirdropGiftsList } from './AirdropGiftsList';
import { KeywordRoutesList } from './KeywordRoutesList';
import { SubscriptionFiltersList } from './SubscriptionFiltersList';

export function AirdropPage() {
  const [activeTab, setActiveTab] = useState<'gifts' | 'keywords' | 'subscriptions'>('gifts');

  const tabs = [
    { id: 'gifts' as const, label: 'لیست ایردراپ‌ها', icon: Gift },
    { id: 'keywords' as const, label: 'کلمات کلیدی تبدیل', icon: Replace },
    { id: 'subscriptions' as const, label: 'فیلترهای اشتراک', icon: Filter },
  ];

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h1 className="text-h1 font-bold text-text">ایردراپ</h1>
        <p className="text-body text-text-subtle mt-1">
          مدیریت ایردراپ‌ها و کلمات کلیدی تبدیل
        </p>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 border-b border-border">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`flex items-center gap-2 px-4 py-2 text-body font-medium transition-colors border-b-2 -mb-px
              ${
                activeTab === tab.id
                  ? 'border-selected text-link'
                  : 'border-transparent text-text-subtle hover:text-text'
              }`}
          >
            <tab.icon className="w-4 h-4" />
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      <div>
        {activeTab === 'gifts' && <AirdropGiftsList />}
        {activeTab === 'keywords' && <KeywordRoutesList />}
        {activeTab === 'subscriptions' && <SubscriptionFiltersList />}
      </div>
    </div>
  );
}

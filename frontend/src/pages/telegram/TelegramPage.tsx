import { useState } from 'react';
import { Users } from 'lucide-react';
import { TelegramAccountsList } from './TelegramAccountsList';

export function TelegramPage() {
  const [activeTab, setActiveTab] = useState<'accounts'>('accounts');

  const tabs = [
    { id: 'accounts' as const, label: 'لیست اکانت‌ها', icon: Users },
  ];

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h1 className="text-h1 font-bold text-text">تلگرام</h1>
        <p className="text-body text-text-subtle mt-1">مدیریت اکانت‌های تلگرام</p>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 border-b border-border">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`flex items-center gap-2 px-4 py-2 text-body font-medium transition-colors border-b-2 -mb-px
              ${activeTab === tab.id
                ? 'border-selected text-link'
                : 'border-transparent text-text-subtle hover:text-text'}`}
          >
            <tab.icon className="w-4 h-4" />
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      <div>
        {activeTab === 'accounts' && <TelegramAccountsList />}
      </div>
    </div>
  );
}

import { useState } from 'react';
import { NavLink, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  Gift,
  User,
  LogOut,
  Moon,
  Sun,
  X,
  Send,
  ChevronDown,
  Eye,
  PlusCircle,
  List,
  Replace,
  Filter,
  Shield,
  Settings,
  Cpu,
  Landmark,
  Newspaper,
  Activity,
  LineChart,
  Star,
  CalendarDays,
  BrainCircuit,
  LayoutTemplate,
} from 'lucide-react';
import { useAuthStore } from '../../store/authStore';
import { useThemeStore } from '../../store/themeStore';
import { Logo } from '../ui/Logo';

interface SidebarProps {
  isOpen: boolean;
  onClose: () => void;
}

interface NavItem {
  to: string;
  icon: React.ComponentType<{ className?: string }>;
  label: string;
}

interface NavGroup {
  id: string;
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  basePath: string;
  children: NavItem[];
}

type NavEntry = NavItem | NavGroup;

function isGroup(entry: NavEntry): entry is NavGroup {
  return 'children' in entry;
}

const navEntries: NavEntry[] = [
  { to: '/dashboard', icon: LayoutDashboard, label: 'داشبورد' },
  {
    id: 'strategies',
    icon: BrainCircuit,
    label: 'استراتژی',
    basePath: '/strategies',
    children: [
      { to: '/strategies', icon: List, label: 'لیست استراتژی‌ها' },
      { to: '/strategies/new', icon: PlusCircle, label: 'تعریف استراتژی جدید' },
      { to: '/strategies/templates', icon: LayoutTemplate, label: 'استراتژی‌های آماده' },
    ],
  },
  {
    id: 'telegram',
    icon: Send,
    label: 'تلگرام',
    basePath: '/telegram',
    children: [
      { to: '/telegram', icon: Eye, label: 'نمای کلی' },
      { to: '/telegram/add', icon: PlusCircle, label: 'افزودن اکانت' },
      { to: '/telegram/accounts', icon: List, label: 'لیست اکانت‌ها' },
      { to: '/telegram/proxies', icon: Shield, label: 'پراکسی‌ها' },
    ],
  },
  {
    id: 'airdrop',
    icon: Gift,
    label: 'تبدیل',
    basePath: '/airdrop',
    children: [
      { to: '/airdrop/gifts', icon: Gift, label: 'لیست ایردراپ‌ها' },
      { to: '/airdrop/keywords', icon: Replace, label: 'کلمات کلیدی تبدیل' },
      { to: '/airdrop/subscriptions', icon: Filter, label: 'فیلترهای اشتراک' },
    ],
  },
  {
    id: 'tse',
    icon: Landmark,
    label: 'بورس تهران',
    basePath: '/tse',
    children: [
      { to: '/tse', icon: Eye, label: 'نمای کلی بازار' },
      { to: '/tse/analysis', icon: Newspaper, label: 'تحلیل روزانه' },
      { to: '/tse/indicators', icon: Activity, label: 'اندیکاتورها' },
      { to: '/tse/markets', icon: LineChart, label: 'لیست بازارها' },
      { to: '/tse/watchlist', icon: Star, label: 'دیده‌بان بازار' },
      { to: '/tse/calendar', icon: CalendarDays, label: 'تقویم بازار' },
    ],
  },
  {
    id: 'backend',
    icon: Settings,
    label: 'مدیریت سیستم',
    basePath: '/backend',
    children: [
      { to: '/backend/configs', icon: Settings, label: 'کانفیگ‌ها' },
      { to: '/backend/workers', icon: Cpu, label: 'مدیریت ورکرها' },
    ],
  },
  { to: '/profile', icon: User, label: 'پروفایل' },
];

export function Sidebar({ isOpen, onClose }: SidebarProps) {
  const { logout, user } = useAuthStore();
  const { theme, toggleTheme } = useThemeStore();
  const location = useLocation();

  // Determine which groups should be open by default based on current path
  const getDefaultOpen = (): Record<string, boolean> => {
    const open: Record<string, boolean> = {};
    for (const entry of navEntries) {
      if (isGroup(entry)) {
        open[entry.id] = location.pathname.startsWith(entry.basePath);
      }
    }
    return open;
  };

  const [openGroups, setOpenGroups] = useState<Record<string, boolean>>(getDefaultOpen);

  const toggleGroup = (id: string) => {
    setOpenGroups((prev) => ({ ...prev, [id]: !prev[id] }));
  };

  const fullName = user
    ? `${user.firstName} ${user.lastName}`.trim() || user.username
    : '';

  return (
    <>
      {/* Overlay for mobile */}
      {isOpen && (
        <div
          className="fixed inset-0 bg-black/50 z-40 lg:hidden"
          onClick={onClose}
        />
      )}

      {/* Sidebar */}
      <aside
        className={`fixed top-0 right-0 z-50 h-full w-64 bg-white dark:bg-surface-900 border-l border-surface-200 dark:border-surface-700
          transition-transform duration-200 ease-in-out
          lg:translate-x-0 lg:static lg:z-auto
          ${isOpen ? 'translate-x-0' : 'translate-x-full lg:translate-x-0'}`}
      >
        <div className="flex flex-col h-full">
          {/* Header */}
          <div className="flex items-center justify-between p-4 border-b border-surface-200 dark:border-surface-700">
            <div className="flex items-center gap-2">
              <Logo size={32} />
              <span className="font-bold text-lg text-surface-900 dark:text-white">Gate</span>
            </div>
            <button
              onClick={onClose}
              className="lg:hidden p-1 rounded hover:bg-surface-100 dark:hover:bg-surface-800"
              aria-label="بستن منو"
            >
              <X className="w-5 h-5" />
            </button>
          </div>

          {/* User info */}
          <div className="p-4 border-b border-surface-200 dark:border-surface-700">
            <p className="text-sm font-medium text-surface-900 dark:text-white">{fullName}</p>
            <p className="text-xs text-surface-500 dark:text-surface-400 mt-0.5" dir="ltr">{user?.phone}</p>
          </div>

          {/* Navigation */}
          <nav className="flex-1 p-3 space-y-1 overflow-y-auto">
            {navEntries.map((entry) => {
              if (isGroup(entry)) {
                const isGroupActive = location.pathname.startsWith(entry.basePath);
                const isExpanded = openGroups[entry.id] ?? false;

                return (
                  <div key={entry.id}>
                    {/* Group Header */}
                    <button
                      onClick={() => toggleGroup(entry.id)}
                      className={`flex items-center justify-between w-full px-3 py-2.5 rounded-lg text-sm font-medium transition-colors
                        ${isGroupActive
                          ? 'text-primary-700 dark:text-primary-300'
                          : 'text-surface-600 hover:bg-surface-100 dark:text-surface-400 dark:hover:bg-surface-800'
                        }`}
                    >
                      <div className="flex items-center gap-3">
                        <entry.icon className="w-5 h-5" />
                        {entry.label}
                      </div>
                      <ChevronDown
                        className={`w-4 h-4 transition-transform duration-200 ${isExpanded ? 'rotate-180' : ''}`}
                      />
                    </button>

                    {/* Sub-items */}
                    {isExpanded && (
                      <div className="mt-1 mr-4 border-r border-surface-200 dark:border-surface-700 space-y-0.5">
                        {entry.children.map((child) => (
                          <NavLink
                            key={child.to}
                            to={child.to}
                            end={child.to === entry.basePath}
                            onClick={onClose}
                            className={({ isActive }) =>
                              `flex items-center gap-2 px-3 py-2 pr-4 rounded-lg text-sm transition-colors
                              ${isActive
                                ? 'bg-primary-50 text-primary-700 dark:bg-primary-950 dark:text-primary-300 font-medium'
                                : 'text-surface-500 hover:bg-surface-100 dark:text-surface-400 dark:hover:bg-surface-800'
                              }`
                            }
                          >
                            <child.icon className="w-4 h-4" />
                            {child.label}
                          </NavLink>
                        ))}
                      </div>
                    )}
                  </div>
                );
              }

              // Simple nav item
              return (
                <NavLink
                  key={entry.to}
                  to={entry.to}
                  onClick={onClose}
                  className={({ isActive }) =>
                    `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors
                    ${isActive
                      ? 'bg-primary-50 text-primary-700 dark:bg-primary-950 dark:text-primary-300'
                      : 'text-surface-600 hover:bg-surface-100 dark:text-surface-400 dark:hover:bg-surface-800'
                    }`
                  }
                >
                  <entry.icon className="w-5 h-5" />
                  {entry.label}
                </NavLink>
              );
            })}
          </nav>

          {/* Footer */}
          <div className="p-3 border-t border-surface-200 dark:border-surface-700 space-y-1">
            <button
              onClick={toggleTheme}
              className="flex items-center gap-3 w-full px-3 py-2.5 rounded-lg text-sm font-medium text-surface-600 hover:bg-surface-100 dark:text-surface-400 dark:hover:bg-surface-800 transition-colors"
            >
              {theme === 'dark' ? <Sun className="w-5 h-5" /> : <Moon className="w-5 h-5" />}
              {theme === 'dark' ? 'تم روشن' : 'تم تاریک'}
            </button>
            <button
              onClick={logout}
              className="flex items-center gap-3 w-full px-3 py-2.5 rounded-lg text-sm font-medium text-danger-500 hover:bg-danger-400/10 transition-colors"
            >
              <LogOut className="w-5 h-5" />
              خروج
            </button>
          </div>
        </div>
      </aside>
    </>
  );
}

import { useState } from 'react';
import { NavLink, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  Gift,
  User,
  LogOut,
  Moon,
  Sun,
  Monitor,
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
  ShieldAlert,
  Star,
  CalendarDays,
  BrainCircuit,
  LayoutTemplate,
  History,
} from 'lucide-react';
import { useAuthStore } from '../../store/authStore';
import { useThemeStore, type Theme } from '../../store/themeStore';
import { Logo } from '../ui/Logo';

interface SidebarProps {
  isOpen: boolean;
  onClose: () => void;
}

interface NavItem {
  to: string;
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  /** Active only on this exact path, not on its sub-paths. */
  end?: boolean;
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
      { to: '/tse/markets', icon: LineChart, label: 'لیست بازارها', end: true },
      { to: '/tse/markets/issues', icon: ShieldAlert, label: 'بررسی نمادها' },
      { to: '/tse/markets/trade-history', icon: History, label: 'پایش سابقه معاملات' },
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

const themeOptions: { value: Theme; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
  { value: 'system', label: 'سیستم', icon: Monitor },
  { value: 'light', label: 'روشن', icon: Sun },
  { value: 'dark', label: 'تاریک', icon: Moon },
];

const itemBase =
  'flex w-full items-center gap-3 rounded-md border-s-3 text-body transition-colors duration-(--duration-fast)';
const itemIdle = 'border-transparent text-nav-text-muted hover:bg-white/8 hover:text-nav-text';
const itemActive = 'border-nav-selected bg-white/10 font-semibold text-nav-text';

export function Sidebar({ isOpen, onClose }: SidebarProps) {
  const { logout, user } = useAuthStore();
  const { theme, setTheme } = useThemeStore();
  const location = useLocation();

  // Groups containing the current page start expanded.
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
      {/* Scrim for the mobile drawer */}
      {isOpen && (
        <div className="fixed inset-0 z-(--z-overlay) bg-scrim lg:hidden" onClick={onClose} aria-hidden="true" />
      )}

      <aside
        id="app-sidebar"
        aria-label="ناوبری اصلی"
        className={`fixed inset-y-0 start-0 z-(--z-modal) flex h-full w-72 flex-col bg-nav text-nav-text
          transition-transform duration-(--duration-slow) ease-standard
          lg:static lg:z-auto lg:w-64
          ${isOpen ? '' : 'max-lg:ltr:-translate-x-full max-lg:rtl:translate-x-full'}`}
      >
        {/* Brand */}
        <div className="flex h-16 shrink-0 items-center justify-between border-b border-nav-border px-4">
          <div className="flex items-center gap-3">
            <Logo size={32} />
            <span className="text-h3">Gate</span>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="-me-2 inline-flex h-11 w-11 items-center justify-center rounded-md text-nav-text-muted hover:bg-white/8 hover:text-nav-text lg:hidden"
            aria-label="بستن منو"
          >
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>

        {/* Signed-in user */}
        <div className="border-b border-nav-border px-4 py-3">
          <p className="truncate text-label text-nav-text">{fullName}</p>
          <p className="text-caption text-nav-text-muted" dir="ltr">
            <span className="block text-end">{user?.phone}</span>
          </p>
        </div>

        <nav className="flex-1 space-y-1 overflow-y-auto p-3">
          {navEntries.map((entry) => {
            if (isGroup(entry)) {
              const isGroupActive = location.pathname.startsWith(entry.basePath);
              const isExpanded = openGroups[entry.id] ?? false;
              const panelId = `nav-group-${entry.id}`;

              return (
                <div key={entry.id}>
                  <button
                    type="button"
                    onClick={() => toggleGroup(entry.id)}
                    aria-expanded={isExpanded}
                    aria-controls={panelId}
                    className={`${itemBase} min-h-11 justify-between px-3 ${
                      isGroupActive ? 'border-transparent font-semibold text-nav-text hover:bg-white/8' : itemIdle
                    }`}
                  >
                    <span className="flex items-center gap-3">
                      <entry.icon className="h-5 w-5 shrink-0" aria-hidden="true" />
                      {entry.label}
                    </span>
                    <ChevronDown
                      className={`h-4 w-4 shrink-0 transition-transform duration-(--duration-base) ${isExpanded ? 'rotate-180' : ''}`}
                      aria-hidden="true"
                    />
                  </button>

                  {isExpanded && (
                    <div id={panelId} className="mt-1 ms-5 space-y-0.5 border-s border-nav-border ps-2">
                      {entry.children.map((child) => (
                        <NavLink
                          key={child.to}
                          to={child.to}
                          end={child.end || child.to === entry.basePath}
                          onClick={onClose}
                          className={({ isActive }) => `${itemBase} min-h-10 px-3 ${isActive ? itemActive : itemIdle}`}
                        >
                          <child.icon className="h-4 w-4 shrink-0" aria-hidden="true" />
                          {child.label}
                        </NavLink>
                      ))}
                    </div>
                  )}
                </div>
              );
            }

            return (
              <NavLink
                key={entry.to}
                to={entry.to}
                onClick={onClose}
                className={({ isActive }) => `${itemBase} min-h-11 px-3 ${isActive ? itemActive : itemIdle}`}
              >
                <entry.icon className="h-5 w-5 shrink-0" aria-hidden="true" />
                {entry.label}
              </NavLink>
            );
          })}
        </nav>

        {/* Footer: theme and sign-out */}
        <div className="space-y-2 border-t border-nav-border p-3">
          <div role="group" aria-label="تم" className="grid grid-cols-3 gap-1 rounded-md bg-black/20 p-1">
            {themeOptions.map((option) => {
              const active = theme === option.value;
              return (
                <button
                  key={option.value}
                  type="button"
                  onClick={() => setTheme(option.value)}
                  aria-pressed={active}
                  className={`inline-flex h-9 items-center justify-center gap-1.5 rounded-sm text-caption transition-colors ${
                    active ? 'bg-white/15 font-semibold text-nav-text' : 'text-nav-text-muted hover:text-nav-text'
                  }`}
                >
                  <option.icon className="h-4 w-4" aria-hidden="true" />
                  {option.label}
                </button>
              );
            })}
          </div>
          <button type="button" onClick={logout} className={`${itemBase} min-h-11 px-3 ${itemIdle}`}>
            <LogOut className="h-5 w-5 shrink-0 rtl:-scale-x-100" aria-hidden="true" />
            خروج
          </button>
        </div>
      </aside>
    </>
  );
}

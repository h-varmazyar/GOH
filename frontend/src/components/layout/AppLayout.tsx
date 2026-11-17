import { useState } from 'react';
import { Outlet } from 'react-router-dom';
import { Menu } from 'lucide-react';
import { Sidebar } from './Sidebar';
import { Logo } from '../ui/Logo';

export function AppLayout() {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  return (
    <div className="flex h-dvh overflow-hidden bg-bg">
      <a
        href="#main"
        className="sr-only z-(--z-toast) rounded-md bg-action px-4 py-2 text-on-action focus:not-sr-only focus:absolute focus:start-4 focus:top-4"
      >
        پرش به محتوای اصلی
      </a>

      <Sidebar isOpen={sidebarOpen} onClose={() => setSidebarOpen(false)} />

      <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
        {/* Mobile header: menu button at the inline start, brand at the end */}
        <header className="sticky top-0 z-(--z-sticky) flex h-14 items-center justify-between border-b border-nav-border bg-nav px-2 text-nav-text lg:hidden">
          <button
            type="button"
            onClick={() => setSidebarOpen(true)}
            className="inline-flex h-11 w-11 items-center justify-center rounded-md hover:bg-white/10"
            aria-label="باز کردن منو"
            aria-expanded={sidebarOpen}
            aria-controls="app-sidebar"
          >
            <Menu className="h-5 w-5" aria-hidden="true" />
          </button>
          <div className="flex items-center gap-2 pe-2">
            <Logo size={28} />
            <span className="text-h4">Gate</span>
          </div>
        </header>

        <main id="main" className="flex-1 overflow-y-auto">
          <div className="mx-auto w-full max-w-[1280px] px-4 py-6 md:px-6 lg:px-8 lg:py-8">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}

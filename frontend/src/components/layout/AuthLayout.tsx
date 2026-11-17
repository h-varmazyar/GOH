import type { ReactNode } from 'react';
import { Logo } from '../ui/Logo';

interface AuthLayoutProps {
  title: string;
  subtitle?: ReactNode;
  children: ReactNode;
  /** Width of the form column. */
  size?: 'sm' | 'md';
}

/**
 * Sign-in screens: a brand hero (gradient.brand, text kept on the dark side and large)
 * beside a neutral form panel. The hero collapses into a compact header on small screens.
 */
export function AuthLayout({ title, subtitle, children, size = 'sm' }: AuthLayoutProps) {
  return (
    <div className="flex min-h-dvh bg-bg">
      <aside className="relative hidden w-[44%] max-w-xl flex-col justify-between overflow-hidden bg-brand-gradient p-12 text-white lg:flex">
        <div className="flex items-center gap-3">
          <Logo size={40} />
          <span className="text-h3">Gate Trading</span>
        </div>
        <div className="max-w-sm space-y-4">
          <p className="text-h1">معاملات خودکار، با آرامش و دقت</p>
          <p className="text-body-lg text-neutral-100">
            استراتژی‌ها، داده‌های بازار و اتوماسیون را در یک فضای کاری متمرکز مدیریت کنید.
          </p>
        </div>
        <p className="text-caption text-neutral-100">© Gate Trading</p>
      </aside>

      <main className="flex flex-1 items-center justify-center px-4 py-10 sm:px-6">
        <div className={`w-full ${size === 'md' ? 'max-w-md' : 'max-w-sm'}`}>
          <div className="mb-8 flex flex-col items-center text-center lg:items-start lg:text-start">
            <Logo size={48} className="mb-5 lg:hidden" />
            <h1 className="text-h2 text-text">{title}</h1>
            {subtitle && <p className="mt-2 text-body text-text-subtle">{subtitle}</p>}
          </div>
          <div className="rounded-lg border border-border bg-surface p-6 shadow-sm sm:p-8">{children}</div>
        </div>
      </main>
    </div>
  );
}

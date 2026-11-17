import { ChevronLeft, ChevronRight } from 'lucide-react';
import { Button } from './Button';

interface PaginationProps {
  page: number;
  totalPages: number;
  totalItems?: number;
  onChange: (page: number) => void;
}

export function Pagination({ page, totalPages, totalItems, onChange }: PaginationProps) {
  if (totalPages <= 1) return null;

  const fa = (n: number) => n.toLocaleString('fa-IR');

  return (
    <nav
      aria-label="صفحه‌بندی"
      className="flex flex-wrap items-center justify-between gap-3 border-t border-border px-4 py-3"
    >
      <span className="text-caption text-text-subtle tabular-nums">
        {totalItems === undefined
          ? `صفحه ${fa(page)} از ${fa(totalPages)}`
          : `${fa(totalItems)} مورد · صفحه ${fa(page)} از ${fa(totalPages)}`}
      </span>
      <div className="flex items-center gap-1">
        {/* Chevrons point in reading direction: "previous" is to the right in RTL. */}
        <Button variant="ghost" size="sm" disabled={page <= 1} onClick={() => onChange(page - 1)}>
          <ChevronRight className="h-4 w-4 ltr:rotate-180" aria-hidden="true" /> قبلی
        </Button>
        <Button variant="ghost" size="sm" disabled={page >= totalPages} onClick={() => onChange(page + 1)}>
          بعدی <ChevronLeft className="h-4 w-4 ltr:rotate-180" aria-hidden="true" />
        </Button>
      </div>
    </nav>
  );
}

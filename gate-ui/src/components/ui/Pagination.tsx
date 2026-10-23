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

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 border-t border-surface-200 dark:border-surface-700">
      <span className="text-xs text-surface-500 dark:text-surface-400">
        {totalItems === undefined ? `صفحه ${page} از ${totalPages}` : `${totalItems.toLocaleString('fa-IR')} مورد · صفحه ${page.toLocaleString('fa-IR')} از ${totalPages.toLocaleString('fa-IR')}`}
      </span>
      <div className="flex items-center gap-1">
        <Button variant="ghost" size="sm" disabled={page <= 1} onClick={() => onChange(page - 1)} aria-label="صفحه قبل">
          <ChevronRight className="w-4 h-4" /> قبلی
        </Button>
        <Button variant="ghost" size="sm" disabled={page >= totalPages} onClick={() => onChange(page + 1)} aria-label="صفحه بعد">
          بعدی <ChevronLeft className="w-4 h-4" />
        </Button>
      </div>
    </div>
  );
}

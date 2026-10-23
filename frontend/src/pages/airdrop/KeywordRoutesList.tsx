import { useEffect, useState } from 'react';
import { Plus, ChevronLeft, ChevronRight, X } from 'lucide-react';
import { Card } from '../../components/ui/Card';
import { Button } from '../../components/ui/Button';
import { Input } from '../../components/ui/Input';
import { keywordRouteService, type KeywordRoute, type Pagination } from '../../services/airdropService';

export function KeywordRoutesList() {
  const [routes, setRoutes] = useState<KeywordRoute[]>([]);
  const [pagination, setPagination] = useState<Pagination | null>(null);
  const [page, setPage] = useState(1);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  // Modal state
  const [showModal, setShowModal] = useState(false);
  const [formData, setFormData] = useState({ phrase: '', route: '' });
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    fetchRoutes();
  }, [page]);

  const fetchRoutes = async () => {
    setIsLoading(true);
    setError('');
    try {
      const res = await keywordRouteService.getRoutes(page);
      setRoutes(res.data || []);
      setPagination(res.pagination);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'خطا در دریافت لیست');
    } finally {
      setIsLoading(false);
    }
  };

  const handleOpenCreate = () => {
    setFormData({ phrase: '', route: '' });
    setShowModal(true);
  };

  const handleSave = async () => {
    setIsSaving(true);
    setError('');
    try {
      await keywordRouteService.createRoute(formData);
      setShowModal(false);
      fetchRoutes();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'خطا در ذخیره');
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <>
      <Card>
        {/* Header with Add button */}
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-medium text-surface-700 dark:text-surface-300">
            لیست کلمات کلیدی و مسیرها
          </h3>
          <Button size="sm" onClick={handleOpenCreate}>
            <Plus className="w-4 h-4" />
            افزودن
          </Button>
        </div>

        {error && (
          <div className="mb-4 p-3 rounded-lg bg-danger-50 dark:bg-danger-950/20 border border-danger-200 dark:border-danger-800">
            <p className="text-sm text-danger-700 dark:text-danger-300">{error}</p>
          </div>
        )}

        {isLoading ? (
          <div className="text-center py-8">
            <p className="text-sm text-surface-500 dark:text-surface-400">در حال بارگذاری...</p>
          </div>
        ) : routes.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-sm text-surface-500 dark:text-surface-400">هیچ موردی یافت نشد</p>
          </div>
        ) : (
          <>
            {/* Table */}
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-surface-200 dark:border-surface-700">
                    <th className="text-right py-3 px-2 font-medium text-surface-600 dark:text-surface-400">
                      عبارت
                    </th>
                    <th className="text-right py-3 px-2 font-medium text-surface-600 dark:text-surface-400">
                      مسیر
                    </th>
                    <th className="text-center py-3 px-2 font-medium text-surface-600 dark:text-surface-400">
                      تاریخ ایجاد
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {routes.map((route) => (
                    <tr
                      key={route.ID}
                      className="border-b border-surface-100 dark:border-surface-800 hover:bg-surface-50 dark:hover:bg-surface-800/50"
                    >
                      <td className="py-3 px-2 text-surface-900 dark:text-surface-100">
                        {route.Phrase}
                      </td>
                      <td className="py-3 px-2 text-surface-900 dark:text-surface-100 font-mono text-xs" dir="ltr">
                        {route.Route}
                      </td>
                      <td className="py-3 px-2 text-center text-surface-500 dark:text-surface-400 text-xs" dir="ltr">
                        {new Date(route.CreatedAt).toLocaleDateString('fa-IR')}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Pagination */}
            {pagination && pagination.total_pages > 1 && (
              <div className="flex items-center justify-between mt-4 pt-4 border-t border-surface-200 dark:border-surface-700">
                <p className="text-xs text-surface-500 dark:text-surface-400">
                  صفحه {pagination.page} از {pagination.total_pages} (مجموع: {pagination.total_items})
                </p>
                <div className="flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={page <= 1}
                  >
                    <ChevronRight className="w-4 h-4" />
                  </Button>
                  <span className="px-3 py-1 text-sm text-surface-700 dark:text-surface-300">
                    {pagination.page}
                  </span>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setPage((p) => Math.min(pagination.total_pages, p + 1))}
                    disabled={page >= pagination.total_pages}
                  >
                    <ChevronLeft className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            )}
          </>
        )}
      </Card>

      {/* Create Modal */}
      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/50" onClick={() => setShowModal(false)} />
          <div className="relative bg-white dark:bg-surface-900 rounded-xl shadow-xl w-full max-w-md border border-surface-200 dark:border-surface-700">
            <div className="flex items-center justify-between p-4 border-b border-surface-200 dark:border-surface-700">
              <h3 className="text-lg font-semibold text-surface-900 dark:text-white">
                افزودن کلمه کلیدی
              </h3>
              <button
                onClick={() => setShowModal(false)}
                className="p-1 rounded hover:bg-surface-100 dark:hover:bg-surface-800"
              >
                <X className="w-5 h-5 text-surface-500" />
              </button>
            </div>
            <div className="p-4 space-y-4">
              <Input
                label="عبارت"
                placeholder="عبارت کلیدی..."
                value={formData.phrase}
                onChange={(e) => setFormData({ ...formData, phrase: e.target.value })}
              />
              <Input
                label="مسیر"
                placeholder="مسیر تبدیل..."
                value={formData.route}
                onChange={(e) => setFormData({ ...formData, route: e.target.value })}
                dir="ltr"
              />
            </div>
            <div className="flex gap-2 p-4 border-t border-surface-200 dark:border-surface-700">
              <Button variant="secondary" className="flex-1" onClick={() => setShowModal(false)}>
                انصراف
              </Button>
              <Button
                className="flex-1"
                onClick={handleSave}
                disabled={!formData.phrase.trim() || !formData.route.trim()}
                isLoading={isSaving}
              >
                ذخیره
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

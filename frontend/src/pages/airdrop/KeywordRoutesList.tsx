import { useEffect, useState } from 'react';
import { Plus, ChevronLeft, ChevronRight, X } from 'lucide-react';
import { Card } from '../../components/ui/Card';
import { Button } from '../../components/ui/Button';
import { Input } from '../../components/ui/Input';
import { keywordRouteService, type KeywordRoute, type Pagination } from '../../services/airdropService';
import { Alert } from '../../components/ui/Alert';

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
          <h3 className="text-body font-medium text-text-muted">
            لیست کلمات کلیدی و مسیرها
          </h3>
          <Button size="sm" onClick={handleOpenCreate}>
            <Plus className="w-4 h-4" />
            افزودن
          </Button>
        </div>

        {error && (
          <Alert tone="error" className="mb-4">{error}</Alert>
        )}

        {isLoading ? (
          <div className="text-center py-8">
            <p className="text-body text-text-subtle">در حال بارگذاری...</p>
          </div>
        ) : routes.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-body text-text-subtle">هیچ موردی یافت نشد</p>
          </div>
        ) : (
          <>
            {/* Table */}
            <div className="overflow-x-auto">
              <table className="w-full text-body">
                <thead>
                  <tr className="border-b border-border">
                    <th className="text-start py-3 px-2 text-label text-text-muted">
                      عبارت
                    </th>
                    <th className="text-start py-3 px-2 text-label text-text-muted">
                      مسیر
                    </th>
                    <th className="text-center py-3 px-2 text-label text-text-muted">
                      تاریخ ایجاد
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {routes.map((route) => (
                    <tr
                      key={route.ID}
                      className="border-b border-border hover:bg-surface-muted"
                    >
                      <td className="py-3 px-2 text-text">
                        {route.Phrase}
                      </td>
                      <td className="py-3 px-2 text-text font-mono text-caption" dir="ltr">
                        {route.Route}
                      </td>
                      <td className="py-3 px-2 text-center text-text-subtle text-caption" dir="ltr">
                        {new Date(route.CreatedAt).toLocaleDateString('fa-IR')}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Pagination */}
            {pagination && pagination.total_pages > 1 && (
              <div className="flex items-center justify-between mt-4 pt-4 border-t border-border">
                <p className="text-caption text-text-subtle">
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
                  <span className="px-3 py-1 text-body text-text-muted">
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
        <div role="dialog" aria-modal="true" className="fixed inset-0 z-(--z-modal) flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-scrim" onClick={() => setShowModal(false)} />
          <div className="relative bg-surface rounded-xl shadow-lg w-full max-w-md border border-border">
            <div className="flex items-center justify-between p-4 border-b border-border">
              <h3 className="text-h3 font-semibold text-text">
                افزودن کلمه کلیدی
              </h3>
              <button
                type="button"
                aria-label="بستن"
                onClick={() => setShowModal(false)}
                className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-md text-text-subtle hover:text-text hover:bg-surface-muted"
              >
                <X className="h-5 w-5" aria-hidden="true" />
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
            <div className="flex gap-2 p-4 border-t border-border">
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

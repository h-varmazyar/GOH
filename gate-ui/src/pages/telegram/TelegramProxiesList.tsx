import { useEffect, useState } from 'react';
import { Plus, ChevronLeft, ChevronRight, Trash2, X, Info, Loader2, Clipboard, Upload, Check } from 'lucide-react';
import { Card } from '../../components/ui/Card';
import { Button } from '../../components/ui/Button';
import { Input } from '../../components/ui/Input';
import { Select } from '../../components/ui/Select';
import { Badge } from '../../components/ui/Badge';
import {
  proxyService,
  type TelegramProxy,
  type ProxyType,
  type ProxyStatus,
  type CreateProxyPayload,
} from '../../services/telegramService';
import type { Pagination } from '../../services/airdropService';

type AddMethod = 'manual' | 'clipboard';

/** Parse MTProto proxy URL like: https://t.me/proxy?server=103.161.35.56&port=443&secret=3XnnAQIAAQAH8AMDhuJMOt0 */
function parseMtprotoUrl(url: string): CreateProxyPayload | null {
  try {
    const urlObj = new URL(url);
    if (!urlObj.searchParams.has('server') || !urlObj.searchParams.has('port')) return null;
    
    const server = urlObj.searchParams.get('server') || '';
    const port = Number(urlObj.searchParams.get('port'));
    const secret = urlObj.searchParams.get('secret') || undefined;
    
    if (!server || !port || isNaN(port)) return null;
    
    return {
      type: 'mtproto',
      server,
      port,
      secret,
    };
  } catch {
    return null;
  }
}

/** Parse SOCKS5 or HTTP proxy string like: socks5://user:pass@server:port or server:port */
function parseSocksHttpProxy(line: string): CreateProxyPayload | null {
  line = line.trim();
  if (!line) return null;
  
  // Try URL parsing first
  try {
    const urlObj = new URL(line);
    const type = urlObj.protocol === 'socks5:' ? 'socks5' : urlObj.protocol === 'http:' ? 'http' : null;
    if (!type) return null;
    
    const server = urlObj.hostname;
    const port = Number(urlObj.port);
    const username = urlObj.username || undefined;
    const password = urlObj.password || undefined;
    
    if (!server || !port || isNaN(port)) return null;
    
    return {
      type,
      server,
      port,
      username,
      password,
    };
  } catch {
    // Try simple host:port format
    const parts = line.split(':');
    if (parts.length === 2) {
      const [server, portStr] = parts;
      const port = Number(portStr.trim());
      if (server && port && !isNaN(port)) {
        return {
          type: 'socks5', // Default to SOCKS5 for simple format
          server: server.trim(),
          port,
        };
      }
    }
  }
  
  return null;
}

/** Extract all proxies from clipboard text */
function parseClipboardText(text: string): CreateProxyPayload[] {
  const lines = text.split(/\r?\n/).map(l => l.trim()).filter(l => l.length > 0);
  const results: CreateProxyPayload[] = [];
  
  for (const line of lines) {
    // Try MTProto URL first
    if (line.includes('t.me/proxy')) {
      const proxy = parseMtprotoUrl(line);
      if (proxy) {
        results.push(proxy);
        continue;
      }
    }
    
    // Try SOCKS5/HTTP
    const proxy = parseSocksHttpProxy(line);
    if (proxy) {
      results.push(proxy);
    }
  }
  
  return results;
}

export function TelegramProxiesList() {
  const [proxies, setProxies] = useState<TelegramProxy[]>([]);
  const [pagination, setPagination] = useState<Pagination | null>(null);
  const [page, setPage] = useState(1);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  // Detail dialog
  const [selectedProxy, setSelectedProxy] = useState<TelegramProxy | null>(null);

  // Create dialog states
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [addMethod, setAddMethod] = useState<AddMethod>('manual');
  const [clipboardText, setClipboardText] = useState('');
  const [parsedProxies, setParsedProxies] = useState<CreateProxyPayload[]>([]);
  const [selectedProxyIndices, setSelectedProxyIndices] = useState<number[]>([]);
  
  // Manual form
  const [manualForm, setManualForm] = useState<CreateProxyPayload>({
    type: 'mtproto',
    server: '',
    port: 443,
  });
  
  const [isSaving, setIsSaving] = useState(false);
  const [isDeleting, setIsDeleting] = useState<number | null>(null);
  const [dialogError, setDialogError] = useState('');

  useEffect(() => {
    fetchProxies();
  }, [page]);

  // Auto-read clipboard when clipboard tab is active
  useEffect(() => {
    if (showCreateModal && addMethod === 'clipboard' && !clipboardText) {
      readClipboardAuto();
    }
  }, [showCreateModal, addMethod]);

  // Reset dialog error when modal opens/closes
  useEffect(() => {
    if (showCreateModal) {
      setDialogError('');
    }
  }, [showCreateModal]);

  // Reset dialog error when detail dialog closes
  useEffect(() => {
    if (!selectedProxy) {
      setDialogError('');
    }
  }, [selectedProxy]);

  const fetchProxies = async () => {
    setIsLoading(true);
    setError('');
    try {
      const res = await proxyService.getProxies(page);
      setProxies(res.data || []);
      setPagination(res.pagination);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'خطا در دریافت لیست پراکسی‌ها');
    } finally {
      setIsLoading(false);
    }
  };

  const readClipboardAuto = async () => {
    try {
      const text = await navigator.clipboard.readText();
      setClipboardText(text);
      const parsed = parseClipboardText(text);
      setParsedProxies(parsed);
      setSelectedProxyIndices(parsed.map((_, idx) => idx)); // Select all by default
    } catch (err) {
      console.error('Failed to read clipboard:', err);
      setClipboardText('');
      setParsedProxies([]);
      setSelectedProxyIndices([]);
    }
  };

  const handleDelete = async (proxyId: number, e?: React.MouseEvent) => {
    e?.stopPropagation();
    if (!confirm('آیا از حذف این پراکسی اطمینان دارید؟')) return;
    setIsDeleting(proxyId);
    try {
      await proxyService.delete(String(proxyId));
      // Close detail dialog if deleting the currently viewed proxy
      if (selectedProxy?.ID === proxyId) {
        setSelectedProxy(null);
      }
      fetchProxies();
    } catch (e: unknown) {
      const errorMessage = e instanceof Error ? e.message : 'خطا در حذف پراکسی';
      // اگر حذف از داخل dialog detail است، خطا را در dialog نمایش بده
      if (selectedProxy?.ID === proxyId) {
        // خطا را در یک state جداگانه برای dialog detail نگه داریم
        // فعلاً چون یک dialog داریم، از dialogError استفاده می‌کنیم
        setDialogError(errorMessage);
      } else {
        setError(errorMessage);
      }
    } finally {
      setIsDeleting(null);
    }
  };

  const handleManualCreate = async () => {
    if (!manualForm.server.trim() || !manualForm.port) return;
    setIsSaving(true);
    setDialogError('');
    try {
      await proxyService.create(manualForm);
      setShowCreateModal(false);
      setManualForm({ type: 'mtproto', server: '', port: 443 });
      fetchProxies();
    } catch (e: unknown) {
      setDialogError(e instanceof Error ? e.message : 'خطا در ایجاد پراکسی');
    } finally {
      setIsSaving(false);
    }
  };

  const handleClipboardImport = async () => {
    if (parsedProxies.length === 0) return;
    
    setIsSaving(true);
    setDialogError('');
    try {
      for (const proxy of parsedProxies.filter((_, idx) => selectedProxyIndices.includes(idx))) {
        await proxyService.create(proxy);
      }
      setShowCreateModal(false);
      setClipboardText('');
      setParsedProxies([]);
      setSelectedProxyIndices([]);
      fetchProxies();
    } catch (e: unknown) {
      setDialogError(e instanceof Error ? e.message : 'خطا در ایجاد پراکسی‌ها');
    } finally {
      setIsSaving(false);
    }
  };

  const toggleProxySelection = (index: number) => {
    setSelectedProxyIndices(prev => 
      prev.includes(index) 
        ? prev.filter(i => i !== index)
        : [...prev, index]
    );
  };

  const selectAllProxies = () => {
    setSelectedProxyIndices(parsedProxies.map((_, idx) => idx));
  };

  const deselectAllProxies = () => {
    setSelectedProxyIndices([]);
  };

  const getTypeBadge = (type: string) => {
    switch (type) {
      case 'mtproto':
        return <Badge variant="primary" size="sm">MTProto</Badge>;
      case 'socks5':
        return <Badge variant="warning" size="sm">SOCKS5</Badge>;
      case 'http':
        return <Badge variant="default" size="sm">HTTP</Badge>;
      default:
        return <Badge variant="default" size="sm">{type}</Badge>;
    }
  };

  const getStatusBadge = (status: ProxyStatus, isEnable: boolean) => {
    if (!isEnable) {
      return <Badge variant="danger" size="sm">غیرفعال</Badge>;
    }
    
    switch (status) {
      case 'active':
        return <Badge variant="success" size="sm">فعال</Badge>;
      case 'inactive':
        return <Badge variant="danger" size="sm">غیرفعال</Badge>;
      case 'unknown':
        return <Badge variant="warning" size="sm">نامعلوم</Badge>;
      default:
        return <Badge variant="default" size="sm">{status}</Badge>;
    }
  };

  const formatDate = (dateStr: string | null) => {
    if (!dateStr) return '—';
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return '—';
    return d.toLocaleDateString('fa-IR') + ' ' + d.toLocaleTimeString('fa-IR', { hour: '2-digit', minute: '2-digit' });
  };

  return (
    <>
      <div className="space-y-4">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-surface-900 dark:text-white">لیست پراکسی‌ها</h2>
            <p className="text-sm text-surface-500 dark:text-surface-400 mt-1">مدیریت پراکسی‌های تلگرام</p>
          </div>
          <Button onClick={() => setShowCreateModal(true)} size="sm">
            <Plus className="w-4 h-4" />
            افزودن پراکسی
          </Button>
        </div>

        {error && (
          <div className="p-3 rounded-lg bg-danger-50 dark:bg-danger-950/20 border border-danger-200 dark:border-danger-800">
            <p className="text-sm text-danger-700 dark:text-danger-300">{error}</p>
          </div>
        )}

        {/* Proxies Table */}
        <Card padding="none">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-surface-200 dark:border-surface-700 bg-surface-50 dark:bg-surface-800/50">
                  <th className="text-center px-4 py-3 text-xs font-medium text-surface-500 dark:text-surface-400">نوع</th>
                  <th className="text-right px-4 py-3 text-xs font-medium text-surface-500 dark:text-surface-400">آدرس سرور</th>
                  <th className="text-center px-4 py-3 text-xs font-medium text-surface-500 dark:text-surface-400">پورت</th>
                  <th className="text-center px-4 py-3 text-xs font-medium text-surface-500 dark:text-surface-400">وضعیت</th>
                  <th className="text-center px-4 py-3 text-xs font-medium text-surface-500 dark:text-surface-400">آخرین بررسی</th>
                  <th className="text-center px-4 py-3 text-xs font-medium text-surface-500 dark:text-surface-400">عملیات</th>
                </tr>
              </thead>
              <tbody>
                {isLoading ? (
                  <tr>
                    <td colSpan={6} className="px-4 py-8 text-center">
                      <Loader2 className="w-5 h-5 animate-spin mx-auto text-surface-400" />
                    </td>
                  </tr>
                ) : proxies.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-4 py-8 text-center text-sm text-surface-500 dark:text-surface-400">
                      هیچ پراکسی‌ای وجود ندارد
                    </td>
                  </tr>
                ) : (
                  proxies.map((proxy) => (
                    <tr
                      key={proxy.ID}
                      onClick={() => setSelectedProxy(proxy)}
                      className="border-b border-surface-100 dark:border-surface-700 hover:bg-surface-50 dark:hover:bg-surface-800/50 transition-colors cursor-pointer"
                    >
                      <td className="px-4 py-3 text-center">
                        {getTypeBadge(proxy.type)}
                      </td>
                      <td className="px-4 py-3">
                        <span className="text-sm font-medium text-surface-900 dark:text-white font-mono" dir="ltr">
                          {proxy.server}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-center">
                        <span className="text-sm text-surface-600 dark:text-surface-400 font-mono" dir="ltr">
                          {proxy.port}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-center">
                        {getStatusBadge(proxy.status, proxy.is_enable)}
                      </td>
                      <td className="px-4 py-3 text-center">
                        <span className="text-xs text-surface-500 dark:text-surface-400">
                          {formatDate(proxy.last_checked_at)}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-center">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => handleDelete(proxy.ID, e)}
                          disabled={isDeleting === proxy.ID}
                          className="text-danger-500 hover:text-danger-700"
                        >
                          <Trash2 className="w-4 h-4" />
                        </Button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {pagination && pagination.total_pages > 1 && (
            <div className="flex items-center justify-between px-4 py-3 border-t border-surface-200 dark:border-surface-700">
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
        </Card>
      </div>

      {/* Detail Dialog */}
      {selectedProxy && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/50" onClick={() => {
            setSelectedProxy(null);
            setDialogError('');
          }} />
          <div className="relative bg-white dark:bg-surface-900 rounded-xl shadow-xl w-full max-w-md border border-surface-200 dark:border-surface-700">
            <div className="flex items-center justify-between p-4 border-b border-surface-200 dark:border-surface-700">
              <div className="flex items-center gap-2">
                <Info className="w-5 h-5 text-primary-500" />
                <h3 className="text-lg font-semibold text-surface-900 dark:text-white">
                  جزئیات پراکسی #{selectedProxy.ID}
                </h3>
              </div>
              <button
                onClick={() => {
                  setSelectedProxy(null);
                  setDialogError('');
                }}
                className="p-1 rounded hover:bg-surface-100 dark:hover:bg-surface-800"
              >
                <X className="w-5 h-5 text-surface-500" />
              </button>
            </div>
            <div className="p-4 space-y-3">
              {/* Detail Dialog Error */}
              {dialogError && selectedProxy && selectedProxy.ID === isDeleting && (
                <div className="p-3 rounded-lg bg-danger-50 dark:bg-danger-950/20 border border-danger-200 dark:border-danger-800 mb-3">
                  <p className="text-sm text-danger-700 dark:text-danger-300">{dialogError}</p>
                </div>
              )}

              <DetailRow label="نوع" value={getTypeBadge(selectedProxy.type)} />
              <DetailRow label="آدرس سرور" value={<span className="font-mono" dir="ltr">{selectedProxy.server}</span>} />
              <DetailRow label="پورت" value={<span className="font-mono" dir="ltr">{selectedProxy.port}</span>} />
              <DetailRow label="وضعیت" value={getStatusBadge(selectedProxy.status, selectedProxy.is_enable)} />
              <DetailRow label="آخرین بررسی" value={formatDate(selectedProxy.last_checked_at)} />
              <DetailRow label="آخرین موفقیت" value={formatDate(selectedProxy.last_success_at)} />
              <DetailRow label="شکست‌های متوالی" value={selectedProxy.consecutive_fails} />
              <DetailRow label="تعداد شکست کل" value={selectedProxy.fail_count} />
              <DetailRow label="اکانت‌های اختصاص‌یافته" value={selectedProxy.assigned_accounts} />
              <DetailRow label="تاریخ ایجاد" value={formatDate(selectedProxy.CreatedAt)} />
              <DetailRow label="آخرین به‌روزرسانی" value={formatDate(selectedProxy.UpdatedAt)} />

              {/* Type-specific fields */}
              {selectedProxy.type === 'mtproto' && (
                <DetailRow
                  label="Secret"
                  value={
                    <span className="font-mono text-xs break-all" dir="ltr">
                      {selectedProxy.secret || '—'}
                    </span>
                  }
                />
              )}
              {(selectedProxy.type === 'socks5' || selectedProxy.type === 'http') && (
                <>
                  <DetailRow label="نام کاربری" value={selectedProxy.username || '—'} />
                  <DetailRow label="رمز عبور" value={selectedProxy.password ? '••••••••' : '—'} />
                </>
              )}
            </div>
            <div className="flex gap-2 p-4 border-t border-surface-200 dark:border-surface-700">
              <Button variant="secondary" className="flex-1" onClick={() => {
                setSelectedProxy(null);
                setDialogError('');
              }}>
                بستن
              </Button>
              <Button
                variant="danger"
                className="flex-1"
                onClick={() => handleDelete(selectedProxy.ID)}
                disabled={isDeleting === selectedProxy.ID}
                isLoading={isDeleting === selectedProxy.ID}
              >
                <Trash2 className="w-4 h-4" />
                حذف پراکسی
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Create Dialog */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/50" onClick={() => {
            setShowCreateModal(false);
            setDialogError('');
            setManualForm({ type: 'mtproto', server: '', port: 443 });
            setClipboardText('');
            setParsedProxies([]);
            setSelectedProxyIndices([]);
          }} />
          <div className="relative bg-white dark:bg-surface-900 rounded-xl shadow-xl w-full max-w-lg border border-surface-200 dark:border-surface-700">
            <div className="flex items-center justify-between p-4 border-b border-surface-200 dark:border-surface-700">
              <h3 className="text-lg font-semibold text-surface-900 dark:text-white">
                افزودن پراکسی جدید
              </h3>
              <button
                onClick={() => {
                  setShowCreateModal(false);
                  setDialogError('');
                  setManualForm({ type: 'mtproto', server: '', port: 443 });
                  setClipboardText('');
                  setParsedProxies([]);
                  setSelectedProxyIndices([]);
                }}
                className="p-1 rounded hover:bg-surface-100 dark:hover:bg-surface-800"
              >
                <X className="w-5 h-5 text-surface-500" />
              </button>
            </div>

            {/* Method Selection */}
            <div className="flex gap-1 p-2 border-b border-surface-200 dark:border-surface-700">
              <Button
                variant={addMethod === 'manual' ? 'primary' : 'ghost'}
                size="sm"
                className="flex-1"
                onClick={() => setAddMethod('manual')}
              >
                افزودن دستی
              </Button>
              <Button
                variant={addMethod === 'clipboard' ? 'primary' : 'ghost'}
                size="sm"
                className="flex-1"
                onClick={() => setAddMethod('clipboard')}
              >
                <Clipboard className="w-4 h-4 ml-1" />
                وارد کردن از کلیپ‌بورد
              </Button>
            </div>

            <div className="p-4 space-y-4 max-h-[60vh] overflow-y-auto">
              {/* Dialog Error */}
              {dialogError && (
                <div className="p-3 rounded-lg bg-danger-50 dark:bg-danger-950/20 border border-danger-200 dark:border-danger-800">
                  <p className="text-sm text-danger-700 dark:text-danger-300">{dialogError}</p>
                </div>
              )}

              {addMethod === 'manual' ? (
                // Manual Form
                <div className="space-y-4">
                  <Select
                    label="نوع پراکسی"
                    value={manualForm.type}
                    onChange={(e) =>
                      setManualForm({
                        ...manualForm,
                        type: e.target.value as ProxyType,
                        secret: undefined,
                        username: undefined,
                        password: undefined,
                      })
                    }
                    options={[
                      { value: 'mtproto', label: 'MTProto' },
                      { value: 'socks5', label: 'SOCKS5' },
                      { value: 'http', label: 'HTTP' },
                    ]}
                  />
                  <Input
                    label="آدرس سرور"
                    placeholder="proxy.example.com"
                    value={manualForm.server}
                    onChange={(e) => setManualForm({ ...manualForm, server: e.target.value })}
                    dir="ltr"
                  />
                  <Input
                    label="پورت"
                    type="number"
                    placeholder="443"
                    value={String(manualForm.port)}
                    onChange={(e) => setManualForm({ ...manualForm, port: Number(e.target.value) })}
                    dir="ltr"
                  />

                  {/* MTProto-specific field */}
                  {manualForm.type === 'mtproto' && (
                    <Input
                      label="Secret"
                      placeholder="ee0123456789abcdef..."
                      value={manualForm.secret || ''}
                      onChange={(e) => setManualForm({ ...manualForm, secret: e.target.value })}
                      dir="ltr"
                    />
                  )}

                  {/* SOCKS5 / HTTP specific fields */}
                  {(manualForm.type === 'socks5' || manualForm.type === 'http') && (
                    <>
                      <Input
                        label="نام کاربری (اختیاری)"
                        placeholder="username"
                        value={manualForm.username || ''}
                        onChange={(e) => setManualForm({ ...manualForm, username: e.target.value })}
                        dir="ltr"
                      />
                      <Input
                        label="رمز عبور (اختیاری)"
                        type="password"
                        placeholder="password"
                        value={manualForm.password || ''}
                        onChange={(e) => setManualForm({ ...manualForm, password: e.target.value })}
                        dir="ltr"
                      />
                    </>
                  )}
                </div>
              ) : (
                // Clipboard Import
                <div className="space-y-4">
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-surface-700 dark:text-surface-300">
                      متن کلیپ‌بورد
                    </label>
                    <textarea
                      className="w-full h-32 px-3 py-2 text-sm border rounded-lg border-surface-300 dark:border-surface-600 bg-surface-50 dark:bg-surface-800 text-surface-900 dark:text-white resize-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                      placeholder="در حال خواندن از کلیپ‌بورد..."
                      value={clipboardText}
                      onChange={(e) => {
                        const text = e.target.value;
                        setClipboardText(text);
                        const parsed = parseClipboardText(text);
                        setParsedProxies(parsed);
                        setSelectedProxyIndices(parsed.map((_, idx) => idx));
                      }}
                      dir="ltr"
                    />
                    <p className="text-xs text-surface-500 dark:text-surface-400">
                      فرمت‌های پشتیبانی شده:
                      <br />• لینک‌های MTProto تلگرام: https://t.me/proxy?server=...&port=...&secret=...
                      <br />• SOCKS5: socks5://user:pass@server:port
                      <br />• HTTP: http://user:pass@server:port
                      <br />• ساده: server:port (پیش‌فرض SOCKS5)
                    </p>
                  </div>

                  {parsedProxies.length > 0 && (
                    <div className="space-y-2">
                      <div className="flex items-center justify-between">
                        <h4 className="text-sm font-medium text-surface-700 dark:text-surface-300">
                          پراکسی‌های شناسایی شده ({parsedProxies.length})
                        </h4>
                        <div className="flex gap-1">
                          <Button variant="ghost" size="sm" onClick={selectAllProxies}>
                            انتخاب همه
                          </Button>
                          <Button variant="ghost" size="sm" onClick={deselectAllProxies}>
                            حذف همه
                          </Button>
                        </div>
                      </div>
                      <div className="space-y-1">
                        {parsedProxies.map((proxy, idx) => (
                          <div
                            key={idx}
                            className={`flex items-center gap-2 p-2 rounded-lg border cursor-pointer transition-colors ${
                              selectedProxyIndices.includes(idx)
                                ? 'border-primary-500 bg-primary-50 dark:bg-primary-950/30'
                                : 'border-surface-200 dark:border-surface-700 hover:bg-surface-50 dark:hover:bg-surface-800'
                            }`}
                            onClick={() => toggleProxySelection(idx)}
                          >
                            <div className="w-4 h-4">
                              {selectedProxyIndices.includes(idx) ? (
                                <Check className="w-4 h-4 text-primary-600 dark:text-primary-400" />
                              ) : (
                                <div className="w-4 h-4 border border-surface-300 dark:border-surface-600 rounded" />
                              )}
                            </div>
                            <div className="flex-1 min-w-0">
                              <div className="flex items-center gap-2">
                                {getTypeBadge(proxy.type)}
                                <span className="text-sm font-mono text-surface-900 dark:text-white truncate" dir="ltr">
                                  {proxy.server}:{proxy.port}
                                </span>
                              </div>
                              <div className="text-xs text-surface-500 dark:text-surface-400 mt-0.5">
                                {proxy.type === 'mtproto'
                                  ? proxy.secret
                                    ? `Secret: ${proxy.secret.substring(0, 20)}...`
                                    : 'No secret'
                                  : proxy.username
                                  ? `Auth: ${proxy.username}`
                                  : 'No authentication'}
                              </div>
                            </div>
                          </div>
                        ))}
                      </div>
                      <p className="text-xs text-surface-500 dark:text-surface-400">
                        {selectedProxyIndices.length} از {parsedProxies.length} پراکسی انتخاب شده است
                      </p>
                    </div>
                  )}

                  {parsedProxies.length === 0 && clipboardText && (
                    <div className="p-3 rounded-lg bg-warning-50 dark:bg-warning-950/20 border border-warning-200 dark:border-warning-800">
                      <p className="text-sm text-warning-700 dark:text-warning-300">
                        هیچ پراکسی معتبری در متن شناسایی نشد.
                      </p>
                    </div>
                  )}
                </div>
              )}
            </div>

            <div className="flex gap-2 p-4 border-t border-surface-200 dark:border-surface-700">
              <Button variant="secondary" className="flex-1" onClick={() => {
                setShowCreateModal(false);
                setDialogError('');
                setManualForm({ type: 'mtproto', server: '', port: 443 });
                setClipboardText('');
                setParsedProxies([]);
                setSelectedProxyIndices([]);
              }}>
                انصراف
              </Button>
              {addMethod === 'manual' ? (
                <Button
                  className="flex-1"
                  onClick={handleManualCreate}
                  disabled={!manualForm.server.trim() || !manualForm.port}
                  isLoading={isSaving}
                >
                  ذخیره
                </Button>
              ) : (
                <Button
                  className="flex-1"
                  onClick={handleClipboardImport}
                  disabled={selectedProxyIndices.length === 0 || isSaving}
                  isLoading={isSaving}
                >
                  <Upload className="w-4 h-4 ml-1" />
                  افزودن {selectedProxyIndices.length} پراکسی
                </Button>
              )}
            </div>
          </div>
        </div>
      )}
    </>
  );
}

/** Small helper component for detail rows */
function DetailRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-4">
      <span className="text-sm text-surface-500 dark:text-surface-400 shrink-0">{label}</span>
      <span className="text-sm text-surface-900 dark:text-white text-left">{value}</span>
    </div>
  );
}

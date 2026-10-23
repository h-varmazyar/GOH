import { useCallback, useEffect, useState } from 'react';
import { FileText, Loader2, Play, RefreshCw, Settings, Square, X } from 'lucide-react';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import {
  workerService,
  type Worker,
  type WorkerSetting,
} from '../../services/backendService';
import {
  ConfigValueInput,
  parseConfigValue,
  serializeConfigValue,
} from '../../components/backend/ConfigValueInput';

export function WorkersPage() {
  const [workers, setWorkers] = useState<Worker[]>([]);
  const [loading, setLoading] = useState(true);
  const [actionId, setActionId] = useState<string | null>(null);
  const [error, setError] = useState('');
  const [logsWorker, setLogsWorker] = useState<Worker | null>(null);
  const [logs, setLogs] = useState('');
  const [logsLoading, setLogsLoading] = useState(false);
  const [settingsWorker, setSettingsWorker] = useState<Worker | null>(null);
  const [settings, setSettings] = useState<WorkerSetting[]>([]);
  const [drafts, setDrafts] = useState<Record<string, string>>({});
  const [settingsLoading, setSettingsLoading] = useState(false);
  const [settingsSaving, setSettingsSaving] = useState(false);


  const loadWorkers = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      setWorkers(await workerService.getAll());
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'خطا در دریافت لیست ورکرها');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timeoutId = window.setTimeout(() => { void loadWorkers(); }, 0);
    return () => window.clearTimeout(timeoutId);
  }, [loadWorkers]);

  const toggleWorker = async (worker: Worker) => {
    setActionId(worker.id);
    setError('');
    try {
      await workerService.setEnabled(worker.name, !worker.enabled);
      await loadWorkers();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'عملیات ورکر انجام نشد');
    } finally {
      setActionId(null);
    }
  };

  const openLogs = async (worker: Worker) => {
    setLogsWorker(worker);
    setLogs('');
    setLogsLoading(true);
    try {
      setLogs(await workerService.getLogs(worker.name));
    } catch (err: unknown) {
      setLogs(err instanceof Error ? err.message : 'خطا در دریافت لاگ‌ها');
    } finally {
      setLogsLoading(false);
    }
  };

  const openSettings = async (worker: Worker) => {
    if (worker.enabled || worker.isActive) return;
    setSettingsWorker(worker);
    setSettings([]);
    setSettingsLoading(true);
    setError('');
    try {
      const result = await workerService.getSettings(worker.section);
      setSettings(result);
      setDrafts(Object.fromEntries(result.map((setting) => [
        setting.key,
        serializeConfigValue(setting.value, setting.type),
      ])));
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'خطا در دریافت تنظیمات ورکر');
      setSettingsWorker(null);
    } finally {
      setSettingsLoading(false);
    }
  };

  const saveSettings = async () => {
    if (!settingsWorker || settingsWorker.enabled || settingsWorker.isActive) return;
    setSettingsSaving(true);
    setError('');
    try {
      const payload = Object.fromEntries(settings.map((setting) => [
        setting.key,
        parseConfigValue(drafts[setting.key] ?? '', setting.type),
      ]));
      await workerService.updateSettings(settingsWorker.section, payload);
      setSettingsWorker(null);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'خطا در ذخیره تنظیمات ورکر');
    } finally {
      setSettingsSaving(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold text-surface-900 dark:text-white">مدیریت ورکرها</h1>
          <p className="text-sm text-surface-500 dark:text-surface-400 mt-1">کنترل وضعیت، لاگ‌ها و تنظیمات ورکرهای Backend</p>
        </div>
        <Button variant="secondary" size="sm" onClick={() => void loadWorkers()} disabled={loading}>
          <RefreshCw className="w-4 h-4" /> بروزرسانی
        </Button>
      </div>

      {error && <Card><p className="text-sm text-danger-500">{error}</p></Card>}

      <Card padding="none">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-surface-200 dark:border-surface-700 bg-surface-50 dark:bg-surface-800/50">
                <th className="text-right px-4 py-3 text-xs font-medium text-surface-500">نام ورکر</th>
                <th className="text-center px-4 py-3 text-xs font-medium text-surface-500">وضعیت</th>
                <th className="text-center px-4 py-3 text-xs font-medium text-surface-500">عملیات</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr><td colSpan={3} className="py-12"><Loader2 className="w-6 h-6 animate-spin mx-auto text-surface-400" /></td></tr>
              ) : workers.length === 0 ? (
                <tr><td colSpan={3} className="py-12 text-center text-sm text-surface-500">ورکری از سرور دریافت نشد</td></tr>
              ) : workers.map((worker) => (
                <tr key={worker.id} className="border-b border-surface-100 dark:border-surface-700 hover:bg-surface-50 dark:hover:bg-surface-800/50">
                  <td className="px-4 py-3">
                    <p className="text-sm font-medium text-surface-900 dark:text-white">{worker.name}</p>
                    {worker.section && <p className="text-xs text-surface-400 mt-1" dir="ltr">{worker.section}</p>}
                  </td>
                  <td className="px-4 py-3 text-center">
                    <div className="flex flex-wrap justify-center gap-2">
                      <Badge variant={worker.enabled ? 'success' : 'default'} size="sm">
                        {worker.enabled ? 'فعال' : 'غیرفعال'}
                      </Badge>
                      <Badge variant={worker.isActive ? 'primary' : 'default'} size="sm">
                        {worker.isActive ? 'در حال اجرا' : 'متوقف'}
                      </Badge>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap justify-center gap-2">
                      <Button
                        size="sm"
                        variant={worker.enabled ? 'danger' : 'primary'}
                        onClick={() => void toggleWorker(worker)}
                        isLoading={actionId === worker.id}
                      >
                        {worker.enabled ? <Square className="w-4 h-4" /> : <Play className="w-4 h-4" />}
                        {worker.enabled ? 'غیرفعال‌سازی' : 'فعال‌سازی'}
                      </Button>
                      <Button variant="secondary" size="sm" onClick={() => void openLogs(worker)}>
                        <FileText className="w-4 h-4" /> لاگ‌ها
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={worker.enabled || worker.isActive}
                        title={worker.enabled || worker.isActive ? 'برای ویرایش تنظیمات ابتدا ورکر را غیرفعال و متوقف کنید' : undefined}
                        onClick={() => void openSettings(worker)}
                      >
                        <Settings className="w-4 h-4" /> تنظیمات
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>


      {logsWorker && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/50" onClick={() => setLogsWorker(null)} />
          <div className="relative w-full max-w-3xl rounded-xl border border-surface-200 dark:border-surface-700 bg-white dark:bg-surface-900 shadow-xl">
            <div className="flex items-center justify-between p-4 border-b border-surface-200 dark:border-surface-700">
              <h2 className="font-semibold text-surface-900 dark:text-white">لاگ‌های {logsWorker.name}</h2>
              <button aria-label="بستن" onClick={() => setLogsWorker(null)} className="p-1 rounded hover:bg-surface-100 dark:hover:bg-surface-800"><X className="w-5 h-5" /></button>
            </div>
            <div className="p-4">
              {logsLoading ? <Loader2 className="w-6 h-6 animate-spin mx-auto my-16 text-surface-400" /> : (
                <pre className="min-h-64 max-h-[60vh] overflow-auto rounded-lg bg-surface-950 p-4 text-xs leading-6 text-surface-100 text-left whitespace-pre-wrap" dir="ltr">
                  {logs || 'لاگی ثبت نشده است'}
                </pre>
              )}
            </div>
          </div>
        </div>
      )}

      {settingsWorker && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/50" onClick={() => !settingsSaving && setSettingsWorker(null)} />
          <div className="relative w-full max-w-2xl rounded-xl border border-surface-200 dark:border-surface-700 bg-white dark:bg-surface-900 shadow-xl">
            <div className="flex items-center justify-between p-4 border-b border-surface-200 dark:border-surface-700">
              <div>
                <h2 className="font-semibold text-surface-900 dark:text-white">تنظیمات {settingsWorker.name}</h2>
                <p className="text-xs text-surface-500 mt-1">مقادیر تنظیمات از سرور دریافت شده‌اند</p>
              </div>
              <button aria-label="بستن" disabled={settingsSaving} onClick={() => setSettingsWorker(null)} className="p-1 rounded hover:bg-surface-100 dark:hover:bg-surface-800 disabled:opacity-50"><X className="w-5 h-5" /></button>
            </div>
            <div className="p-4 max-h-[60vh] overflow-y-auto space-y-4">
              {settingsLoading ? <Loader2 className="w-6 h-6 animate-spin mx-auto my-16 text-surface-400" /> : settings.length === 0 ? (
                <p className="py-10 text-center text-sm text-surface-500">تنظیمی برای این ورکر دریافت نشد</p>
              ) : settings.map((setting) => (
                <div key={setting.key}>
                  <div className="flex items-center justify-between gap-2 mb-1.5">
                    <label className="text-sm font-medium text-surface-700 dark:text-surface-300" dir="ltr">{setting.key}</label>
                    {setting.type && <span className="text-xs text-surface-400" dir="ltr">{setting.type}</span>}
                  </div>
                  <ConfigValueInput
                    config={setting}
                    value={drafts[setting.key] ?? ''}
                    onChange={(value) => setDrafts((current) => ({ ...current, [setting.key]: value }))}
                    disabled={!setting.editable || settingsSaving}
                    ariaLabel={`مقدار ${setting.key}`}
                  />
                  {setting.description && <p className="text-xs text-surface-500 mt-1">{setting.description}</p>}
                </div>
              ))}
            </div>
            <div className="flex justify-end gap-2 p-4 border-t border-surface-200 dark:border-surface-700">
              <Button variant="secondary" onClick={() => setSettingsWorker(null)} disabled={settingsSaving}>انصراف</Button>
              <Button onClick={() => void saveSettings()} isLoading={settingsSaving} disabled={settingsLoading || settings.length === 0}>ذخیره تنظیمات</Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
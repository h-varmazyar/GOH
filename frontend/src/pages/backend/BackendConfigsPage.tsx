import { useState } from 'react';
import { Loader2, RefreshCw, Save } from 'lucide-react';
import { Card } from '../../components/ui/Card';
import { Button } from '../../components/ui/Button';
import { Input } from '../../components/ui/Input';
import {
  backendConfigService,
  type BackendConfig,
} from '../../services/backendService';
import {
  ConfigValueInput,
  parseConfigValue,
  serializeConfigValue,
} from '../../components/backend/ConfigValueInput';

export function BackendConfigsPage() {
  const [section, setSection] = useState('');
  const [loadedSection, setLoadedSection] = useState('');
  const [configs, setConfigs] = useState<BackendConfig[]>([]);
  const [drafts, setDrafts] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const [savingKey, setSavingKey] = useState<string | null>(null);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const loadConfigs = async () => {
    const targetSection = section.trim();
    if (!targetSection) {
      setError('نام بخش کانفیگ را وارد کنید');
      return;
    }
    setLoading(true);
    setError('');
    setSuccess('');
    try {
      const result = await backendConfigService.getSection(targetSection);
      setConfigs(result);
      setDrafts(Object.fromEntries(result.map((config) => [
        config.key,
        serializeConfigValue(config.value, config.type),
      ])));
      setLoadedSection(targetSection);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'خطا در دریافت کانفیگ‌ها');
    } finally {
      setLoading(false);
    }
  };

  const saveConfig = async (config: BackendConfig) => {
    if (!loadedSection) return;
    setSavingKey(config.key);
    setError('');
    setSuccess('');
    try {
      const value = parseConfigValue(drafts[config.key] ?? '', config.type);
      await backendConfigService.updateSection(loadedSection, { [config.key]: value });
      setConfigs((current) => current.map((item) => item.key === config.key ? { ...item, value } : item));
      setSuccess(`کانفیگ «${config.key}» بروزرسانی شد`);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'خطا در بروزرسانی کانفیگ');
    } finally {
      setSavingKey(null);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold text-surface-900 dark:text-white">کانفیگ‌های Backend</h1>
          <p className="text-sm text-surface-500 dark:text-surface-400 mt-1">نمایش و بروزرسانی تنظیمات داینامیک سمت سرور</p>
        </div>
        <div className="flex flex-wrap items-end gap-3">
          <div className="w-full sm:w-72">
            <Input
              label="نام بخش کانفیگ"
              value={section}
              onChange={(event) => setSection(event.target.value)}
              onKeyDown={(event) => { if (event.key === 'Enter') void loadConfigs(); }}
              placeholder="برای نمونه: worker.candle-polling"
              dir="ltr"
            />
          </div>
          <Button variant="secondary" size="sm" onClick={() => void loadConfigs()} disabled={loading}>
            <RefreshCw className="w-4 h-4" /> دریافت تنظیمات
          </Button>
        </div>
      </div>

      {error && <Card><p className="text-sm text-danger-500">{error}</p></Card>}
      {success && <Card><p className="text-sm text-success-500">{success}</p></Card>}

      <Card padding="none">
        {loading ? (
          <div className="py-12"><Loader2 className="w-6 h-6 animate-spin mx-auto text-surface-400" /></div>
        ) : !loadedSection ? (
          <p className="py-12 text-center text-sm text-surface-500">برای مشاهده تنظیمات، نام بخش را وارد کنید</p>
        ) : configs.length === 0 ? (
          <p className="py-12 text-center text-sm text-surface-500">کانفیگی از سرور دریافت نشد</p>
        ) : (
          <div className="divide-y divide-surface-200 dark:divide-surface-700">
            {configs.map((config) => (
              <div key={config.key} className="p-4 grid gap-3 lg:grid-cols-[minmax(180px,1fr)_minmax(300px,2fr)_auto] lg:items-start">
                <div>
                  <p className="font-medium text-surface-900 dark:text-white" dir="ltr">{config.key}</p>
                  {config.description && <p className="mt-1 text-xs text-surface-500 dark:text-surface-400">{config.description}</p>}
                  {config.type && <span className="inline-block mt-2 text-xs text-surface-400" dir="ltr">{config.type}</span>}
                </div>
                <ConfigValueInput
                  config={config}
                  value={drafts[config.key] ?? ''}
                  onChange={(value) => setDrafts((current) => ({ ...current, [config.key]: value }))}
                  disabled={!config.editable || savingKey === config.key}
                  ariaLabel={`مقدار ${config.key}`}
                />
                <Button
                  size="sm"
                  onClick={() => void saveConfig(config)}
                  isLoading={savingKey === config.key}
                  disabled={!config.editable || drafts[config.key] === serializeConfigValue(config.value, config.type)}
                >
                  <Save className="w-4 h-4" /> ذخیره
                </Button>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}
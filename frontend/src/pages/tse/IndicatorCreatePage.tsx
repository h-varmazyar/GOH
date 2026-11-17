import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { ArrowRight, Check, Plus, Search } from 'lucide-react';
import { Alert } from '../../components/ui/Alert';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import { Input } from '../../components/ui/Input';
import { IndicatorParamFields } from '../../components/tse/IndicatorParamFields';
import { toDraft, validateDraft, type ParamDraft } from '../../components/tse/indicatorParams';
import { MarketScopeField } from '../../components/tse/MarketScopeField';
import { marketIdsOf, marketScopeError, type MarketScopeValue } from '../../components/tse/marketScope';
import { errorMessage } from '../../components/tse/format';
import { indicatorService, timeframeLabel, type IndicatorCatalog, type IndicatorDefinition } from '../../services/indicatorService';

/** Picks an indicator from the catalog and adds it in one or more timeframes. */
export function IndicatorCreatePage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [catalog, setCatalog] = useState<IndicatorCatalog | null>(null);
  const [loadError, setLoadError] = useState('');
  const [query, setQuery] = useState('');
  // The params of the selected indicator; a newly selected indicator starts from its defaults.
  const [form, setForm] = useState<{ type: string; draft: ParamDraft; errors: Record<string, string> }>({ type: '', draft: {}, errors: {} });
  const [timeframeIds, setTimeframeIds] = useState<number[]>([]);
  const [markets, setMarkets] = useState<MarketScopeValue>({ scope: 'ALL', markets: [] });
  const [formErrors, setFormErrors] = useState<{ timeframes?: string; market?: string }>({});
  const [submitError, setSubmitError] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  const selectedType = searchParams.get('type') || '';
  const selected = catalog?.indicators.find((item) => item.type === selectedType) ?? null;

  useEffect(() => {
    indicatorService
      .catalog()
      .then(setCatalog)
      .catch((e: unknown) => setLoadError(errorMessage(e, 'خطا در دریافت فهرست اندیکاتورها')));
  }, []);

  if (selected && form.type !== selected.type) {
    setForm({ type: selected.type, draft: toDraft(selected), errors: {} });
    setSubmitError('');
  }

  const groups = useMemo(() => {
    if (!catalog) return [];
    const q = query.trim().toLowerCase();
    return catalog.categories
      .map((category) => ({
        ...category,
        items: catalog.indicators.filter(
          (item) => item.category === category.key && (!q || `${item.type} ${item.name} ${item.title}`.toLowerCase().includes(q)),
        ),
      }))
      .filter((group) => group.items.length > 0);
  }, [catalog, query]);

  const choose = (definition: IndicatorDefinition) => setSearchParams({ type: definition.type }, { replace: true });

  const toggleTimeframe = (id: number) => {
    setTimeframeIds((current) => (current.includes(id) ? current.filter((item) => item !== id) : [...current, id]));
    setFormErrors((current) => ({ ...current, timeframes: undefined }));
  };

  const submit = async () => {
    if (!selected) return;
    const { params, errors } = validateDraft(selected, form.draft);
    const nextFormErrors = {
      timeframes: timeframeIds.length === 0 ? 'حداقل یک بازه زمانی انتخاب کنید.' : undefined,
      market: marketScopeError(markets),
    };
    setForm((current) => ({ ...current, errors }));
    setFormErrors(nextFormErrors);
    if (!params || nextFormErrors.timeframes || nextFormErrors.market) return;

    setIsSaving(true);
    setSubmitError('');
    try {
      const created = await indicatorService.create({
        type: selected.type,
        params,
        timeframe_ids: timeframeIds,
        scope: markets.scope,
        market_ids: marketIdsOf(markets),
      });
      navigate('/tse/indicators', {
        state: { notice: `${selected.title} در ${created.length.toLocaleString('fa-IR')} بازه زمانی افزوده شد.` },
      });
    } catch (e) {
      setSubmitError(errorMessage(e, 'خطا در افزودن اندیکاتور'));
    } finally {
      setIsSaving(false);
    }
  };

  const categoryLabel = (key: string) => catalog?.categories.find((category) => category.key === key)?.label ?? key;

  return (
    <div className="space-y-4">
      <div>
        <Link to="/tse/indicators" className="inline-flex items-center gap-1 text-body text-link hover:underline">
          <ArrowRight className="h-4 w-4" aria-hidden="true" />
          اندیکاتورها
        </Link>
        <h1 className="mt-2 text-h3 font-semibold text-text">افزودن اندیکاتور</h1>
        <p className="mt-1 text-body text-text-subtle">اندیکاتور را انتخاب کنید، پارامترها را تنظیم کنید و بازه‌های زمانی محاسبه را مشخص کنید.</p>
      </div>

      {loadError && <Alert tone="error">{loadError}</Alert>}

      <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,5fr)_minmax(0,7fr)]">
        <Card padding="none">
          <div className="border-b border-border p-4">
            <Input leadingIcon={<Search />} aria-label="جستجوی اندیکاتور" placeholder="نام یا کد اندیکاتور..." value={query} onChange={(e) => setQuery(e.target.value)} />
          </div>
          {!catalog && !loadError && <p className="p-4 text-body text-text-subtle">در حال دریافت...</p>}
          {catalog && groups.length === 0 && <p className="p-4 text-body text-text-subtle">اندیکاتوری پیدا نشد.</p>}
          <div className="max-h-[70vh] overflow-y-auto">
            {groups.map((group) => (
              <section key={group.key} aria-label={group.label}>
                <h2 className="sticky top-0 z-[1] bg-surface-muted px-4 py-2 text-label text-text-muted">{group.label}</h2>
                <ul>
                  {group.items.map((item) => {
                    const active = item.type === selectedType;
                    return (
                      <li key={item.type}>
                        <button
                          type="button"
                          aria-pressed={active}
                          onClick={() => choose(item)}
                          className={`flex w-full items-start gap-3 border-s-3 px-4 py-3 text-start transition-colors ${
                            active ? 'border-selected bg-selected/10' : 'border-transparent hover:bg-surface-muted'
                          }`}
                        >
                          <span className="min-w-0 flex-1">
                            <span className="block text-body font-medium text-text">{item.title}</span>
                            <span className="mt-0.5 block text-caption text-text-subtle" dir="ltr">{item.name}</span>
                          </span>
                          {active && <Check className="mt-0.5 h-4 w-4 shrink-0 text-selected" aria-hidden="true" />}
                        </button>
                      </li>
                    );
                  })}
                </ul>
              </section>
            ))}
          </div>
        </Card>

        {!selected ? (
          <Card>
            <p className="text-body text-text-subtle">{catalog ? 'یک اندیکاتور از فهرست انتخاب کنید.' : ''}</p>
          </Card>
        ) : (
          <Card className="space-y-6">
            <div>
              <div className="flex flex-wrap items-center gap-2">
                <h2 className="text-h3 font-semibold text-text">{selected.title}</h2>
                <Badge>{categoryLabel(selected.category)}</Badge>
              </div>
              <p className="mt-1 text-caption text-text-subtle" dir="ltr">{selected.name} · {selected.type}</p>
              <p className="mt-3 text-body text-text-muted">{selected.description}</p>
              <p className="mt-3 text-caption text-text-subtle">
                خروجی‌ها:{' '}
                <span dir="ltr" className="font-mono">{selected.outputs.join(', ')}</span>
              </p>
            </div>

            <fieldset className="space-y-3">
              <legend className="text-body font-semibold text-text">پارامترها</legend>
              <IndicatorParamFields definition={selected} draft={form.draft} errors={form.errors} onChange={(draft) => setForm((current) => ({ ...current, draft }))} />
            </fieldset>

            <fieldset>
              <legend className="text-body font-semibold text-text">بازه‌های زمانی</legend>
              <p className="mt-1 text-caption text-text-subtle">برای هر بازه انتخاب‌شده یک اندیکاتور جداگانه افزوده می‌شود.</p>
              <div className="mt-3 flex flex-wrap gap-2">
                {catalog?.timeframes.map((timeframe) => {
                  const checked = timeframeIds.includes(timeframe.id);
                  return (
                    <label
                      key={timeframe.id}
                      className={`inline-flex h-10 cursor-pointer items-center gap-2 rounded-md border px-3 text-body transition-colors has-focus-visible:ring-2 has-focus-visible:ring-focus-ring ${
                        checked ? 'border-selected bg-selected/10 text-text' : 'border-border-strong text-text-muted hover:bg-surface-muted'
                      }`}
                    >
                      <input type="checkbox" className="h-4 w-4 accent-action" checked={checked} onChange={() => toggleTimeframe(timeframe.id)} />
                      {timeframeLabel(timeframe.label)}
                    </label>
                  );
                })}
              </div>
              {formErrors.timeframes && <p className="mt-2 text-caption text-error">{formErrors.timeframes}</p>}
            </fieldset>

            <MarketScopeField
              value={markets}
              error={formErrors.market}
              onChange={(value) => {
                setMarkets(value);
                setFormErrors((current) => ({ ...current, market: undefined }));
              }}
            />

            {submitError && <Alert tone="error">{submitError}</Alert>}

            <div className="flex flex-wrap justify-end gap-2 border-t border-border pt-4">
              <Button variant="ghost" onClick={() => navigate('/tse/indicators')}>انصراف</Button>
              <Button isLoading={isSaving} loadingLabel="در حال افزودن..." onClick={submit}>
                <Plus className="h-4 w-4" aria-hidden="true" />
                افزودن اندیکاتور
              </Button>
            </div>
          </Card>
        )}
      </div>
    </div>
  );
}

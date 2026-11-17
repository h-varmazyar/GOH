import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ArrowRight, Save } from 'lucide-react';
import { Alert } from '../../components/ui/Alert';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Card } from '../../components/ui/Card';
import { IndicatorParamFields } from '../../components/tse/IndicatorParamFields';
import { MarketScopeField } from '../../components/tse/MarketScopeField';
import { toDraft, validateDraft, type ParamDraft } from '../../components/tse/indicatorParams';
import { marketIdsOf, marketScopeError, type MarketScopeValue } from '../../components/tse/marketScope';
import { errorMessage } from '../../components/tse/format';
import {
  indicatorService,
  timeframeLabel,
  type IndicatorCatalog,
  type IndicatorDefinition,
  type TseIndicator,
} from '../../services/indicatorService';

/** Changes the params and markets of an indicator; its type and timeframe stay. */
export function IndicatorEditPage() {
  const { indicatorId } = useParams();
  const [loaded, setLoaded] = useState<{ item: TseIndicator; catalog: IndicatorCatalog } | null>(null);
  const [loadError, setLoadError] = useState('');

  useEffect(() => {
    let cancelled = false;
    Promise.all([indicatorService.get(Number(indicatorId)), indicatorService.catalog()])
      .then(([item, catalog]) => {
        if (!cancelled) setLoaded({ item, catalog });
      })
      .catch((e: unknown) => {
        if (!cancelled) setLoadError(errorMessage(e, 'خطا در دریافت اندیکاتور'));
      });
    return () => {
      cancelled = true;
    };
  }, [indicatorId]);

  const definition = loaded?.catalog.indicators.find((item) => item.type === loaded.item.type);

  return (
    <div className="space-y-4">
      <div>
        <Link to="/tse/indicators" className="inline-flex items-center gap-1 text-body text-link hover:underline">
          <ArrowRight className="h-4 w-4" aria-hidden="true" />
          اندیکاتورها
        </Link>
        <h1 className="mt-2 text-h3 font-semibold text-text">ویرایش اندیکاتور</h1>
        <p className="mt-1 text-body text-text-subtle">پارامترها و نمادهای محاسبه را تغییر دهید. نوع اندیکاتور و بازه زمانی ثابت است.</p>
      </div>
      {loadError && <Alert tone="error">{loadError}</Alert>}
      {!loaded && !loadError && <p className="text-body text-text-subtle">در حال دریافت...</p>}
      {loaded && !definition && <Alert tone="error">اندیکاتور {loaded.item.type} در فهرست اندیکاتورها نیست و قابل ویرایش نیست.</Alert>}
      {loaded && definition && (
        <EditForm key={loaded.item.id} item={loaded.item} definition={definition} categoryLabel={loaded.catalog.categories.find((c) => c.key === definition.category)?.label ?? ''} />
      )}
    </div>
  );
}

function EditForm({ item, definition, categoryLabel }: { item: TseIndicator; definition: IndicatorDefinition; categoryLabel: string }) {
  const navigate = useNavigate();
  const [draft, setDraft] = useState<ParamDraft>(() => toDraft(definition, item.params));
  const [paramErrors, setParamErrors] = useState<Record<string, string>>({});
  const [markets, setMarkets] = useState<MarketScopeValue>({ scope: item.scope, markets: item.markets });
  const [marketError, setMarketError] = useState<string | undefined>();
  const [saveError, setSaveError] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  const save = async () => {
    const { params, errors } = validateDraft(definition, draft);
    const nextMarketError = marketScopeError(markets);
    setParamErrors(errors);
    setMarketError(nextMarketError);
    if (!params || nextMarketError) return;
    setIsSaving(true);
    setSaveError('');
    try {
      const updated = await indicatorService.update(item.id, { params, scope: markets.scope, market_ids: marketIdsOf(markets) });
      navigate('/tse/indicators', { state: { notice: `${updated.title} در بازه ${timeframeLabel(updated.timeframe)} ذخیره شد.` } });
    } catch (e) {
      setSaveError(errorMessage(e, 'خطا در ذخیره اندیکاتور'));
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Card className="space-y-6">
      <div>
        <div className="flex flex-wrap items-center gap-2">
          <h2 className="text-h3 font-semibold text-text">{definition.title}</h2>
          {categoryLabel && <Badge>{categoryLabel}</Badge>}
          <Badge variant="primary">بازه {timeframeLabel(item.timeframe)}</Badge>
          {!item.enabled && <Badge variant="warning">غیرفعال</Badge>}
        </div>
        <p className="mt-1 text-caption text-text-subtle" dir="ltr">{definition.name} · {definition.type}</p>
        <p className="mt-3 text-body text-text-muted">{definition.description}</p>
      </div>

      <fieldset className="space-y-3">
        <legend className="text-body font-semibold text-text">پارامترها</legend>
        <IndicatorParamFields definition={definition} draft={draft} errors={paramErrors} onChange={setDraft} />
      </fieldset>

      <MarketScopeField
        value={markets}
        error={marketError}
        onChange={(value) => {
          setMarkets(value);
          setMarketError(undefined);
        }}
      />

      {saveError && <Alert tone="error">{saveError}</Alert>}

      <div className="flex flex-wrap justify-end gap-2 border-t border-border pt-4">
        <Button variant="ghost" onClick={() => navigate('/tse/indicators')}>انصراف</Button>
        <Button isLoading={isSaving} loadingLabel="در حال ذخیره..." onClick={save}>
          <Save className="h-4 w-4" aria-hidden="true" />
          ذخیره تغییرات
        </Button>
      </div>
    </Card>
  );
}

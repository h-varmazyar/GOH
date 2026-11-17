import { Input } from '../ui/Input';
import { Select } from '../ui/Select';
import type { IndicatorDefinition } from '../../services/indicatorService';
import type { ParamDraft } from './indicatorParams';

const fa = (value: number) => value.toLocaleString('fa-IR');

const rangeHint = (min?: number, max?: number) => {
  if (min !== undefined && max !== undefined) return `از ${fa(min)} تا ${fa(max)}`;
  if (min !== undefined) return `حداقل ${fa(min)}`;
  if (max !== undefined) return `حداکثر ${fa(max)}`;
  return undefined;
};

/** The fields of the params of an indicator. */
export function IndicatorParamFields({
  definition,
  draft,
  errors,
  onChange,
}: {
  definition: IndicatorDefinition;
  draft: ParamDraft;
  errors: Record<string, string>;
  onChange: (draft: ParamDraft) => void;
}) {
  if (definition.params.length === 0) {
    return <p className="text-body text-text-subtle">این اندیکاتور پارامتری ندارد.</p>;
  }
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      {definition.params.map((spec) =>
        spec.kind === 'SOURCE' ? (
          <Select
            key={spec.key}
            label={spec.label}
            value={draft[spec.key] ?? ''}
            onChange={(e) => onChange({ ...draft, [spec.key]: e.target.value })}
            options={spec.options ?? []}
          />
        ) : (
          <Input
            key={spec.key}
            label={spec.label}
            type="number"
            inputMode={spec.kind === 'INT' ? 'numeric' : 'decimal'}
            step={spec.kind === 'INT' ? 1 : 'any'}
            min={spec.min}
            max={spec.max}
            dir="ltr"
            value={draft[spec.key] ?? ''}
            error={errors[spec.key]}
            hint={rangeHint(spec.min, spec.max)}
            onChange={(e) => onChange({ ...draft, [spec.key]: e.target.value })}
          />
        ),
      )}
    </div>
  );
}

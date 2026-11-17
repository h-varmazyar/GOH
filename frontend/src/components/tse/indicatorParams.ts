import type { IndicatorDefinition, IndicatorParams } from '../../services/indicatorService';

/** Param values as typed in the form, keyed by param key. */
export type ParamDraft = Record<string, string>;

export const toDraft = (definition: IndicatorDefinition, params: IndicatorParams = {}): ParamDraft =>
  Object.fromEntries(definition.params.map((param) => [param.key, String(params[param.key] ?? param.default)]));

const fa = (value: number) => value.toLocaleString('fa-IR');

/** Checks the draft against the param specs; params is set when every param is valid. */
export function validateDraft(definition: IndicatorDefinition, draft: ParamDraft): { params?: IndicatorParams; errors: Record<string, string> } {
  const errors: Record<string, string> = {};
  const params: IndicatorParams = {};
  for (const spec of definition.params) {
    const raw = (draft[spec.key] ?? '').trim();
    if (spec.kind === 'SOURCE') {
      params[spec.key] = raw || String(spec.default);
      continue;
    }
    const value = Number(raw);
    if (raw === '' || !Number.isFinite(value)) {
      errors[spec.key] = 'یک عدد وارد کنید.';
    } else if (spec.kind === 'INT' && !Number.isInteger(value)) {
      errors[spec.key] = 'عدد صحیح وارد کنید.';
    } else if (spec.min !== undefined && value < spec.min) {
      errors[spec.key] = `حداقل ${fa(spec.min)} است.`;
    } else if (spec.max !== undefined && value > spec.max) {
      errors[spec.key] = `حداکثر ${fa(spec.max)} است.`;
    } else {
      params[spec.key] = value;
    }
  }
  if (definition.type === 'MACD' && !errors.fast_period && !errors.slow_period && Number(params.fast_period) >= Number(params.slow_period)) {
    errors.fast_period = 'دوره سریع باید از دوره کند کمتر باشد.';
  }
  return Object.keys(errors).length > 0 ? { errors } : { params, errors };
}

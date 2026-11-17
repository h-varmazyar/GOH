import { api } from './api';
import type { PaginatedResponse } from './airdropService';

export type IndicatorCategory = 'TREND' | 'MOMENTUM' | 'VOLATILITY' | 'VOLUME' | 'PRICE_ACTION';

/** INT and DECIMAL params are numbers; SOURCE params are one of the options. */
export type IndicatorParamKind = 'INT' | 'DECIMAL' | 'SOURCE';

export type IndicatorParamValue = number | string;
export type IndicatorParams = Record<string, IndicatorParamValue>;

export interface IndicatorParamSpec {
  key: string;
  label: string;
  description?: string;
  kind: IndicatorParamKind;
  default: IndicatorParamValue;
  min?: number;
  max?: number;
  options?: { value: string; label: string }[];
}

/** An indicator of the indicators package that can be added. */
export interface IndicatorDefinition {
  type: string;
  name: string;
  title: string;
  description: string;
  category: IndicatorCategory;
  params: IndicatorParamSpec[];
  outputs: string[];
}

export interface IndicatorTimeframe {
  id: number;
  label: string;
  duration_seconds: number;
}

export interface IndicatorCatalog {
  indicators: IndicatorDefinition[];
  categories: { key: IndicatorCategory; label: string }[];
  timeframes: IndicatorTimeframe[];
}

/** ALL: every TSE market; GROUP: a group of markets; MARKET: one market. */
export type IndicatorScope = 'ALL' | 'GROUP' | 'MARKET';

export const scopeLabel: Record<IndicatorScope, string> = {
  ALL: 'همه نمادها',
  GROUP: 'گروهی از نمادها',
  MARKET: 'یک نماد',
};

export interface MarketRef {
  id: number;
  label: string;
}

/** A TSE market, with the details of its main instrument. */
export interface IndicatorMarket {
  id: number;
  label: string;
  name: string;
  market_type: string;
  industry_group_name: string;
  instruments: number;
  /** Every instrument of the market is delisted. */
  delisted: boolean;
}

export interface IndicatorMarketFilter {
  q?: string;
  /** Industry group code. */
  group?: string;
  include_delisted?: boolean;
}

/** The markets of an indicator: none for ALL, one for MARKET and the group for GROUP. */
export interface MarketSelection {
  scope: IndicatorScope;
  market_ids: number[];
}

/** An indicator added to TSE markets. */
export interface TseIndicator {
  id: number;
  type: string;
  name: string;
  title: string;
  category: IndicatorCategory | '';
  params: IndicatorParams;
  outputs: string[];
  timeframe_id: number;
  timeframe: string;
  scope: IndicatorScope;
  /** The market of MARKET or the markets of GROUP; empty for ALL. */
  markets: MarketRef[];
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface IndicatorFilter {
  q?: string;
  category?: string;
  timeframe_id?: number;
  /** Indicators of the market and of the groups that include it. */
  market_id?: number;
  scope?: IndicatorScope;
  enabled?: boolean;
}

export interface CreateIndicator extends MarketSelection {
  type: string;
  params: IndicatorParams;
  timeframe_ids: number[];
}

export interface UpdateIndicator extends MarketSelection {
  params: IndicatorParams;
}

/** Persian labels of timeframe labels, such as 1h. */
export const timeframeLabel = (label: string) => {
  const match = /^(\d+)([mhdwM])$/.exec(label);
  if (!match) return label;
  const unit = { m: 'دقیقه', h: 'ساعت', d: 'روز', w: 'هفته', M: 'ماه' }[match[2] as 'm' | 'h' | 'd' | 'w' | 'M'];
  return `${Number(match[1]).toLocaleString('fa-IR')} ${unit}`;
};

/** The defaults of the params of an indicator. */
export const defaultParams = (definition: IndicatorDefinition): IndicatorParams =>
  Object.fromEntries(definition.params.map((param) => [param.key, param.default]));

/** A short text of the params, such as "period 14 · source CLOSE". */
export const formatParams = (params: IndicatorParams, definition?: IndicatorDefinition) => {
  const entries = Object.entries(params);
  if (entries.length === 0) return 'بدون پارامتر';
  return entries
    .map(([key, value]) => {
      const spec = definition?.params.find((param) => param.key === key);
      const shown = spec?.kind === 'SOURCE' ? String(value) : typeof value === 'number' ? value.toLocaleString('fa-IR') : value;
      return `${spec?.label ?? key}: ${shown}`;
    })
    .join(' · ');
};

const toQuery = (filter: IndicatorFilter | IndicatorMarketFilter, page: number, pageSize: number) => {
  const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
  Object.entries(filter).forEach(([key, value]) => {
    if (value !== undefined && value !== '') params.set(key, String(value));
  });
  return params.toString();
};

export const indicatorService = {
  catalog: () => api.get<IndicatorCatalog>('/v1/tse/indicators/catalog'),

  list: (filter: IndicatorFilter, page = 1, pageSize = 20) =>
    api.get<PaginatedResponse<TseIndicator>>(`/v1/tse/indicators?${toQuery(filter, page, pageSize)}`),

  get: (id: number) => api.get<TseIndicator>(`/v1/tse/indicators/${id}`),

  /** TSE markets, one row per market, by label. */
  markets: (filter: IndicatorMarketFilter, page = 1, pageSize = 30) =>
    api.get<PaginatedResponse<IndicatorMarket>>(`/v1/tse/indicators/markets?${toQuery(filter, page, pageSize)}`),

  /** Adds the indicator in each timeframe; nothing is added when one exists already. */
  create: (body: CreateIndicator) => api.post<TseIndicator[]>('/v1/tse/indicators', body),

  /** Replaces the params and markets; the type and timeframe do not change. */
  update: (id: number, body: UpdateIndicator) => api.put<TseIndicator>(`/v1/tse/indicators/${id}`, body),

  setEnabled: (id: number, enabled: boolean) =>
    api.patch<TseIndicator>(`/v1/tse/indicators/${id}/enabled`, { enabled }),

  remove: (id: number) => api.delete<void>(`/v1/tse/indicators/${id}`),
};

import { api } from './api';
import type { PaginatedResponse } from './airdropService';

export type TseMarketType =
  | 'BOURSE'
  | 'IFB_FIRST'
  | 'IFB_SECOND'
  | 'IFB_BASE'
  | 'IFB_NOAFARIN'
  | 'IFB_AGREEMENT'
  | 'IFB';

export type TseInstrumentKind = 'STOCK' | 'FUND' | 'RIGHT';

export const marketTypeOptions: { value: TseMarketType; label: string }[] = [
  { value: 'BOURSE', label: 'بورس' },
  { value: 'IFB_FIRST', label: 'بازار اول فرابورس' },
  { value: 'IFB_SECOND', label: 'بازار دوم فرابورس' },
  { value: 'IFB_BASE', label: 'بازار پایه فرابورس' },
  { value: 'IFB_NOAFARIN', label: 'بازار نوآفرین فرابورس' },
  { value: 'IFB_AGREEMENT', label: 'بازار توافقی فرابورس' },
  { value: 'IFB', label: 'فرابورس (سایر)' },
];

export const kindOptions: { value: TseInstrumentKind; label: string }[] = [
  { value: 'STOCK', label: 'سهام' },
  { value: 'FUND', label: 'صندوق قابل معامله' },
  { value: 'RIGHT', label: 'حق تقدم' },
];

export interface TseMarket {
  id: number;
  market_id: number;
  ins_code: string;
  instrument_id: string;
  company_isin: string;
  symbol: string;
  market_label: string;
  /** Number of insCodes linked to the market of this insCode. */
  market_members: number;
  name: string;
  english_name: string;
  market_type: TseMarketType | '';
  market_type_label: string;
  kind: TseInstrumentKind | '';
  kind_label: string;
  instrument_type: number;
  board_code: string;
  industry_group_code: string;
  industry_group_name: string;
  industry_sub_code: string;
  delisted: boolean;
  manually_added: boolean;
  insert_date: string | null;
  release_date: string | null;
  release_date_source: string;
  acceptance_date: string | null;
  total_shares: number | null;
  base_volume: number | null;
  nav: number | null;
  eps: number | null;
  sector_pe: number | null;
  psr: number | null;
  info_updated_at: string | null;
  data_collection_enabled: boolean;
  evaluation_enabled: boolean;
}

export interface TseMarketFilter {
  q?: string;
  market_id?: number;
  market_type?: string;
  kind?: string;
  group?: string;
  delisted?: boolean;
  data_collection?: boolean;
  evaluation?: boolean;
}

export interface TseMarketFlags {
  data_collection_enabled?: boolean;
  evaluation_enabled?: boolean;
}

export interface IndustryGroup {
  code: string;
  name: string;
}

const toQuery = (filter: TseMarketFilter | TseIssueFilter, page: number, pageSize: number) => {
  const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
  for (const [key, value] of Object.entries(filter)) {
    if (value !== undefined && value !== '') params.set(key, String(value));
  }
  return params.toString();
};

export type TseIssueSeverity = 'ERROR' | 'WARNING' | 'INFO';
export type TseIssueStatus = 'OPEN' | 'IGNORED' | 'RESOLVED';

export const issueSeverityOptions: { value: TseIssueSeverity; label: string }[] = [
  { value: 'ERROR', label: 'خطا' },
  { value: 'WARNING', label: 'هشدار' },
  { value: 'INFO', label: 'اطلاع' },
];

export const issueStatusOptions: { value: TseIssueStatus; label: string }[] = [
  { value: 'OPEN', label: 'نیازمند تصمیم' },
  { value: 'IGNORED', label: 'تأییدشده' },
  { value: 'RESOLVED', label: 'رفع‌شده' },
];

export interface TseMarketIssue {
  id: number;
  type: string;
  type_label: string;
  severity: TseIssueSeverity;
  status: TseIssueStatus;
  symbol: string;
  name: string;
  detail: string;
  /** insCodes the issue is about. */
  ins_codes: string[];
  /** Markets the issue is about; their insCodes are in instruments too. */
  market_ids: number[];
  note: string;
  created_at: string;
  detected_at: string;
  resolved_at: string | null;
  instruments: TseMarket[];
}

export interface TseIssueFilter {
  q?: string;
  type?: string;
  severity?: string;
  status?: string;
  ins_code?: string;
  market_id?: number;
}

export interface TseIssueSummary {
  types: { type: string; label: string }[];
  counts: { type: string; severity: TseIssueSeverity; status: TseIssueStatus; count: number }[];
}

export const tseMarketService = {
  list: (filter: TseMarketFilter, page = 1, pageSize = 20) =>
    api.get<PaginatedResponse<TseMarket>>(`/v1/tse/markets?${toQuery(filter, page, pageSize)}`),

  get: (id: number) => api.get<TseMarket>(`/v1/tse/markets/${id}`),

  industryGroups: () =>
    api.get<{ data: IndustryGroup[] }>('/v1/tse/markets/industry-groups'),

  setFlags: (id: number, flags: TseMarketFlags) =>
    api.patch<TseMarket>(`/v1/tse/markets/${id}/flags`, flags),

  /** Changes the given markets, or every market matching filter when ids is empty. */
  bulkSetFlags: (flags: TseMarketFlags, target: { ids: number[] } | { filter: TseMarketFilter }) =>
    api.patch<{ updated: number }>('/v1/tse/markets/flags', { ...flags, ...target }),

  create: (symbol: string, insCode: string) =>
    api.post<TseMarket>('/v1/tse/markets', { symbol, ins_code: insCode }),

  /** Reads the insCode again from TSETMC and the listing sources. */
  refresh: (id: number) => api.post<TseMarket>(`/v1/tse/markets/${id}/refresh`, {}),

  setDelisted: (id: number, delisted: boolean) =>
    api.patch<TseMarket>(`/v1/tse/markets/${id}/delisted`, { delisted }),

  /** Moves the insCode to the market with this label, or to a new market with it. */
  link: (id: number, marketLabel: string) =>
    api.patch<TseMarket>(`/v1/tse/markets/${id}/market`, { market_label: marketLabel }),

  /**
   * Deletes the insCode with its prices, or with wholeMarket every insCode of its
   * market. A market left without insCodes is deleted too.
   */
  remove: (id: number, wholeMarket = false) =>
    api.delete<{ deleted_ins_codes: number; deleted_markets: number }>(
      `/v1/tse/markets/${id}${wholeMarket ? '?whole_market=true' : ''}`,
    ),
};

export const tseMarketIssueService = {
  list: (filter: TseIssueFilter, page = 1, pageSize = 20) =>
    api.get<PaginatedResponse<TseMarketIssue>>(`/v1/tse/market-issues?${toQuery(filter, page, pageSize)}`),

  summary: () => api.get<TseIssueSummary>('/v1/tse/market-issues/summary'),

  /** Checks the stored markets again; issues no longer found are resolved. */
  detect: () => api.post<{ found: number; resolved: number }>('/v1/tse/market-issues/detect', {}),

  /** Opens or ignores the given issues, or every issue matching filter. */
  setStatus: (status: 'OPEN' | 'IGNORED', target: { ids: number[] } | { filter: TseIssueFilter }, note?: string) =>
    api.patch<{ updated: number }>('/v1/tse/market-issues/status', { status, note, ...target }),
};

/** Number of open issues, from an issue summary. */
export const openIssueCount = (summary: TseIssueSummary | null, type?: string) =>
  (summary?.counts || [])
    .filter((count) => count.status === 'OPEN' && (!type || count.type === type))
    .reduce((total, count) => total + count.count, 0);

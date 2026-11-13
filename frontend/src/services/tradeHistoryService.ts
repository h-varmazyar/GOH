import { api } from './api';
import type { Pagination, PaginatedResponse } from './airdropService';

/**
 * MISSING is a trading day whose trade history was not fetched yet; MISMATCHED
 * selects fetched days whose totals differ from the closing price. The others
 * are stored results.
 */
export type TradeDayStatus = 'FETCHED' | 'EMPTY' | 'FAILED' | 'MISSING' | 'MISMATCHED';
export type TradeJobStatus = 'PENDING' | 'RUNNING' | 'DONE' | 'FAILED' | 'CANCELED';

export const tradeDayStatusOptions: { value: TradeDayStatus; label: string }[] = [
  { value: 'FETCHED', label: 'دریافت‌شده' },
  { value: 'EMPTY', label: 'پاسخ خالی' },
  { value: 'FAILED', label: 'خطا' },
  { value: 'MISSING', label: 'دریافت‌نشده' },
  { value: 'MISMATCHED', label: 'مغایرت با قیمت پایانی' },
];

export const tradeJobStatusLabel: Record<TradeJobStatus, string> = {
  PENDING: 'در صف',
  RUNNING: 'در حال اجرا',
  DONE: 'پایان‌یافته',
  FAILED: 'ناموفق',
  CANCELED: 'لغوشده',
};

export interface TradeHistoryJob {
  id: number;
  status: TradeJobStatus;
  title: string;
  all_markets: boolean;
  tse_market_ids: number[];
  dates: string[];
  from: string | null;
  to: string | null;
  day_statuses: TradeDayStatus[];
  total_markets: number;
  done_markets: number;
  planned_days: number;
  processed_days: number;
  fetched_days: number;
  empty_days: number;
  failed_days: number;
  current_market: string;
  cancel_requested: boolean;
  error: string;
  worker: string;
  created_at: string;
  started_at: string | null;
  heartbeat_at: string | null;
  finished_at: string | null;
}

/** Counts of the trading days (daily closing price rows) of a market. */
export interface TradeDayCounts {
  total_days: number;
  fetched_days: number;
  empty_days: number;
  failed_days: number;
  missing_days: number;
  /** Fetched days whose trade count or volume, canceled trades left out, differ from the closing price. */
  mismatched_days: number;
  first_date: string | null;
  last_date: string | null;
}

export interface TradeHistorySummary extends TradeDayCounts {
  tse_market_id: number;
  ins_code: string;
  symbol: string;
  active_jobs: TradeHistoryJob[];
}

/** A trading day with its closing price figures and the totals of its fetched trades. */
export interface TradeDay {
  date: string;
  /** 0 is Sunday. */
  weekday: number;
  status: Exclude<TradeDayStatus, 'MISMATCHED'>;
  mismatched: boolean;
  /** From the closing price of the day. */
  trade_count: number;
  volume: number;
  value: number;
  /** Totals of the received trades; the net figures leave canceled trades out. */
  history_trade_count: number;
  history_canceled_count: number;
  history_volume: number;
  history_net_volume: number;
  history_net_value: number;
  attempts: number;
  error: string;
  job_id: number | null;
  fetched_at: string | null;
}

export interface TradeDayFilter {
  status?: TradeDayStatus | '';
  from?: string;
  to?: string;
}

/** Empty problem selects markets with missing or failed days. */
export type TradeGapProblem = '' | 'missing' | 'failed' | 'empty' | 'mismatched' | 'any' | 'all';

export const gapProblemOptions: { value: TradeGapProblem; label: string }[] = [
  { value: '', label: 'روز دریافت‌نشده یا خطادار' },
  { value: 'missing', label: 'روز دریافت‌نشده' },
  { value: 'failed', label: 'روز خطادار' },
  { value: 'empty', label: 'روز با پاسخ خالی' },
  { value: 'mismatched', label: 'مغایرت با قیمت پایانی' },
  { value: 'any', label: 'هر نوع مشکل (شامل خالی و مغایرت)' },
  { value: 'all', label: 'همه نمادها' },
];

export interface TradeGapFilter {
  q?: string;
  problem?: TradeGapProblem;
  delisted?: boolean;
  data_collection?: boolean;
}

export interface TradeGap extends TradeDayCounts {
  tse_market_id: number;
  market_id: number;
  ins_code: string;
  symbol: string;
  name: string;
  delisted: boolean;
  data_collection_enabled: boolean;
  last_fetched_at: string | null;
}

export interface TradeGapTotals {
  markets: number;
  total_days: number;
  fetched_days: number;
  empty_days: number;
  failed_days: number;
  missing_days: number;
  mismatched_days: number;
  /** Markets without daily closing prices; they have no trading days yet. */
  without_days: number;
}

/** Exactly one of tse_market_ids, all_markets and gaps selects the markets. */
export interface CreateTradeJob {
  tse_market_ids?: number[];
  all_markets?: boolean;
  gaps?: TradeGapFilter;
  /** Fetched whatever their status; from, to and day_statuses are then ignored. */
  dates?: string[];
  from?: string;
  to?: string;
  /** Defaults to MISSING and FAILED. */
  day_statuses?: TradeDayStatus[];
}

export interface FetchDayResult {
  date: string;
  status: TradeDayStatus;
  trade_count: number;
  canceled_count: number;
  volume: number;
  net_volume: number;
  attempts: number;
  error: string;
}

const toQuery = (params: object, page?: number, pageSize?: number) => {
  const query = new URLSearchParams();
  if (page) query.set('page', String(page));
  if (pageSize) query.set('page_size', String(pageSize));
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== '' && value !== false) query.set(key, String(value));
  }
  return query.toString();
};

const base = '/v1/tse/trade-history';

export const tradeHistoryService = {
  summary: (tseMarketId: number) => api.get<TradeHistorySummary>(`${base}/markets/${tseMarketId}/summary`),

  days: (tseMarketId: number, filter: TradeDayFilter, page = 1, pageSize = 30) =>
    api.get<PaginatedResponse<TradeDay>>(`${base}/markets/${tseMarketId}/days?${toQuery(filter, page, pageSize)}`),

  /** Fetches one day right away and records the result. */
  fetchDay: (tseMarketId: number, date: string) =>
    api.post<FetchDayResult>(`${base}/markets/${tseMarketId}/days/${date}/fetch`, {}),

  gaps: (filter: TradeGapFilter, page = 1, pageSize = 20) =>
    api.get<{ data: TradeGap[]; pagination: Pagination; totals: TradeGapTotals }>(`${base}/gaps?${toQuery(filter, page, pageSize)}`),

  /** Queues a collection for the tse-trade-history worker. */
  createJob: (job: CreateTradeJob) => api.post<TradeHistoryJob>(`${base}/jobs`, job),

  jobs: (filter: { active?: boolean; status?: TradeJobStatus; tse_market_id?: number }, page = 1, pageSize = 10) =>
    api.get<PaginatedResponse<TradeHistoryJob>>(`${base}/jobs?${toQuery(filter, page, pageSize)}`),

  cancelJob: (id: number) => api.post<TradeHistoryJob>(`${base}/jobs/${id}/cancel`, {}),
};

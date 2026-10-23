import { api } from './api';

// --- Pagination ---

export interface Pagination {
  page: number;
  page_size: number;
  total_items: number;
  total_pages: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: Pagination;
}

// --- Gifts (Airdrops) ---

export interface AirdropGift {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  DeletedAt: string | null;
  TelegramMessageID: number;
  TelegramChatID: number;
  RawMessage: string;
  SearchPhrase: string;
  Route: string;
  GiftCode: string;
  Currency: string;
  Status: string;
  ErrorMessage: string;
}

export const airdropService = {
  /** Get paginated list of airdrop gifts */
  getGifts: (page: number, pageSize: number = 20) =>
    api.get<PaginatedResponse<AirdropGift>>(`/v1/airdrops/gifts?page=${page}&page_size=${pageSize}`),

  /** Get single gift detail */
  getGiftDetail: (id: number) =>
    api.get<AirdropGift>(`/v1/airdrops/gifts/${id}`),

  /** Retry a failed airdrop gift */
  retryGift: (id: number, giftCode?: string) =>
    api.post<AirdropGift>(`/v1/airdrops/gifts/${id}/retry`, { giftCode }),
};

// --- Keyword Routes (کلمات کلیدی تبدیل) ---

export interface KeywordRoute {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  DeletedAt: string | null;
  Phrase: string;
  Route: string;
}

export interface KeywordRoutePayload {
  phrase: string;
  route: string;
}

export const keywordRouteService = {
  /** Get paginated list of keyword routes */
  getRoutes: (page: number, pageSize: number = 20) =>
    api.get<PaginatedResponse<KeywordRoute>>(`/v1/airdrops/routes?page=${page}&page_size=${pageSize}`),

  /** Create a new keyword route */
  createRoute: (payload: KeywordRoutePayload) =>
    api.post<KeywordRoute>('/v1/airdrops/routes', payload),
};

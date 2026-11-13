import { api } from './api';
import type { PaginatedResponse } from './airdropService';

export interface TelegramSessionResponse {
  SessionId: string;
  NextStep: string;
  Failed: boolean;
}

export type TelegramAccountType = 'bot' | 'client';

/** Telegram account returned from the API list */
export interface TelegramAccountListItem {
  ID: string;
  CreatedAt: string;
  UpdatedAt: string;
  DeletedAt: string | null;
  Role: string;
  Status: string;
  DatabaseDirectory: string;
  FilesDirectory: string;
  BotToken: string | null;
  Mobile: string;
  Username: string;
  AccountName: string;
  user_id: number;
}

/** Subscription filter returned from API */
export interface SubscriptionFilter {
  ID: string;
  AccountID: string;
  ChatID: number;
  CreatedAt: string;
  UpdatedAt: string;
  /** Joined from account */
  AccountUsername?: string;
  AccountPhone?: string;
}

export interface CreateSubscriptionFilterPayload {
  account_id: string;
  chat_id: number;
}

/** Normalize API response to handle both PascalCase and snake_case field names */
function normalizeResponse(data: Record<string, unknown>): TelegramSessionResponse {
  return {
    SessionId: (data.SessionId ?? data.session_id ?? '') as string,
    NextStep: (data.NextStep ?? data.next_step ?? '') as string,
    Failed: (data.Failed ?? data.failed ?? false) as boolean,
  };
}

export const telegramService = {
  /** Step 1: Create a new telegram account session */
  addAccount: async (session_type: TelegramAccountType): Promise<TelegramSessionResponse> => {
    const raw = await api.post<Record<string, unknown>>('/v1/telegram/sessions', {
      session_type: session_type,
    });
    return normalizeResponse(raw);
  },

  /** Step 2: Submit phone number */
  submitPhone: async (sessionId: string, phoneNumber: string): Promise<TelegramSessionResponse> => {
    const raw = await api.post<Record<string, unknown>>('/v1/telegram/sessions/phone', {
      session_id: sessionId,
      phone_number: phoneNumber,
    });
    return normalizeResponse(raw);
  },

  /** Step 2 (bot): Submit bot token from @BotFather */
  submitBotToken: async (sessionId: string, botToken: string): Promise<TelegramSessionResponse> => {
    const raw = await api.post<Record<string, unknown>>('/v1/telegram/sessions/bot-token', {
      session_id: sessionId,
      bot_token: botToken,
    });
    return normalizeResponse(raw);
  },

  /** Step 3: Submit OTP code */
  submitOtp: async (sessionId: string, code: string): Promise<TelegramSessionResponse> => {
    const raw = await api.post<Record<string, unknown>>('/v1/telegram/sessions/code', {
      session_id: sessionId,
      code: code,
    });
    return normalizeResponse(raw);
  },

  /** Step 4: Submit 2FA password */
  submitPassword: async (sessionId: string, password: string): Promise<TelegramSessionResponse> => {
    const raw = await api.post<Record<string, unknown>>('/v1/telegram/sessions/password', {
      session_id: sessionId,
      password,
    });
    return normalizeResponse(raw);
  },

  /** Get paginated list of telegram accounts */
  getAccounts: (page: number, pageSize: number = 20) =>
    api.get<PaginatedResponse<TelegramAccountListItem>>(
      `/v1/telegram/accounts?page=${page}&page_size=${pageSize}`
    ),
};

// --- Proxy Types ---

export type ProxyType = 'mtproto' | 'socks5' | 'http';
export type ProxyStatus = 'active' | 'inactive' | 'unknown';

export interface TelegramProxy {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  DeletedAt: string | null;
  server: string;
  port: number;
  type: ProxyType;
  status: ProxyStatus;
  is_enable: boolean;
  secret?: string;
  username?: string;
  password?: string;
  last_checked_at: string | null;
  last_success_at: string | null;
  fail_count: number;
  consecutive_fails: number;
  assigned_accounts: number;
}

export interface CreateProxyPayload {
  type: ProxyType;
  server: string;
  port: number;
  secret?: string;
  username?: string;
  password?: string;
}

export const proxyService = {
  /** Get paginated list of telegram proxies */
  getProxies: (page: number, pageSize: number = 20) =>
    api.get<PaginatedResponse<TelegramProxy>>(
      `/v1/telegram/proxies?page=${page}&page_size=${pageSize}`
    ),

  /** Create a new proxy */
  create: (payload: CreateProxyPayload) =>
    api.post<TelegramProxy>('/v1/telegram/proxies', payload),

  /** Delete a proxy by ID */
  delete: (proxyId: string) =>
    api.delete(`/v1/telegram/proxies/${proxyId}`),
};

export const subscriptionFilterService = {
  /** Get paginated list of subscription filters */
  getFilters: (page: number, pageSize: number = 20) =>
    api.get<PaginatedResponse<SubscriptionFilter>>(
      `/v1/telegram/subscription-filters?page=${page}&page_size=${pageSize}`
    ),

  /** Create a new subscription filter */
  create: (payload: CreateSubscriptionFilterPayload) =>
    api.post<SubscriptionFilter>('/v1/telegram/subscription-filters', payload),

  /** Delete a subscription filter by ID */
  delete: (filterId: string) =>
    api.delete(`/v1/telegram/subscription-filters/${filterId}`),
};

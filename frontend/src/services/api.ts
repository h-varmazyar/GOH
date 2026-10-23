import { tokenStorage, buildStoredAuth } from './tokenStorage';

const BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8575';

export class ApiError extends Error {
  status: number;
  data: unknown;

  constructor(message: string, status: number, data: unknown = null) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.data = data;
  }
}

interface RequestOptions extends Omit<RequestInit, 'body'> {
  body?: unknown;
  /** When true (default), Authorization header is added if a token is stored. */
  auth?: boolean;
  /** Internal flag to prevent infinite refresh loops. */
  _isRetry?: boolean;
}

/** Hooks for the auth store to react to global auth events (e.g., 401). */
type AuthEventListener = () => void;
let onUnauthorized: AuthEventListener | null = null;
export const setUnauthorizedHandler = (handler: AuthEventListener | null) => {
  onUnauthorized = handler;
};

/**
 * Refresh access token. Coalesces concurrent refreshes into a single request.
 */
let refreshPromise: Promise<string | null> | null = null;
async function refreshAccessToken(): Promise<string | null> {
  if (refreshPromise) return refreshPromise;

  refreshPromise = (async () => {
    const stored = tokenStorage.get();
    if (!stored?.refreshToken) return null;

    try {
      const res = await fetch(`${BASE_URL}/v1/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: stored.refreshToken }),
      });

      if (!res.ok) return null;

      const data = (await res.json()) as {
        AccessToken: string;
        RefreshToken: string;
        ExpiresIn: number;
        HashedID: string;
      };

      tokenStorage.set(buildStoredAuth(data));
      return data.AccessToken;
    } catch {
      return null;
    } finally {
      refreshPromise = null;
    }
  })();

  return refreshPromise;
}

export async function apiRequest<T = unknown>(
  path: string,
  options: RequestOptions = {}
): Promise<T> {
  const { body, auth = true, headers = {}, _isRetry = false, ...rest } = options;

  const finalHeaders: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(headers as Record<string, string>),
  };

  if (auth) {
    // Proactively refresh if expired
    if (tokenStorage.isExpired() && !_isRetry) {
      await refreshAccessToken();
    }
    const stored = tokenStorage.get();
    if (stored?.accessToken) {
      finalHeaders['Authorization'] = `Bearer ${stored.accessToken}`;
    }
  }

  const url = path.startsWith('http') ? path : `${BASE_URL}${path}`;

  let response: Response;
  try {
    response = await fetch(url, {
      ...rest,
      headers: finalHeaders,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  } catch (e) {
    throw new ApiError(
      'خطا در ارتباط با سرور. اتصال اینترنت و در دسترس بودن سرور را بررسی کنید.',
      0,
      e
    );
  }

  // Handle 401 with one refresh attempt
  if (response.status === 401 && auth && !_isRetry) {
    const newToken = await refreshAccessToken();
    if (newToken) {
      return apiRequest<T>(path, { ...options, _isRetry: true });
    }
    // Refresh failed - clear auth and notify listener
    tokenStorage.clear();
    onUnauthorized?.();
  }

  let data: unknown = null;
  const contentType = response.headers.get('content-type') || '';
  if (contentType.includes('application/json')) {
    try {
      data = await response.json();
    } catch {
      data = null;
    }
  } else {
    try {
      data = await response.text();
    } catch {
      data = null;
    }
  }

  if (!response.ok) {
    const message = extractErrorMessage(data) || `خطای ${response.status}`;
    throw new ApiError(message, response.status, data);
  }

  return data as T;
}

function extractErrorMessage(data: unknown): string | null {
  if (!data || typeof data !== 'object') return null;
  const obj = data as Record<string, unknown>;
  for (const key of ['message', 'error', 'Message', 'Error']) {
    const val = obj[key];
    if (typeof val === 'string' && val.trim()) return val;
  }
  return null;
}

export const api = {
  get: <T = unknown>(path: string, options: RequestOptions = {}) =>
    apiRequest<T>(path, { ...options, method: 'GET' }),
  post: <T = unknown>(path: string, body?: unknown, options: RequestOptions = {}) =>
    apiRequest<T>(path, { ...options, method: 'POST', body }),
  put: <T = unknown>(path: string, body?: unknown, options: RequestOptions = {}) =>
    apiRequest<T>(path, { ...options, method: 'PUT', body }),
  patch: <T = unknown>(path: string, body?: unknown, options: RequestOptions = {}) =>
    apiRequest<T>(path, { ...options, method: 'PATCH', body }),
  delete: <T = unknown>(path: string, options: RequestOptions = {}) =>
    apiRequest<T>(path, { ...options, method: 'DELETE' }),
};

export { BASE_URL };

/**
 * Centralized token storage. Used by both the API client (to attach
 * Authorization headers) and the auth store (to persist auth state).
 */

const STORAGE_KEY = 'gate_auth';

export interface StoredAuth {
  accessToken: string;
  refreshToken: string;
  hashedId: string;
  expiresAt: number; // ms epoch when access token expires
}

export const tokenStorage = {
  get(): StoredAuth | null {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return null;
      const data = JSON.parse(raw) as StoredAuth;
      if (!data.accessToken || !data.refreshToken) return null;
      return data;
    } catch {
      return null;
    }
  },

  set(auth: StoredAuth) {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(auth));
  },

  clear() {
    localStorage.removeItem(STORAGE_KEY);
  },

  isExpired(buffer = 30): boolean {
    const auth = this.get();
    if (!auth) return true;
    // Treat as expired `buffer` seconds before actual expiration
    return Date.now() >= auth.expiresAt - buffer * 1000;
  },
};

/**
 * Build StoredAuth from server response.
 */
export function buildStoredAuth(response: {
  AccessToken: string;
  RefreshToken: string;
  ExpiresIn: number;
  HashedID: string;
}): StoredAuth {
  return {
    accessToken: response.AccessToken,
    refreshToken: response.RefreshToken,
    hashedId: response.HashedID,
    expiresAt: Date.now() + response.ExpiresIn * 1000,
  };
}

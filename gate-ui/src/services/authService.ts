import { api } from './api';

export interface RegisterPayload {
  first_name: string;
  last_name: string;
  username: string;
  mobile: string;
  password: string;
}

export interface LoginPayload {
  mobile: string;
  password: string;
}

/**
 * Server response shape for login/register success.
 */
export interface AuthTokenResponse {
  AccessToken: string;
  RefreshToken: string;
  ExpiresIn: number; // seconds
  HashedID: string;
}

export interface UserDto {
  id?: string | number;
  hashed_id?: string;
  HashedID?: string;
  first_name?: string;
  last_name?: string;
  username?: string;
  mobile?: string;
  role?: string;
  is_verified?: boolean;
  created_at?: string;
}

export const authService = {
  register: (payload: RegisterPayload) =>
    api.post<AuthTokenResponse>('/v1/auth/register', payload, { auth: false }),

  login: (payload: LoginPayload) =>
    api.post<AuthTokenResponse>('/v1/auth/login', payload, { auth: false }),

  // Refresh access token using refresh token
  refreshToken: (refreshToken: string) =>
    api.post<AuthTokenResponse>(
      '/v1/auth/refresh',
      { refresh_token: refreshToken },
      { auth: false }
    ),

  // Placeholders - will be wired to real endpoints when provided
  forgotPassword: (mobile: string) =>
    api.post<{ message?: string }>('/v1/auth/forgot-password', { mobile }, { auth: false }),

  resetPassword: (token: string, password: string) =>
    api.post<{ message?: string }>(
      '/v1/auth/reset-password',
      { token, password },
      { auth: false }
    ),

  verifyMobile: (code: string) =>
    api.post<{ message?: string }>('/v1/auth/verify-mobile', { code }),

  resendMobileCode: () =>
    api.post<{ message?: string }>('/v1/auth/resend-mobile-code', {}),

  changePassword: (oldPassword: string, newPassword: string) =>
    api.post<{ message?: string }>('/v1/auth/change-password', {
      old_password: oldPassword,
      new_password: newPassword,
    }),
  
  // Logout (server-side) — expects Authorization header
  logout: () => api.delete<{ message?: string }>('/v1/auth/logout'),

  // Get user info (some backends expose this under /v1/user/info)
  userInfo: () => api.get<UserDto>('/v1/user/info'),

  me: () => api.get<UserDto>('/v1/auth/me'),
};

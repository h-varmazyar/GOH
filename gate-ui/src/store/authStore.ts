import { create } from 'zustand';
import type { User, UserRole } from '../types';
import { authService, type UserDto } from '../services/authService';
import { ApiError, setUnauthorizedHandler } from '../services/api';
import { tokenStorage, buildStoredAuth } from '../services/tokenStorage';

interface AuthStore {
  user: User | null;
  hashedId: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;

  login: (mobile: string, password: string) => Promise<{ success: boolean; error?: string }>;
  register: (data: RegisterData) => Promise<{ success: boolean; error?: string }>;
  forgotPassword: (mobile: string) => Promise<{ success: boolean; error?: string }>;
  sendPhoneVerification: () => Promise<{ success: boolean; error?: string }>;
  verifyPhone: (code: string) => Promise<{ success: boolean; error?: string }>;
  fetchProfile: () => Promise<void>;
  logout: () => void;
  updateProfile: (data: Partial<User>) => void;
  changePassword: (oldPassword: string, newPassword: string) => Promise<{ success: boolean; error?: string }>;
}

export interface RegisterData {
  firstName: string;
  lastName: string;
  username: string;
  mobile: string;
  password: string;
}

const PROFILE_STORAGE_KEY = 'gate_user';

const loadProfile = (): User | null => {
  try {
    const raw = localStorage.getItem(PROFILE_STORAGE_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as User;
  } catch {
    return null;
  }
};

const saveProfile = (user: User | null) => {
  if (user) {
    localStorage.setItem(PROFILE_STORAGE_KEY, JSON.stringify(user));
  } else {
    localStorage.removeItem(PROFILE_STORAGE_KEY);
  }
};

const mapUser = (dto: UserDto, fallback: Partial<User> = {}): User => ({
  id: String(dto.id ?? fallback.id ?? ''),
  username: dto.username ?? fallback.username ?? '',
  firstName: dto.first_name ?? fallback.firstName ?? '',
  lastName: dto.last_name ?? fallback.lastName ?? '',
  phone: dto.mobile ?? fallback.phone ?? '',
  role: ((dto.role as UserRole) ?? fallback.role ?? 'user'),
  isVerified: dto.is_verified ?? fallback.isVerified ?? false,
  createdAt: dto.created_at ?? fallback.createdAt ?? new Date().toISOString(),
});

const errorMessage = (e: unknown): string => {
  if (e instanceof ApiError) return e.message;
  if (e instanceof Error) return e.message;
  return 'خطای ناشناخته';
};

const storedAuth = tokenStorage.get();
const initialProfile = storedAuth ? loadProfile() : null;

export const useAuthStore = create<AuthStore>((set, get) => ({
  user: initialProfile,
  hashedId: storedAuth?.hashedId ?? null,
  isAuthenticated: !!storedAuth,
  isLoading: false,

  login: async (mobile, password) => {
    set({ isLoading: true });
    try {
      const res = await authService.login({ mobile, password });
      tokenStorage.set(buildStoredAuth(res));

      // Build a tentative user from inputs (server returns only tokens here)
      const tentativeUser: User = {
        id: res.HashedID,
        username: '',
        firstName: '',
        lastName: '',
        phone: mobile,
        role: 'user',
        isVerified: true,
        createdAt: new Date().toISOString(),
      };
      saveProfile(tentativeUser);
      set({
        user: tentativeUser,
        hashedId: res.HashedID,
        isAuthenticated: true,
        isLoading: false,
      });

      // Fetch the full profile in the background
      get().fetchProfile().catch(() => { /* non-fatal */ });

      return { success: true };
    } catch (e) {
      set({ isLoading: false });
      return { success: false, error: errorMessage(e) };
    }
  },

  register: async (data) => {
    set({ isLoading: true });
    try {
      const res = await authService.register({
        first_name: data.firstName,
        last_name: data.lastName,
        username: data.username,
        mobile: data.mobile,
        password: data.password,
      });
      tokenStorage.set(buildStoredAuth(res));

      const tentativeUser: User = {
        id: res.HashedID,
        username: data.username,
        firstName: data.firstName,
        lastName: data.lastName,
        phone: data.mobile,
        role: 'user',
        isVerified: false,
        createdAt: new Date().toISOString(),
      };
      saveProfile(tentativeUser);
      set({
        user: tentativeUser,
        hashedId: res.HashedID,
        isAuthenticated: true,
        isLoading: false,
      });

      get().fetchProfile().catch(() => { /* non-fatal */ });

      return { success: true };
    } catch (e) {
      set({ isLoading: false });
      return { success: false, error: errorMessage(e) };
    }
  },

  forgotPassword: async (mobile) => {
    set({ isLoading: true });
    try {
      await authService.forgotPassword(mobile);
      set({ isLoading: false });
      return { success: true };
    } catch (e) {
      set({ isLoading: false });
      return { success: false, error: errorMessage(e) };
    }
  },

  sendPhoneVerification: async () => {
    try {
      await authService.resendMobileCode();
      return { success: true };
    } catch (e) {
      return { success: false, error: errorMessage(e) };
    }
  },

  verifyPhone: async (code) => {
    set({ isLoading: true });
    try {
      await authService.verifyMobile(code);
      const user = get().user;
      if (user) {
        const updated = { ...user, isVerified: true };
        saveProfile(updated);
        set({ user: updated });
      }
      set({ isLoading: false });
      return { success: true };
    } catch (e) {
      set({ isLoading: false });
      return { success: false, error: errorMessage(e) };
    }
  },

  fetchProfile: async () => {
    try {
      // Prefer /v1/user/info if available, fallback to /v1/auth/me
      let dto;
      try {
        dto = await authService.userInfo();
      } catch {
        dto = await authService.me();
      }
      const merged = mapUser(dto, get().user || {});
      saveProfile(merged);
      set({ user: merged });
    } catch {
      // Ignore - profile fetch is best-effort
    }
  },

  logout: () => {
    // Attempt server-side logout but always clear local state
    authService.logout().catch(() => {
      /* non-fatal */
    });
    tokenStorage.clear();
    saveProfile(null);
    set({ user: null, hashedId: null, isAuthenticated: false });
  },

  updateProfile: (data) => {
    set((state) => {
      const updated = state.user ? { ...state.user, ...data } : null;
      saveProfile(updated);
      return { user: updated };
    });
  },

  changePassword: async (oldPassword, newPassword) => {
    try {
      await authService.changePassword(oldPassword, newPassword);
      return { success: true };
    } catch (e) {
      return { success: false, error: errorMessage(e) };
    }
  },
}));

// Wire up global 401 handler so the API client can force logout
setUnauthorizedHandler(() => {
  saveProfile(null);
  useAuthStore.setState({ user: null, hashedId: null, isAuthenticated: false });
});

'use client';

import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  useMemo,
} from 'react';
import type { ReactNode } from 'react';
import type { User, RegisterRequest } from '@/types';
import { api } from '@/lib/api';

// ─── Types ────────────────────────────────────────────────────────────────────

export interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (data: RegisterRequest) => Promise<void>;
  logout: () => Promise<void>;
}

// ─── Context ──────────────────────────────────────────────────────────────────

const AuthContext = createContext<AuthContextType | undefined>(undefined);

// ─── Provider ─────────────────────────────────────────────────────────────────

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Check existing session on mount
  useEffect(() => {
    let cancelled = false;

    async function checkAuth() {
      try {
        const response = await api.getMe();
        if (!cancelled) {
          setUser(response.data);
        }
      } catch {
        // Not authenticated — that is fine
        if (!cancelled) {
          setUser(null);
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    checkAuth();

    return () => {
      cancelled = true;
    };
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    // BFF login returns { data: User } with full user info and sets auth cookie
    const response = await api.login({ email, password });
    setUser(response.data);
  }, []);

  const register = useCallback(async (data: RegisterRequest) => {
    // BFF register returns { data: User } with full user info and sets auth cookie
    const response = await api.register(data);
    setUser(response.data);
  }, []);

  const logout = useCallback(async () => {
    try {
      await fetch(
        `${process.env.NEXT_PUBLIC_BFF_URL || 'http://localhost:3001'}/api/auth/logout`,
        {
          method: 'POST',
          credentials: 'include',
        },
      );
    } catch {
      // Ignore logout errors — clear state regardless
    }
    setUser(null);
  }, []);

  const value = useMemo<AuthContextType>(
    () => ({
      user,
      isLoading,
      isAuthenticated: user !== null,
      login,
      register,
      logout,
    }),
    [user, isLoading, login, register, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

// ─── Hook ─────────────────────────────────────────────────────────────────────

export function useAuth(): AuthContextType {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}

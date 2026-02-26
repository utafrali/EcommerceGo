'use client';

import React, {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
} from 'react';
import { useRouter, usePathname } from 'next/navigation';
import type { User, LoginRequest } from '@/types';
import { authApi, setToken, clearToken } from '@/lib/api';

// ─── Types ─────────────────────────────────────────────────────────────────

interface AuthContextValue {
  user: User | null;
  isAuthenticated: boolean;
  isAdmin: boolean;
  isLoading: boolean;
  login: (data: LoginRequest) => Promise<void>;
  logout: () => void;
  error: string | null;
}

// ─── Context ───────────────────────────────────────────────────────────────

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

// ─── Provider ──────────────────────────────────────────────────────────────

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const router = useRouter();
  const pathname = usePathname();

  const isAuthenticated = !!user;
  const isAdmin = user?.role === 'admin';

  // Check for existing session on mount
  useEffect(() => {
    const checkAuth = async () => {
      try {
        const token =
          typeof window !== 'undefined'
            ? localStorage.getItem('cms_auth_token')
            : null;

        if (!token) {
          setIsLoading(false);
          return;
        }

        setToken(token);
        const userData = await authApi.getMe();
        setUser(userData);
      } catch {
        // Token is invalid or expired — clear it
        clearToken();
        setUser(null);
      } finally {
        setIsLoading(false);
      }
    };

    checkAuth();
  }, []);

  // Redirect to login if not authenticated (and not already on login page)
  // Redirect to dashboard if authenticated and on login page
  useEffect(() => {
    if (isLoading) return;
    if (!isAuthenticated && pathname !== '/login') {
      router.replace('/login');
    } else if (isAuthenticated && pathname === '/login') {
      router.replace('/dashboard');
    }
  }, [isLoading, isAuthenticated, pathname, router]);

  const login = useCallback(async (data: LoginRequest) => {
    setError(null);
    try {
      const authResponse = await authApi.login(data);
      setToken(authResponse.access_token);
      setUser(authResponse.user);
      router.push('/dashboard');
    } catch (err) {
      const message =
        err instanceof Error ? err.message : 'Login failed. Please try again.';
      setError(message);
      throw err;
    }
  }, [router]);

  const logout = useCallback(() => {
    clearToken();
    setUser(null);
    router.push('/login');
  }, [router]);

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated,
        isAdmin,
        isLoading,
        login,
        logout,
        error,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

// ─── Hook ──────────────────────────────────────────────────────────────────

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}

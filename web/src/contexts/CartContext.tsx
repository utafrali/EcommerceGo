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
import type { Cart, AddCartItemRequest } from '@/types';
import { api } from '@/lib/api';
import { useAuth } from '@/contexts/AuthContext';

// ─── Types ────────────────────────────────────────────────────────────────────

export interface CartContextType {
  cart: Cart | null;
  itemCount: number;
  isLoading: boolean;
  addItem: (item: AddCartItemRequest) => Promise<void>;
  updateItem: (productId: string, quantity: number) => Promise<void>;
  removeItem: (productId: string) => Promise<void>;
  refreshCart: () => Promise<void>;
}

// ─── Context ──────────────────────────────────────────────────────────────────

const CartContext = createContext<CartContextType | undefined>(undefined);

// ─── Provider ─────────────────────────────────────────────────────────────────

export function CartProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated, isLoading: authLoading } = useAuth();
  const [cart, setCart] = useState<Cart | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const refreshCart = useCallback(async () => {
    if (!isAuthenticated) {
      setCart(null);
      return;
    }
    setIsLoading(true);
    try {
      const response = await api.getCart();
      setCart(response.data);
    } catch {
      setCart(null);
    } finally {
      setIsLoading(false);
    }
  }, [isAuthenticated]);

  // Load cart when authentication state changes
  useEffect(() => {
    if (authLoading) return;

    if (isAuthenticated) {
      refreshCart();
    } else {
      setCart(null);
    }
  }, [isAuthenticated, authLoading, refreshCart]);

  const addItem = useCallback(
    async (item: AddCartItemRequest) => {
      const response = await api.addToCart(item);
      setCart(response.data);
    },
    [],
  );

  const updateItem = useCallback(
    async (productId: string, quantity: number) => {
      const response = await api.updateCartItem(productId, quantity);
      setCart(response.data);
    },
    [],
  );

  const removeItem = useCallback(async (productId: string) => {
    await api.removeFromCart(productId);
    // DELETE returns 204 (no body), so re-fetch the cart
    await refreshCart();
  }, [refreshCart]);

  const itemCount = useMemo(() => {
    if (!cart?.items) return 0;
    return cart.items.reduce((sum, item) => sum + item.quantity, 0);
  }, [cart]);

  const value = useMemo<CartContextType>(
    () => ({
      cart,
      itemCount,
      isLoading,
      addItem,
      updateItem,
      removeItem,
      refreshCart,
    }),
    [cart, itemCount, isLoading, addItem, updateItem, removeItem, refreshCart],
  );

  return <CartContext.Provider value={value}>{children}</CartContext.Provider>;
}

// ─── Hook ─────────────────────────────────────────────────────────────────────

export function useCart(): CartContextType {
  const context = useContext(CartContext);
  if (context === undefined) {
    throw new Error('useCart must be used within a CartProvider');
  }
  return context;
}

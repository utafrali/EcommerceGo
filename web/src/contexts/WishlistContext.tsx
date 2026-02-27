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
import { api } from '@/lib/api';
import { useAuth } from '@/contexts/AuthContext';

// ─── Types ────────────────────────────────────────────────────────────────────

export interface WishlistContextType {
  wishlistIds: Set<string>;
  count: number;
  isLoading: boolean;
  toggle: (productId: string) => Promise<void>;
  isInWishlist: (productId: string) => boolean;
}

// ─── Context ──────────────────────────────────────────────────────────────────

const WishlistContext = createContext<WishlistContextType | undefined>(
  undefined,
);

// ─── Provider ─────────────────────────────────────────────────────────────────

export function WishlistProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated, isLoading: authLoading } = useAuth();
  const [wishlistIds, setWishlistIds] = useState<Set<string>>(new Set());
  const [isLoading, setIsLoading] = useState(false);

  /**
   * Fetch all wishlist item IDs from the API.
   * We paginate through all pages to build the complete Set of product IDs.
   */
  const fetchWishlist = useCallback(async () => {
    if (!isAuthenticated) {
      setWishlistIds(new Set());
      return;
    }

    setIsLoading(true);
    try {
      const ids = new Set<string>();
      let page = 1;
      let hasMore = true;

      while (hasMore) {
        const response = await api.getWishlist(page);
        const data = response.data;

        if (data.items && data.items.length > 0) {
          for (const item of data.items) {
            ids.add(item.product_id);
          }
          // Check if there are more pages
          const totalPages = Math.ceil(data.total / data.per_page);
          hasMore = page < totalPages;
          page++;
        } else {
          hasMore = false;
        }
      }

      setWishlistIds(ids);
    } catch {
      // On error (e.g. 401), clear the wishlist
      setWishlistIds(new Set());
    } finally {
      setIsLoading(false);
    }
  }, [isAuthenticated]);

  // Reload wishlist when authentication state changes
  useEffect(() => {
    if (authLoading) return;

    if (isAuthenticated) {
      fetchWishlist();
    } else {
      setWishlistIds(new Set());
    }
  }, [isAuthenticated, authLoading, fetchWishlist]);

  /**
   * Toggle a product in the wishlist.
   * Uses optimistic updates: immediately update the Set, then call the API.
   * Reverts on error.
   */
  const toggle = useCallback(
    async (productId: string) => {
      const wasInWishlist = wishlistIds.has(productId);

      // Optimistic update
      setWishlistIds((prev) => {
        const next = new Set(prev);
        if (wasInWishlist) {
          next.delete(productId);
        } else {
          next.add(productId);
        }
        return next;
      });

      try {
        if (wasInWishlist) {
          await api.removeFromWishlist(productId);
        } else {
          await api.addToWishlist(productId);
        }
      } catch {
        // Revert optimistic update on error
        setWishlistIds((prev) => {
          const reverted = new Set(prev);
          if (wasInWishlist) {
            reverted.add(productId);
          } else {
            reverted.delete(productId);
          }
          return reverted;
        });
        throw new Error(
          wasInWishlist
            ? 'Failed to remove from wishlist'
            : 'Failed to add to wishlist',
        );
      }
    },
    [wishlistIds],
  );

  const isInWishlist = useCallback(
    (productId: string) => wishlistIds.has(productId),
    [wishlistIds],
  );

  const count = wishlistIds.size;

  const value = useMemo<WishlistContextType>(
    () => ({
      wishlistIds,
      count,
      isLoading,
      toggle,
      isInWishlist,
    }),
    [wishlistIds, count, isLoading, toggle, isInWishlist],
  );

  return (
    <WishlistContext.Provider value={value}>{children}</WishlistContext.Provider>
  );
}

// ─── Hook ─────────────────────────────────────────────────────────────────────

export function useWishlist(): WishlistContextType {
  const context = useContext(WishlistContext);
  if (context === undefined) {
    throw new Error('useWishlist must be used within a WishlistProvider');
  }
  return context;
}

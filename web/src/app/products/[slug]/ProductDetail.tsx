'use client';

import { useState, useEffect, useMemo, useCallback } from 'react';
import Link from 'next/link';
import type { Product, ProductVariant, ReviewSummary } from '@/types';
import { ImageGallery, RatingStars, PriceDisplay, QuantitySelector, Badge } from '@/components/ui';
import { WishlistButton } from '@/components/ui/WishlistButton';
import { useCart } from '@/contexts/CartContext';
import { useToast } from '@/components/ui/Toast';
import { cn } from '@/lib/utils';

// ─── Props ────────────────────────────────────────────────────────────────────

interface ProductDetailProps {
  product: Product;
  reviewSummary: ReviewSummary;
}

// ─── Recently Viewed Tracking ─────────────────────────────────────────────────

const RECENTLY_VIEWED_KEY = 'recentlyViewed';
const MAX_RECENTLY_VIEWED = 20;

interface RecentlyViewedItem {
  id: string;
  slug: string;
  name: string;
  base_price: number;
  currency: string;
  images?: Product['images'];
  category?: Product['category'];
  timestamp: number;
}

function trackRecentlyViewed(product: Product) {
  try {
    const stored = localStorage.getItem(RECENTLY_VIEWED_KEY);
    const items: RecentlyViewedItem[] = stored ? JSON.parse(stored) : [];

    // Remove existing entry for this product (dedup by id)
    const filtered = items.filter((item) => item.id !== product.id);

    // Prepend current product
    filtered.unshift({
      id: product.id,
      slug: product.slug,
      name: product.name,
      base_price: product.base_price,
      currency: product.currency,
      images: product.images,
      category: product.category,
      timestamp: Date.now(),
    });

    // Keep only the most recent entries
    const trimmed = filtered.slice(0, MAX_RECENTLY_VIEWED);

    localStorage.setItem(RECENTLY_VIEWED_KEY, JSON.stringify(trimmed));
  } catch {
    // localStorage may be unavailable -- silently ignore
  }
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function calculateDiscountPercentage(originalPrice: number, salePrice: number): number | null {
  if (originalPrice <= 0 || salePrice >= originalPrice) return null;
  return Math.round(((originalPrice - salePrice) / originalPrice) * 100);
}

// ─── Component ────────────────────────────────────────────────────────────────

export function ProductDetail({ product, reviewSummary }: ProductDetailProps) {
  const { addItem } = useCart();
  const { toast } = useToast();

  const [quantity, setQuantity] = useState(1);
  const [selectedVariant, setSelectedVariant] = useState<ProductVariant | null>(null);
  const [isAddingToCart, setIsAddingToCart] = useState(false);

  // Track recently viewed on mount
  useEffect(() => {
    trackRecentlyViewed(product);
  }, [product]);

  // Extract unique attribute keys and their values from variants
  const attributeOptions = useMemo(() => {
    if (!product.variants || product.variants.length === 0) return {};

    const options: Record<string, string[]> = {};
    for (const variant of product.variants) {
      if (!variant.is_active) continue;
      for (const [key, value] of Object.entries(variant.attributes)) {
        if (!options[key]) options[key] = [];
        if (!options[key].includes(value)) {
          options[key].push(value);
        }
      }
    }
    return options;
  }, [product.variants]);

  // Track selected attribute values
  const [selectedAttributes, setSelectedAttributes] = useState<Record<string, string>>(() => {
    // Initialize with first active variant's attributes, or empty
    if (product.variants && product.variants.length > 0) {
      const firstActive = product.variants.find((v) => v.is_active);
      return firstActive?.attributes ? { ...firstActive.attributes } : {};
    }
    return {};
  });

  // Find matching variant when attributes change
  useEffect(() => {
    if (!product.variants || product.variants.length === 0) {
      setSelectedVariant(null);
      return;
    }

    const match = product.variants.find((variant) => {
      if (!variant.is_active) return false;
      return Object.entries(selectedAttributes).every(
        ([key, value]) => variant.attributes[key] === value,
      );
    });

    setSelectedVariant(match || null);
  }, [selectedAttributes, product.variants]);

  const handleAttributeSelect = useCallback((key: string, value: string) => {
    setSelectedAttributes((prev) => ({ ...prev, [key]: value }));
  }, []);

  // Determine displayed price
  const displayPrice =
    selectedVariant?.price !== null && selectedVariant?.price !== undefined
      ? selectedVariant.price
      : product.base_price;

  // Calculate discount percentage (variant price vs base price)
  const discountPercent = useMemo(() => {
    if (selectedVariant?.price !== null && selectedVariant?.price !== undefined) {
      return calculateDiscountPercentage(product.base_price, selectedVariant.price);
    }
    return null;
  }, [product.base_price, selectedVariant]);

  // Build the image URL for the cart
  const primaryImageUrl = useMemo(() => {
    if (product.images && product.images.length > 0) {
      const primary = product.images.find((img) => img.is_primary);
      return primary?.url || product.images[0].url;
    }
    return '';
  }, [product.images]);

  // Add to cart handler
  const handleAddToCart = useCallback(async () => {
    setIsAddingToCart(true);
    try {
      const variant = selectedVariant || product.variants?.[0];
      await addItem({
        product_id: product.id,
        variant_id: variant?.id || product.id,
        name: product.name,
        sku: variant?.sku || product.slug,
        price: displayPrice,
        quantity,
        image_url: primaryImageUrl,
      });
      toast.success(`${product.name} sepete eklendi`);
    } catch {
      toast.error('Ürün sepete eklenemedi. Lütfen tekrar deneyin.');
    } finally {
      setIsAddingToCart(false);
    }
  }, [addItem, product.id, product.name, product.slug, product.variants, selectedVariant, displayPrice, quantity, primaryImageUrl, toast]);

  const hasVariants = Object.keys(attributeOptions).length > 0;
  const images = product.images || [];

  return (
    <>
      <div className="grid grid-cols-1 gap-8 lg:grid-cols-2">
        {/* Left: Image Gallery */}
        <div>
          <ImageGallery images={images} />
        </div>

        {/* Right: Product Info */}
        <div className="flex flex-col">
          {/* Brand */}
          {product.brand && (
            <Link
              href={`/products?brand_id=${product.brand.id}`}
              className="mb-1 text-sm font-medium text-brand hover:text-brand-light transition-colors"
            >
              {product.brand.name}
            </Link>
          )}

          {/* Product Name + Wishlist */}
          <div className="flex items-start justify-between gap-4">
            <h1 className="mb-3 text-2xl font-bold text-stone-900 sm:text-3xl">
              {product.name}
            </h1>
            <WishlistButton productId={product.id} size="md" />
          </div>

          {/* Rating Summary */}
          {reviewSummary.total_count > 0 && (
            <div className="mb-4">
              <RatingStars
                rating={reviewSummary.average_rating}
                count={reviewSummary.total_count}
                size="md"
              />
            </div>
          )}

          {/* Price + Discount Badge */}
          <div className="mb-6 flex items-center gap-3">
            <PriceDisplay price={displayPrice} currency={product.currency} size="lg" />
            {discountPercent && discountPercent > 0 && (
              <span className="rounded-full bg-brand px-2.5 py-0.5 text-xs font-semibold text-white">
                -{discountPercent}%
              </span>
            )}
          </div>

          {/* Status Badge */}
          {product.status !== 'published' && (
            <div className="mb-4">
              <Badge
                variant={product.status === 'draft' ? 'warning' : 'error'}
                size="md"
              >
                {product.status === 'draft' ? 'Yakında' : product.status}
              </Badge>
            </div>
          )}

          {/* Variant Selectors */}
          {hasVariants && (
            <div className="mb-6 space-y-4">
              {Object.entries(attributeOptions).map(([attrKey, values]) => (
                <div key={attrKey}>
                  <label className="mb-2 block text-sm font-medium text-stone-700 capitalize">
                    {attrKey}
                    {selectedAttributes[attrKey] && (
                      <span className="ml-2 font-normal text-stone-500">
                        : {selectedAttributes[attrKey]}
                      </span>
                    )}
                  </label>
                  <div className="flex flex-wrap gap-2">
                    {values.map((value) => {
                      const isSelected = selectedAttributes[attrKey] === value;
                      // For color attributes, render color swatches
                      if (attrKey.toLowerCase() === 'color') {
                        return (
                          <button
                            key={value}
                            type="button"
                            onClick={() => handleAttributeSelect(attrKey, value)}
                            aria-label={`${value} rengini seç`}
                            title={value}
                            className={cn(
                              'h-9 w-9 rounded-full border-2 transition-all',
                              isSelected
                                ? 'border-brand ring-2 ring-brand ring-offset-1'
                                : 'border-stone-300 hover:border-stone-400',
                            )}
                            style={{ backgroundColor: value.toLowerCase() }}
                          />
                        );
                      }
                      // Default: pill/button selector
                      return (
                        <button
                          key={value}
                          type="button"
                          onClick={() => handleAttributeSelect(attrKey, value)}
                          className={cn(
                            'rounded-md border px-4 py-2 text-sm font-medium transition-all',
                            isSelected
                              ? 'border-brand bg-brand-lighter text-brand'
                              : 'border-stone-300 bg-white text-stone-700 hover:border-stone-400 hover:bg-stone-50',
                          )}
                        >
                          {value}
                        </button>
                      );
                    })}
                  </div>
                </div>
              ))}

              {/* Selected variant SKU */}
              {selectedVariant && (
                <p className="text-xs text-stone-500">
                  SKU: {selectedVariant.sku}
                </p>
              )}
            </div>
          )}

          {/* Quantity + Add to Cart */}
          <div className="flex items-center gap-4">
            <QuantitySelector
              value={quantity}
              onChange={setQuantity}
              min={1}
              max={99}
              disabled={isAddingToCart}
            />
            <button
              type="button"
              onClick={handleAddToCart}
              disabled={isAddingToCart || product.status !== 'published'}
              className={cn(
                'flex flex-1 items-center justify-center gap-2 rounded-lg px-6 py-3 text-base font-semibold text-white transition-colors',
                isAddingToCart || product.status !== 'published'
                  ? 'cursor-not-allowed bg-stone-400'
                  : 'bg-brand hover:bg-brand-light active:bg-rose-800',
              )}
            >
              {isAddingToCart ? (
                <>
                  <LoadingSpinner />
                  Ekleniyor...
                </>
              ) : (
                <>
                  <CartIcon />
                  Sepete Ekle
                </>
              )}
            </button>
          </div>

          {/* Short Description Preview */}
          {product.description && (
            <div className="mt-6 border-t border-stone-200 pt-6">
              <p className="text-sm text-stone-600 line-clamp-3">
                {product.description}
              </p>
            </div>
          )}

          {/* Shipping / Returns / Security Info */}
          <div className="mt-6 space-y-3 border-t border-stone-200 pt-6">
            <div className="flex items-center gap-3 text-sm text-stone-600">
              <TruckIcon /> <span>500 TL üzeri ücretsiz kargo</span>
            </div>
            <div className="flex items-center gap-3 text-sm text-stone-600">
              <RefreshIcon /> <span>30 gün iade garantisi</span>
            </div>
            <div className="flex items-center gap-3 text-sm text-stone-600">
              <ShieldIcon /> <span>Güvenli ödeme garantisi</span>
            </div>
          </div>
        </div>
      </div>

      {/* Mobile Sticky Add to Cart Bar */}
      <div className="fixed bottom-0 left-0 right-0 z-40 border-t border-stone-200 bg-white p-4 shadow-lg lg:hidden">
        <div className="flex items-center gap-3">
          <PriceDisplay price={displayPrice} currency={product.currency} size="md" />
          <button
            onClick={handleAddToCart}
            disabled={isAddingToCart || product.status !== 'published'}
            className="flex-1 rounded-lg bg-brand py-3 text-sm font-semibold text-white transition-colors hover:bg-brand-light disabled:bg-stone-300"
          >
            {isAddingToCart ? 'Ekleniyor...' : 'Sepete Ekle'}
          </button>
        </div>
      </div>
    </>
  );
}

// ─── Icon Components ──────────────────────────────────────────────────────────

function CartIcon() {
  return (
    <svg
      width={20}
      height={20}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <circle cx={9} cy={21} r={1} />
      <circle cx={20} cy={21} r={1} />
      <path d="M1 1h4l2.68 13.39a2 2 0 002 1.61h9.72a2 2 0 002-1.61L23 6H6" />
    </svg>
  );
}

function LoadingSpinner() {
  return (
    <svg
      className="h-5 w-5 animate-spin"
      viewBox="0 0 24 24"
      fill="none"
    >
      <circle
        className="opacity-25"
        cx={12}
        cy={12}
        r={10}
        stroke="currentColor"
        strokeWidth={4}
      />
      <path
        className="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      />
    </svg>
  );
}

function TruckIcon() {
  return (
    <svg
      width={20}
      height={20}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.5}
      strokeLinecap="round"
      strokeLinejoin="round"
      className="flex-shrink-0 text-stone-500"
    >
      <rect x={1} y={3} width={15} height={13} />
      <polygon points="16 8 20 8 23 11 23 16 16 16 16 8" />
      <circle cx={5.5} cy={18.5} r={2.5} />
      <circle cx={18.5} cy={18.5} r={2.5} />
    </svg>
  );
}

function RefreshIcon() {
  return (
    <svg
      width={20}
      height={20}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.5}
      strokeLinecap="round"
      strokeLinejoin="round"
      className="flex-shrink-0 text-stone-500"
    >
      <polyline points="23 4 23 10 17 10" />
      <polyline points="1 20 1 14 7 14" />
      <path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15" />
    </svg>
  );
}

function ShieldIcon() {
  return (
    <svg
      width={20}
      height={20}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.5}
      strokeLinecap="round"
      strokeLinejoin="round"
      className="flex-shrink-0 text-stone-500"
    >
      <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
      <polyline points="9 12 11 14 15 10" />
    </svg>
  );
}

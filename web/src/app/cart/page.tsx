'use client';

import { useState, useEffect, useCallback, useMemo } from 'react';
import Image from 'next/image';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useCart } from '@/contexts/CartContext';
import { useAuth } from '@/contexts/AuthContext';
import { api } from '@/lib/api';
import { formatPrice, calculateDiscount, cn } from '@/lib/utils';
import {
  QuantitySelector,
  PriceDisplay,
  Badge,
  useToast,
  EmptyState,
  CartIcon,
} from '@/components/ui';
import type { Product, Campaign } from '@/types';

// ─── Types ────────────────────────────────────────────────────────────────────

interface CartProductMap {
  [productId: string]: Product;
}

// ─── Constants ────────────────────────────────────────────────────────────────

const FREE_SHIPPING_THRESHOLD = 5000; // cents
const FLAT_SHIPPING_RATE = 499; // cents

// ─── Cart Page Component ──────────────────────────────────────────────────────

export default function CartPage() {
  const router = useRouter();
  const { cart, isLoading: cartLoading, updateItem, removeItem } = useCart();
  const { isAuthenticated, isLoading: authLoading } = useAuth();
  const { toast } = useToast();

  // Product details fetched for each cart item
  const [products, setProducts] = useState<CartProductMap>({});
  const [productsLoading, setProductsLoading] = useState(false);

  // Coupon state
  const [couponCode, setCouponCode] = useState('');
  const [couponLoading, setCouponLoading] = useState(false);
  const [appliedCampaign, setAppliedCampaign] = useState<Campaign | null>(null);

  // Item-level loading states (for quantity changes and removals)
  const [updatingItems, setUpdatingItems] = useState<Set<string>>(new Set());

  // ── Fetch product details for all cart items ────────────────────────────

  useEffect(() => {
    if (!cart?.items || cart.items.length === 0) {
      setProducts({});
      return;
    }

    let cancelled = false;

    async function fetchProducts() {
      setProductsLoading(true);
      try {
        const results = await Promise.allSettled(
          cart!.items.map((item) => api.getProduct(item.product_id)),
        );

        if (cancelled) return;

        const productMap: CartProductMap = {};
        results.forEach((result, index) => {
          if (result.status === 'fulfilled') {
            productMap[cart!.items[index].product_id] = result.value.data;
          }
        });
        setProducts(productMap);
      } catch {
        // Silently handle — individual products may fail
      } finally {
        if (!cancelled) {
          setProductsLoading(false);
        }
      }
    }

    fetchProducts();

    return () => {
      cancelled = true;
    };
  }, [cart]);

  // ── Computed values ─────────────────────────────────────────────────────

  const subtotal = useMemo(() => {
    if (!cart?.items) return 0;
    return cart.items.reduce((sum, item) => {
      const product = products[item.product_id];
      if (!product) return sum;
      return sum + product.base_price * item.quantity;
    }, 0);
  }, [cart, products]);

  const discountAmount = useMemo(() => {
    if (!appliedCampaign) return 0;
    return calculateDiscount(subtotal, appliedCampaign.type, appliedCampaign.discount_value);
  }, [subtotal, appliedCampaign]);

  const shippingCost = useMemo(() => {
    const afterDiscount = subtotal - discountAmount;
    if (afterDiscount >= FREE_SHIPPING_THRESHOLD) return 0;
    return FLAT_SHIPPING_RATE;
  }, [subtotal, discountAmount]);

  const total = useMemo(() => {
    return subtotal - discountAmount + shippingCost;
  }, [subtotal, discountAmount, shippingCost]);

  // ── Handlers ────────────────────────────────────────────────────────────

  const handleQuantityChange = useCallback(
    async (productId: string, newQuantity: number) => {
      setUpdatingItems((prev) => new Set(prev).add(productId));
      try {
        await updateItem(productId, newQuantity);
      } catch {
        toast.error('Miktar güncellenemedi. Lütfen tekrar deneyin.');
      } finally {
        setUpdatingItems((prev) => {
          const next = new Set(prev);
          next.delete(productId);
          return next;
        });
      }
    },
    [updateItem, toast],
  );

  const handleRemoveItem = useCallback(
    async (productId: string) => {
      setUpdatingItems((prev) => new Set(prev).add(productId));
      try {
        await removeItem(productId);
        toast.success('Ürün sepetten kaldırıldı.');
      } catch {
        toast.error('Ürün kaldırılamadı. Lütfen tekrar deneyin.');
      } finally {
        setUpdatingItems((prev) => {
          const next = new Set(prev);
          next.delete(productId);
          return next;
        });
      }
    },
    [removeItem, toast],
  );

  const handleApplyCoupon = useCallback(async () => {
    const code = couponCode.trim();
    if (!code) return;

    setCouponLoading(true);
    try {
      const response = await api.validateCoupon(code);
      const campaign = response.data;

      // Check minimum order amount
      if (campaign.min_order_amount > 0 && subtotal < campaign.min_order_amount) {
        toast.error(
          `Bu kupon için minimum ${formatPrice(campaign.min_order_amount)} sipariş tutarı gereklidir.`,
        );
        setCouponLoading(false);
        return;
      }

      setAppliedCampaign(campaign);
      toast.success(`"${campaign.code}" kuponu başarıyla uygulandı!`);
    } catch {
      toast.error('Geçersiz veya süresi dolmuş kupon kodu.');
    } finally {
      setCouponLoading(false);
    }
  }, [couponCode, subtotal, toast]);

  const handleRemoveCoupon = useCallback(() => {
    setAppliedCampaign(null);
    setCouponCode('');
    toast.info('Kupon kaldırıldı.');
  }, [toast]);

  const handleProceedToCheckout = useCallback(() => {
    if (!isAuthenticated) {
      router.push('/auth/login?returnUrl=/cart');
      return;
    }
    router.push('/checkout');
  }, [isAuthenticated, router]);

  // ── Loading state ───────────────────────────────────────────────────────

  if (authLoading || cartLoading) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <h1 className="text-3xl font-bold tracking-tight text-stone-900">
          Sepetim
        </h1>
        <CartSkeleton />
      </div>
    );
  }

  // ── Empty cart ──────────────────────────────────────────────────────────

  const isEmpty = !cart?.items || cart.items.length === 0;

  if (isEmpty) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <h1 className="text-3xl font-bold tracking-tight text-stone-900">
          Sepetim
        </h1>
        <EmptyCart />
      </div>
    );
  }

  // ── Main cart view ──────────────────────────────────────────────────────

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <h1 className="text-3xl font-bold tracking-tight text-stone-900">
        Sepetim
      </h1>

      <div className="mt-8 lg:grid lg:grid-cols-12 lg:gap-x-12">
        {/* Cart items list */}
        <section aria-label="Sepet ürünleri" className="lg:col-span-7">
          <ul aria-label="Sepet ürünleri" className="divide-y divide-stone-200 border-b border-t border-stone-200">
            {cart.items.map((item) => {
              const product = products[item.product_id];
              const isUpdating = updatingItems.has(item.product_id);

              return (
                <CartItemRow
                  key={item.product_id}
                  productId={item.product_id}
                  quantity={item.quantity}
                  product={product}
                  isLoading={productsLoading && !product}
                  isUpdating={isUpdating}
                  onQuantityChange={handleQuantityChange}
                  onRemove={handleRemoveItem}
                />
              );
            })}
          </ul>

          {/* Continue shopping link */}
          <div className="mt-6">
            <Link
              href="/products"
              className="inline-flex items-center gap-1.5 text-sm font-medium text-brand hover:text-brand-light transition-colors"
            >
              <ArrowLeftIcon />
              Alışverişe Devam Et
            </Link>
          </div>
        </section>

        {/* Order summary sidebar */}
        <section
          aria-label="Order summary"
          className="mt-10 lg:col-span-5 lg:mt-0"
        >
          <div className="rounded-lg bg-stone-50 px-6 py-6">
            <h2 className="text-lg font-semibold text-stone-900">
              Sipariş Özeti
            </h2>

            {/* Coupon input */}
            <div className="mt-6">
              <label
                htmlFor="coupon-code"
                className="block text-sm font-medium text-stone-700"
              >
                Kupon / Kampanya Kodu
              </label>
              {appliedCampaign ? (
                <div className="mt-2 flex items-center gap-2">
                  <Badge variant="success" size="md">
                    {appliedCampaign.code}
                  </Badge>
                  <span className="text-sm text-green-700">
                    {appliedCampaign.type === 'percentage'
                      ? `${appliedCampaign.discount_value}% off`
                      : `${formatPrice(appliedCampaign.discount_value)} off`}
                  </span>
                  <button
                    type="button"
                    onClick={handleRemoveCoupon}
                    className="ml-auto text-sm text-stone-500 hover:text-red-600 transition-colors"
                  >
                    Kaldır
                  </button>
                </div>
              ) : (
                <div className="mt-2 flex gap-2">
                  <input
                    id="coupon-code"
                    type="text"
                    value={couponCode}
                    onChange={(e) => setCouponCode(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') handleApplyCoupon();
                    }}
                    placeholder="Kodu girin"
                    className="flex-1 rounded-md border border-stone-300 px-3 py-2.5 text-base sm:text-sm shadow-sm placeholder:text-stone-400 focus:border-brand focus:outline-none focus:ring-1 focus:ring-brand"
                  />
                  <button
                    type="button"
                    onClick={handleApplyCoupon}
                    disabled={couponLoading || !couponCode.trim()}
                    className={cn(
                      'rounded-md px-4 py-2 text-sm font-medium transition-colors',
                      couponLoading || !couponCode.trim()
                        ? 'cursor-not-allowed bg-stone-200 text-stone-400'
                        : 'bg-stone-900 text-white hover:bg-stone-800',
                    )}
                  >
                    {couponLoading ? 'Uygulanıyor...' : 'Uygula'}
                  </button>
                </div>
              )}
            </div>

            {/* Summary lines */}
            <dl className="mt-6 space-y-4" role="status" aria-live="polite">
              <div className="flex items-center justify-between">
                <dt className="text-sm text-stone-600">Ara Toplam</dt>
                <dd className="text-sm font-medium text-stone-900">
                  {formatPrice(subtotal)}
                </dd>
              </div>

              {discountAmount > 0 && (
                <div className="flex items-center justify-between">
                  <dt className="text-sm text-green-600">İndirim</dt>
                  <dd className="text-sm font-medium text-green-600">
                    -{formatPrice(discountAmount)}
                  </dd>
                </div>
              )}

              <div className="flex items-center justify-between border-t border-stone-200 pt-4">
                <dt className="text-sm text-stone-600">Kargo</dt>
                <dd className="text-sm font-medium text-stone-900">
                  {shippingCost === 0 ? (
                    <span className="text-green-600">Ücretsiz</span>
                  ) : (
                    formatPrice(shippingCost)
                  )}
                </dd>
              </div>

              {shippingCost > 0 && (
                <p className="text-xs text-stone-500">
                  {formatPrice(FREE_SHIPPING_THRESHOLD)} üzeri siparişlerde ücretsiz kargo
                </p>
              )}

              <div className="flex items-center justify-between border-t border-stone-200 pt-4">
                <dt className="text-base font-semibold text-stone-900">
                  Toplam
                </dt>
                <dd className="text-base font-semibold text-stone-900">
                  {formatPrice(total)}
                </dd>
              </div>
            </dl>

            {/* Proceed to Checkout */}
            <button
              type="button"
              onClick={handleProceedToCheckout}
              disabled={productsLoading || subtotal === 0}
              className={cn(
                'mt-6 w-full rounded-md px-6 py-3 text-base font-medium text-white shadow-sm transition-colors',
                productsLoading || subtotal === 0
                  ? 'cursor-not-allowed bg-stone-300'
                  : 'bg-brand hover:bg-brand-light focus:outline-none focus:ring-2 focus:ring-brand focus:ring-offset-2',
              )}
            >
              {!isAuthenticated ? 'Ödemeye geçmek için giriş yap' : 'Ödemeye Geç'}
            </button>

            {!isAuthenticated && (
              <p className="mt-2 text-center text-xs text-stone-500">
                Ödemeye geçmeden önce giriş yapmanız gerekecek.
              </p>
            )}
          </div>
        </section>
      </div>
    </div>
  );
}

// ─── Cart Item Row ────────────────────────────────────────────────────────────

interface CartItemRowProps {
  productId: string;
  quantity: number;
  product: Product | undefined;
  isLoading: boolean;
  isUpdating: boolean;
  onQuantityChange: (productId: string, quantity: number) => void;
  onRemove: (productId: string) => void;
}

function CartItemRow({
  productId,
  quantity,
  product,
  isLoading,
  isUpdating,
  onQuantityChange,
  onRemove,
}: CartItemRowProps) {
  if (isLoading || !product) {
    return (
      <li className="flex gap-4 py-6">
        <div className="h-24 w-24 animate-pulse rounded-md bg-stone-200" />
        <div className="flex flex-1 flex-col gap-2">
          <div className="h-4 w-1/3 animate-pulse rounded bg-stone-200" />
          <div className="h-3 w-1/4 animate-pulse rounded bg-stone-200" />
          <div className="h-3 w-1/5 animate-pulse rounded bg-stone-200" />
        </div>
      </li>
    );
  }

  const imageUrl = product.images?.find((img) => img.is_primary)?.url
    || product.images?.[0]?.url
    || `https://picsum.photos/seed/${product.slug}/800/800`;

  const lineTotal = product.base_price * quantity;

  return (
    <li
      className={cn(
        'flex gap-4 py-6 transition-opacity',
        isUpdating && 'opacity-60 pointer-events-none',
      )}
    >
      {/* Product image */}
      <div className="relative h-24 w-24 flex-shrink-0 overflow-hidden rounded-md border border-stone-200">
        <Image
          src={imageUrl}
          alt={product.name}
          fill
          sizes="96px"
          className="object-cover"
        />
      </div>

      {/* Product info */}
      <div className="flex flex-1 flex-col">
        <div className="flex justify-between">
          <div>
            <h3 className="text-sm font-medium text-stone-900">
              <Link
                href={`/products/${product.slug}`}
                className="hover:text-brand transition-colors"
              >
                {product.name}
              </Link>
            </h3>

            {/* Variant info / category / brand */}
            <div className="mt-1 flex flex-wrap items-center gap-2">
              {product.category && (
                <Badge variant="default" size="sm">
                  {product.category.name}
                </Badge>
              )}
              {product.brand && (
                <span className="text-xs text-stone-500">
                  {product.brand.name}
                </span>
              )}
            </div>

            {/* Unit price */}
            <div className="mt-1">
              <PriceDisplay price={product.base_price} size="sm" />
            </div>
          </div>

          {/* Line total (desktop) */}
          <div className="hidden sm:block text-right">
            <span className="text-sm font-semibold text-stone-900">
              {formatPrice(lineTotal)}
            </span>
          </div>
        </div>

        {/* Bottom row: quantity + remove + line total (mobile) */}
        <div className="mt-3 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <QuantitySelector
              value={quantity}
              onChange={(newQty) => onQuantityChange(productId, newQty)}
              min={1}
              max={99}
              disabled={isUpdating}
            />
            <button
              type="button"
              onClick={() => onRemove(productId)}
              disabled={isUpdating}
              aria-label="Ürünü kaldır"
              className="text-sm font-medium text-red-600 hover:text-red-500 transition-colors disabled:cursor-not-allowed disabled:text-red-300"
            >
              Kaldır
            </button>
          </div>

          {/* Line total (mobile) */}
          <span className="text-sm font-semibold text-stone-900 sm:hidden">
            {formatPrice(lineTotal)}
          </span>
        </div>
      </div>
    </li>
  );
}

// ─── Empty Cart ───────────────────────────────────────────────────────────────

function EmptyCart() {
  return (
    <EmptyState
      icon={<CartIcon className="text-brand" />}
      iconBgClass="bg-brand/10"
      heading="Sepetiniz boş"
      message="Harika ürünleri keşfedin ve alışverişe başlayın. Aradığınız şey bir tık uzağınızda!"
      primaryAction={{
        label: 'Ürünleri Keşfet',
        href: '/products',
      }}
      className="mt-16"
    />
  );
}

// ─── Cart Skeleton ────────────────────────────────────────────────────────────

function CartSkeleton() {
  return (
    <div className="mt-8 lg:grid lg:grid-cols-12 lg:gap-x-12">
      <div className="lg:col-span-7">
        <div className="divide-y divide-stone-200 border-b border-t border-stone-200">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="flex gap-4 py-6">
              <div className="h-24 w-24 animate-pulse rounded-md bg-stone-200" />
              <div className="flex flex-1 flex-col gap-2">
                <div className="h-4 w-2/5 animate-pulse rounded bg-stone-200" />
                <div className="h-3 w-1/4 animate-pulse rounded bg-stone-200" />
                <div className="h-3 w-1/6 animate-pulse rounded bg-stone-200" />
                <div className="mt-auto h-9 w-28 animate-pulse rounded bg-stone-200" />
              </div>
            </div>
          ))}
        </div>
      </div>
      <div className="mt-10 lg:col-span-5 lg:mt-0">
        <div className="rounded-lg bg-stone-50 px-6 py-6">
          <div className="h-6 w-1/3 animate-pulse rounded bg-stone-200" />
          <div className="mt-6 space-y-4">
            <div className="h-10 w-full animate-pulse rounded bg-stone-200" />
            <div className="h-4 w-full animate-pulse rounded bg-stone-200" />
            <div className="h-4 w-full animate-pulse rounded bg-stone-200" />
            <div className="h-4 w-full animate-pulse rounded bg-stone-200" />
            <div className="h-12 w-full animate-pulse rounded bg-stone-200" />
          </div>
        </div>
      </div>
    </div>
  );
}

// ─── Icons ────────────────────────────────────────────────────────────────────

function ArrowLeftIcon() {
  return (
    <svg
      width={16}
      height={16}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      className="flex-shrink-0"
    >
      <path d="M19 12H5M12 19l-7-7 7-7" />
    </svg>
  );
}

function ArrowRightIcon() {
  return (
    <svg
      width={16}
      height={16}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      className="flex-shrink-0"
    >
      <path d="M5 12h14M12 5l7 7-7 7" />
    </svg>
  );
}

'use client';

import { Suspense, useState, useEffect, useCallback } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { useAuth } from '@/contexts/AuthContext';
import { useCart } from '@/contexts/CartContext';
import { useToast } from '@/components/ui/Toast';
import { api } from '@/lib/api';
import { formatPrice } from '@/lib/utils';
import { cn } from '@/lib/utils';
import { CHECKOUT_STEPS } from '@/lib/constants';
import type { Address, CheckoutSession } from '@/types';

// ---- Country options for dropdown ------------------------------------------------

const COUNTRIES = [
  { code: 'US', name: 'United States' },
  { code: 'CA', name: 'Canada' },
  { code: 'GB', name: 'United Kingdom' },
  { code: 'DE', name: 'Germany' },
  { code: 'FR', name: 'France' },
  { code: 'AU', name: 'Australia' },
  { code: 'JP', name: 'Japan' },
  { code: 'BR', name: 'Brazil' },
  { code: 'IN', name: 'India' },
  { code: 'TR', name: 'Turkey' },
] as const;

// ---- Form field error map --------------------------------------------------------

interface ShippingErrors {
  line1?: string;
  city?: string;
  state?: string;
  postal_code?: string;
  country?: string;
}

// ---- Step progress indicator -----------------------------------------------------

function StepIndicator({ currentIndex }: { currentIndex: number }) {
  return (
    <nav aria-label="Checkout progress" className="mb-10">
      <ol className="flex items-center justify-center gap-2 sm:gap-4">
        {CHECKOUT_STEPS.map((step, idx) => {
          const isCompleted = idx < currentIndex;
          const isCurrent = idx === currentIndex;

          return (
            <li key={step.id} className="flex items-center gap-2 sm:gap-4">
              {/* Step circle + label */}
              <div className="flex items-center gap-2">
                <span
                  className={cn(
                    'flex h-8 w-8 items-center justify-center rounded-full text-sm font-semibold',
                    isCompleted && 'bg-green-600 text-white',
                    isCurrent && 'bg-gray-900 text-white',
                    !isCompleted && !isCurrent && 'bg-gray-200 text-gray-500',
                  )}
                >
                  {isCompleted ? (
                    <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                    </svg>
                  ) : (
                    idx + 1
                  )}
                </span>
                <span
                  className={cn(
                    'hidden text-sm font-medium sm:inline',
                    isCurrent ? 'text-gray-900' : 'text-gray-500',
                  )}
                >
                  {step.label}
                </span>
              </div>

              {/* Connector line */}
              {idx < CHECKOUT_STEPS.length - 1 && (
                <div
                  className={cn(
                    'h-0.5 w-8 sm:w-12',
                    idx < currentIndex ? 'bg-green-600' : 'bg-gray-200',
                  )}
                />
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}

// ---- Shared input class ----------------------------------------------------------

const inputCls =
  'mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-base sm:text-sm text-gray-900 shadow-sm placeholder:text-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-gray-500';

const labelCls = 'block text-sm font-medium text-gray-700';

// ---- Shipping Step ---------------------------------------------------------------

function ShippingStep({
  address,
  onChange,
  onSubmit,
  errors,
}: {
  address: Address;
  onChange: (field: keyof Address, value: string) => void;
  onSubmit: () => void;
  errors: ShippingErrors;
}) {
  return (
    <div className="mx-auto max-w-lg">
      <h2 className="text-xl font-semibold text-gray-900">Shipping Address</h2>
      <p className="mt-1 text-sm text-gray-500">Where should we deliver your order?</p>

      <div className="mt-6 space-y-4">
        {/* Line 1 */}
        <div>
          <label htmlFor="line1" className={labelCls}>
            Address Line 1 <span className="text-red-500">*</span>
          </label>
          <input
            id="line1"
            type="text"
            value={address.line1}
            onChange={(e) => onChange('line1', e.target.value)}
            placeholder="123 Main Street"
            aria-invalid={!!errors.line1}
            aria-describedby={errors.line1 ? 'error-line1' : undefined}
            className={cn(inputCls, errors.line1 && 'border-red-500 focus:border-red-500 focus:ring-red-500')}
          />
          {errors.line1 && (
            <p id="error-line1" className="mt-1 text-xs text-red-600" role="alert">
              {errors.line1}
            </p>
          )}
        </div>

        {/* Line 2 */}
        <div>
          <label htmlFor="line2" className={labelCls}>
            Address Line 2 <span className="text-gray-400">(optional)</span>
          </label>
          <input
            id="line2"
            type="text"
            value={address.line2 || ''}
            onChange={(e) => onChange('line2', e.target.value)}
            placeholder="Apt, Suite, Unit, etc."
            className={inputCls}
          />
        </div>

        {/* City + State row */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label htmlFor="city" className={labelCls}>
              City <span className="text-red-500">*</span>
            </label>
            <input
              id="city"
              type="text"
              value={address.city}
              onChange={(e) => onChange('city', e.target.value)}
              placeholder="New York"
              aria-invalid={!!errors.city}
              aria-describedby={errors.city ? 'error-city' : undefined}
              className={cn(inputCls, errors.city && 'border-red-500 focus:border-red-500 focus:ring-red-500')}
            />
            {errors.city && (
              <p id="error-city" className="mt-1 text-xs text-red-600" role="alert">
                {errors.city}
              </p>
            )}
          </div>
          <div>
            <label htmlFor="state" className={labelCls}>
              State / Province <span className="text-red-500">*</span>
            </label>
            <input
              id="state"
              type="text"
              value={address.state}
              onChange={(e) => onChange('state', e.target.value)}
              placeholder="NY"
              aria-invalid={!!errors.state}
              aria-describedby={errors.state ? 'error-state' : undefined}
              className={cn(inputCls, errors.state && 'border-red-500 focus:border-red-500 focus:ring-red-500')}
            />
            {errors.state && (
              <p id="error-state" className="mt-1 text-xs text-red-600" role="alert">
                {errors.state}
              </p>
            )}
          </div>
        </div>

        {/* Postal code + Country row */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label htmlFor="postal_code" className={labelCls}>
              Postal Code <span className="text-red-500">*</span>
            </label>
            <input
              id="postal_code"
              type="text"
              value={address.postal_code}
              onChange={(e) => onChange('postal_code', e.target.value)}
              placeholder="10001"
              aria-invalid={!!errors.postal_code}
              aria-describedby={errors.postal_code ? 'error-postal' : undefined}
              className={cn(inputCls, errors.postal_code && 'border-red-500 focus:border-red-500 focus:ring-red-500')}
            />
            {errors.postal_code && (
              <p id="error-postal" className="mt-1 text-xs text-red-600" role="alert">
                {errors.postal_code}
              </p>
            )}
          </div>
          <div>
            <label htmlFor="country" className={labelCls}>
              Country <span className="text-red-500">*</span>
            </label>
            <select
              id="country"
              value={address.country}
              onChange={(e) => onChange('country', e.target.value)}
              aria-invalid={!!errors.country}
              aria-describedby={errors.country ? 'error-country' : undefined}
              className={cn(inputCls, errors.country && 'border-red-500 focus:border-red-500 focus:ring-red-500')}
            >
              <option value="">Select a country</option>
              {COUNTRIES.map((c) => (
                <option key={c.code} value={c.code}>
                  {c.name}
                </option>
              ))}
            </select>
            {errors.country && (
              <p id="error-country" className="mt-1 text-xs text-red-600" role="alert">
                {errors.country}
              </p>
            )}
          </div>
        </div>
      </div>

      <button
        type="button"
        onClick={onSubmit}
        className="mt-8 w-full rounded-md bg-gray-900 px-4 py-3 text-sm font-semibold text-white shadow-sm hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2"
      >
        Continue to Review
      </button>
    </div>
  );
}

// ---- Review Step -----------------------------------------------------------------

function ReviewStep({
  session,
  address,
  onPay,
  onEditShipping,
}: {
  session: CheckoutSession;
  address: Address;
  onPay: () => void;
  onEditShipping: () => void;
}) {
  const countryName =
    COUNTRIES.find((c) => c.code === address.country)?.name || address.country;

  return (
    <div className="mx-auto max-w-2xl">
      <h2 className="text-xl font-semibold text-gray-900">Order Review</h2>
      <p className="mt-1 text-sm text-gray-500">Review your order before payment.</p>

      {/* Items table */}
      <div className="mt-6 rounded-lg border border-gray-200">
        <div className="divide-y divide-gray-200">
          {/* Header */}
          <div className="hidden grid-cols-12 gap-4 px-4 py-3 text-xs font-medium uppercase text-gray-500 sm:grid">
            <span className="col-span-5">Item</span>
            <span className="col-span-2 text-right">Price</span>
            <span className="col-span-2 text-center">Qty</span>
            <span className="col-span-3 text-right">Total</span>
          </div>

          {/* Items */}
          {session.items.map((item) => (
            <div
              key={item.product_id}
              className="grid grid-cols-1 gap-2 px-4 py-3 text-sm sm:grid-cols-12 sm:gap-4 sm:items-center"
            >
              <span className="font-medium text-gray-900 sm:col-span-5">
                {item.product_name}
              </span>
              <span className="text-gray-600 sm:col-span-2 sm:text-right">
                {formatPrice(item.unit_price)}
              </span>
              <span className="text-gray-600 sm:col-span-2 sm:text-center">
                x{item.quantity}
              </span>
              <span className="font-medium text-gray-900 sm:col-span-3 sm:text-right">
                {formatPrice(item.total_price)}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* Shipping address */}
      <div className="mt-6 rounded-lg border border-gray-200 p-4">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold text-gray-900">Shipping Address</h3>
          <button
            type="button"
            onClick={onEditShipping}
            className="text-sm font-medium text-gray-600 hover:text-gray-900 hover:underline"
          >
            Edit
          </button>
        </div>
        <p className="mt-2 text-sm text-gray-600">
          {address.line1}
          {address.line2 && <>, {address.line2}</>}
          <br />
          {address.city}, {address.state} {address.postal_code}
          <br />
          {countryName}
        </p>
      </div>

      {/* Campaign code */}
      {session.campaign_code && (
        <div className="mt-4 rounded-lg border border-green-200 bg-green-50 p-3">
          <p className="text-sm text-green-800">
            Coupon applied: <span className="font-semibold">{session.campaign_code}</span>
          </p>
        </div>
      )}

      {/* Summary totals */}
      <div className="mt-6 space-y-2 border-t border-gray-200 pt-4">
        <div className="flex justify-between text-sm text-gray-600">
          <span>Subtotal</span>
          <span>{formatPrice(session.subtotal)}</span>
        </div>
        {session.discount > 0 && (
          <div className="flex justify-between text-sm text-green-600">
            <span>Discount</span>
            <span>-{formatPrice(session.discount)}</span>
          </div>
        )}
        <div className="flex justify-between text-sm text-gray-600">
          <span>Shipping</span>
          <span>{session.shipping_cost === 0 ? 'Free' : formatPrice(session.shipping_cost)}</span>
        </div>
        <div className="flex justify-between border-t border-gray-200 pt-2 text-base font-semibold text-gray-900">
          <span>Total</span>
          <span>{formatPrice(session.total)}</span>
        </div>
      </div>

      <button
        type="button"
        onClick={onPay}
        className="mt-8 w-full rounded-md bg-gray-900 px-4 py-3 text-sm font-semibold text-white shadow-sm hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2"
      >
        Continue to Payment
      </button>
    </div>
  );
}

// ---- Payment Step ----------------------------------------------------------------

function PaymentStep({
  total,
  isProcessing,
  onPay,
}: {
  total: number;
  isProcessing: boolean;
  onPay: () => void;
}) {
  return (
    <div className="mx-auto max-w-lg">
      {/* Test mode badge */}
      <div className="mb-6 rounded-lg border-2 border-dashed border-yellow-400 bg-yellow-50 p-4 text-center">
        <span className="inline-flex items-center gap-2 text-sm font-bold uppercase tracking-wide text-yellow-700">
          <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          Test Mode -- No real charges
        </span>
      </div>

      <h2 className="text-xl font-semibold text-gray-900">Payment Details</h2>
      <p className="mt-1 text-sm text-gray-500">This is a mock payment for testing.</p>

      <div className="mt-6 space-y-4">
        {/* Card number */}
        <div>
          <label htmlFor="card_number" className={labelCls}>Card Number</label>
          <input
            id="card_number"
            type="text"
            readOnly
            value="4242 4242 4242 4242"
            className={cn(inputCls, 'bg-gray-50 text-gray-500')}
          />
        </div>

        {/* Expiry + CVV */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label htmlFor="expiry" className={labelCls}>Expiry</label>
            <input
              id="expiry"
              type="text"
              readOnly
              value="12/28"
              className={cn(inputCls, 'bg-gray-50 text-gray-500')}
            />
          </div>
          <div>
            <label htmlFor="cvv" className={labelCls}>CVV</label>
            <input
              id="cvv"
              type="text"
              readOnly
              value="123"
              className={cn(inputCls, 'bg-gray-50 text-gray-500')}
            />
          </div>
        </div>
      </div>

      <button
        type="button"
        onClick={onPay}
        disabled={isProcessing}
        className={cn(
          'mt-8 flex w-full items-center justify-center gap-2 rounded-md px-4 py-3 text-sm font-semibold text-white shadow-sm focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2',
          isProcessing
            ? 'cursor-not-allowed bg-gray-400'
            : 'bg-gray-900 hover:bg-gray-700',
        )}
      >
        {isProcessing ? (
          <>
            {/* Spinner */}
            <svg className="h-5 w-5 animate-spin" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx={12} cy={12} r={10} stroke="currentColor" strokeWidth={4} />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
            </svg>
            Processing...
          </>
        ) : (
          `Pay ${formatPrice(total)}`
        )}
      </button>
    </div>
  );
}

// ---- Confirmation Step -----------------------------------------------------------

function ConfirmationStep({
  session,
}: {
  session: CheckoutSession;
}) {
  // Derive orderId from session_id (the backend may use it as the order reference)
  const orderId = session.session_id;

  return (
    <div className="mx-auto max-w-lg text-center">
      {/* Success icon */}
      <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-green-100">
        <svg className="h-8 w-8 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
        </svg>
      </div>

      <h2 className="mt-4 text-2xl font-bold text-gray-900">Order placed successfully!</h2>
      <p className="mt-2 text-sm text-gray-600">
        Thank you for your purchase. Your order has been confirmed.
      </p>

      {/* Order ID */}
      <div className="mt-6 rounded-lg border border-gray-200 bg-gray-50 p-4">
        <p className="text-xs font-medium uppercase text-gray-500">Order ID</p>
        <p className="mt-1 break-all font-mono text-sm text-gray-900">{orderId}</p>
      </div>

      {/* Order summary */}
      <div className="mt-6 rounded-lg border border-gray-200 p-4 text-left">
        <h3 className="text-sm font-semibold text-gray-900">Order Summary</h3>
        <div className="mt-3 divide-y divide-gray-100">
          {session.items.map((item) => (
            <div key={item.product_id} className="flex justify-between py-2 text-sm">
              <span className="text-gray-600">
                {item.product_name} x{item.quantity}
              </span>
              <span className="font-medium text-gray-900">{formatPrice(item.total_price)}</span>
            </div>
          ))}
        </div>
        <div className="mt-3 flex justify-between border-t border-gray-200 pt-3 text-sm font-semibold text-gray-900">
          <span>Total</span>
          <span>{formatPrice(session.total)}</span>
        </div>
      </div>

      {/* Action links */}
      <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:justify-center">
        <Link
          href={`/orders/${orderId}`}
          className="inline-flex items-center justify-center rounded-md bg-gray-900 px-6 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2"
        >
          View Order
        </Link>
        <Link
          href="/products"
          className="inline-flex items-center justify-center rounded-md border border-gray-300 bg-white px-6 py-2.5 text-sm font-semibold text-gray-700 shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2"
        >
          Continue Shopping
        </Link>
      </div>
    </div>
  );
}

// ---- Main Checkout Page ----------------------------------------------------------

function CheckoutContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { user, isAuthenticated, isLoading: authLoading } = useAuth();
  const { cart, itemCount, refreshCart } = useCart();
  const { toast } = useToast();

  // Current step index (0-3)
  const [stepIndex, setStepIndex] = useState(0);

  // Shipping address form state
  const [address, setAddress] = useState<Address>({
    line1: '',
    line2: '',
    city: '',
    state: '',
    postal_code: '',
    country: '',
  });
  const [shippingErrors, setShippingErrors] = useState<ShippingErrors>({});

  // Checkout session from the backend
  const [session, setSession] = useState<CheckoutSession | null>(null);

  // Processing state for payment
  const [isProcessing, setIsProcessing] = useState(false);

  // Loading state for checkout initiation
  const [isInitiating, setIsInitiating] = useState(false);

  // ---- Auth gate: redirect to login if not authenticated ----
  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      router.replace('/auth/login?returnUrl=/checkout');
    }
  }, [authLoading, isAuthenticated, router]);

  // ---- Empty cart gate: redirect to cart page ----
  useEffect(() => {
    if (!authLoading && isAuthenticated && !cart?.items?.length && stepIndex === 0 && !session) {
      router.replace('/cart');
    }
  }, [authLoading, isAuthenticated, cart, stepIndex, session, router]);

  // ---- Get campaign code from URL params or localStorage ----
  const getCampaignCode = useCallback((): string | undefined => {
    const fromUrl = searchParams.get('coupon') || searchParams.get('campaign');
    if (fromUrl) return fromUrl;

    if (typeof window !== 'undefined') {
      return localStorage.getItem('campaign_code') || undefined;
    }
    return undefined;
  }, [searchParams]);

  // ---- Shipping form validation ----
  const validateShipping = useCallback((): boolean => {
    const errs: ShippingErrors = {};
    if (!address.line1.trim()) errs.line1 = 'Address line 1 is required.';
    if (!address.city.trim()) errs.city = 'City is required.';
    if (!address.state.trim()) errs.state = 'State is required.';
    if (!address.postal_code.trim()) errs.postal_code = 'Postal code is required.';
    if (!address.country) errs.country = 'Please select a country.';
    setShippingErrors(errs);
    return Object.keys(errs).length === 0;
  }, [address]);

  // ---- Handle address field change ----
  const handleAddressChange = useCallback(
    (field: keyof Address, value: string) => {
      setAddress((prev) => ({ ...prev, [field]: value }));
      // Clear the error for this field when user types
      setShippingErrors((prev) => {
        if (prev[field as keyof ShippingErrors]) {
          const next = { ...prev };
          delete next[field as keyof ShippingErrors];
          return next;
        }
        return prev;
      });
    },
    [],
  );

  // ---- Step 1 -> Step 2: initiate checkout & set shipping ----
  const handleShippingSubmit = useCallback(async () => {
    if (!validateShipping()) return;

    setIsInitiating(true);
    try {
      // 1. Initiate checkout (creates session, locks cart items)
      const campaignCode = getCampaignCode();
      const initResponse = await api.initiateCheckout(
        campaignCode ? { campaign_code: campaignCode } : undefined,
      );
      const checkoutSession = initResponse.data;

      // 2. Set shipping address on the session
      const shippingResponse = await api.setShipping(checkoutSession.session_id, {
        shipping_address: address,
      });
      setSession(shippingResponse.data);

      // Move to review step
      setStepIndex(1);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : 'Failed to initiate checkout. Please try again.';
      toast.error(message);
    } finally {
      setIsInitiating(false);
    }
  }, [validateShipping, getCampaignCode, address, toast]);

  // ---- Step 2 -> Step 3: go to payment ----
  const handleContinueToPayment = useCallback(() => {
    setStepIndex(2);
  }, []);

  // ---- Step 3: process payment ----
  const handleProcessPayment = useCallback(async () => {
    if (!session) return;

    setIsProcessing(true);
    try {
      const paymentResponse = await api.processPayment(session.session_id);
      setSession(paymentResponse.data);

      // Clear the cart after successful payment
      await refreshCart();

      // Clear stored campaign code
      if (typeof window !== 'undefined') {
        localStorage.removeItem('campaign_code');
      }

      // Move to confirmation step
      setStepIndex(3);
      toast.success('Payment processed successfully!');
    } catch (err) {
      const message =
        err instanceof Error ? err.message : 'Payment failed. Please try again.';
      toast.error(message);
    } finally {
      setIsProcessing(false);
    }
  }, [session, refreshCart, toast]);

  // ---- Navigate back to shipping ----
  const handleEditShipping = useCallback(() => {
    setSession(null);
    setStepIndex(0);
  }, []);

  // ---- Guard renders ----

  // Show nothing while checking auth status
  if (authLoading) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
        <div className="flex items-center justify-center py-24">
          <svg className="h-8 w-8 animate-spin text-gray-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx={12} cy={12} r={10} stroke="currentColor" strokeWidth={4} />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
          </svg>
        </div>
      </div>
    );
  }

  // Not authenticated -- redirect in progress
  if (!isAuthenticated) {
    return null;
  }

  // ---- Render ----
  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
      <h1 className="mb-2 text-center text-3xl font-bold tracking-tight text-gray-900">
        Checkout
      </h1>

      {/* Step indicator */}
      <StepIndicator currentIndex={stepIndex} />

      {/* Step content */}
      {stepIndex === 0 && (
        <>
          {isInitiating ? (
            <div className="flex flex-col items-center justify-center py-16">
              <svg className="h-8 w-8 animate-spin text-gray-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx={12} cy={12} r={10} stroke="currentColor" strokeWidth={4} />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
              </svg>
              <p className="mt-4 text-sm text-gray-500">Setting up your checkout...</p>
            </div>
          ) : (
            <ShippingStep
              address={address}
              onChange={handleAddressChange}
              onSubmit={handleShippingSubmit}
              errors={shippingErrors}
            />
          )}
        </>
      )}

      {stepIndex === 1 && session && (
        <ReviewStep
          session={session}
          address={address}
          onPay={handleContinueToPayment}
          onEditShipping={handleEditShipping}
        />
      )}

      {stepIndex === 2 && session && (
        <PaymentStep
          total={session.total}
          isProcessing={isProcessing}
          onPay={handleProcessPayment}
        />
      )}

      {stepIndex === 3 && session && (
        <ConfirmationStep session={session} />
      )}
    </div>
  );
}

export default function CheckoutPage() {
  return (
    <Suspense
      fallback={
        <div className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
          <div className="flex items-center justify-center py-24">
            <div className="h-8 w-8 animate-spin rounded-full border-4 border-gray-300 border-t-gray-900" />
          </div>
        </div>
      }
    >
      <CheckoutContent />
    </Suspense>
  );
}

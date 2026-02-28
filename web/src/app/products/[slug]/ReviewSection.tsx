'use client';

import { useState, useCallback } from 'react';
import Link from 'next/link';
import type { Review, ReviewSummary, CreateReviewRequest } from '@/types';
import { RatingStars, Pagination, ChatBubbleIcon } from '@/components/ui';
import { useAuth } from '@/contexts/AuthContext';
import { useToast } from '@/components/ui/Toast';
import { api } from '@/lib/api';
import { cn, formatDate } from '@/lib/utils';

// ─── Props ────────────────────────────────────────────────────────────────────

interface ReviewSectionProps {
  productId: string;
  initialReviews: Review[];
  reviewSummary: ReviewSummary;
  totalPages: number;
}

// ─── Component ────────────────────────────────────────────────────────────────

export function ReviewSection({
  productId,
  initialReviews,
  reviewSummary,
  totalPages: initialTotalPages,
}: ReviewSectionProps) {
  const { isAuthenticated } = useAuth();
  const { toast } = useToast();

  const [reviews, setReviews] = useState<Review[]>(initialReviews);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(initialTotalPages);
  const [isLoadingPage, setIsLoadingPage] = useState(false);

  // Pagination handler
  const handlePageChange = useCallback(
    async (page: number) => {
      setIsLoadingPage(true);
      try {
        const response = await api.getProductReviews(productId, page);
        setReviews(response.data);
        setCurrentPage(page);
        setTotalPages(response.total_pages);
      } catch {
        toast.error('Değerlendirmeler yüklenemedi.');
      } finally {
        setIsLoadingPage(false);
      }
    },
    [productId, toast],
  );

  // Handle new review submitted
  const handleReviewSubmitted = useCallback((newReview: Review) => {
    setReviews((prev) => [newReview, ...prev]);
  }, []);

  return (
    <div className="space-y-8">
      {/* Review Summary */}
      <div className="flex flex-col items-start gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-4">
          <div className="text-center">
            <div className="text-4xl font-bold text-stone-900">
              {reviewSummary.average_rating.toFixed(1)}
            </div>
            <RatingStars rating={reviewSummary.average_rating} size="md" />
            <div className="mt-1 text-sm text-stone-500">
              {reviewSummary.total_count} değerlendirme
            </div>
          </div>
        </div>
      </div>

      {/* Write Review Form or Auth Gate */}
      {isAuthenticated ? (
        <WriteReviewForm
          productId={productId}
          onReviewSubmitted={handleReviewSubmitted}
        />
      ) : (
        <div className="rounded-lg border border-stone-200 bg-stone-50 p-6 text-center">
          <p className="text-stone-600">
            Deneyiminizi paylaşmak ister misiniz?{' '}
            <Link
              href="/auth/login"
              className="font-medium text-brand hover:text-brand-light transition-colors"
            >
              Değerlendirme yazmak için giriş yapın
            </Link>
          </p>
        </div>
      )}

      {/* Review List */}
      <div className={cn('space-y-6', isLoadingPage && 'opacity-50 pointer-events-none')}>
        {reviews.length === 0 ? (
          <div className="py-12 text-center">
            <div className="mb-3 inline-flex h-12 w-12 items-center justify-center rounded-full bg-stone-100">
              <ChatBubbleIcon className="text-stone-400" />
            </div>
            <h4 className="mb-1 text-base font-medium text-stone-700">
              Henüz değerlendirme yok
            </h4>
            <p className="text-sm text-stone-500">
              Bu ürünü ilk değerlendiren siz olun!
            </p>
          </div>
        ) : (
          reviews.map((review) => (
            <ReviewCard key={review.id} review={review} />
          ))
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="pt-4">
          <Pagination
            currentPage={currentPage}
            totalPages={totalPages}
            onPageChange={handlePageChange}
          />
        </div>
      )}
    </div>
  );
}

// ─── Review Card ──────────────────────────────────────────────────────────────

function ReviewCard({ review }: { review: Review }) {
  // Generate user initial from user_id (placeholder since we don't have user name)
  const initial = review.user_id.charAt(0).toUpperCase();

  return (
    <div className="border-b border-stone-100 pb-6 last:border-b-0">
      <div className="flex items-start gap-4">
        {/* Avatar */}
        <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full bg-brand-lighter text-sm font-semibold text-brand">
          {initial}
        </div>

        {/* Review Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <RatingStars rating={review.rating} size="sm" />
            <span className="text-xs text-stone-400">
              {formatDate(review.created_at)}
            </span>
          </div>

          {review.title && (
            <h4 className="text-sm font-semibold text-stone-900 mb-1">
              {review.title}
            </h4>
          )}

          {review.body && (
            <p className="text-sm text-stone-700 leading-relaxed">
              {review.body}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

// ─── Write Review Form ────────────────────────────────────────────────────────

interface WriteReviewFormProps {
  productId: string;
  onReviewSubmitted: (review: Review) => void;
}

function WriteReviewForm({ productId, onReviewSubmitted }: WriteReviewFormProps) {
  const { toast } = useToast();

  const [isOpen, setIsOpen] = useState(false);
  const [rating, setRating] = useState(0);
  const [hoverRating, setHoverRating] = useState(0);
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};
    if (rating === 0) newErrors.rating = 'Lütfen bir puan seçin';
    if (!title.trim()) newErrors.title = 'Lütfen bir başlık girin';
    if (!body.trim()) newErrors.body = 'Lütfen bir değerlendirme yazın';
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!validate()) return;

      setIsSubmitting(true);
      try {
        const data: CreateReviewRequest = {
          rating,
          title: title.trim(),
          body: body.trim(),
        };
        const response = await api.createReview(productId, data);
        onReviewSubmitted(response.data);
        toast.success('Değerlendirmeniz başarıyla gönderildi!');

        // Reset form
        setRating(0);
        setTitle('');
        setBody('');
        setErrors({});
        setIsOpen(false);
      } catch {
        toast.error('Değerlendirme gönderilemedi. Lütfen tekrar deneyin.');
      } finally {
        setIsSubmitting(false);
      }
    },
    [rating, title, body, productId, onReviewSubmitted, toast],
  );

  if (!isOpen) {
    return (
      <button
        type="button"
        onClick={() => setIsOpen(true)}
        className="rounded-lg border-2 border-dashed border-stone-300 px-6 py-4 text-sm font-medium text-stone-600 transition-colors hover:border-brand hover:text-brand"
      >
        Değerlendirme Yaz
      </button>
    );
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="rounded-lg border border-stone-200 bg-white p-6 shadow-sm"
    >
      <h3 className="mb-4 text-lg font-semibold text-stone-900">
        Değerlendirme Yaz
      </h3>

      {/* Star Rating Selector */}
      <div className="mb-4">
        <label className="mb-2 block text-sm font-medium text-stone-700">
          Puanınız
        </label>
        <div className="flex gap-1">
          {[1, 2, 3, 4, 5].map((star) => (
            <button
              key={star}
              type="button"
              onClick={() => setRating(star)}
              onMouseEnter={() => setHoverRating(star)}
              onMouseLeave={() => setHoverRating(0)}
              aria-label={`${star} yıldız ver`}
              className="transition-transform hover:scale-110"
            >
              <svg
                width={28}
                height={28}
                viewBox="0 0 24 24"
                fill={(hoverRating || rating) >= star ? 'currentColor' : '#D1D5DB'}
                className={cn(
                  'transition-colors',
                  (hoverRating || rating) >= star
                    ? 'text-yellow-400'
                    : 'text-stone-300',
                )}
              >
                <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
              </svg>
            </button>
          ))}
        </div>
        {errors.rating && (
          <p className="mt-1 text-xs text-red-600">{errors.rating}</p>
        )}
      </div>

      {/* Title Input */}
      <div className="mb-4">
        <label
          htmlFor="review-title"
          className="mb-1 block text-sm font-medium text-stone-700"
        >
          Başlık
        </label>
        <input
          id="review-title"
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Deneyiminizi özetleyin"
          maxLength={100}
          className={cn(
            'w-full rounded-md border px-3 py-2 text-sm text-stone-900 placeholder-stone-400 transition-colors focus:border-brand focus:outline-none focus:ring-1 focus:ring-brand',
            errors.title ? 'border-red-300' : 'border-stone-300',
          )}
        />
        {errors.title && (
          <p className="mt-1 text-xs text-red-600">{errors.title}</p>
        )}
      </div>

      {/* Body Textarea */}
      <div className="mb-4">
        <label
          htmlFor="review-body"
          className="mb-1 block text-sm font-medium text-stone-700"
        >
          Değerlendirme
        </label>
        <textarea
          id="review-body"
          value={body}
          onChange={(e) => setBody(e.target.value)}
          placeholder="Bu ürün hakkındaki görüşlerinizi paylaşın"
          rows={4}
          maxLength={2000}
          className={cn(
            'w-full rounded-md border px-3 py-2 text-sm text-stone-900 placeholder-stone-400 transition-colors focus:border-brand focus:outline-none focus:ring-1 focus:ring-brand resize-y',
            errors.body ? 'border-red-300' : 'border-stone-300',
          )}
        />
        {errors.body && (
          <p className="mt-1 text-xs text-red-600">{errors.body}</p>
        )}
      </div>

      {/* Actions */}
      <div className="flex items-center gap-3">
        <button
          type="submit"
          disabled={isSubmitting}
          className={cn(
            'rounded-md px-5 py-2 text-sm font-medium text-white transition-colors',
            isSubmitting
              ? 'cursor-not-allowed bg-stone-400'
              : 'bg-brand hover:bg-brand-light',
          )}
        >
          {isSubmitting ? 'Gönderiliyor...' : 'Gönder'}
        </button>
        <button
          type="button"
          onClick={() => {
            setIsOpen(false);
            setErrors({});
          }}
          disabled={isSubmitting}
          className="rounded-md px-4 py-2 text-sm font-medium text-stone-600 hover:text-stone-800 transition-colors"
        >
          Vazgeç
        </button>
      </div>
    </form>
  );
}

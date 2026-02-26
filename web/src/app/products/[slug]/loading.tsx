import { ProductDetailSkeleton } from '@/components/ui';

export default function Loading() {
  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <ProductDetailSkeleton />
    </div>
  );
}

import type { Metadata } from 'next';
import { WishlistClient } from './WishlistClient';

export const metadata: Metadata = {
  title: 'Favorilerim | EcommerceGo',
  description: 'Favori ürünlerinizi görüntüleyin ve yönetin.',
};

export default function WishlistPage() {
  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <h1 className="mb-8 text-2xl font-bold text-stone-900">Favorilerim</h1>
      <WishlistClient />
    </div>
  );
}

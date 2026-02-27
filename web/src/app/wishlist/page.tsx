import type { Metadata } from 'next';
import { WishlistClient } from './WishlistClient';

export const metadata: Metadata = {
  title: 'My Wishlist | EcommerceGo',
  description: 'View and manage your wishlist items',
};

export default function WishlistPage() {
  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <h1 className="mb-8 text-2xl font-bold text-stone-900">My Wishlist</h1>
      <WishlistClient />
    </div>
  );
}

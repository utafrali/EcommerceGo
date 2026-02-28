import { Metadata } from 'next';
import Link from 'next/link';

export const metadata: Metadata = {
  title: 'Hesabım | EcommerceGo',
  description: 'Hesap ayarlarınızı ve tercihlerinizi yönetin',
};

export default function AccountPage() {
  return (
    <div className="container mx-auto px-4 py-12">
      <div className="max-w-4xl mx-auto">
        <h1 className="text-3xl font-bold text-stone-900 mb-8">Hesabım</h1>

        <div className="grid gap-6 md:grid-cols-2">
          {/* Profile */}
          <Link
            href="/account/profile"
            className="block p-6 bg-white rounded-lg border border-stone-200 hover:border-brand hover:shadow-md transition-all"
          >
            <div className="flex items-center gap-4">
              <div className="flex-shrink-0 w-12 h-12 bg-brand-lighter rounded-full flex items-center justify-center">
                <svg className="w-6 h-6 text-brand" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                </svg>
              </div>
              <div>
                <h2 className="text-lg font-semibold text-stone-900">Profil</h2>
                <p className="text-sm text-stone-600">Kişisel bilgilerinizi görüntüleyin ve düzenleyin</p>
              </div>
            </div>
          </Link>

          {/* Orders */}
          <Link
            href="/orders"
            className="block p-6 bg-white rounded-lg border border-stone-200 hover:border-brand hover:shadow-md transition-all"
          >
            <div className="flex items-center gap-4">
              <div className="flex-shrink-0 w-12 h-12 bg-brand-lighter rounded-full flex items-center justify-center">
                <svg className="w-6 h-6 text-brand" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z" />
                </svg>
              </div>
              <div>
                <h2 className="text-lg font-semibold text-stone-900">Siparişlerim</h2>
                <p className="text-sm text-stone-600">Siparişlerinizi takip edin ve yönetin</p>
              </div>
            </div>
          </Link>

          {/* Wishlist */}
          <Link
            href="/wishlist"
            className="block p-6 bg-white rounded-lg border border-stone-200 hover:border-brand hover:shadow-md transition-all"
          >
            <div className="flex items-center gap-4">
              <div className="flex-shrink-0 w-12 h-12 bg-brand-lighter rounded-full flex items-center justify-center">
                <svg className="w-6 h-6 text-brand" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" />
                </svg>
              </div>
              <div>
                <h2 className="text-lg font-semibold text-stone-900">Favorilerim</h2>
                <p className="text-sm text-stone-600">Kaydettiğiniz ürünleri görüntüleyin</p>
              </div>
            </div>
          </Link>

          {/* Settings */}
          <Link
            href="/account/settings"
            className="block p-6 bg-white rounded-lg border border-stone-200 hover:border-brand hover:shadow-md transition-all"
          >
            <div className="flex items-center gap-4">
              <div className="flex-shrink-0 w-12 h-12 bg-brand-lighter rounded-full flex items-center justify-center">
                <svg className="w-6 h-6 text-brand" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
              </div>
              <div>
                <h2 className="text-lg font-semibold text-stone-900">Ayarlar</h2>
                <p className="text-sm text-stone-600">Şifre ve tercihleri yönetin</p>
              </div>
            </div>
          </Link>
        </div>
      </div>
    </div>
  );
}

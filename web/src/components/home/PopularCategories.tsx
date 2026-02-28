// ─── Popular Categories (Modanisa style) ──────────────────────────────────────
// Horizontal pill chip navigation for popular category searches.

import Link from 'next/link';

const POPULAR_CATEGORIES = [
  { label: 'Tesettür Giyim', href: '/products?search=tesettur+giyim' },
  { label: 'Tesettür Elbise', href: '/products?search=tesettur+elbise' },
  { label: 'Tesettür Abiye', href: '/products?search=tesettur+abiye' },
  { label: 'Sırt Çantası', href: '/products?search=sirt+cantasi' },
  { label: 'Topuklu Ayakkabı', href: '/products?search=topuklu+ayakkabi' },
  { label: 'Trençkot', href: '/products?search=trenkot' },
  { label: 'Nikah Elbisesi', href: '/products?search=nikah+elbisesi' },
  { label: 'Ferace', href: '/products?search=ferace' },
  { label: 'Başörtüsü', href: '/products?search=basortust' },
  { label: 'Tunik', href: '/products?search=tunik' },
  { label: 'Pardesü', href: '/products?search=pardest' },
];

export function PopularCategories() {
  return (
    <section className="border-t border-gray-100 bg-white py-8">
      <div className="mx-auto max-w-screen-xl px-4 sm:px-6">
        <h2 className="mb-4 text-base font-bold text-gray-900">Popüler Kategori</h2>
        <div className="flex flex-wrap gap-2">
          {POPULAR_CATEGORIES.map((cat) => (
            <Link
              key={cat.label}
              href={cat.href}
              className="rounded border border-gray-300 bg-white px-4 py-2 text-sm text-gray-700 transition-colors hover:border-brand hover:text-brand"
            >
              {cat.label}
            </Link>
          ))}
        </div>
      </div>
    </section>
  );
}

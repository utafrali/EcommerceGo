'use client';

import Link from 'next/link';

export default function Footer() {
  const currentYear = new Date().getFullYear();

  return (
    <footer className="bg-white border-t border-gray-100">

      {/* ── Main Footer Grid ──────────────────────────────────────────────── */}
      <div className="mx-auto max-w-screen-xl px-4 py-12 sm:px-6">
        <div className="grid grid-cols-1 gap-10 sm:grid-cols-2 lg:grid-cols-4">

          {/* EcommerceGo */}
          <div>
            <Link href="/" className="text-xl font-black tracking-tight text-gray-900">
              Ecommerce<span className="text-brand">Go</span>
            </Link>
            <ul className="mt-5 space-y-3">
              <li><a href="#" className="text-sm text-gray-500 hover:text-gray-900 transition-colors">Kampanyalar</a></li>
              <li><a href="#" className="text-sm text-gray-500 hover:text-gray-900 transition-colors">Hakkımızda</a></li>
              <li><a href="#" className="text-sm text-gray-500 hover:text-gray-900 transition-colors">İletişim</a></li>
              <li><a href="#" className="text-sm text-gray-500 hover:text-gray-900 transition-colors">Kariyer</a></li>
              <li><a href="#" className="text-sm text-gray-500 hover:text-gray-900 transition-colors">Kurumsal</a></li>
            </ul>
          </div>

          {/* Sıkça Sorulan Sorular */}
          <div>
            <h3 className="text-sm font-bold text-gray-900">Sıkça Sorulan Sorular</h3>
            <ul className="mt-5 space-y-3">
              <li><a href="#" className="text-sm text-gray-500 hover:text-gray-900 transition-colors">Sipariş</a></li>
              <li><a href="#" className="text-sm text-gray-500 hover:text-gray-900 transition-colors">Teslimat ve Kargo</a></li>
              <li><a href="#" className="text-sm text-gray-500 hover:text-gray-900 transition-colors">İade, İptal ve Değişim</a></li>
              <li><a href="#" className="text-sm text-gray-500 hover:text-gray-900 transition-colors">Ödeme Seçenekleri</a></li>
              <li><Link href="/orders" className="text-sm text-gray-500 hover:text-gray-900 transition-colors">İşlem Rehberi</Link></li>
            </ul>
          </div>

          {/* Kategoriler */}
          <div>
            <h3 className="text-sm font-bold text-gray-900">Kategoriler</h3>
            <ul className="mt-5 space-y-3">
              <li><Link href="/products?category=elbise" className="text-sm text-gray-500 hover:text-gray-900 transition-colors">Elbise</Link></li>
              <li><Link href="/products?category=giyim" className="text-sm text-gray-500 hover:text-gray-900 transition-colors">Giyim</Link></li>
              <li><Link href="/products?category=aksesuar" className="text-sm text-gray-500 hover:text-gray-900 transition-colors">Aksesuar</Link></li>
              <li><Link href="/products?category=ayakkabi" className="text-sm text-gray-500 hover:text-gray-900 transition-colors">Ayakkabı &amp; Çanta</Link></li>
              <li><Link href="/products?on_sale=true" className="text-sm text-brand-accent hover:text-orange-600 transition-colors font-medium">Fırsat Ürünleri</Link></li>
            </ul>
          </div>

          {/* Bizi Takip Edin */}
          <div>
            <h3 className="text-sm font-bold text-gray-900">Bizi Takip Edin</h3>

            {/* Social icons */}
            <div className="mt-4 flex flex-wrap gap-2">
              {/* Facebook */}
              <a href="#" target="_blank" rel="noopener noreferrer" aria-label="Facebook"
                className="flex h-9 w-9 items-center justify-center rounded-full bg-[#1877F2] text-white hover:opacity-90 transition-opacity">
                <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                  <path d="M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z"/>
                </svg>
              </a>
              {/* Twitter/X */}
              <a href="#" target="_blank" rel="noopener noreferrer" aria-label="X (Twitter)"
                className="flex h-9 w-9 items-center justify-center rounded-full bg-black text-white hover:opacity-90 transition-opacity">
                <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                  <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z"/>
                </svg>
              </a>
              {/* Instagram */}
              <a href="#" target="_blank" rel="noopener noreferrer" aria-label="Instagram"
                className="flex h-9 w-9 items-center justify-center rounded-full bg-gradient-to-tr from-[#f09433] via-[#e6683c] via-[#dc2743] via-[#cc2366] to-[#bc1888] text-white hover:opacity-90 transition-opacity">
                <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                  <path fillRule="evenodd" d="M12.315 2c2.43 0 2.784.013 3.808.06 1.064.049 1.791.218 2.427.465a4.902 4.902 0 011.772 1.153 4.902 4.902 0 011.153 1.772c.247.636.416 1.363.465 2.427.048 1.067.06 1.407.06 4.123v.08c0 2.643-.012 2.987-.06 4.043-.049 1.064-.218 1.791-.465 2.427a4.902 4.902 0 01-1.153 1.772 4.902 4.902 0 01-1.772 1.153c-.636.247-1.363.416-2.427.465-1.067.048-1.407.06-4.123.06h-.08c-2.643 0-2.987-.012-4.043-.06-1.064-.049-1.791-.218-2.427-.465a4.902 4.902 0 01-1.772-1.153 4.902 4.902 0 01-1.153-1.772c-.247-.636-.416-1.363-.465-2.427-.047-1.024-.06-1.379-.06-3.808v-.63c0-2.43.013-2.784.06-3.808.049-1.064.218-1.791.465-2.427a4.902 4.902 0 011.153-1.772A4.902 4.902 0 015.45 2.525c.636-.247 1.363-.416 2.427-.465C8.901 2.013 9.256 2 11.685 2h.63zm-.081 1.802h-.468c-2.456 0-2.784.011-3.807.058-.975.045-1.504.207-1.857.344-.467.182-.8.398-1.15.748-.35.35-.566.683-.748 1.15-.137.353-.3.882-.344 1.857-.047 1.023-.058 1.351-.058 3.807v.468c0 2.456.011 2.784.058 3.807.045.975.207 1.504.344 1.857.182.466.399.8.748 1.15.35.35.683.566 1.15.748.353.137.882.3 1.857.344 1.054.048 1.37.058 4.041.058h.08c2.597 0 2.917-.01 3.96-.058.976-.045 1.505-.207 1.858-.344.466-.182.8-.398 1.15-.748.35-.35.566-.683.748-1.15.137-.353.3-.882.344-1.857.048-1.055.058-1.37.058-4.041v-.08c0-2.597-.01-2.917-.058-3.96-.045-.976-.207-1.505-.344-1.858a3.097 3.097 0 00-.748-1.15 3.098 3.098 0 00-1.15-.748c-.353-.137-.882-.3-1.857-.344-1.023-.047-1.351-.058-3.807-.058zM12 6.865a5.135 5.135 0 110 10.27 5.135 5.135 0 010-10.27zm0 1.802a3.333 3.333 0 100 6.666 3.333 3.333 0 000-6.666zm5.338-3.205a1.2 1.2 0 110 2.4 1.2 1.2 0 010-2.4z" clipRule="evenodd"/>
                </svg>
              </a>
              {/* Pinterest */}
              <a href="#" target="_blank" rel="noopener noreferrer" aria-label="Pinterest"
                className="flex h-9 w-9 items-center justify-center rounded-full bg-[#E60023] text-white hover:opacity-90 transition-opacity">
                <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                  <path d="M12 0C5.373 0 0 5.373 0 12c0 5.084 3.163 9.426 7.627 11.174-.105-.949-.2-2.405.042-3.441.218-.937 1.407-5.965 1.407-5.965s-.359-.719-.359-1.782c0-1.668.967-2.914 2.171-2.914 1.023 0 1.518.769 1.518 1.69 0 1.029-.655 2.568-.994 3.995-.283 1.194.599 2.169 1.777 2.169 2.133 0 3.772-2.249 3.772-5.495 0-2.873-2.064-4.882-5.012-4.882-3.414 0-5.418 2.561-5.418 5.207 0 1.031.397 2.138.893 2.738a.36.36 0 0 1 .083.345l-.333 1.36c-.053.22-.174.267-.402.161-1.499-.698-2.436-2.889-2.436-4.649 0-3.785 2.75-7.262 7.929-7.262 4.163 0 7.398 2.967 7.398 6.931 0 4.136-2.607 7.464-6.227 7.464-1.216 0-2.359-.632-2.75-1.378l-.748 2.853c-.271 1.043-1.002 2.35-1.492 3.146C9.57 23.812 10.763 24 12 24c6.627 0 12-5.373 12-12S18.627 0 12 0z"/>
                </svg>
              </a>
              {/* YouTube */}
              <a href="#" target="_blank" rel="noopener noreferrer" aria-label="YouTube"
                className="flex h-9 w-9 items-center justify-center rounded-full bg-[#FF0000] text-white hover:opacity-90 transition-opacity">
                <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                  <path d="M23.498 6.186a3.016 3.016 0 0 0-2.122-2.136C19.505 3.545 12 3.545 12 3.545s-7.505 0-9.377.505A3.017 3.017 0 0 0 .502 6.186C0 8.07 0 12 0 12s0 3.93.502 5.814a3.016 3.016 0 0 0 2.122 2.136c1.871.505 9.376.505 9.376.505s7.505 0 9.377-.505a3.015 3.015 0 0 0 2.122-2.136C24 15.93 24 12 24 12s0-3.93-.502-5.814zM9.545 15.568V8.432L15.818 12l-6.273 3.568z"/>
                </svg>
              </a>
              {/* GitHub */}
              <a href="https://github.com/utafrali/EcommerceGo" target="_blank" rel="noopener noreferrer" aria-label="GitHub"
                className="flex h-9 w-9 items-center justify-center rounded-full bg-gray-900 text-white hover:opacity-90 transition-opacity">
                <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                  <path fillRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" clipRule="evenodd"/>
                </svg>
              </a>
            </div>

            {/* App store badges */}
            <div className="mt-5 space-y-2">
              <p className="text-xs font-semibold text-gray-700">Uygulamalarımız</p>
              <div className="flex flex-col gap-2">
                <a href="#" className="inline-flex items-center gap-2 rounded border border-gray-300 bg-white px-3 py-1.5 text-xs text-gray-800 hover:border-gray-400 transition-colors w-fit">
                  <svg className="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                    <path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.8-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z"/>
                  </svg>
                  <span>App Store&apos;dan İndir</span>
                </a>
                <a href="#" className="inline-flex items-center gap-2 rounded border border-gray-300 bg-white px-3 py-1.5 text-xs text-gray-800 hover:border-gray-400 transition-colors w-fit">
                  <svg className="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                    <path d="M3.18 23.76c.33.19.71.24 1.08.14L13.71 12 4.26.1C3.89 0 3.51.05 3.18.24 2.48.64 2 1.42 2 2.33v19.34c0 .91.48 1.69 1.18 2.09zM16.55 15.33l-2.8-2.8 2.8-2.8 3.53 2.03c.99.57.99 2 0 2.57l-3.53 2.03-.0 -.03zM5.05 22.67l9.53-9.53-2.9-2.9-6.63 12.43zM5.05 1.33L14.68 10.9l-2.9 2.9L5.05 1.33z"/>
                  </svg>
                  <span>Google Play&apos;den İndir</span>
                </a>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* ── Footer Bottom ─────────────────────────────────────────────────── */}
      <div className="border-t border-gray-100">
        <div className="mx-auto flex max-w-screen-xl flex-col items-center justify-between gap-3 px-4 py-5 sm:flex-row sm:px-6">
          <p className="text-sm text-gray-400">
            Telif hakkı &copy; {currentYear} EcommerceGo. Tüm hakları saklıdır.
          </p>
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2 rounded border border-gray-200 px-3 py-1.5 text-xs text-gray-500">
              <span>🇹🇷</span>
              <span>Teslimat Ülkesi: Türkiye</span>
            </div>
            <div className="flex items-center gap-2 rounded border border-gray-200 px-3 py-1.5 text-xs text-gray-500">
              <span>Dil: Türkçe</span>
            </div>
          </div>
        </div>
      </div>
    </footer>
  );
}

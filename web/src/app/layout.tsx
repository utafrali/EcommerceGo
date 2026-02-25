import type { Metadata } from 'next';
import Link from 'next/link';
import './globals.css';

export const metadata: Metadata = {
  title: 'EcommerceGo',
  description: 'AI-driven open-source e-commerce platform',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="min-h-screen flex flex-col">
        <header className="border-b border-gray-200 bg-white">
          <nav className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
            <div className="flex h-16 items-center justify-between">
              <Link
                href="/"
                className="text-xl font-bold text-gray-900"
              >
                EcommerceGo
              </Link>

              <div className="flex items-center gap-6">
                <Link
                  href="/products"
                  className="text-sm font-medium text-gray-700 hover:text-gray-900"
                >
                  Products
                </Link>
                <Link
                  href="/cart"
                  className="text-sm font-medium text-gray-700 hover:text-gray-900"
                >
                  Cart
                </Link>
                <Link
                  href="/auth/login"
                  className="text-sm font-medium text-gray-700 hover:text-gray-900"
                >
                  Sign In
                </Link>
              </div>
            </div>
          </nav>
        </header>

        <main className="flex-1">{children}</main>

        <footer className="border-t border-gray-200 bg-white py-8">
          <div className="mx-auto max-w-7xl px-4 text-center text-sm text-gray-500">
            EcommerceGo &mdash; AI-driven open-source e-commerce platform
          </div>
        </footer>
      </body>
    </html>
  );
}

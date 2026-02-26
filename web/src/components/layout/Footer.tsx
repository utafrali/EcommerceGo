import Link from 'next/link';

export default function Footer() {
  const currentYear = new Date().getFullYear();

  return (
    <footer className="border-t border-gray-200 bg-white">
      <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
        <div className="grid grid-cols-1 gap-8 sm:grid-cols-3">
          {/* Brand */}
          <div>
            <Link href="/" className="text-lg font-bold text-gray-900">
              EcommerceGo
            </Link>
            <p className="mt-2 text-sm text-gray-500">
              AI-driven open-source e-commerce platform built with Go
              microservices and Next.js.
            </p>
          </div>

          {/* Quick Links */}
          <div>
            <h3 className="text-sm font-semibold text-gray-900">
              Quick Links
            </h3>
            <ul className="mt-3 space-y-2">
              <li>
                <Link
                  href="/products"
                  className="text-sm text-gray-500 hover:text-gray-900 transition-colors"
                >
                  Products
                </Link>
              </li>
              <li>
                <Link
                  href="/cart"
                  className="text-sm text-gray-500 hover:text-gray-900 transition-colors"
                >
                  Cart
                </Link>
              </li>
              <li>
                <Link
                  href="/orders"
                  className="text-sm text-gray-500 hover:text-gray-900 transition-colors"
                >
                  Orders
                </Link>
              </li>
            </ul>
          </div>

          {/* About */}
          <div>
            <h3 className="text-sm font-semibold text-gray-900">About</h3>
            <ul className="mt-3 space-y-2">
              <li>
                <a
                  href="https://github.com/utafrali/EcommerceGo"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sm text-gray-500 hover:text-gray-900 transition-colors"
                >
                  GitHub
                </a>
              </li>
            </ul>
          </div>
        </div>

        {/* Copyright */}
        <div className="mt-8 border-t border-gray-200 pt-6 text-center text-sm text-gray-500">
          &copy; {currentYear} EcommerceGo. All rights reserved.
        </div>
      </div>
    </footer>
  );
}

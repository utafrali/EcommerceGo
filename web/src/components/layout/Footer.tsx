'use client';

import Link from 'next/link';

export default function Footer() {
  const currentYear = new Date().getFullYear();

  return (
    <footer className="bg-stone-900 text-stone-300">
      {/* Newsletter Section — prominent with background and incentive */}
      <div className="bg-stone-100">
        <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
          <div className="flex flex-col items-center justify-between gap-6 sm:flex-row">
            <div className="text-center sm:text-left">
              <h3 className="text-xl font-bold text-stone-900 sm:text-2xl">
                Subscribe to our newsletter
              </h3>
              <p className="mt-2 text-sm text-stone-600">
                Get the latest deals and updates delivered to your inbox.
              </p>
              <p className="mt-1 text-sm font-semibold text-brand">
                Get 10% off your first order
              </p>
            </div>
            <form className="flex w-full max-w-md gap-2" onSubmit={(e) => e.preventDefault()}>
              <input
                type="email"
                placeholder="Enter your email"
                className="flex-1 rounded-full border border-stone-300 bg-white px-5 py-3 text-sm text-stone-900 placeholder-stone-400 shadow-sm focus:border-brand focus:outline-none focus:ring-2 focus:ring-brand/20"
              />
              <button
                type="submit"
                className="rounded-full bg-brand px-6 py-3 text-sm font-semibold text-white shadow-sm transition-all duration-200 hover:bg-brand-light hover:shadow-md focus:outline-none focus:ring-2 focus:ring-brand focus:ring-offset-2"
              >
                Subscribe
              </button>
            </form>
          </div>
        </div>
      </div>

      {/* Main Footer Grid */}
      <div className="mx-auto max-w-7xl px-4 py-14 sm:px-6 lg:px-8">
        <div className="grid grid-cols-1 gap-10 sm:grid-cols-2 lg:grid-cols-4">
          {/* Categories */}
          <div>
            <h3 className="text-sm font-semibold uppercase tracking-wider text-white">
              Categories
            </h3>
            <ul className="mt-4 space-y-3">
              <li>
                <Link
                  href="/products?category=electronics"
                  className="text-sm text-stone-400 transition-colors hover:text-white"
                >
                  Electronics
                </Link>
              </li>
              <li>
                <Link
                  href="/products?category=clothing"
                  className="text-sm text-stone-400 transition-colors hover:text-white"
                >
                  Clothing
                </Link>
              </li>
              <li>
                <Link
                  href="/products?category=home-kitchen"
                  className="text-sm text-stone-400 transition-colors hover:text-white"
                >
                  Home &amp; Kitchen
                </Link>
              </li>
              <li>
                <Link
                  href="/products?category=sports"
                  className="text-sm text-stone-400 transition-colors hover:text-white"
                >
                  Sports
                </Link>
              </li>
              <li>
                <Link
                  href="/products?category=books"
                  className="text-sm text-stone-400 transition-colors hover:text-white"
                >
                  Books
                </Link>
              </li>
            </ul>
          </div>

          {/* Customer Service */}
          <div>
            <h3 className="text-sm font-semibold uppercase tracking-wider text-white">
              Customer Service
            </h3>
            <ul className="mt-4 space-y-3">
              <li>
                <a
                  href="#"
                  className="text-sm text-stone-400 transition-colors hover:text-white"
                >
                  Help Center
                </a>
              </li>
              <li>
                <a
                  href="#"
                  className="text-sm text-stone-400 transition-colors hover:text-white"
                >
                  Shipping Info
                </a>
              </li>
              <li>
                <a
                  href="#"
                  className="text-sm text-stone-400 transition-colors hover:text-white"
                >
                  Returns
                </a>
              </li>
              <li>
                <a
                  href="#"
                  className="text-sm text-stone-400 transition-colors hover:text-white"
                >
                  Contact Us
                </a>
              </li>
            </ul>
          </div>

          {/* Quick Links */}
          <div>
            <h3 className="text-sm font-semibold uppercase tracking-wider text-white">
              Quick Links
            </h3>
            <ul className="mt-4 space-y-3">
              <li>
                <Link
                  href="/products"
                  className="text-sm text-stone-400 transition-colors hover:text-white"
                >
                  Products
                </Link>
              </li>
              <li>
                <Link
                  href="/cart"
                  className="text-sm text-stone-400 transition-colors hover:text-white"
                >
                  Cart
                </Link>
              </li>
              <li>
                <Link
                  href="/orders"
                  className="text-sm text-stone-400 transition-colors hover:text-white"
                >
                  Orders
                </Link>
              </li>
            </ul>
          </div>

          {/* About */}
          <div>
            <h3 className="text-sm font-semibold uppercase tracking-wider text-white">
              About
            </h3>
            <ul className="mt-4 space-y-3">
              <li>
                <a
                  href="#"
                  className="text-sm text-stone-400 transition-colors hover:text-white"
                >
                  About Us
                </a>
              </li>
              <li>
                <a
                  href="#"
                  className="text-sm text-stone-400 transition-colors hover:text-white"
                >
                  Careers
                </a>
              </li>
              <li>
                <a
                  href="#"
                  className="text-sm text-stone-400 transition-colors hover:text-white"
                >
                  Blog
                </a>
              </li>
            </ul>
          </div>
        </div>

        {/* Brand & Social */}
        <div className="mt-14 flex flex-col items-center justify-between gap-6 border-t border-stone-800 pt-10 sm:flex-row">
          <div>
            <Link href="/" className="text-xl font-extrabold text-white tracking-tight">
              Ecommerce<span className="text-brand-light">Go</span>
            </Link>
            <p className="mt-2 max-w-xs text-sm leading-relaxed text-stone-400">
              AI-driven open-source e-commerce platform built with Go
              microservices and Next.js.
            </p>
          </div>

          {/* Social Icons — larger with hover effects */}
          <div className="flex items-center gap-5">
            <a
              href="https://github.com/utafrali/EcommerceGo"
              target="_blank"
              rel="noopener noreferrer"
              className="text-stone-400 transition-all duration-200 hover:text-white hover:scale-110"
              aria-label="GitHub"
            >
              <svg
                className="h-6 w-6"
                fill="currentColor"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  fillRule="evenodd"
                  d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z"
                  clipRule="evenodd"
                />
              </svg>
            </a>
            <a
              href="https://x.com"
              target="_blank"
              rel="noopener noreferrer"
              className="text-stone-400 transition-all duration-200 hover:text-white hover:scale-110"
              aria-label="X (Twitter)"
            >
              <svg
                className="h-6 w-6"
                fill="currentColor"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
              </svg>
            </a>
            <a
              href="#"
              target="_blank"
              rel="noopener noreferrer"
              className="text-stone-400 transition-all duration-200 hover:text-white hover:scale-110"
              aria-label="Instagram"
            >
              <svg
                className="h-6 w-6"
                fill="currentColor"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  fillRule="evenodd"
                  d="M12.315 2c2.43 0 2.784.013 3.808.06 1.064.049 1.791.218 2.427.465a4.902 4.902 0 011.772 1.153 4.902 4.902 0 011.153 1.772c.247.636.416 1.363.465 2.427.048 1.067.06 1.407.06 4.123v.08c0 2.643-.012 2.987-.06 4.043-.049 1.064-.218 1.791-.465 2.427a4.902 4.902 0 01-1.153 1.772 4.902 4.902 0 01-1.772 1.153c-.636.247-1.363.416-2.427.465-1.067.048-1.407.06-4.123.06h-.08c-2.643 0-2.987-.012-4.043-.06-1.064-.049-1.791-.218-2.427-.465a4.902 4.902 0 01-1.772-1.153 4.902 4.902 0 01-1.153-1.772c-.247-.636-.416-1.363-.465-2.427-.047-1.024-.06-1.379-.06-3.808v-.63c0-2.43.013-2.784.06-3.808.049-1.064.218-1.791.465-2.427a4.902 4.902 0 011.153-1.772A4.902 4.902 0 015.45 2.525c.636-.247 1.363-.416 2.427-.465C8.901 2.013 9.256 2 11.685 2h.63zm-.081 1.802h-.468c-2.456 0-2.784.011-3.807.058-.975.045-1.504.207-1.857.344-.467.182-.8.398-1.15.748-.35.35-.566.683-.748 1.15-.137.353-.3.882-.344 1.857-.047 1.023-.058 1.351-.058 3.807v.468c0 2.456.011 2.784.058 3.807.045.975.207 1.504.344 1.857.182.466.399.8.748 1.15.35.35.683.566 1.15.748.353.137.882.3 1.857.344 1.054.048 1.37.058 4.041.058h.08c2.597 0 2.917-.01 3.96-.058.976-.045 1.505-.207 1.858-.344.466-.182.8-.398 1.15-.748.35-.35.566-.683.748-1.15.137-.353.3-.882.344-1.857.048-1.055.058-1.37.058-4.041v-.08c0-2.597-.01-2.917-.058-3.96-.045-.976-.207-1.505-.344-1.858a3.097 3.097 0 00-.748-1.15 3.098 3.098 0 00-1.15-.748c-.353-.137-.882-.3-1.857-.344-1.023-.047-1.351-.058-3.807-.058zM12 6.865a5.135 5.135 0 110 10.27 5.135 5.135 0 010-10.27zm0 1.802a3.333 3.333 0 100 6.666 3.333 3.333 0 000-6.666zm5.338-3.205a1.2 1.2 0 110 2.4 1.2 1.2 0 010-2.4z"
                  clipRule="evenodd"
                />
              </svg>
            </a>
          </div>
        </div>
      </div>

      {/* Payment Methods & Copyright */}
      <div className="bg-stone-950">
        <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
          {/* Payment badges */}
          <div className="flex flex-wrap items-center justify-center gap-3 pb-4">
            <span className="rounded border border-stone-700 bg-stone-800 px-3 py-1.5 text-xs font-medium text-stone-300">
              Visa
            </span>
            <span className="rounded border border-stone-700 bg-stone-800 px-3 py-1.5 text-xs font-medium text-stone-300">
              Mastercard
            </span>
            <span className="rounded border border-stone-700 bg-stone-800 px-3 py-1.5 text-xs font-medium text-stone-300">
              PayPal
            </span>
            <span className="rounded border border-stone-700 bg-stone-800 px-3 py-1.5 text-xs font-medium text-stone-300">
              Apple Pay
            </span>
            <span className="rounded border border-stone-700 bg-stone-800 px-3 py-1.5 text-xs font-medium text-stone-300">
              Google Pay
            </span>
          </div>
          <p className="text-center text-sm text-stone-500">
            &copy; {currentYear} EcommerceGo. All rights reserved.
          </p>
        </div>
      </div>
    </footer>
  );
}

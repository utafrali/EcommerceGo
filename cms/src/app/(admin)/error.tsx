'use client';

import Link from 'next/link';

export default function AdminError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="flex items-center justify-center p-12">
      <div className="text-center">
        <h2 className="text-xl font-semibold text-gray-900 mb-3">Something went wrong</h2>
        <p className="text-gray-600 mb-6">
          An error occurred while loading this page. Please try again.
        </p>
        <div className="flex gap-3 justify-center">
          <button
            onClick={reset}
            className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700"
          >
            Try Again
          </button>
          <Link
            href="/dashboard"
            className="px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300"
          >
            Go to Dashboard
          </Link>
        </div>
      </div>
    </div>
  );
}

'use client';

import { useState, type FormEvent } from 'react';
import { useAuth } from '@/contexts/AuthContext';

export default function LoginPage() {
  const { login, error: authError, isLoading } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [localError, setLocalError] = useState<string | null>(null);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setLocalError(null);
    setSubmitting(true);

    try {
      await login({ email, password });
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : 'Invalid credentials. Please try again.');
    } finally {
      setSubmitting(false);
    }
  };

  const displayError = localError || authError;

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-100">
        <div className="text-gray-500">Loading...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100">
      <div className="w-full max-w-md">
        {/* Logo / Title */}
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-gray-900">EcommerceGo</h1>
          <p className="mt-2 text-gray-600">Admin Login</p>
        </div>

        {/* Login Card */}
        <div className="bg-white rounded-lg shadow-md p-8">
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Error Display */}
            {displayError && (
              <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-md text-sm">
                {displayError}
              </div>
            )}

            {/* Email Field */}
            <div>
              <label
                htmlFor="email"
                className="block text-sm font-medium text-gray-700 mb-1"
              >
                Email Address
              </label>
              <input
                id="email"
                name="email"
                type="email"
                autoComplete="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm
                         placeholder-gray-400 focus:outline-none focus:ring-2
                         focus:ring-indigo-500 focus:border-indigo-500 text-sm"
                placeholder="admin@example.com"
              />
            </div>

            {/* Password Field */}
            <div>
              <label
                htmlFor="password"
                className="block text-sm font-medium text-gray-700 mb-1"
              >
                Password
              </label>
              <input
                id="password"
                name="password"
                type="password"
                autoComplete="current-password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm
                         placeholder-gray-400 focus:outline-none focus:ring-2
                         focus:ring-indigo-500 focus:border-indigo-500 text-sm"
                placeholder="Enter your password"
              />
            </div>

            {/* Submit Button */}
            <button
              type="submit"
              disabled={submitting}
              className="w-full flex justify-center py-2.5 px-4 border border-transparent
                       rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600
                       hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2
                       focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed
                       transition-colors"
            >
              {submitting ? 'Signing in...' : 'Sign In'}
            </button>
          </form>
        </div>

        {/* Footer */}
        <p className="mt-6 text-center text-xs text-gray-500">
          EcommerceGo CMS &mdash; Administration Panel
        </p>
      </div>
    </div>
  );
}

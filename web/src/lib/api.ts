const BFF_URL = process.env.NEXT_PUBLIC_BFF_URL || 'http://localhost:3001';

/**
 * Fetch data from the BFF layer.
 * Works in both server components (SSR) and client components.
 */
export async function fetchAPI<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const url = `${BFF_URL}${path}`;

  const response = await fetch(url, {
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      ...options?.headers,
    },
    ...options,
  });

  if (!response.ok) {
    const errorBody = await response.json().catch(() => null);
    const message =
      errorBody?.error?.message ||
      `Request failed with status ${response.status}`;
    throw new Error(message);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}

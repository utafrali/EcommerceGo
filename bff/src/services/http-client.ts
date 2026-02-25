import { config } from '../config.js';

export class ApiError extends Error {
  constructor(
    public readonly statusCode: number,
    public readonly code: string,
    message: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

interface RequestOptions {
  method?: string;
  body?: unknown;
  token?: string;
  query?: Record<string, string | undefined>;
}

/**
 * Forward a request to the API Gateway and return the parsed JSON response.
 * Throws ApiError for non-2xx responses.
 */
export async function apiRequest<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const { method = 'GET', body, token, query } = options;

  const url = new URL(path, config.gatewayUrl);

  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined) {
        url.searchParams.set(key, value);
      }
    }
  }

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    Accept: 'application/json',
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const fetchOptions: RequestInit = {
    method,
    headers,
  };

  if (body !== undefined && method !== 'GET') {
    fetchOptions.body = JSON.stringify(body);
  }

  const response = await fetch(url.toString(), fetchOptions);

  if (!response.ok) {
    let errorBody: { error?: { code?: string; message?: string } } | undefined;
    try {
      errorBody = (await response.json()) as {
        error?: { code?: string; message?: string };
      };
    } catch {
      // response body is not JSON
    }

    const code = errorBody?.error?.code || `HTTP_${response.status}`;
    const message =
      errorBody?.error?.message || `Request failed with status ${response.status}`;

    throw new ApiError(response.status, code, message);
  }

  // 204 No Content
  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

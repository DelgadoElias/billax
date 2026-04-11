/**
 * API client wrapper with automatic JWT token handling
 */

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    public requestId?: string
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

interface ErrorResponse {
  error?: {
    code: string
    message: string
    request_id?: string
  }
}

/**
 * Fetch wrapper that:
 * - Prefixes /v1 to all paths
 * - Adds Bearer token to Authorization header
 * - Parses error responses
 */
export async function apiCall<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const token = localStorage.getItem('jwt_token')

  const url = `/v1${path}`
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  if (options.headers) {
    Object.assign(headers, options.headers)
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const response = await fetch(url, {
    ...options,
    headers,
  })

  // Parse response body
  const data = (await response.json()) as unknown as ErrorResponse | T

  if (!response.ok) {
    const error = data as ErrorResponse
    const errorCode = error.error?.code || 'unknown_error'
    const errorMessage = error.error?.message || 'An error occurred'
    const requestId = error.error?.request_id

    throw new ApiError(response.status, errorCode, errorMessage, requestId)
  }

  return data as T
}

/**
 * GET request
 */
export function apiGet<T>(path: string): Promise<T> {
  return apiCall<T>(path, { method: 'GET' })
}

/**
 * POST request
 */
export function apiPost<T>(path: string, data: unknown): Promise<T> {
  return apiCall<T>(path, {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

/**
 * PATCH request
 */
export function apiPatch<T>(path: string, data: unknown): Promise<T> {
  return apiCall<T>(path, {
    method: 'PATCH',
    body: JSON.stringify(data),
  })
}

/**
 * DELETE request
 */
export function apiDelete(path: string): Promise<void> {
  return apiCall<void>(path, { method: 'DELETE' })
}

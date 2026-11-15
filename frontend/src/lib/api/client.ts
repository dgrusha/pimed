/**
 * Minimal typed fetch wrapper for the Go /api.
 *
 * - Always sends the session cookie (`credentials: 'same-origin'`).
 * - Every non-GET request sends `Content-Type: application/json` with a JSON body
 *   (defaults to `{}`) — the backend rejects anything else with 415 (CSRF defence).
 * - Non-2xx responses become an `ApiError` carrying the status and the server's
 *   `{"error": "..."}` message.
 * - A 401 also notifies the handler registered with `onUnauthorized` (the auth
 *   store expires the session), unless the call opts out with `allow401` because
 *   401 is an expected answer there (login, setup, session probe).
 */

export class ApiError extends Error {
	readonly status: number;

	constructor(status: number, message: string) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
	}
}

export function isApiError(err: unknown, status?: number): err is ApiError {
	return err instanceof ApiError && (status === undefined || err.status === status);
}

type Method = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';

const BASE = '/api';

/** Message used when the backend cannot be reached at all. */
export const UNREACHABLE_MESSAGE = 'Cannot reach the server. Check your connection.';

export interface RequestOptions {
	/** 401 is an expected answer for this call: don't notify `onUnauthorized`. */
	allow401?: boolean;
}

let unauthorizedHandler: ((err: ApiError) => void) | null = null;

/**
 * Register the global handler for unexpected 401 responses (session expired or
 * revoked). Returns a function that unregisters it.
 */
export function onUnauthorized(handler: (err: ApiError) => void): () => void {
	unauthorizedHandler = handler;
	return () => {
		if (unauthorizedHandler === handler) unauthorizedHandler = null;
	};
}

export async function request<T>(
	method: Method,
	path: string,
	body?: unknown,
	options: RequestOptions = {}
): Promise<T> {
	const init: RequestInit = {
		method,
		credentials: 'same-origin',
		headers: { Accept: 'application/json' }
	};

	if (method !== 'GET') {
		init.headers = { ...init.headers, 'Content-Type': 'application/json' };
		init.body = JSON.stringify(body ?? {});
	}

	let res: Response;
	try {
		res = await fetch(BASE + path, init);
	} catch {
		// Network failure, server down, VPN dropped...
		throw new ApiError(0, UNREACHABLE_MESSAGE);
	}

	if (res.status === 204) return undefined as T;

	const data = await res.json().catch(() => null);

	if (!res.ok) {
		let message: string;
		if (data && typeof data === 'object' && typeof data.error === 'string') {
			message = data.error;
		} else if (res.status === 502 || res.status === 503 || res.status === 504) {
			// Proxy (Caddy / Vite dev server) could not reach the Go backend.
			message = UNREACHABLE_MESSAGE;
		} else {
			message = `Request failed (${res.status})`;
		}
		const err = new ApiError(res.status, message);
		if (res.status === 401 && !options.allow401) unauthorizedHandler?.(err);
		throw err;
	}

	return data as T;
}

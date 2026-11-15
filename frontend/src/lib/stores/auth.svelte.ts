/**
 * Global auth state (Svelte 5 runes).
 *
 * The root layout calls `auth.init()` once on startup; route guarding is derived
 * from `needsSetup` / `user` (see `routeFor` below and src/routes/+layout.svelte).
 * Unexpected 401s from any API call expire the session centrally (see
 * `onUnauthorized` in $lib/api/client.ts), so pages need not handle them.
 */
import * as api from '$lib/api';
import type { User } from '$lib/api';

class AuthState {
	/** The signed-in user, or null when anonymous. */
	user = $state<User | null>(null);
	/** True while the server reports that no account exists yet. */
	needsSetup = $state(false);
	/** True while the initial status/session check is in flight. */
	loading = $state(true);
	/** Set when the startup check failed for a reason other than 401 (e.g. server down). */
	error = $state<string | null>(null);
	/** One-shot message for the login screen (e.g. "setup already completed"). */
	notice = $state<string | null>(null);

	isAdmin = $derived(this.user?.role === 'admin');

	constructor() {
		api.onUnauthorized(() => {
			if (this.user) this.expire('Your session has expired. Please sign in again.');
		});
	}

	async init(): Promise<void> {
		this.loading = true;
		this.error = null;
		try {
			const { needsSetup } = await api.setupStatus();
			this.needsSetup = needsSetup;
			if (needsSetup) {
				this.user = null;
				return;
			}
			try {
				this.user = await api.me();
			} catch (err) {
				if (api.isApiError(err, 401)) this.user = null;
				else throw err;
			}
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Something went wrong.';
		} finally {
			this.loading = false;
		}
	}

	/** Called after a successful login or first-run setup. */
	signedIn(user: User): void {
		this.needsSetup = false;
		this.notice = null;
		this.user = user;
	}

	/** Forget the local session (e.g. after any 401 from the API). */
	expire(message?: string): void {
		this.user = null;
		if (message) this.notice = message;
	}

	/**
	 * Someone else completed first-run setup meanwhile: leave the setup screen and
	 * show `message` on the login screen (the route guard redirects there).
	 */
	setupClosed(message: string): void {
		this.needsSetup = false;
		this.user = null;
		this.notice = message;
	}

	/**
	 * Sign out on the server, then forget the local session. If the request fails
	 * the error is rethrown and local state is kept: the server session (and its
	 * HttpOnly cookie) is still active, so pretending otherwise would be wrong.
	 */
	async logout(): Promise<void> {
		await api.logout();
		this.user = null;
	}

	/** Consume the one-shot login notice. */
	takeNotice(): string | null {
		const n = this.notice;
		this.notice = null;
		return n;
	}
}

export const auth = new AuthState();

/**
 * Where the current auth state allows the user to be. Returns the path to
 * redirect to, or null when `pathname` is fine as is.
 */
export function routeFor(pathname: string): string | null {
	const path = pathname.replace(/\/+$/, '') || '/';

	if (auth.needsSetup) return path === '/setup' ? null : '/setup';

	if (!auth.user) return path === '/login' ? null : '/login';

	if (path === '/login' || path === '/setup') return '/';
	if (path === '/users' && !auth.isAdmin) return '/';
	return null;
}

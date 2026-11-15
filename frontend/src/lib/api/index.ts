import { request } from './client';
import type { CreateUserInput, Credentials, SetupStatus, User } from './types';

export { ApiError, isApiError, onUnauthorized, UNREACHABLE_MESSAGE } from './client';
export type { CreateUserInput, Credentials, Role, SetupStatus, User } from './types';

// ── First-run setup ────────────────────────────────────
export function setupStatus(): Promise<SetupStatus> {
	return request<SetupStatus>('GET', '/setup/status');
}

export async function setup(input: Credentials): Promise<User> {
	const { user } = await request<{ user: User }>('POST', '/setup', input, { allow401: true });
	return user;
}

// ── Authentication ─────────────────────────────────────
export async function login(input: Credentials): Promise<User> {
	// 401 here means wrong credentials, not an expired session.
	const { user } = await request<{ user: User }>('POST', '/auth/login', input, { allow401: true });
	return user;
}

export function logout(): Promise<void> {
	return request<void>('POST', '/auth/logout', {});
}

export async function me(): Promise<User> {
	// 401 here just means "not signed in" (startup probe).
	const { user } = await request<{ user: User }>('GET', '/auth/me', undefined, { allow401: true });
	return user;
}

// ── Users (admin) ──────────────────────────────────────
export async function listUsers(): Promise<User[]> {
	const { users } = await request<{ users: User[] }>('GET', '/users');
	return users;
}

export async function createUser(input: CreateUserInput): Promise<User> {
	const { user } = await request<{ user: User }>('POST', '/users', input);
	return user;
}

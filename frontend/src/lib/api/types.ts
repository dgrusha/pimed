/** Types mirroring docs/api.md (v0). */

export type Role = 'admin' | 'user';

export interface User {
	id: number;
	username: string;
	role: Role;
	/** RFC 3339 timestamp. */
	createdAt: string;
}

export interface Credentials {
	username: string;
	password: string;
}

export interface CreateUserInput extends Credentials {
	role?: Role;
}

export interface SetupStatus {
	needsSetup: boolean;
}

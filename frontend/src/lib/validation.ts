/** Client-side validation mirroring the API contract (docs/api.md). */

export const USERNAME_MIN = 3;
export const USERNAME_MAX = 32;
export const PASSWORD_MIN = 8;
export const PASSWORD_MAX = 128;
const USERNAME_RE = /^[a-zA-Z0-9_.-]+$/;

export function validateUsername(value: string): string | null {
	if (!value) return 'Username is required.';
	if (value.length < USERNAME_MIN || value.length > USERNAME_MAX)
		return `Username must be ${USERNAME_MIN}–${USERNAME_MAX} characters.`;
	if (!USERNAME_RE.test(value)) return 'Use only letters, numbers, dot, dash and underscore.';
	return null;
}

export function validatePassword(value: string): string | null {
	if (!value) return 'Password is required.';
	// Count code points, not UTF-16 units, so emoji etc. count as one character.
	const len = [...value].length;
	if (len < PASSWORD_MIN) return `Password must be at least ${PASSWORD_MIN} characters.`;
	if (len > PASSWORD_MAX) return `Password must be at most ${PASSWORD_MAX} characters.`;
	return null;
}

export function validateConfirm(password: string, confirm: string): string | null {
	if (!confirm) return 'Please confirm your password.';
	if (password !== confirm) return 'Passwords do not match.';
	return null;
}

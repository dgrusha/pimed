<script lang="ts">
	import * as api from '$lib/api';
	import type { Credentials } from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';
	import AuthLayout from '$lib/components/templates/AuthLayout.svelte';
	import AuthForm from '$lib/components/organisms/AuthForm.svelte';

	// One-shot message from a redirect (e.g. setup already completed, session expired).
	const notice = auth.takeNotice();

	let error = $state<string | null>(null);

	async function handleLogin(credentials: Credentials) {
		error = null;
		try {
			const user = await api.login(credentials);
			auth.signedIn(user); // route guard takes us to /
		} catch (err) {
			if (api.isApiError(err, 401)) {
				error = 'Invalid username or password.';
			} else if (api.isApiError(err, 429)) {
				error = 'Too many sign-in attempts. Please wait a few minutes and try again.';
			} else if (api.isApiError(err, 503) && err.message !== api.UNREACHABLE_MESSAGE) {
				// The backend itself answered: it is busy hashing other passwords.
				error = 'Server busy — please try again in a moment.';
			} else {
				error = err instanceof Error ? err.message : 'Sign-in failed. Please try again.';
			}
		}
	}
</script>

<svelte:head>
	<title>Sign in · PiMed</title>
</svelte:head>

<AuthLayout title="Sign in" subtitle="Welcome back to your PiMed.">
	<AuthForm mode="login" submitLabel="Sign in" {notice} {error} onsubmit={handleLogin}>
		{#snippet footer()}
			Accounts are created by your PiMed administrator.
		{/snippet}
	</AuthForm>
</AuthLayout>

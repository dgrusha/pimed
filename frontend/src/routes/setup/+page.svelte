<script lang="ts">
	import * as api from '$lib/api';
	import type { Credentials } from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';
	import { toasts } from '$lib/stores/toast.svelte';
	import AuthLayout from '$lib/components/templates/AuthLayout.svelte';
	import AuthForm from '$lib/components/organisms/AuthForm.svelte';

	let error = $state<string | null>(null);

	async function handleSetup(credentials: Credentials) {
		error = null;
		try {
			const user = await api.setup(credentials);
			toasts.success(`Welcome, ${user.username}!`);
			auth.signedIn(user); // route guard takes us to /
		} catch (err) {
			if (api.isApiError(err, 409)) {
				// Someone else finished setup first; the guard redirects to /login.
				auth.setupClosed('Setup has already been completed. Please sign in.');
				return;
			}
			error = err instanceof Error ? err.message : 'Setup failed. Please try again.';
		}
	}
</script>

<svelte:head>
	<title>Welcome · PiMed</title>
</svelte:head>

<AuthLayout
	title="Welcome to PiMed"
	subtitle="Create the first account. It becomes the administrator of this PiMed."
>
	<AuthForm mode="setup" submitLabel="Create admin account" {error} onsubmit={handleSetup} />
</AuthLayout>

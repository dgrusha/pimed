<script lang="ts">
	import type { Snippet } from 'svelte';
	import { auth } from '$lib/stores/auth.svelte';
	import { toasts } from '$lib/stores/toast.svelte';
	import DashboardLayout from '$lib/components/templates/DashboardLayout.svelte';

	let { children }: { children: Snippet } = $props();

	async function handleLogout() {
		try {
			await auth.logout();
		} catch {
			// The server session is still valid, so stay signed in and say so.
			toasts.error('Sign-out failed — your session is still active.');
			return;
		}
		toasts.show('You have been signed out.');
	}
</script>

<!-- The root layout guarantees a signed-in user here; the guard keeps TypeScript honest. -->
{#if auth.user}
	<DashboardLayout user={auth.user} onlogout={handleLogout}>
		{@render children()}
	</DashboardLayout>
{/if}

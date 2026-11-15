<script lang="ts">
	import '$lib/styles/global.css';
	import { onMount, type Snippet } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { auth, routeFor } from '$lib/stores/auth.svelte';
	import Spinner from '$lib/components/atoms/Spinner.svelte';
	import Button from '$lib/components/atoms/Button.svelte';
	import Icon from '$lib/components/atoms/Icon.svelte';
	import Alert from '$lib/components/molecules/Alert.svelte';
	import ToastRegion from '$lib/components/organisms/ToastRegion.svelte';

	let { children }: { children: Snippet } = $props();

	onMount(() => {
		auth.init();
	});

	// Route guard: derived from auth state, so login/logout/setup/expiry all redirect
	// automatically without pages having to navigate themselves.
	let redirect = $derived(auth.loading || auth.error ? null : routeFor(page.url.pathname));

	$effect(() => {
		if (redirect) goto(redirect, { replaceState: true });
	});
</script>

{#if auth.loading || redirect}
	<div class="splash" aria-busy="true">
		<span class="mark" aria-hidden="true">Pi</span>
		<Spinner size={22} label="Loading PiMed" />
	</div>
{:else if auth.error}
	<div class="splash">
		<div class="offline">
			<Alert kind="error" title="Can't connect to PiMed">{auth.error}</Alert>
			<Button variant="secondary" onclick={() => auth.init()}>
				<Icon name="refresh" size={18} />
				Try again
			</Button>
		</div>
	</div>
{:else}
	{@render children()}
{/if}

<ToastRegion />

<style>
	.splash {
		min-height: 100vh;
		min-height: 100dvh;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: var(--space-6);
		padding: calc(var(--safe-top) + var(--space-6)) var(--space-4)
			calc(var(--safe-bottom) + var(--space-6));
		color: var(--color-text-secondary);
		/* Don't flash a spinner on fast connections. */
		animation: fade-in var(--duration-normal) var(--ease-out) 250ms both;
	}

	.mark {
		display: grid;
		place-items: center;
		width: 56px;
		height: 56px;
		border-radius: 16px;
		font-size: 1.375rem;
		font-weight: var(--weight-bold);
		color: #fff;
		background: linear-gradient(135deg, var(--color-accent), var(--color-accent-2));
		box-shadow: 0 12px 36px rgba(79, 140, 255, 0.4);
	}

	.offline {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-4);
		width: min(420px, 100%);
	}

	.offline > :global(.alert) {
		width: 100%;
	}

	@keyframes fade-in {
		from {
			opacity: 0;
		}
	}
</style>

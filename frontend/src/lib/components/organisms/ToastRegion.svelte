<script lang="ts">
	import { fly } from 'svelte/transition';
	import { backOut, cubicIn } from 'svelte/easing';
	import { prefersReducedMotion } from 'svelte/motion';
	import { toasts } from '$lib/stores/toast.svelte';
	import Toast from '../molecules/Toast.svelte';

	let distance = $derived(prefersReducedMotion.current ? 0 : -24);
</script>

<div class="region" aria-live="polite" aria-relevant="additions">
	{#each toasts.items as toast (toast.id)}
		<div
			in:fly={{ y: distance, duration: 320, easing: backOut }}
			out:fly={{ y: distance, duration: 180, easing: cubicIn }}
		>
			<Toast kind={toast.kind} message={toast.message} ondismiss={() => toasts.dismiss(toast.id)} />
		</div>
	{/each}
</div>

<style>
	.region {
		position: fixed;
		top: calc(var(--safe-top) + var(--space-4));
		left: 50%;
		z-index: var(--z-toast);
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-2);
		transform: translateX(-50%);
		pointer-events: none;
	}
</style>

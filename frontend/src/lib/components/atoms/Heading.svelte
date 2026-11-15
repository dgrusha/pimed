<script lang="ts">
	import type { Snippet } from 'svelte';

	type Size = 'sm' | 'md' | 'lg' | 'xl' | 'xxl';

	interface Props {
		level?: 1 | 2 | 3 | 4;
		/** Visual size, independent of the semantic level. */
		size?: Size;
		id?: string;
		class?: string;
		children: Snippet;
	}

	let { level = 2, size, id, class: className, children }: Props = $props();

	const defaultSize: Record<1 | 2 | 3 | 4, Size> = { 1: 'xl', 2: 'lg', 3: 'md', 4: 'sm' };
	let resolvedSize = $derived(size ?? defaultSize[level]);
</script>

<svelte:element this={`h${level}`} {id} class={['heading', resolvedSize, className]}>
	{@render children()}
</svelte:element>

<style>
	.heading {
		font-weight: var(--weight-bold);
		line-height: var(--leading-tight);
		letter-spacing: var(--tracking-tight);
		color: var(--color-text);
		text-wrap: balance;
	}

	.sm {
		font-size: var(--text-md);
		font-weight: var(--weight-semibold);
	}
	.md {
		font-size: var(--text-lg);
		font-weight: var(--weight-semibold);
	}
	.lg {
		font-size: var(--text-xl);
	}
	.xl {
		font-size: var(--text-2xl);
	}
	.xxl {
		font-size: var(--text-3xl);
	}
</style>

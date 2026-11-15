<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	interface Props extends HTMLAttributes<HTMLElement> {
		as?: 'div' | 'section' | 'article';
		padding?: 'sm' | 'md' | 'lg';
		/** Stronger, more opaque glass for primary surfaces (e.g. the auth card). */
		strong?: boolean;
		children: Snippet;
	}

	let {
		as = 'div',
		padding = 'md',
		strong = false,
		class: className,
		children,
		...rest
	}: Props = $props();
</script>

<svelte:element this={as} class={['card', `pad-${padding}`, { strong }, className]} {...rest}>
	{@render children()}
</svelte:element>

<style>
	.card {
		position: relative;
		background: var(--color-glass);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-xl);
		box-shadow: var(--shadow-md);
		-webkit-backdrop-filter: blur(var(--blur-md)) saturate(var(--glass-saturate));
		backdrop-filter: blur(var(--blur-md)) saturate(var(--glass-saturate));
	}

	/* Subtle top highlight, like light catching the edge of glass. */
	.card::before {
		content: '';
		position: absolute;
		inset: 0;
		border-radius: inherit;
		background: linear-gradient(180deg, rgba(255, 255, 255, 0.06), transparent 40%);
		pointer-events: none;
	}

	.strong {
		background: var(--color-glass-strong);
		box-shadow: var(--shadow-lg);
		-webkit-backdrop-filter: blur(var(--blur-lg)) saturate(var(--glass-saturate));
		backdrop-filter: blur(var(--blur-lg)) saturate(var(--glass-saturate));
	}

	.pad-sm {
		padding: var(--space-4);
	}
	.pad-md {
		padding: var(--space-6);
	}
	.pad-lg {
		padding: clamp(var(--space-6), 5vw, var(--space-10));
	}
</style>

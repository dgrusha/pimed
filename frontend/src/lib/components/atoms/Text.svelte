<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		as?: 'p' | 'span' | 'div' | 'small';
		size?: 'xs' | 'sm' | 'md' | 'lg';
		tone?: 'default' | 'secondary' | 'tertiary' | 'danger' | 'success';
		weight?: 'regular' | 'medium' | 'semibold';
		id?: string;
		class?: string;
		children: Snippet;
	}

	let {
		as = 'p',
		size = 'md',
		tone = 'default',
		weight = 'regular',
		id,
		class: className,
		children
	}: Props = $props();
</script>

<svelte:element this={as} {id} class={['text', size, tone, weight, className]}>
	{@render children()}
</svelte:element>

<style>
	.text {
		line-height: var(--leading-normal);
	}

	.xs {
		font-size: var(--text-xs);
	}
	.sm {
		font-size: var(--text-sm);
	}
	.md {
		font-size: var(--text-md);
	}
	.lg {
		font-size: var(--text-lg);
	}

	.default {
		color: var(--color-text);
	}
	.secondary {
		color: var(--color-text-secondary);
	}
	.tertiary {
		color: var(--color-text-tertiary);
	}
	.danger {
		color: var(--color-danger);
	}
	.success {
		color: var(--color-success);
	}

	.regular {
		font-weight: var(--weight-regular);
	}
	.medium {
		font-weight: var(--weight-medium);
	}
	.semibold {
		font-weight: var(--weight-semibold);
	}
</style>

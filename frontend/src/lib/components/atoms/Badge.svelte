<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		tone?: 'neutral' | 'accent' | 'success' | 'warning';
		children: Snippet;
	}

	let { tone = 'neutral', children }: Props = $props();
</script>

<span class={['badge', tone]}>{@render children()}</span>

<style>
	.badge {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		padding: 2px var(--space-2);
		border-radius: var(--radius-full);
		font-size: var(--text-xs);
		font-weight: var(--weight-semibold);
		letter-spacing: 0.02em;
		line-height: 1.5;
		white-space: nowrap;
	}

	.neutral {
		background: var(--color-surface-hover);
		color: var(--color-text-secondary);
	}
	.accent {
		background: var(--color-accent-soft);
		color: var(--color-accent-hover);
	}
	.success {
		background: var(--color-success-soft);
		color: var(--color-success);
	}
	.warning {
		background: var(--color-warning-soft);
		color: var(--color-warning);
	}
</style>

<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';
	import Spinner from './Spinner.svelte';

	interface Props extends HTMLButtonAttributes {
		variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
		size?: 'sm' | 'md' | 'lg';
		/** Shows a spinner and disables the button. */
		loading?: boolean;
		block?: boolean;
		children: Snippet;
	}

	let {
		variant = 'primary',
		size = 'md',
		loading = false,
		block = false,
		disabled = false,
		type = 'button',
		children,
		class: className,
		...rest
	}: Props = $props();
</script>

<button
	{type}
	class={['btn', variant, size, { block, loading }, className]}
	disabled={disabled || loading}
	aria-busy={loading || undefined}
	{...rest}
>
	<!-- Content stays in the layout while loading so the button never changes size. -->
	<span class="content">{@render children()}</span>
	{#if loading}
		<span class="spinner"><Spinner size={size === 'sm' ? 14 : 18} /></span>
	{/if}
</button>

<style>
	.btn {
		position: relative;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: var(--space-2);
		min-height: 44px; /* Apple HIG minimum touch target */
		padding: 0 var(--space-5);
		border: 1px solid transparent;
		border-radius: var(--radius-md);
		font-size: var(--text-md);
		font-weight: var(--weight-semibold);
		letter-spacing: -0.01em;
		white-space: nowrap;
		cursor: pointer;
		user-select: none;
		touch-action: manipulation;
		transition:
			background-color var(--duration-fast) var(--ease-standard),
			border-color var(--duration-fast) var(--ease-standard),
			transform var(--duration-fast) var(--ease-standard),
			box-shadow var(--duration-fast) var(--ease-standard),
			opacity var(--duration-fast) var(--ease-standard);
	}

	.btn:active:not(:disabled) {
		transform: scale(0.97);
	}

	.btn:disabled {
		cursor: not-allowed;
		opacity: 0.55;
	}

	.btn.loading {
		cursor: progress;
		opacity: 0.85;
	}

	.content {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
	}

	.loading .content {
		visibility: hidden;
	}

	.spinner {
		position: absolute;
		inset: 0;
		display: grid;
		place-items: center;
	}

	.block {
		display: flex;
		width: 100%;
	}

	.sm {
		min-height: 36px;
		padding: 0 var(--space-3);
		font-size: var(--text-sm);
		border-radius: var(--radius-sm);
	}

	.lg {
		min-height: 52px;
		padding: 0 var(--space-6);
		font-size: var(--text-lg);
	}

	.primary {
		background: var(--color-accent);
		color: var(--color-text-on-accent);
		box-shadow: 0 6px 20px rgba(79, 140, 255, 0.25);
	}

	.primary:hover:not(:disabled) {
		background: var(--color-accent-hover);
	}

	.primary:active:not(:disabled) {
		background: var(--color-accent-active);
	}

	.secondary {
		background: var(--color-surface);
		border-color: var(--color-border);
		color: var(--color-text);
	}

	.secondary:hover:not(:disabled) {
		background: var(--color-surface-hover);
		border-color: var(--color-border-strong);
	}

	.ghost {
		background: transparent;
		color: var(--color-text-secondary);
	}

	.ghost:hover:not(:disabled) {
		background: var(--color-surface);
		color: var(--color-text);
	}

	.danger {
		background: var(--color-danger-soft);
		color: var(--color-danger);
	}

	.danger:hover:not(:disabled) {
		background: rgba(255, 90, 95, 0.24);
	}

	.btn:focus-visible {
		box-shadow: var(--focus-ring);
	}
</style>

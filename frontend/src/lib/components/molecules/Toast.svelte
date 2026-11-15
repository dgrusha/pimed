<script lang="ts">
	import Icon from '../atoms/Icon.svelte';
	import type { ToastKind } from '$lib/stores/toast.svelte';

	interface Props {
		kind?: ToastKind;
		message: string;
		ondismiss?: () => void;
	}

	let { kind = 'info', message, ondismiss }: Props = $props();
</script>

<div class={['toast', kind]}>
	<span class="icon">
		<Icon name={kind === 'success' ? 'check' : kind === 'error' ? 'alert' : 'info'} size={20} />
	</span>
	<p class="message">{message}</p>
	{#if ondismiss}
		<button type="button" class="close" onclick={ondismiss} aria-label="Dismiss notification">
			<Icon name="x" size={16} />
		</button>
	{/if}
</div>

<style>
	.toast {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		width: min(420px, calc(100vw - var(--space-8)));
		padding: var(--space-3) var(--space-3) var(--space-3) var(--space-4);
		border-radius: var(--radius-lg);
		background: var(--color-glass-strong);
		border: 1px solid var(--color-border-strong);
		box-shadow: var(--shadow-lg);
		-webkit-backdrop-filter: blur(var(--blur-md)) saturate(var(--glass-saturate));
		backdrop-filter: blur(var(--blur-md)) saturate(var(--glass-saturate));
		pointer-events: auto;
	}

	.icon {
		flex: none;
	}
	.success .icon {
		color: var(--color-success);
	}
	.error .icon {
		color: var(--color-danger);
	}
	.info .icon {
		color: var(--color-accent);
	}

	.message {
		flex: 1;
		min-width: 0;
		font-size: var(--text-sm);
		font-weight: var(--weight-medium);
	}

	.close {
		display: grid;
		place-items: center;
		flex: none;
		width: 32px;
		height: 32px;
		padding: 0;
		border: 0;
		border-radius: 50%;
		background: transparent;
		color: var(--color-text-secondary);
		cursor: pointer;
		transition: background-color var(--duration-fast) var(--ease-standard);
	}

	.close:hover {
		background: var(--color-surface-hover);
		color: var(--color-text);
	}
</style>

<script lang="ts">
	import type { Snippet } from 'svelte';
	import Icon, { type IconName } from '../atoms/Icon.svelte';

	interface Props {
		kind?: 'info' | 'success' | 'warning' | 'error';
		title?: string;
		children: Snippet;
	}

	let { kind = 'info', title, children }: Props = $props();

	const icons: Record<NonNullable<Props['kind']>, IconName> = {
		info: 'info',
		success: 'check',
		warning: 'alert',
		error: 'alert'
	};
</script>

<div class={['alert', kind]} role={kind === 'error' ? 'alert' : 'status'}>
	<span class="icon"><Icon name={icons[kind]} size={20} /></span>
	<div class="body">
		{#if title}<p class="title">{title}</p>{/if}
		<div class="message">{@render children()}</div>
	</div>
</div>

<style>
	.alert {
		display: flex;
		gap: var(--space-3);
		align-items: flex-start;
		padding: var(--space-3) var(--space-4);
		border-radius: var(--radius-md);
		border: 1px solid transparent;
		font-size: var(--text-sm);
		line-height: var(--leading-normal);
	}

	.icon {
		flex: none;
		margin-top: 1px;
	}

	.body {
		min-width: 0;
	}

	.title {
		font-weight: var(--weight-semibold);
		color: var(--color-text);
	}

	.message {
		color: var(--color-text-secondary);
	}

	.info {
		background: var(--color-info-soft);
		border-color: rgba(79, 140, 255, 0.3);
	}
	.info .icon {
		color: var(--color-info);
	}

	.success {
		background: var(--color-success-soft);
		border-color: rgba(52, 199, 123, 0.3);
	}
	.success .icon {
		color: var(--color-success);
	}

	.warning {
		background: var(--color-warning-soft);
		border-color: rgba(255, 179, 64, 0.3);
	}
	.warning .icon {
		color: var(--color-warning);
	}

	.error {
		background: var(--color-danger-soft);
		border-color: rgba(255, 90, 95, 0.32);
	}
	.error .icon {
		color: var(--color-danger);
	}
	.error .message {
		color: var(--color-text);
	}
</style>

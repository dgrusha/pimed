<script lang="ts">
	import type { HTMLSelectAttributes } from 'svelte/elements';

	interface Option {
		value: string;
		label: string;
	}

	interface Props extends Omit<HTMLSelectAttributes, 'value'> {
		value?: string;
		options: Option[];
		invalid?: boolean;
	}

	let {
		value = $bindable(''),
		options,
		invalid = false,
		class: className,
		...rest
	}: Props = $props();
</script>

<div class={['wrap', className]}>
	<select class="select" bind:value aria-invalid={invalid || undefined} {...rest}>
		{#each options as opt (opt.value)}
			<option value={opt.value}>{opt.label}</option>
		{/each}
	</select>
	<svg class="chevron" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
		<path d="m6 9 6 6 6-6" />
	</svg>
</div>

<style>
	.wrap {
		position: relative;
	}

	.select {
		width: 100%;
		min-height: 48px;
		padding: 0 calc(var(--space-4) + 24px) 0 var(--space-4);
		font-size: max(16px, var(--text-md));
		color: var(--color-text);
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		appearance: none;
		cursor: pointer;
		transition:
			border-color var(--duration-fast) var(--ease-standard),
			box-shadow var(--duration-fast) var(--ease-standard);
	}

	.select:hover {
		border-color: var(--color-border-strong);
	}

	.select:focus-visible {
		border-color: var(--color-accent);
		box-shadow: var(--focus-ring);
	}

	.select option {
		background: var(--color-bg-elevated);
		color: var(--color-text);
	}

	.chevron {
		position: absolute;
		right: var(--space-4);
		top: 50%;
		width: 18px;
		height: 18px;
		transform: translateY(-50%);
		fill: none;
		stroke: var(--color-text-secondary);
		stroke-width: 2;
		stroke-linecap: round;
		stroke-linejoin: round;
		pointer-events: none;
	}
</style>

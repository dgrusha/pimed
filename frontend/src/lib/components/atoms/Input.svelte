<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';

	interface Props extends Omit<HTMLInputAttributes, 'value'> {
		value?: string;
		invalid?: boolean;
	}

	let { value = $bindable(''), invalid = false, class: className, ...rest }: Props = $props();
</script>

<input
	class={['input', className]}
	bind:value
	aria-invalid={invalid || undefined}
	autocapitalize="off"
	autocorrect="off"
	spellcheck="false"
	{...rest}
/>

<style>
	.input {
		width: 100%;
		min-height: 48px;
		padding: 0 var(--space-4);
		/* >= 16px prevents iOS Safari from zooming on focus. */
		font-size: max(16px, var(--text-md));
		color: var(--color-text);
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		appearance: none;
		transition:
			border-color var(--duration-fast) var(--ease-standard),
			background-color var(--duration-fast) var(--ease-standard),
			box-shadow var(--duration-fast) var(--ease-standard);
	}

	.input::placeholder {
		color: var(--color-text-tertiary);
	}

	.input:hover {
		border-color: var(--color-border-strong);
	}

	.input:focus,
	.input:focus-visible {
		border-color: var(--color-accent);
		background: rgba(255, 255, 255, 0.07);
		box-shadow: var(--focus-ring);
	}

	.input[aria-invalid='true'] {
		border-color: var(--color-danger);
	}

	.input[aria-invalid='true']:focus {
		box-shadow: 0 0 0 3px rgba(255, 90, 95, 0.4);
	}

	.input:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	/* Keep autofill on-theme in WebKit. */
	.input:-webkit-autofill {
		-webkit-text-fill-color: var(--color-text);
		-webkit-box-shadow: 0 0 0 100px #1a1d28 inset;
		caret-color: var(--color-text);
	}
</style>

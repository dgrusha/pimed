<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';
	import Input from '../atoms/Input.svelte';

	interface Props extends Omit<HTMLInputAttributes, 'value'> {
		label: string;
		value?: string;
		/** Validation message; when set the input is marked invalid and described by it. */
		error?: string | null;
		hint?: string;
	}

	let { label, value = $bindable(''), error = null, hint, id, ...rest }: Props = $props();

	const uid = $props.id();
	let inputId = $derived(id ?? `field-${uid}`);
	let hintId = $derived(`${inputId}-hint`);
	let errorId = $derived(`${inputId}-error`);
	// Only one of error / hint is rendered below, so only reference that one.
	let describedBy = $derived(error ? errorId : hint ? hintId : undefined);
</script>

<div class="field">
	<label class="label" for={inputId}>{label}</label>
	<Input id={inputId} bind:value invalid={!!error} aria-describedby={describedBy} {...rest} />
	<!-- Reserve space for one line of feedback so errors never shift the layout. -->
	<div class="feedback">
		{#if error}
			<p id={errorId} class="error" role="alert">{error}</p>
		{:else if hint}
			<p id={hintId} class="hint">{hint}</p>
		{/if}
	</div>
</div>

<style>
	.field {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.label {
		font-size: var(--text-sm);
		font-weight: var(--weight-medium);
		color: var(--color-text-secondary);
	}

	.feedback {
		min-height: 1.25rem;
		margin-top: calc(var(--space-1) * -1);
	}

	.error,
	.hint {
		font-size: var(--text-xs);
		line-height: 1.4;
	}

	.error {
		color: var(--color-danger);
	}

	.hint {
		color: var(--color-text-tertiary);
	}
</style>

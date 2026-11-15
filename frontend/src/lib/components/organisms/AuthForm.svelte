<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { Credentials } from '$lib/api';
	import { validateConfirm, validatePassword, validateUsername } from '$lib/validation';
	import Button from '../atoms/Button.svelte';
	import FormField from '../molecules/FormField.svelte';
	import Alert from '../molecules/Alert.svelte';

	/**
	 * Username/password form shared by first-run setup and login.
	 * - `setup` mode: confirm field, full client-side validation (mirrors the API contract),
	 *   `new-password` autocomplete.
	 * - `login` mode: only checks that both fields are filled, `current-password` autocomplete.
	 * The page owns the API call; this component owns field state and the submitting flag.
	 */
	interface Props {
		mode: 'setup' | 'login';
		submitLabel: string;
		/** Form-level error from the server (shown above the submit button). */
		error?: string | null;
		/** Informational message shown above the fields (e.g. after a redirect). */
		notice?: string | null;
		onsubmit: (credentials: Credentials) => Promise<void>;
		footer?: Snippet;
	}

	let { mode, submitLabel, error = null, notice = null, onsubmit, footer }: Props = $props();

	let username = $state('');
	let password = $state('');
	let confirm = $state('');
	let submitting = $state(false);
	/** Field errors are only shown after the first submit attempt (no nagging while typing). */
	let attempted = $state(false);

	let errors = $derived.by(() => {
		if (mode === 'login') {
			return {
				username: username.trim() ? null : 'Enter your username.',
				password: password ? null : 'Enter your password.',
				confirm: null
			};
		}
		return {
			username: validateUsername(username.trim()),
			password: validatePassword(password),
			confirm: validateConfirm(password, confirm)
		};
	});

	let valid = $derived(!errors.username && !errors.password && !errors.confirm);

	let form: HTMLFormElement;

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		if (submitting) return;
		attempted = true;
		if (!valid) {
			// Move focus to the first invalid field.
			form.querySelector<HTMLInputElement>('[aria-invalid="true"]')?.focus();
			return;
		}
		submitting = true;
		try {
			await onsubmit({ username: username.trim(), password });
		} finally {
			submitting = false;
		}
	}
</script>

<form class="form" bind:this={form} onsubmit={handleSubmit} novalidate>
	{#if notice}
		<Alert kind="info">{notice}</Alert>
	{/if}

	<FormField
		label="Username"
		name="username"
		bind:value={username}
		error={attempted ? errors.username : null}
		hint={mode === 'setup' ? '3–32 characters: letters, numbers, . _ -' : undefined}
		autocomplete="username"
		inputmode="text"
		enterkeyhint="next"
		maxlength={mode === 'setup' ? 32 : undefined}
		required
		disabled={submitting}
	/>

	<FormField
		label="Password"
		name="password"
		type="password"
		bind:value={password}
		error={attempted ? errors.password : null}
		hint={mode === 'setup' ? 'At least 8 characters.' : undefined}
		autocomplete={mode === 'setup' ? 'new-password' : 'current-password'}
		enterkeyhint={mode === 'setup' ? 'next' : 'go'}
		required
		disabled={submitting}
	/>

	{#if mode === 'setup'}
		<FormField
			label="Confirm password"
			name="confirm"
			type="password"
			bind:value={confirm}
			error={attempted ? errors.confirm : null}
			autocomplete="new-password"
			enterkeyhint="go"
			required
			disabled={submitting}
		/>
	{/if}

	{#if error}
		<Alert kind="error">{error}</Alert>
	{/if}

	<Button type="submit" size="lg" block loading={submitting}>{submitLabel}</Button>

	{#if footer}
		<div class="footer">{@render footer()}</div>
	{/if}
</form>

<style>
	.form {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.form > :global(.btn) {
		margin-top: var(--space-2);
	}

	.footer {
		margin-top: var(--space-2);
		text-align: center;
		font-size: var(--text-sm);
		color: var(--color-text-tertiary);
	}
</style>

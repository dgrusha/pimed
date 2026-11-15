<script lang="ts">
	import { isApiError, type CreateUserInput, type Role } from '$lib/api';
	import { validatePassword, validateUsername } from '$lib/validation';
	import Button from '../atoms/Button.svelte';
	import Icon from '../atoms/Icon.svelte';
	import Select from '../atoms/Select.svelte';
	import FormField from '../molecules/FormField.svelte';
	import Alert from '../molecules/Alert.svelte';

	/**
	 * Admin form for creating an account. The page performs the request and resolves
	 * `true` when the user was created (the form then resets) or `false` when it
	 * handled the failure itself (e.g. 401/403; the form is left untouched). Any
	 * thrown `ApiError` is mapped here (409 → inline "username taken", others →
	 * form alert).
	 */
	interface Props {
		onsubmit: (input: CreateUserInput) => Promise<boolean>;
	}

	let { onsubmit }: Props = $props();

	const roleOptions = [
		{ value: 'user', label: 'User' },
		{ value: 'admin', label: 'Admin' }
	];

	let username = $state('');
	let password = $state('');
	let role = $state<string>('user');
	let submitting = $state(false);
	let attempted = $state(false);
	let takenUsername = $state<string | null>(null);
	let formError = $state<string | null>(null);

	const roleId = $props.id();

	let usernameError = $derived(
		takenUsername !== null && takenUsername === username.trim().toLowerCase()
			? 'That username is already taken.'
			: attempted
				? validateUsername(username.trim())
				: null
	);
	let passwordError = $derived(attempted ? validatePassword(password) : null);

	let form: HTMLFormElement;

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		if (submitting) return;
		attempted = true;
		formError = null;
		if (validateUsername(username.trim()) || validatePassword(password)) {
			form.querySelector<HTMLInputElement>('[aria-invalid="true"]')?.focus();
			return;
		}

		submitting = true;
		try {
			const created = await onsubmit({ username: username.trim(), password, role: role as Role });
			if (!created) return;
			username = '';
			password = '';
			role = 'user';
			attempted = false;
			takenUsername = null;
		} catch (err) {
			if (isApiError(err, 409)) {
				takenUsername = username.trim().toLowerCase();
				form.querySelector<HTMLInputElement>('input[name="new-username"]')?.focus();
			} else {
				formError = err instanceof Error ? err.message : 'Could not create the user.';
			}
		} finally {
			submitting = false;
		}
	}
</script>

<form class="form" bind:this={form} onsubmit={handleSubmit} novalidate>
	<FormField
		label="Username"
		name="new-username"
		bind:value={username}
		error={usernameError}
		hint="3–32 characters: letters, numbers, . _ -"
		autocomplete="off"
		maxlength={32}
		required
		disabled={submitting}
	/>

	<FormField
		label="Password"
		name="new-password"
		type="password"
		bind:value={password}
		error={passwordError}
		hint="At least 8 characters."
		autocomplete="new-password"
		required
		disabled={submitting}
	/>

	<div class="field">
		<label class="label" for="role-{roleId}">Role</label>
		<Select
			id="role-{roleId}"
			name="role"
			bind:value={role}
			options={roleOptions}
			disabled={submitting}
		/>
	</div>

	{#if formError}
		<Alert kind="error">{formError}</Alert>
	{/if}

	<Button type="submit" block loading={submitting}>
		<Icon name="plus" size={18} />
		Create user
	</Button>
</form>

<style>
	.form {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		margin-bottom: var(--space-3);
	}

	.label {
		font-size: var(--text-sm);
		font-weight: var(--weight-medium);
		color: var(--color-text-secondary);
	}
</style>

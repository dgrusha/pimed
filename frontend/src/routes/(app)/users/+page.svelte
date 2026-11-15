<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import * as api from '$lib/api';
	import type { CreateUserInput, User } from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';
	import { toasts } from '$lib/stores/toast.svelte';
	import Button from '$lib/components/atoms/Button.svelte';
	import Heading from '$lib/components/atoms/Heading.svelte';
	import Icon from '$lib/components/atoms/Icon.svelte';
	import Spinner from '$lib/components/atoms/Spinner.svelte';
	import Text from '$lib/components/atoms/Text.svelte';
	import Alert from '$lib/components/molecules/Alert.svelte';
	import Card from '$lib/components/molecules/Card.svelte';
	import CreateUserForm from '$lib/components/organisms/CreateUserForm.svelte';
	import UserTable from '$lib/components/organisms/UserTable.svelte';

	let users = $state<User[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let highlightId = $state<number | null>(null);

	/**
	 * Shared handling for auth failures on admin endpoints. Returns true if handled.
	 * 401 is already handled centrally (the API client expires the session).
	 */
	function handleAuthError(err: unknown): boolean {
		if (api.isApiError(err, 401)) return true;
		if (api.isApiError(err, 403)) {
			toasts.error('You need administrator rights for that.');
			goto('/', { replaceState: true });
			return true;
		}
		return false;
	}

	async function refresh() {
		loadError = null;
		try {
			users = await api.listUsers();
		} catch (err) {
			if (!handleAuthError(err)) {
				loadError = err instanceof Error ? err.message : 'Could not load users.';
			}
		} finally {
			loading = false;
		}
	}

	/** Resolves true once the user exists, false if the failure was handled here. */
	async function handleCreate(input: CreateUserInput): Promise<boolean> {
		try {
			const user = await api.createUser(input);
			toasts.success(`User “${user.username}” created.`);
			highlightId = user.id;
			setTimeout(() => {
				if (highlightId === user.id) highlightId = null;
			}, 2500);
			await refresh();
			return true;
		} catch (err) {
			if (handleAuthError(err)) return false; // we are navigating away; keep the form as is
			throw err; // 400 / 409 etc. are shown inline by the form
		}
	}

	onMount(refresh);
</script>

<svelte:head>
	<title>Users · PiMed</title>
</svelte:head>

<div class="page">
	<header class="header">
		<a class="back" href="/">
			<Icon name="chevron-right" size={16} class="flip" />
			Home
		</a>
		<Heading level={1} size="xl">Users</Heading>
		<Text tone="secondary">Everyone who can sign in to this PiMed.</Text>
	</header>

	<div class="grid">
		<Card as="section" padding="md" aria-labelledby="list-heading">
			<div class="card-head">
				<Heading level={2} size="md" id="list-heading">Accounts</Heading>
				{#if !loading}
					<Text as="span" size="sm" tone="tertiary">{users.length} total</Text>
				{/if}
			</div>

			{#if loading}
				<div class="state"><Spinner size={22} label="Loading users" /></div>
			{:else if loadError}
				<div class="error">
					<Alert kind="error">{loadError}</Alert>
					<Button variant="secondary" size="sm" onclick={refresh}>
						<Icon name="refresh" size={16} />
						Retry
					</Button>
				</div>
			{:else}
				<UserTable {users} currentUserId={auth.user?.id} {highlightId} />
			{/if}
		</Card>

		<Card as="section" padding="md" aria-labelledby="create-heading">
			<div class="card-head">
				<Heading level={2} size="md" id="create-heading">New user</Heading>
			</div>
			<CreateUserForm onsubmit={handleCreate} />
		</Card>
	</div>
</div>

<style>
	.page {
		display: flex;
		flex-direction: column;
		gap: var(--space-6);
		padding: 0 var(--gutter);
	}

	.header {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.back {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		align-self: flex-start;
		margin-left: calc(var(--space-1) * -1);
		padding: var(--space-1);
		border-radius: var(--radius-sm);
		font-size: var(--text-sm);
		font-weight: var(--weight-medium);
	}

	.back :global(.flip) {
		transform: scaleX(-1);
	}

	.grid {
		display: grid;
		gap: var(--space-5);
		align-items: start;
	}

	@media (min-width: 900px) {
		.grid {
			grid-template-columns: minmax(0, 1.6fr) minmax(320px, 1fr);
		}
	}

	.card-head {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: var(--space-3);
		margin-bottom: var(--space-4);
	}

	.state {
		display: grid;
		place-items: center;
		min-height: 160px;
		color: var(--color-text-secondary);
	}

	.error {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: var(--space-3);
	}
</style>

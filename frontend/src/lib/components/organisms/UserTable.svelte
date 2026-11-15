<script lang="ts">
	import type { User } from '$lib/api';
	import Avatar from '../atoms/Avatar.svelte';
	import Badge from '../atoms/Badge.svelte';

	interface Props {
		users: User[];
		/** Id of the signed-in user, marked "you". */
		currentUserId?: number;
		/** Id of a row to highlight briefly (e.g. just created). */
		highlightId?: number | null;
	}

	let { users, currentUserId, highlightId = null }: Props = $props();

	const dateFormat = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' });

	function formatDate(iso: string): string {
		const d = new Date(iso);
		return Number.isNaN(d.getTime()) ? iso : dateFormat.format(d);
	}
</script>

<div class="wrap">
	<table class="table">
		<caption class="visually-hidden">Users</caption>
		<thead>
			<tr>
				<th scope="col">User</th>
				<th scope="col">Role</th>
				<th scope="col" class="created">Created</th>
			</tr>
		</thead>
		<tbody>
			{#each users as user (user.id)}
				<tr class:highlight={user.id === highlightId}>
					<td>
						<div class="user">
							<Avatar name={user.username} size={32} />
							<span class="name">{user.username}</span>
							{#if user.id === currentUserId}<span class="you">you</span>{/if}
						</div>
					</td>
					<td>
						<Badge tone={user.role === 'admin' ? 'accent' : 'neutral'}>
							{user.role === 'admin' ? 'Admin' : 'User'}
						</Badge>
					</td>
					<td class="created">
						<time datetime={user.createdAt}>{formatDate(user.createdAt)}</time>
					</td>
				</tr>
			{:else}
				<tr>
					<td colspan="3" class="empty">No users yet.</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>

<style>
	.wrap {
		overflow-x: auto;
		-webkit-overflow-scrolling: touch;
	}

	.table {
		width: 100%;
		border-collapse: collapse;
		font-size: var(--text-sm);
	}

	th {
		padding: var(--space-2) var(--space-3);
		text-align: left;
		font-size: var(--text-xs);
		font-weight: var(--weight-semibold);
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--color-text-tertiary);
		border-bottom: 1px solid var(--color-border);
	}

	td {
		padding: var(--space-3);
		border-bottom: 1px solid rgba(255, 255, 255, 0.05);
		vertical-align: middle;
	}

	tbody tr {
		transition: background-color var(--duration-slow) var(--ease-out);
	}

	tbody tr:hover {
		background: rgba(255, 255, 255, 0.03);
	}

	tbody tr:last-child td {
		border-bottom: 0;
	}

	tr.highlight {
		background: var(--color-accent-soft);
	}

	.user {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		min-width: 0;
	}

	.name {
		font-weight: var(--weight-semibold);
		word-break: break-all;
	}

	.you {
		font-size: var(--text-xs);
		color: var(--color-text-tertiary);
	}

	.created {
		white-space: nowrap;
		color: var(--color-text-secondary);
	}

	.empty {
		padding: var(--space-8) var(--space-3);
		text-align: center;
		color: var(--color-text-tertiary);
	}

	@media (max-width: 420px) {
		.created {
			display: none;
		}
	}
</style>

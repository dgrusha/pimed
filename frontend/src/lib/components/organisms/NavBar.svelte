<script lang="ts">
	import type { User } from '$lib/api';
	import Avatar from '../atoms/Avatar.svelte';
	import Badge from '../atoms/Badge.svelte';
	import Button from '../atoms/Button.svelte';
	import Icon from '../atoms/Icon.svelte';

	interface Props {
		user: User;
		onlogout: () => Promise<void>;
	}

	let { user, onlogout }: Props = $props();

	let loggingOut = $state(false);

	async function handleLogout() {
		loggingOut = true;
		try {
			await onlogout();
		} finally {
			loggingOut = false;
		}
	}
</script>

<header class="navbar">
	<div class="inner">
		<a class="brand" href="/" aria-label="PiMed home">
			<span class="mark" aria-hidden="true">Pi</span>
			<span class="wordmark">PiMed</span>
		</a>

		<div class="right">
			<div class="who">
				<Avatar name={user.username} size={34} />
				<div class="identity">
					<span class="name">{user.username}</span>
					<Badge tone={user.role === 'admin' ? 'accent' : 'neutral'}>
						{user.role === 'admin' ? 'Admin' : 'User'}
					</Badge>
				</div>
			</div>

			<Button variant="secondary" size="sm" loading={loggingOut} onclick={handleLogout}>
				<Icon name="logout" size={16} />
				<span class="logout-label">Log out</span>
			</Button>
		</div>
	</div>
</header>

<style>
	.navbar {
		position: sticky;
		top: 0;
		z-index: var(--z-navbar);
		padding-top: var(--safe-top);
		background: rgba(7, 8, 12, 0.6);
		border-bottom: 1px solid var(--color-border);
		-webkit-backdrop-filter: blur(var(--blur-md)) saturate(var(--glass-saturate));
		backdrop-filter: blur(var(--blur-md)) saturate(var(--glass-saturate));
	}

	.inner {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-4);
		max-width: var(--content-max);
		height: var(--navbar-height);
		margin: 0 auto;
		padding: 0 max(var(--space-4), var(--safe-right)) 0 max(var(--space-4), var(--safe-left));
	}

	.brand {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-1);
		border-radius: var(--radius-sm);
		color: var(--color-text);
		text-decoration: none;
	}

	.mark {
		display: grid;
		place-items: center;
		width: 32px;
		height: 32px;
		border-radius: 9px;
		font-size: 0.875rem;
		font-weight: var(--weight-bold);
		color: #fff;
		background: linear-gradient(135deg, var(--color-accent), var(--color-accent-2));
		box-shadow: 0 4px 14px rgba(79, 140, 255, 0.35);
	}

	.wordmark {
		font-size: var(--text-lg);
		font-weight: var(--weight-bold);
		letter-spacing: -0.03em;
	}

	.right {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		min-width: 0;
	}

	.who {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		min-width: 0;
	}

	.identity {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		min-width: 0;
		line-height: 1.2;
	}

	.name {
		max-width: 16ch;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-size: var(--text-sm);
		font-weight: var(--weight-semibold);
	}

	@media (max-width: 480px) {
		.identity {
			display: none;
		}
		.logout-label {
			position: absolute;
			width: 1px;
			height: 1px;
			overflow: hidden;
			clip: rect(0 0 0 0);
			white-space: nowrap;
		}
	}
</style>

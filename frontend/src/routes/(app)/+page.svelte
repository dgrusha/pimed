<script lang="ts">
	import { auth } from '$lib/stores/auth.svelte';
	import { toasts } from '$lib/stores/toast.svelte';
	import type { IconName } from '$lib/components/atoms/Icon.svelte';
	import Heading from '$lib/components/atoms/Heading.svelte';
	import Text from '$lib/components/atoms/Text.svelte';
	import Tile from '$lib/components/molecules/Tile.svelte';
	import CardRow from '$lib/components/organisms/CardRow.svelte';

	interface TileDef {
		title: string;
		description: string;
		icon: IconName;
		hue: number;
		href?: string;
	}

	const comingSoon: TileDef[] = [
		{ title: 'Files', description: 'Browse, upload & download', icon: 'files', hue: 215 },
		{ title: 'Videos', description: 'Your movies and clips', icon: 'video', hue: 345 },
		{ title: 'Music', description: 'Albums and playlists', icon: 'music', hue: 285 },
		{ title: 'Devices', description: 'Connected laptops & phones', icon: 'devices', hue: 165 },
		{ title: 'Console', description: 'Terminal into the Pi', icon: 'terminal', hue: 30 }
	];

	let tiles = $derived<TileDef[]>(
		auth.isAdmin
			? [
					...comingSoon,
					{
						title: 'Users',
						description: 'Manage accounts',
						icon: 'users',
						hue: 200,
						href: '/users'
					}
				]
			: comingSoon
	);

	function greeting(): string {
		const h = new Date().getHours();
		if (h < 5) return 'Good night';
		if (h < 12) return 'Good morning';
		if (h < 18) return 'Good afternoon';
		return 'Good evening';
	}
</script>

<svelte:head>
	<title>Home · PiMed</title>
</svelte:head>

<div class="home">
	<header class="hero">
		<Text size="sm" tone="tertiary" weight="semibold" class="eyebrow">{greeting()}</Text>
		<Heading level={1} size="xxl">Hi, {auth.user?.username}</Heading>
		<Text tone="secondary">Your private media, right here on your Pi.</Text>
	</header>

	<CardRow label="Library">
		{#each tiles as tile (tile.title)}
			<li>
				{#if tile.href}
					<Tile
						title={tile.title}
						description={tile.description}
						icon={tile.icon}
						hue={tile.hue}
						href={tile.href}
					/>
				{:else}
					<Tile
						title={tile.title}
						description={tile.description}
						icon={tile.icon}
						hue={tile.hue}
						badge="Coming soon"
						onclick={() => toasts.show(`${tile.title} is coming soon.`)}
					/>
				{/if}
			</li>
		{/each}
	</CardRow>
</div>

<style>
	.home {
		display: flex;
		flex-direction: column;
		gap: var(--space-6);
	}

	.hero {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		padding: 0 var(--gutter);
		animation: rise var(--duration-slow) var(--ease-out) both;
	}

	.hero :global(.eyebrow) {
		letter-spacing: 0.06em;
		text-transform: uppercase;
	}

	@keyframes rise {
		from {
			opacity: 0;
			transform: translateY(10px);
		}
	}
</style>

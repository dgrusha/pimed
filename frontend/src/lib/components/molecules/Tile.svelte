<script lang="ts">
	import Icon, { type IconName } from '../atoms/Icon.svelte';
	import Badge from '../atoms/Badge.svelte';

	/**
	 * Large PS4-style dashboard card. Renders a link when `href` is set, otherwise a
	 * button (e.g. a "Coming soon" tile). Focus/hover scale + glow.
	 */
	interface Props {
		title: string;
		description?: string;
		icon: IconName;
		href?: string;
		onclick?: () => void;
		/** Small badge in the corner, e.g. "Coming soon". */
		badge?: string;
		/** Hue for the tile's gradient artwork. */
		hue?: number;
	}

	let { title, description, icon, href, onclick, badge, hue = 220 }: Props = $props();
</script>

{#snippet inner()}
	<span class="art" aria-hidden="true">
		<Icon name={icon} size={44} strokeWidth={1.5} />
	</span>
	<span class="meta">
		<span class="title">{title}</span>
		{#if description}<span class="description">{description}</span>{/if}
	</span>
	{#if badge}<span class="badge"><Badge>{badge}</Badge></span>{/if}
{/snippet}

{#if href}
	<a class="tile" {href} style:--hue={hue} data-tile>
		{@render inner()}
	</a>
{:else}
	<button type="button" class="tile inert" style:--hue={hue} {onclick} data-tile>
		{@render inner()}
	</button>
{/if}

<style>
	.tile {
		position: relative;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		gap: var(--space-4);
		width: 100%;
		aspect-ratio: 4 / 5;
		padding: var(--space-5);
		border-radius: var(--radius-xl);
		color: var(--color-text);
		font: inherit;
		text-align: left;
		text-decoration: none;
		overflow: hidden;
		isolation: isolate;
		background:
			radial-gradient(120% 90% at 0% 0%, hsl(var(--hue) 80% 60% / 0.35), transparent 60%),
			linear-gradient(
				160deg,
				hsl(var(--hue) 40% 18% / 0.9),
				hsl(calc(var(--hue) + 30) 35% 10% / 0.9)
			);
		border: 1px solid var(--color-border);
		box-shadow: var(--shadow-md);
		transform: translateZ(0) scale(1);
		transition:
			transform var(--duration-slow) var(--ease-spring),
			box-shadow var(--duration-normal) var(--ease-out),
			border-color var(--duration-normal) var(--ease-out),
			filter var(--duration-normal) var(--ease-out);
		cursor: pointer;
		-webkit-user-select: none;
		user-select: none;
		touch-action: manipulation;
	}

	.tile:hover {
		text-decoration: none;
	}

	.tile.inert {
		align-items: stretch;
	}

	.tile.inert .art,
	.tile.inert .meta {
		opacity: 0.7;
	}

	@media (hover: hover) {
		.tile:hover {
			transform: scale(1.04);
			box-shadow: var(--shadow-lg);
			border-color: var(--color-border-strong);
		}
	}

	.tile:focus-visible {
		transform: scale(1.07);
		border-color: rgba(255, 255, 255, 0.5);
		box-shadow:
			0 0 0 2px rgba(255, 255, 255, 0.85),
			0 24px 60px hsl(var(--hue) 80% 50% / 0.35);
		z-index: 1;
	}

	.tile:active {
		transform: scale(0.98);
	}

	.art {
		display: grid;
		place-items: center;
		width: 72px;
		height: 72px;
		border-radius: var(--radius-lg);
		background: rgba(255, 255, 255, 0.1);
		box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.12);
		color: #fff;
	}

	.meta {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}

	.title {
		font-size: var(--text-xl);
		font-weight: var(--weight-bold);
		letter-spacing: var(--tracking-tight);
		line-height: var(--leading-tight);
	}

	.description {
		font-size: var(--text-sm);
		color: var(--color-text-secondary);
	}

	.badge {
		position: absolute;
		top: var(--space-4);
		right: var(--space-4);
	}
</style>

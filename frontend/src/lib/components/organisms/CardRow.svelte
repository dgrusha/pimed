<script lang="ts">
	import type { Snippet } from 'svelte';

	/**
	 * PS4-style horizontal row of tiles.
	 * - Left/Right (and Home/End) move focus between tiles; the focused tile scrolls into view.
	 * - Horizontally scrollable with scroll-snap on touch devices.
	 * Children should render elements carrying the `data-tile` attribute (see Tile).
	 */
	interface Props {
		label: string;
		children: Snippet;
	}

	let { label, children }: Props = $props();

	const headingId = $props.id();
	let row: HTMLElement;

	function tiles(): HTMLElement[] {
		return Array.from(row.querySelectorAll<HTMLElement>('[data-tile]'));
	}

	function handleKeydown(event: KeyboardEvent) {
		const items = tiles();
		const current = items.indexOf(document.activeElement as HTMLElement);
		if (current === -1) return;

		let next = current;
		switch (event.key) {
			case 'ArrowRight':
				next = Math.min(items.length - 1, current + 1);
				break;
			case 'ArrowLeft':
				next = Math.max(0, current - 1);
				break;
			case 'Home':
				next = 0;
				break;
			case 'End':
				next = items.length - 1;
				break;
			default:
				return;
		}
		event.preventDefault();
		if (next === current) return;

		const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
		items[next].focus({ preventScroll: true });
		items[next].scrollIntoView({
			behavior: reduce ? 'auto' : 'smooth',
			block: 'nearest',
			inline: 'center'
		});
	}
</script>

<section class="section" aria-labelledby="row-{headingId}">
	<h2 id="row-{headingId}" class="label">{label}</h2>
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<ul class="row" bind:this={row} onkeydown={handleKeydown}>
		{@render children()}
	</ul>
</section>

<style>
	.section {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.label {
		padding: 0 var(--gutter, var(--space-4));
		font-size: var(--text-sm);
		font-weight: var(--weight-semibold);
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--color-text-tertiary);
	}

	.row {
		display: grid;
		grid-auto-flow: column;
		grid-auto-columns: clamp(180px, 44vw, 240px);
		gap: var(--space-4);
		margin: 0;
		/* Vertical padding leaves room for the focus scale + glow. */
		padding: var(--space-5) var(--gutter, var(--space-4)) var(--space-8);
		list-style: none;
		overflow-x: auto;
		overscroll-behavior-x: contain;
		scroll-snap-type: x mandatory;
		scroll-padding-inline: var(--gutter, var(--space-4));
		scrollbar-width: none;
		-webkit-overflow-scrolling: touch;
	}

	.row::-webkit-scrollbar {
		display: none;
	}

	.row > :global(li) {
		scroll-snap-align: start;
	}
</style>

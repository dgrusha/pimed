<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { User } from '$lib/api';
	import NavBar from '../organisms/NavBar.svelte';

	/** Sticky translucent navbar over a subtle ambient backdrop, with page content below. */
	interface Props {
		user: User;
		onlogout: () => Promise<void>;
		children: Snippet;
	}

	let { user, onlogout, children }: Props = $props();
</script>

<div class="shell">
	<div class="ambient" aria-hidden="true"></div>
	<NavBar {user} {onlogout} />
	<main class="content" id="main">
		{@render children()}
	</main>
</div>

<style>
	.shell {
		position: relative;
		min-height: 100vh;
		min-height: 100dvh;
		isolation: isolate;
	}

	.ambient {
		position: fixed;
		inset: 0;
		z-index: -1;
		background:
			radial-gradient(70% 50% at 15% 0%, rgba(47, 91, 255, 0.22), transparent 70%),
			radial-gradient(60% 50% at 100% 20%, rgba(122, 60, 255, 0.16), transparent 70%),
			radial-gradient(80% 60% at 50% 110%, rgba(0, 179, 255, 0.1), transparent 70%), var(--color-bg);
	}

	.content {
		/* Pages read --gutter so full-bleed rows can align with the rest of the content. */
		--gutter: max(var(--space-4), var(--safe-left), calc((100vw - var(--content-max)) / 2));
		padding: var(--space-8) 0 calc(var(--safe-bottom) + var(--space-12));
	}

	@media (min-width: 768px) {
		.content {
			--gutter: max(var(--space-8), calc((100vw - var(--content-max)) / 2));
			padding-top: var(--space-12);
		}
	}
</style>

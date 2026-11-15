<script lang="ts">
	import type { Snippet } from 'svelte';
	import Card from '../molecules/Card.svelte';
	import Heading from '../atoms/Heading.svelte';
	import Text from '../atoms/Text.svelte';

	/** Centered glass card floating over an ambient gradient. */
	interface Props {
		title: string;
		subtitle?: string;
		children: Snippet;
	}

	let { title, subtitle, children }: Props = $props();
</script>

<div class="page">
	<div class="ambient" aria-hidden="true">
		<span class="orb one"></span>
		<span class="orb two"></span>
		<span class="orb three"></span>
	</div>

	<main class="main">
		<div class="brand" aria-hidden="true">
			<span class="mark">Pi</span>
			<span class="wordmark">PiMed</span>
		</div>

		<Card strong padding="lg" class="auth-card">
			<header class="header">
				<Heading level={1} size="xl">{title}</Heading>
				{#if subtitle}<Text tone="secondary">{subtitle}</Text>{/if}
			</header>
			{@render children()}
		</Card>
	</main>
</div>

<style>
	.page {
		position: relative;
		min-height: 100vh;
		min-height: 100dvh;
		display: grid;
		place-items: center;
		padding: calc(var(--safe-top) + var(--space-6)) max(var(--space-4), var(--safe-right))
			calc(var(--safe-bottom) + var(--space-6)) max(var(--space-4), var(--safe-left));
		overflow: hidden;
		isolation: isolate;
	}

	.ambient {
		position: fixed;
		inset: 0;
		z-index: -1;
		background:
			radial-gradient(80% 60% at 50% 0%, rgba(79, 140, 255, 0.12), transparent 70%), var(--color-bg);
		overflow: hidden;
	}

	.orb {
		position: absolute;
		width: 60vmax;
		height: 60vmax;
		border-radius: 50%;
		filter: blur(80px);
		opacity: 0.45;
		will-change: transform;
	}

	.one {
		top: -25vmax;
		left: -20vmax;
		background: radial-gradient(circle, #2f5bff, transparent 65%);
		animation: drift-a 22s var(--ease-standard) infinite alternate;
	}

	.two {
		bottom: -30vmax;
		right: -20vmax;
		background: radial-gradient(circle, #7a3cff, transparent 65%);
		animation: drift-b 26s var(--ease-standard) infinite alternate;
	}

	.three {
		top: 30%;
		left: 40%;
		width: 40vmax;
		height: 40vmax;
		opacity: 0.25;
		background: radial-gradient(circle, #00b3ff, transparent 65%);
		animation: drift-a 30s var(--ease-standard) infinite alternate-reverse;
	}

	@keyframes drift-a {
		to {
			transform: translate3d(8vmax, 6vmax, 0) scale(1.1);
		}
	}

	@keyframes drift-b {
		to {
			transform: translate3d(-10vmax, -5vmax, 0) scale(0.95);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.orb {
			animation: none !important;
		}
	}

	.main {
		width: 100%;
		max-width: 440px;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-6);
		animation: rise var(--duration-slow) var(--ease-out) both;
	}

	@keyframes rise {
		from {
			opacity: 0;
			transform: translateY(12px) scale(0.98);
		}
	}

	.brand {
		display: flex;
		align-items: center;
		gap: var(--space-3);
	}

	.mark {
		display: grid;
		place-items: center;
		width: 44px;
		height: 44px;
		border-radius: 13px;
		font-size: 1.125rem;
		font-weight: var(--weight-bold);
		color: #fff;
		background: linear-gradient(135deg, var(--color-accent), var(--color-accent-2));
		box-shadow: 0 10px 30px rgba(79, 140, 255, 0.4);
	}

	.wordmark {
		font-size: var(--text-xl);
		font-weight: var(--weight-bold);
		letter-spacing: -0.03em;
	}

	.main :global(.auth-card) {
		width: 100%;
	}

	.header {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		margin-bottom: var(--space-6);
	}
</style>

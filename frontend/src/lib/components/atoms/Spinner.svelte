<script lang="ts">
	interface Props {
		size?: number;
		/** Accessible label; omit when the spinner is purely decorative (e.g. inside a busy button). */
		label?: string;
	}

	let { size = 18, label }: Props = $props();
</script>

<span
	class="spinner"
	style:--size="{size}px"
	role={label ? 'status' : undefined}
	aria-label={label}
	aria-hidden={label ? undefined : 'true'}
></span>

<style>
	.spinner {
		display: inline-block;
		flex: none;
		width: var(--size);
		height: var(--size);
		border-radius: 50%;
		border: 2px solid currentColor;
		border-right-color: transparent;
		opacity: 0.85;
		animation: spin 0.7s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.spinner {
			/* Keep a slow, gentle rotation so loading is still perceivable. */
			animation: spin 1.6s linear infinite !important;
		}
	}
</style>

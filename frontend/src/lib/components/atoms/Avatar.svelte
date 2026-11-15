<script lang="ts">
	interface Props {
		name: string;
		size?: number;
	}

	let { name, size = 36 }: Props = $props();

	let initials = $derived(
		name
			.split(/[._\-\s]+/)
			.filter(Boolean)
			.slice(0, 2)
			.map((part) => part[0])
			.join('')
			.toUpperCase() || '?'
	);

	// Stable hue per name so each user keeps "their" color.
	let hue = $derived([...name].reduce((h, ch) => (h * 31 + ch.charCodeAt(0)) % 360, 7));
</script>

<span class="avatar" style:--size="{size}px" style:--hue={hue} aria-hidden="true">
	{initials}
</span>

<style>
	.avatar {
		display: inline-grid;
		place-items: center;
		flex: none;
		width: var(--size);
		height: var(--size);
		border-radius: 50%;
		font-size: calc(var(--size) * 0.4);
		font-weight: var(--weight-semibold);
		letter-spacing: 0.02em;
		color: #fff;
		background: linear-gradient(
			135deg,
			hsl(var(--hue) 75% 58%),
			hsl(calc(var(--hue) + 40) 70% 45%)
		);
		box-shadow:
			inset 0 0 0 1px rgba(255, 255, 255, 0.18),
			var(--shadow-sm);
		user-select: none;
	}
</style>

<script lang="ts" module>
	// Inline SVG paths (24x24, stroke based, SF Symbols-ish weight).
	const PATHS = {
		files: 'M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z',
		video:
			'M4 6h11a1 1 0 0 1 1 1v10a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1zm12 4.5 5-3v9l-5-3',
		music: 'M9 18V5l11-2v13M9 18a3 3 0 1 1-6 0 3 3 0 0 1 6 0zm11-2a3 3 0 1 1-6 0 3 3 0 0 1 6 0z',
		devices:
			'M4 5h12a1 1 0 0 1 1 1v8H3V6a1 1 0 0 1 1-1zM1 17h16M19 9h3a1 1 0 0 1 1 1v9a1 1 0 0 1-1 1h-3a1 1 0 0 1-1-1v-9a1 1 0 0 1 1-1z',
		terminal:
			'M4 4h16a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1zm3 5 3 3-3 3m5 0h5',
		users:
			'M16 19v-1a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v1M9 10a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7zm13 9v-1a4 4 0 0 0-3-3.87M16 3.13a3.5 3.5 0 0 1 0 6.75',
		user: 'M20 21v-1.5a4.5 4.5 0 0 0-4.5-4.5h-7A4.5 4.5 0 0 0 4 19.5V21M12 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8z',
		logout: 'M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4m7 14 5-5-5-5m5 5H9',
		lock: 'M6 11h12a1 1 0 0 1 1 1v8a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1v-8a1 1 0 0 1 1-1zm2 0V7a4 4 0 1 1 8 0v4',
		plus: 'M12 5v14M5 12h14',
		check: 'M5 12.5 10 17 19 7',
		x: 'M6 6l12 12M18 6 6 18',
		info: 'M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20zm0-6v-5m0-3h.01',
		alert: 'M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20zm0-14v5m0 3h.01',
		'chevron-right': 'm9 6 6 6-6 6',
		refresh: 'M21 12a9 9 0 1 1-2.64-6.36M21 4v5h-5'
	} as const;

	export type IconName = keyof typeof PATHS;
</script>

<script lang="ts">
	interface Props {
		name: IconName;
		size?: number;
		strokeWidth?: number;
		/** Accessible label; when omitted the icon is decorative. */
		label?: string;
		class?: string;
	}

	let { name, size = 20, strokeWidth = 1.8, label, class: className }: Props = $props();
</script>

<svg
	class={className}
	width={size}
	height={size}
	viewBox="0 0 24 24"
	fill="none"
	stroke="currentColor"
	stroke-width={strokeWidth}
	stroke-linecap="round"
	stroke-linejoin="round"
	role={label ? 'img' : undefined}
	aria-label={label}
	aria-hidden={label ? undefined : 'true'}
	focusable="false"
>
	<path d={PATHS[name]} />
</svg>

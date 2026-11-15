/** Tiny toast queue (Svelte 5 runes). */

export type ToastKind = 'success' | 'error' | 'info';

export interface ToastItem {
	id: number;
	kind: ToastKind;
	message: string;
}

const DEFAULT_TIMEOUT = 3500;

class ToastState {
	items = $state<ToastItem[]>([]);
	#next = 1;

	show(message: string, kind: ToastKind = 'info', timeout = DEFAULT_TIMEOUT): number {
		const id = this.#next++;
		this.items.push({ id, kind, message });
		if (timeout > 0) setTimeout(() => this.dismiss(id), timeout);
		return id;
	}

	success(message: string): number {
		return this.show(message, 'success');
	}

	error(message: string): number {
		return this.show(message, 'error', 6000);
	}

	dismiss(id: number): void {
		this.items = this.items.filter((t) => t.id !== id);
	}
}

export const toasts = new ToastState();

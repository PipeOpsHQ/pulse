import { writable } from 'svelte/store';

export const mobileMenuOpen = writable(false);
export const sidebarCollapsed = writable(false);

export function toggleMobileMenu() {
  mobileMenuOpen.update(open => !open);
}

export function closeMobileMenu() {
  mobileMenuOpen.set(false);
}

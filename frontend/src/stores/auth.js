import { writable } from 'svelte/store';

const initialToken = typeof localStorage !== 'undefined' ? localStorage.getItem('token') : null;
const initialUser = typeof localStorage !== 'undefined' ? JSON.parse(localStorage.getItem('user') || 'null') : null;

export const token = writable(initialToken);
export const user = writable(initialUser);
export const isAuthenticated = writable(!!initialToken);

export function login(newToken, newUser) {
  token.set(newToken);
  user.set(newUser);
  isAuthenticated.set(true);

  if (typeof localStorage !== 'undefined') {
    localStorage.setItem('token', newToken);
    localStorage.setItem('user', JSON.stringify(newUser));
  }
}

export function logout() {
  token.set(null);
  user.set(null);
  isAuthenticated.set(false);

  if (typeof localStorage !== 'undefined') {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
  }
}

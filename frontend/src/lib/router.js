export function navigate(path, { replace = false } = {}) {
  if (typeof window === 'undefined') return;

  if (replace) {
    history.replaceState(null, '', path);
  } else {
    history.pushState(null, '', path);
  }

  // Trigger popstate so the App listener can update the path state
  window.dispatchEvent(new PopStateEvent('popstate'));
}

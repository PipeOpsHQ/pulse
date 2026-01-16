<script>
  import { onMount, onDestroy } from 'svelte';
  import { navigate } from 'svelte-routing';
  import { isAuthenticated } from '../stores/auth';

  let authenticated = false;
  let loading = true;

  function isPublicRoute(path) {
    return path === '/login' || path.startsWith('/status/');
  }

  const unsubscribe = isAuthenticated.subscribe(value => {
    authenticated = value;
    if (!loading && !value) {
      const currentPath = window.location.pathname;
      // Don't redirect if it's a public route
      if (!isPublicRoute(currentPath)) {
        navigate('/login', { replace: true });
      }
    }
  });

  onMount(() => {
    loading = false;
    const currentPath = window.location.pathname;
    // If it's a public route, don't check authentication
    if (isPublicRoute(currentPath)) {
      return;
    }
    if (!authenticated) {
      navigate('/login', { replace: true });
    }
  });

  onDestroy(() => {
    unsubscribe();
  });
</script>

{#if authenticated && !isPublicRoute(window.location.pathname)}
  <slot />
{/if}

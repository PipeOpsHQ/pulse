<script>
  import { onMount } from 'svelte';
  import { navigate } from 'svelte-routing';
  import { isAuthenticated } from '../stores/auth';

  let { children } = $props();
  let loading = $state(true);
  let authenticated = $state(false);

  function isPublicRoute(path) {
    return path === '/login' || path.startsWith('/status/');
  }

  onMount(() => {
    loading = false;
    const currentPath = window.location.pathname;

    // Subscribe to changes
    const unsubscribe = isAuthenticated.subscribe(value => {
      authenticated = value;
      if (!loading && !value && !isPublicRoute(window.location.pathname)) {
        navigate('/login', { replace: true });
      }
    });

    if (!isPublicRoute(currentPath) && !authenticated) {
      navigate('/login', { replace: true });
    }

    return () => unsubscribe();
  });
</script>

{#if authenticated}
  {@render children()}
{/if}

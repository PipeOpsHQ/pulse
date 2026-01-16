<script>
  import { onMount } from 'svelte';
  import { navigate } from 'svelte-routing';

  export let id;

  onMount(() => {
    const currentPath = window.location.pathname;
    // Don't redirect if it's a status page
    if (currentPath.startsWith('/status/')) {
      return;
    }
    // Check if it's a UUID (project id format)
    const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
    if (uuidRegex.test(id)) {
      navigate(`/projects/${id}`, { replace: true });
    } else {
      navigate('/', { replace: true });
    }
  });
</script>

<div class="flex h-96 items-center justify-center">
  <div class="h-10 w-10 animate-spin rounded-full border-2 border-pulse-500 border-t-transparent"></div>
</div>

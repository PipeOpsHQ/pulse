<script>
  import { navigate } from "../lib/router";
  let { to, class: className = "", children } = $props();

  function handleClick(e) {
    if (
      e.button === 0 &&
      !e.ctrlKey &&
      !e.metaKey &&
      !e.shiftKey &&
      !e.altKey
    ) {
      e.preventDefault();
      navigate(to);
    }
  }
</script>

<a href={to} class={className} onclick={handleClick}>
  {@render children()}
</a>

<script>
  import { toggleMobileMenu } from '../stores/ui';
  import { navigate } from 'svelte-routing';
  import { onMount } from 'svelte';
  import {
    Menu,
    Search,
    Bell,
    Settings,
    ChevronRight,
    Command
  } from 'lucide-svelte';

  export let breadcrumbs = [];
  export let showSearch = true;

  let searchInput;

  onMount(() => {
    const handleKeydown = (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        searchInput?.focus();
      }
    };
    window.addEventListener('keydown', handleKeydown);
    return () => window.removeEventListener('keydown', handleKeydown);
  });
</script>

<header class="sticky top-0 z-50 h-16 border-b border-white/[0.08] bg-black/40 backdrop-blur-2xl shadow-[0_1px_0_rgba(255,255,255,0.05)]">
  <div class="mx-auto flex h-full w-full max-w-7xl items-center justify-between px-6">
    <div class="flex items-center gap-4">
      <button
        class="flex h-10 w-10 items-center justify-center rounded-lg text-slate-400 hover:bg-white/5 hover:text-white lg:hidden"
        on:click={toggleMobileMenu}
        aria-label="Toggle Menu"
      >
        <Menu size={24} />
      </button>

      <nav class="flex flex-1 items-center gap-2 text-sm overflow-x-auto no-scrollbar whitespace-nowrap pr-4" aria-label="Breadcrumb">
        {#each breadcrumbs as crumb, i}
          {#if i < breadcrumbs.length - 1}
            <a href={crumb.path} class="text-slate-400 transition-colors hover:text-white">{crumb.label}</a>
            <ChevronRight size={14} class="text-slate-600" />
          {:else}
            <span class="font-medium text-white">{crumb.label}</span>
          {/if}
        {/each}
      </nav>
    </div>

    <div class="flex items-center gap-6">
      {#if showSearch}
        <div class="group relative hidden sm:flex items-center">
          <Search size={16} class="absolute left-3 text-slate-500 transition-colors group-focus-within:text-pulse-500" />
          <input
            bind:this={searchInput}
            type="text"
            placeholder="Search resources..."
            class="h-9 w-64 rounded-xl border border-white/[0.08] bg-white/[0.04] pl-10 pr-12 text-sm text-white placeholder-slate-500 outline-none transition-all duration-200 focus:border-pulse-500/50 focus:bg-white/[0.06] focus:ring-2 focus:ring-pulse-500/20 md:w-80"
          />
          <div class="absolute right-2 flex items-center gap-1 rounded border border-white/10 bg-black/40 px-1.5 py-0.5 text-[10px] font-medium text-slate-500">
            <Command size={10} />
            <span>K</span>
          </div>
        </div>
      {/if}

      <div class="flex items-center gap-2 border-l border-white/10 pl-6">
        <button
          class="flex h-9 w-9 items-center justify-center rounded-lg text-slate-400 transition-all hover:bg-white/5 hover:text-white"
          aria-label="Notifications"
          on:click={() => navigate('/settings')}
        >
          <Bell size={18} />
        </button>
        <button
          class="flex h-9 w-9 items-center justify-center rounded-lg text-slate-400 transition-all hover:bg-white/5 hover:text-white"
          aria-label="Settings"
          on:click={() => navigate('/settings')}
        >
          <Settings size={18} />
        </button>
      </div>
    </div>
  </div>
</header>


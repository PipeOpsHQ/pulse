<script>
  import Link from "./Link.svelte";
  import { user, logout } from "../stores/auth";
  import {
    mobileMenuOpen,
    sidebarCollapsed,
    closeMobileMenu,
  } from "../stores/ui";
  import {
    LayoutDashboard,
    AlertCircle,
    Settings,
    LogOut,
    ChevronLeft,
    ChevronRight,
    Activity,
    Lock,
    BarChart3,
    Shield,
    Route,
  } from "lucide-svelte";

  export let currentPath = "";

  function handleLogout() {
    logout();
    window.location.href = "/login";
  }
</script>

<aside
  class="fixed inset-y-0 left-0 z-[1000] flex h-screen shrink-0 transition-all duration-300 ease-in-out"
  class:w-64={!$sidebarCollapsed}
  class:w-20={$sidebarCollapsed}
>
  <!-- Mobile Overlay -->
  {#if $mobileMenuOpen}
    <div
      role="button"
      tabindex="0"
      class="fixed inset-0 z-[1001] bg-black/60 backdrop-blur-sm transition-opacity duration-300 lg:hidden"
      on:click={closeMobileMenu}
      on:keydown={(e) => e.key === "Escape" && closeMobileMenu()}
    ></div>
  {/if}

  <!-- Sidebar Content -->
  <div
    class="relative z-[1002] flex h-full w-full flex-col border-r border-white/10 bg-black/80 backdrop-blur-xl transition-transform duration-300 lg:translate-x-0"
    class:translate-x-0={$mobileMenuOpen}
    class:-translate-x-full={!$mobileMenuOpen}
    class:lg:w-64={!$sidebarCollapsed}
    class:lg:w-20={$sidebarCollapsed}
  >
    <!-- Header/Logo -->
    <div class="flex h-16 items-center justify-between px-4">
      <Link to="/" class="flex items-center gap-3 overflow-hidden font-bold">
        <div
          class="flex h-9 w-9 items-center justify-center rounded-lg bg-gradient-to-br from-pulse-600 to-indigo-600 shadow-lg shadow-pulse-600/30"
        >
          <Activity size={20} class="text-white" />
        </div>
        {#if !$sidebarCollapsed}
          <span
            class="whitespace-nowrap bg-gradient-to-r from-white to-slate-400 bg-clip-text text-xl font-bold text-transparent"
          >
            Pulse
          </span>
        {/if}
      </Link>

      <button
        class="hidden h-7 w-7 items-center justify-center rounded-md border border-white/10 bg-white/5 text-slate-400 hover:text-white lg:flex"
        on:click={() => sidebarCollapsed.update((c) => !c)}
      >
        {#if $sidebarCollapsed}
          <ChevronRight size={16} />
        {:else}
          <ChevronLeft size={16} />
        {/if}
      </button>

      <!-- Mobile Close Button -->
      <button
        class="flex h-9 w-9 items-center justify-center text-slate-400 lg:hidden"
        on:click={closeMobileMenu}
      >
        <ChevronLeft size={24} />
      </button>
    </div>

    <!-- Navigation -->
    <nav class="flex-1 space-y-1 overflow-y-auto p-3">
      {#if !$sidebarCollapsed}
        <div
          class="mb-2 px-3 text-[10px] font-bold uppercase tracking-wider text-slate-500"
        >
          Overview
        </div>
      {/if}

      <Link
        to="/"
        class="group flex items-center rounded-lg px-3 py-2.5 transition-colors {currentPath ===
        '/'
          ? 'bg-pulse-500/10 text-pulse-400'
          : 'text-slate-400 hover:bg-white/5 hover:text-white'}"
      >
        <LayoutDashboard size={20} class="shrink-0" />
        {#if !$sidebarCollapsed}
          <span class="ml-3 text-sm font-medium">Dashboard</span>
        {/if}
      </Link>

      <Link
        to="/issues"
        class="group flex items-center rounded-lg px-3 py-2.5 transition-colors {currentPath.startsWith(
          '/issues',
        ) || currentPath.startsWith('/errors')
          ? 'bg-pulse-500/10 text-pulse-400'
          : 'text-slate-400 hover:bg-white/5 hover:text-white'}"
      >
        <AlertCircle size={20} class="shrink-0" />
        {#if !$sidebarCollapsed}
          <span class="ml-3 text-sm font-medium">Issues</span>
        {/if}
      </Link>

      <Link
        to="/traces"
        class="group flex items-center rounded-lg px-3 py-2.5 transition-colors {currentPath ===
        '/traces'
          ? 'bg-pulse-500/10 text-pulse-400'
          : 'text-slate-400 hover:bg-white/5 hover:text-white'}"
      >
        <Route size={20} class="shrink-0" />
        {#if !$sidebarCollapsed}
          <span class="ml-3 text-sm font-medium">Traces</span>
        {/if}
      </Link>

      <Link
        to="/trace-analytics"
        class="group flex items-center rounded-lg px-3 py-2.5 transition-colors {currentPath ===
        '/trace-analytics'
          ? 'bg-pulse-500/10 text-pulse-400'
          : 'text-slate-400 hover:bg-white/5 hover:text-white'}"
      >
        <BarChart3 size={20} class="shrink-0" />
        {#if !$sidebarCollapsed}
          <span class="ml-3 text-sm font-medium">Trace Analytics</span>
        {/if}
      </Link>

      <Link
        to="/projects"
        class="group flex items-center rounded-lg px-3 py-2.5 transition-colors {currentPath ===
        '/projects'
          ? 'bg-pulse-500/10 text-pulse-400'
          : 'text-slate-400 hover:bg-white/5 hover:text-white'}"
      >
        <Activity size={20} class="shrink-0" />
        {#if !$sidebarCollapsed}
          <span class="ml-3 text-sm font-medium">Projects</span>
        {/if}
      </Link>

      <Link
        to="/insights"
        class="group flex items-center rounded-lg px-3 py-2.5 transition-colors {currentPath ===
        '/insights'
          ? 'bg-pulse-500/10 text-pulse-400'
          : 'text-slate-400 hover:bg-white/5 hover:text-white'}"
      >
        <BarChart3 size={20} class="shrink-0" />
        {#if !$sidebarCollapsed}
          <span class="ml-3 text-sm font-medium">Insights</span>
        {/if}
      </Link>

      {#if !$sidebarCollapsed}
        <div
          class="mb-2 mt-6 px-3 text-[10px] font-bold uppercase tracking-wider text-slate-500"
        >
          Management
        </div>
      {/if}

      <Link
        to="/settings"
        class="group flex items-center rounded-lg px-3 py-2.5 transition-colors {currentPath ===
        '/settings'
          ? 'bg-pulse-500/10 text-pulse-400'
          : 'text-slate-400 hover:bg-white/5 hover:text-white'}"
      >
        <Settings size={20} class="shrink-0" />
        {#if !$sidebarCollapsed}
          <span class="ml-3 text-sm font-medium">Settings</span>
        {/if}
      </Link>

      <Link
        to="/security-vault"
        class="group flex items-center rounded-lg px-3 py-2.5 transition-colors {currentPath ===
        '/security-vault'
          ? 'bg-pulse-500/10 text-pulse-400'
          : 'text-slate-400 hover:bg-white/5 hover:text-white'}"
      >
        <Lock size={20} class="shrink-0" />
        {#if !$sidebarCollapsed}
          <span class="ml-3 text-sm font-medium">Security Vault</span>
        {/if}
      </Link>

      <Link
        to="/admin"
        class="group flex items-center rounded-lg px-3 py-2.5 transition-colors {currentPath ===
        '/admin'
          ? 'bg-pulse-500/10 text-pulse-400'
          : 'text-slate-400 hover:bg-white/5 hover:text-white'}"
      >
        <Shield size={20} class="shrink-0" />
        {#if !$sidebarCollapsed}
          <span class="ml-3 text-sm font-medium">Admin</span>
        {/if}
      </Link>
    </nav>

    <!-- Footer -->
    <div class="border-t border-white/10 p-4">
      <div
        class="mb-4 flex items-center gap-3 rounded-xl bg-white/5 p-2 transition-all"
        class:justify-center={$sidebarCollapsed}
      >
        <div
          class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-pulse-600 font-bold text-white shadow-lg shadow-pulse-600/20"
        >
          {$user?.email?.[0]?.toUpperCase() || "U"}
        </div>
        {#if !$sidebarCollapsed}
          <div class="min-w-0 flex-1 overflow-hidden">
            <div class="truncate text-xs font-medium text-white">
              {$user?.email}
            </div>
            <div class="text-[10px] text-slate-500">Administrator</div>
          </div>
        {/if}
      </div>

      <button
        on:click={handleLogout}
        class="flex w-full items-center justify-center gap-2 rounded-lg border border-white/10 py-2.5 text-sm font-medium text-slate-400 transition-all hover:border-red-500/30 hover:bg-red-500/10 hover:text-red-500"
        class:px-0={$sidebarCollapsed}
      >
        <LogOut size={18} />
        {#if !$sidebarCollapsed}
          <span>Sign Out</span>
        {/if}
      </button>
    </div>
  </div>
</aside>

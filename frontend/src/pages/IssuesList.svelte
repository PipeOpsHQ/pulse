<script>
  import { onMount, onDestroy } from "svelte";
  import { navigate } from "../lib/router";
  import Link from "../components/Link.svelte";
  import { api } from "../lib/api";
  import { getErrorLevelColor, getIssueStatusColor } from "../lib/statusColors";
  import {
    Search,
    ArrowUpDown,
    Zap,
    Users,
    ChevronLeft,
    ChevronRight,
    Filter,
    Activity,
    Inbox,
    Target,
  } from "lucide-svelte";

  let issues = [];
  let projects = [];
  let loading = true;
  let activeTab = "unresolved";
  let sortBy = "lastSeen";
  let searchQuery = "";
  let selectedProjectId = "";
  let meta = { limit: 50, cursor: "", has_more: false };
  let filteredIssues = [];
  let nextCursor = "";

  const tabs = [
    { id: "unresolved", label: "Unresolved", icon: Inbox },
    { id: "resolved", label: "Resolved", icon: Target },
    { id: "ignored", label: "Ignored", icon: Filter },
  ];

  let refreshInterval = null;

  // Refresh data when page becomes visible (e.g., returning from issue detail)
  function visibilityHandler() {
    if (!document.hidden) {
      loadIssues("", false); // Background refresh, reset cursor
    }
  }

  onMount(async () => {
    await Promise.all([
      loadProjects(),
      loadIssues("", true), // Initial load, empty cursor
    ]);
    document.addEventListener("visibilitychange", visibilityHandler);

    // Set up real-time polling every 30 seconds (optimized from 10s)
    refreshInterval = setInterval(() => {
      if (!document.hidden) {
        loadIssues("", false); // Background refresh, reset cursor
      }
    }, 30000);
  });

  onDestroy(() => {
    document.removeEventListener("visibilitychange", visibilityHandler);
    if (refreshInterval) {
      clearInterval(refreshInterval);
    }
  });

  async function loadProjects() {
    try {
      const response = await api.get("/projects");
      projects = Array.isArray(response) ? response : response?.data || [];
    } catch (error) {
      console.error("Failed to load projects:", error);
      projects = [];
    }
  }

  async function loadIssues(cursor = "", showLoading = false) {
    if (showLoading) {
      loading = true;
    }
    try {
      const status = activeTab === "unresolved" ? "unresolved" : activeTab;
      let url = `/errors?status=${status}&limit=50&use_cursor=true`;
      if (cursor) {
        url += `&cursor=${encodeURIComponent(cursor)}`;
      }
      if (selectedProjectId) {
        url += `&projectId=${selectedProjectId}`;
      }
      const response = await api.get(url, { ttl: 5000 }); // Cache for 5 seconds

      const rawIssues = response?.data || response || [];
      meta = response?.meta || { limit: 50, cursor: "", has_more: false };
      nextCursor = meta.cursor || "";
      const hasMore = meta.has_more || false;

      // If loading with cursor (pagination), append to existing issues
      const newIssues = Array.isArray(rawIssues)
        ? rawIssues.map((issue) => {
            const project = projects.find((p) => p.id === issue.project_id);
            return {
              ...issue,
              projectName: project ? project.name : "Unknown Project",
              // Use actual stats from API
              eventCount: issue.event_count || 1,
              userCount: issue.user_count || 0,
            };
          })
        : [];

      // If cursor is empty, replace issues (new load or refresh)
      // If cursor exists, append (load more)
      if (!cursor) {
        issues = newIssues;
      } else {
        issues = [...issues, ...newIssues];
      }
    } catch (error) {
      console.error("Failed to load issues:", error);
    } finally {
      if (showLoading) {
        loading = false;
      }
    }
  }

  function handleTabChange(tabId) {
    activeTab = tabId;
    issues = [];
    nextCursor = "";
    loadIssues("", true);
  }

  function handleLoadMore() {
    if (nextCursor && !loading) {
      loadIssues(nextCursor, false);
    }
  }

  function handleProjectChange() {
    issues = [];
    nextCursor = "";
    loadIssues("", true);
  }

  // Filter and sort issues reactively
  $: filteredIssues = (() => {
    // First filter
    const filtered = searchQuery
      ? (issues || []).filter(
          (issue) =>
            issue.message?.toLowerCase().includes(searchQuery.toLowerCase()) ||
            issue.projectName
              ?.toLowerCase()
              .includes(searchQuery.toLowerCase()),
        )
      : issues || [];

    // Then sort (create new array, don't mutate original)
    const sorted = [...filtered];
    if (sortBy === "events") {
      sorted.sort((a, b) => b.eventCount - a.eventCount);
    } else if (sortBy === "users") {
      sorted.sort((a, b) => b.userCount - a.userCount);
    } else if (sortBy === "firstSeen") {
      sorted.sort((a, b) => new Date(a.created_at) - new Date(b.created_at));
    } else {
      sorted.sort((a, b) => new Date(b.created_at) - new Date(a.created_at));
    }
    return sorted;
  })();

  function getLevelColorClass(level) {
    const colors = getErrorLevelColor(level);
    return `${colors.bg} ${colors.text} ${colors.border} border`;
  }

  function getStatusColorClass(status) {
    const colors = getIssueStatusColor(status);
    return `${colors.bg} ${colors.text} ${colors.border} border`;
  }

  function formatDate(dateString) {
    const date = new Date(dateString);
    const now = new Date();
    const diff = now - date;
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return "just now";
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    if (days < 7) return `${days}d ago`;
    return date.toLocaleDateString(undefined, {
      month: "short",
      day: "numeric",
    });
  }

  function getIssueUrl(issue) {
    return `/errors/${issue.id}`;
  }
</script>

<div class="animate-in fade-in slide-in-from-bottom-4 duration-500">
  <div
    class="mb-5 flex flex-col justify-between gap-3 sm:flex-row sm:items-center"
  >
    <div>
      <h1 class="text-xl font-semibold tracking-tight text-white mb-0.5">
        Issues
      </h1>
      <p class="text-xs text-slate-400">
        Track and manage application errors across all environments
      </p>
    </div>

    <div class="grid grid-cols-2 gap-2 sm:flex sm:items-center sm:gap-3">
      <div class="relative">
        <Filter
          size={14}
          class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500"
        />
        <select
          bind:value={selectedProjectId}
          on:change={handleProjectChange}
          class="h-9 w-full sm:w-44 rounded-lg border border-white/[0.08] bg-white/[0.04] pl-8 pr-3 text-xs font-medium text-white outline-none focus:border-pulse-500/50 focus:ring-1 focus:ring-pulse-500/20 focus:bg-white/[0.06] active:scale-[0.98] transition-all duration-200 appearance-none cursor-pointer"
        >
          <option value="">All Projects</option>
          {#each projects as project}
            <option value={project.id}>{project.name}</option>
          {/each}
        </select>
      </div>

      <div class="relative">
        <ArrowUpDown
          size={12}
          class="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-500"
        />
        <select
          bind:value={sortBy}
          class="h-9 w-full sm:w-40 rounded-lg border border-white/[0.08] bg-white/[0.04] pl-8 pr-3 text-xs font-medium text-white outline-none focus:border-pulse-500/50 focus:ring-1 focus:ring-pulse-500/20 focus:bg-white/[0.06] active:scale-[0.98] transition-all duration-200 appearance-none cursor-pointer"
        >
          <option value="lastSeen">Last Seen</option>
          <option value="firstSeen">First Seen</option>
          <option value="events">Events</option>
          <option value="users">Users</option>
        </select>
      </div>
    </div>
  </div>

  <!-- Filters Bar -->
  <div
    class="mb-3 overflow-hidden rounded-lg border border-white/[0.08] bg-gradient-to-br from-white/[0.03] to-white/[0.01] backdrop-blur-xl"
  >
    <div class="flex flex-col border-b border-white/[0.08] lg:flex-row">
      <div
        class="flex overflow-x-auto no-scrollbar border-b border-white/[0.08] px-1.5 lg:border-b-0 lg:border-r"
      >
        {#each tabs as tab}
          <button
            class="relative flex items-center gap-1.5 px-4 py-3 text-[10px] font-semibold uppercase tracking-wider transition-all duration-200 whitespace-nowrap"
            class:text-pulse-400={activeTab === tab.id}
            class:text-slate-400={activeTab !== tab.id}
            class:hover:text-white={activeTab !== tab.id}
            on:click={() => handleTabChange(tab.id)}
          >
            <svelte:component this={tab.icon} size={12} />
            <span>{tab.label}</span>
            {#if activeTab === tab.id}
              <div
                class="absolute bottom-0 left-0 h-0.5 w-full bg-gradient-to-r from-pulse-500 to-pulse-400"
              ></div>
            {/if}
          </button>
        {/each}
      </div>

      <div class="flex flex-1 items-center px-4 py-2.5 lg:py-0">
        <Search size={14} class="text-slate-500 mr-2.5" />
        <input
          type="text"
          placeholder="Filter issues..."
          class="flex-1 bg-transparent text-xs text-white placeholder-slate-500 outline-none font-medium"
          bind:value={searchQuery}
        />
      </div>
    </div>
  </div>

  <!-- Issues List -->
  <div
    class="overflow-hidden rounded-lg border border-white/[0.08] bg-gradient-to-br from-white/[0.03] to-white/[0.01] backdrop-blur-xl"
  >
    {#if loading}
      <div class="divide-y divide-white/5 text-slate-500">
        {#each Array(8) as _}
          <div class="h-20 animate-pulse bg-white/5"></div>
        {/each}
      </div>
    {:else if filteredIssues.length === 0}
      <div class="flex flex-col items-center justify-center py-20">
        <div
          class="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-pulse-500/10 text-pulse-400"
        >
          <Activity size={32} />
        </div>
        <h3 class="mb-1 text-base font-semibold text-white">All clear here!</h3>
        <p class="text-sm text-slate-500">
          No issues match your current filters.
        </p>
      </div>
    {:else}
      <div class="divide-y divide-white/[0.06]">
        {#each filteredIssues as issue}
          {@const levelColors = getErrorLevelColor(issue.level)}
          {@const statusColors = getIssueStatusColor(issue.status)}
          <Link
            to={getIssueUrl(issue)}
            class="group flex items-center gap-3 sm:gap-4 px-4 py-3.5 transition-all duration-200 hover:bg-white/[0.03]"
          >
            <div
              class="h-10 w-0.5 shrink-0 rounded-full transition-all duration-200 group-hover:w-1 {levelColors.dot}"
            ></div>

            <div class="flex-1 min-w-0">
              <div
                class="truncate text-xs font-semibold text-white group-hover:text-pulse-400 transition-colors leading-tight"
              >
                {issue.message || "No message"}
              </div>
              <div
                class="mt-1 flex items-center gap-2 text-[9px] text-slate-500"
              >
                <span
                  class="rounded px-1.5 py-0.5 text-xs font-bold uppercase tracking-tight {getStatusColorClass(
                    issue.status,
                  )}"
                >
                  {statusColors.icon}
                  {issue.status}
                </span>
                <span class="font-medium text-slate-400"
                  >{issue.projectName}</span
                >
                {#if issue.environment}
                  <span class="rounded bg-white/5 px-1.5 py-0.5"
                    >{issue.environment}</span
                  >
                {/if}
                <span>•</span>
                <span>{formatDate(issue.created_at)}</span>
              </div>
            </div>

            <div class="hidden items-center gap-6 sm:flex">
              <div class="flex flex-col items-center">
                <Zap size={12} class="mb-0.5 text-slate-600" />
                <span class="text-[10px] font-bold text-white"
                  >{issue.eventCount}</span
                >
              </div>
              <div class="flex flex-col items-center">
                <Users size={12} class="mb-0.5 text-slate-600" />
                <span class="text-[10px] font-bold text-white"
                  >{issue.userCount}</span
                >
              </div>
            </div>

            <div class="hidden w-24 sm:block">
              <svg
                width="64"
                height="24"
                viewBox="0 0 64 24"
                class="text-pulse-600 opacity-30"
              >
                <path
                  d="M0,24 L8,18 L16,22 L24,10 L32,15 L40,5 L48,20 L64,12"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.5"
                />
              </svg>
            </div>

            <ChevronRight
              size={18}
              class="text-slate-700 transition-transform group-hover:translate-x-1 group-hover:text-white"
            />
          </Link>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Load More Button (Cursor-based pagination) -->
  {#if nextCursor && meta.has_more}
    <div class="mt-8 flex items-center justify-center border-t border-white/10 pt-6">
      <button
        class="rounded-lg border border-white/10 bg-white/5 px-6 py-2 text-sm text-slate-400 transition-all hover:bg-white/10 hover:text-white disabled:opacity-30"
        disabled={loading}
        on:click={handleLoadMore}
      >
          <ChevronLeft size={20} />
        </button>
        <button
          class="flex h-10 w-10 items-center justify-center rounded-lg border border-white/10 bg-white/5 text-slate-400 transition-all hover:bg-white/10 hover:text-white disabled:opacity-30"
          disabled={meta.offset + meta.limit >= meta.total}
          on:click={() => handlePageChange(meta.offset + meta.limit)}
        >
          {loading ? "Loading..." : "Load More"}
        </button>
    </div>
  {/if}
</div>

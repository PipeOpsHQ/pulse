<script>
  import { onMount } from "svelte";
  import { navigate } from "../lib/router";
  import Link from "../components/Link.svelte";
  import { api } from "../lib/api";
  import { toast } from "../stores/toast";
  import { Activity, Search, Timer, ChevronRight, Filter } from "lucide-svelte";

  let traces = [];
  let projects = [];
  let loading = true;
  let traceQuery = "";
  let selectedProject = "";
  let loadingMore = false;
  let hasMore = true;
  let limit = 50;

  async function loadTraces(loadMore = false) {
    if (!loadMore) {
      loading = true;
      traces = [];
      hasMore = true;
    } else {
      loadingMore = true;
    }

    try {
      const offset = loadMore ? traces.length : 0;
      let url = `/traces?limit=${limit}&offset=${offset}&query=${encodeURIComponent(traceQuery)}`;
      if (selectedProject) {
        url += `&project_id=${selectedProject}`;
      }
      const res = await api.get(url);
      const newTraces = res || [];

      if (loadMore) {
        traces = [...traces, ...newTraces];
      } else {
        traces = newTraces;
      }

      if (newTraces.length < limit) {
        hasMore = false;
      }
    } catch (err) {
      toast.fromHttpError(err);
    } finally {
      loading = false;
      loadingMore = false;
    }
  }

  async function loadProjects() {
    try {
      const res = await api.get("/projects");
      projects = res || [];
    } catch (err) {
      console.error("Failed to load projects:", err);
    }
  }

  function handleSearch(e) {
    if (e.key === "Enter") {
      loadTraces();
    }
  }

  function handleScroll(e) {
    const { scrollTop, scrollHeight, clientHeight } = e.target;
    if (
      scrollHeight - scrollTop <= clientHeight + 100 &&
      !loadingMore &&
      hasMore &&
      !loading
    ) {
      loadTraces(true);
    }
  }

  function formatDuration(trace) {
    const duration =
      new Date(trace.timestamp).getTime() -
      new Date(trace.start_timestamp).getTime();
    return duration >= 1000
      ? (duration / 1000).toFixed(2) + "s"
      : duration.toFixed(0) + "ms";
  }

  function getStatusColor(status) {
    const colors = {
      ok: "text-emerald-400 bg-emerald-500/10 border-emerald-500/20",
      error: "text-red-400 bg-red-500/10 border-red-500/20",
      cancelled: "text-yellow-400 bg-yellow-500/10 border-yellow-500/20",
      unknown: "text-slate-400 bg-slate-500/10 border-slate-500/20",
    };
    return colors[status?.toLowerCase()] || colors.unknown;
  }

  onMount(() => {
    loadProjects();
    loadTraces();
  });
</script>

<div
  class="min-h-screen bg-gradient-to-br from-[#0a0a0b] via-[#0f0f12] to-[#0a0a0b] p-8"
>
  <div class="mx-auto max-w-7xl">
    <!-- Header -->
    <div class="mb-8">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-3xl font-black text-white tracking-tight">
            Performance Traces
          </h1>
          <p class="mt-2 text-sm text-slate-400">
            Monitor distributed tracing and transaction performance across all
            projects
          </p>
        </div>
        <div class="flex items-center gap-2">
          <div class="rounded-lg bg-white/5 border border-white/10 px-4 py-2">
            <div
              class="text-[10px] text-slate-500 uppercase tracking-wider font-bold"
            >
              Total Traces
            </div>
            <div class="text-2xl font-black text-white mt-1">
              {traces.length}
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Search and Filter Bar -->
    <div class="mb-6 flex items-center gap-3">
      <div class="relative flex-1 max-w-md">
        <div
          class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none"
        >
          <Search size={18} class="text-slate-500" />
        </div>
        <input
          type="text"
          bind:value={traceQuery}
          on:keydown={handleSearch}
          placeholder="Search traces by name, operation, or description..."
          class="block w-full pl-12 pr-4 py-3 border border-white/10 rounded-xl bg-black/20 text-sm text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/50 focus:border-indigo-500/50 transition-all"
        />
      </div>
      <div class="relative">
        <div
          class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none"
        >
          <Filter size={16} class="text-slate-500" />
        </div>
        <select
          bind:value={selectedProject}
          on:change={() => loadTraces()}
          class="block pl-11 pr-10 py-3 border border-white/10 rounded-xl bg-black/20 text-sm text-white focus:outline-none focus:ring-2 focus:ring-indigo-500/50 focus:border-indigo-500/50 transition-all appearance-none cursor-pointer"
        >
          <option value="">All Projects</option>
          {#each projects as project}
            <option value={project.id}>{project.name}</option>
          {/each}
        </select>
      </div>
      <button
        on:click={() => loadTraces()}
        class="px-6 py-3 bg-indigo-500 hover:bg-indigo-400 text-white text-sm font-bold rounded-xl transition-colors"
      >
        Search
      </button>
      <button
        on:click={() => {
          traceQuery = "";
          selectedProject = "";
          loadTraces();
        }}
        class="px-6 py-3 bg-white/5 hover:bg-white/10 border border-white/10 text-white text-sm font-bold rounded-xl transition-colors"
      >
        Reset
      </button>
    </div>

    <!-- Traces List -->
    <div
      class="rounded-xl border border-white/10 bg-white/5 backdrop-blur-xl overflow-hidden"
    >
      {#if loading}
        <div class="flex flex-col items-center justify-center py-32">
          <div
            class="h-12 w-12 animate-spin rounded-full border-4 border-indigo-500/20 border-t-indigo-500"
          ></div>
          <p class="mt-4 text-sm text-slate-400">Loading traces...</p>
        </div>
      {:else if traces.length === 0}
        <div
          class="flex flex-col items-center justify-center py-32 text-center"
        >
          <div class="mb-6 rounded-full bg-white/5 p-6 text-slate-500">
            <Timer size={48} />
          </div>
          <p class="text-lg font-medium text-slate-400">No traces found</p>
          <p class="mt-2 text-sm text-slate-600 max-w-md">
            {#if traceQuery}
              Try adjusting your search query or clearing filters.
            {:else}
              Configure your Sentry SDK to send performance data (transactions).
            {/if}
          </p>
        </div>
      {:else}
        <div
          class="max-h-[calc(100vh-300px)] overflow-y-auto scrollbar-thin scrollbar-thumb-white/10"
          on:scroll={handleScroll}
        >
          <table class="w-full text-left border-collapse">
            <thead
              class="sticky top-0 bg-[#0a0a0b] z-10 border-b border-white/10"
            >
              <tr
                class="text-[10px] font-bold uppercase tracking-wider text-slate-500"
              >
                <th class="px-6 py-4">Transaction</th>
                <th class="px-6 py-4">Project</th>
                <th class="px-6 py-4">Operation</th>
                <th class="px-6 py-4">Status</th>
                <th class="px-6 py-4">Duration</th>
                <th class="px-6 py-4">Time</th>
                <th class="px-6 py-4"></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-white/5">
              {#each traces as trace}
                <tr class="group hover:bg-white/5 transition-colors">
                  <td class="px-6 py-4">
                    <div
                      class="text-sm font-bold text-white group-hover:text-indigo-400 transition-colors"
                    >
                      {trace.description || trace.name || "Unknown Transaction"}
                    </div>
                    <div class="text-xs text-slate-500 font-mono mt-0.5">
                      {trace.trace_id.substring(0, 16)}...
                    </div>
                  </td>
                  <td class="px-6 py-4">
                    <Link
                      to="/projects/{trace.project_id}"
                      class="text-xs text-pulse-400 hover:text-pulse-300 font-medium"
                    >
                      Project
                    </Link>
                  </td>
                  <td class="px-6 py-4">
                    <span
                      class="inline-flex items-center rounded-md bg-white/10 px-2 py-1 text-xs font-medium text-slate-300 ring-1 ring-inset ring-white/10"
                    >
                      {trace.op || "default"}
                    </span>
                  </td>
                  <td class="px-6 py-4">
                    {#if trace.status && trace.status.toLowerCase() !== "ok" && trace.status.toLowerCase() !== "unset"}
                      <span
                        class="inline-flex items-center rounded-md px-2 py-1 text-xs font-bold border {getStatusColor(
                          trace.status,
                        )}"
                      >
                        {trace.status}
                      </span>
                    {:else}
                      <span class="text-xs text-slate-600">—</span>
                    {/if}
                  </td>
                  <td class="px-6 py-4">
                    <span class="text-sm font-mono text-emerald-400">
                      {formatDuration(trace)}
                    </span>
                  </td>
                  <td
                    class="px-6 py-4 text-xs text-slate-400 whitespace-nowrap"
                  >
                    {new Date(trace.timestamp).toLocaleString()}
                  </td>
                  <td class="px-6 py-4">
                    <Link
                      to="/projects/{trace.project_id}?tab=traces"
                      class="text-slate-500 hover:text-white transition-colors"
                    >
                      <ChevronRight size={16} />
                    </Link>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>

          {#if loadingMore}
            <div
              class="py-6 flex justify-center border-t border-white/5 bg-[#0a0a0b]/50"
            >
              <div
                class="h-6 w-6 animate-spin rounded-full border-2 border-indigo-500/20 border-t-indigo-500"
              ></div>
            </div>
          {:else if !hasMore && traces.length > 0}
            <div
              class="py-8 text-center text-[10px] text-slate-600 font-bold uppercase tracking-widest border-t border-white/5 bg-[#0a0a0b]/10"
            >
              End of traces
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </div>
</div>

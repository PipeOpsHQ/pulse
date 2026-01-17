<script>
  import { onMount } from "svelte";
  import { navigate } from "../lib/router";
  import Link from "../components/Link.svelte";
  import { Activity, Clock, ArrowRight, ChevronRight } from "lucide-svelte";
  import { apiGet } from "../lib/api.js";
  import { toast } from "../stores/toast.js";

  let traces = [];
  let loading = true;
  let projectId = "";

  onMount(async () => {
    // Get project ID from URL
    const path = window.location.pathname;
    const match = path.match(/\/projects\/([^\/]+)\/traces/);
    if (match) {
      projectId = match[1];
      await loadTraces();
    } else {
      loading = false;
    }
  });

  async function loadTraces() {
    try {
      const data = await apiGet(`/projects/${projectId}/traces`);
      traces = data || [];
    } catch (error) {
      console.error("Failed to load traces:", error);
      toast.error("Failed to load traces");
    } finally {
      loading = false;
    }
  }

  function formatDuration(start, end) {
    if (!start || !end) return "N/A";
    const startTime = new Date(start).getTime();
    const endTime = new Date(end).getTime();
    const duration = endTime - startTime;
    if (duration < 1000) return `${duration}ms`;
    return `${(duration / 1000).toFixed(2)}s`;
  }

  function formatDate(dateString) {
    if (!dateString) return "N/A";
    return new Date(dateString).toLocaleString();
  }
</script>

<div
  class="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 p-4 sm:p-6 lg:p-8"
>
  <div class="mx-auto max-w-7xl">
    <div class="mb-8">
      <h1 class="text-3xl font-bold text-white mb-2">Performance Traces</h1>
      <p class="text-slate-400">
        View and analyze performance traces for your project
      </p>
    </div>

    {#if loading}
      <div class="pulse-card p-12 text-center">
        <div
          class="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-pulse-400"
        ></div>
        <p class="mt-4 text-slate-400">Loading traces...</p>
      </div>
    {:else if traces.length === 0}
      <div class="pulse-card p-12 text-center">
        <Activity size={48} class="mx-auto mb-4 text-slate-600" />
        <p class="text-slate-400">No traces found</p>
        <p class="text-sm text-slate-500 mt-2">
          Traces will appear here when your application sends performance data
        </p>
      </div>
    {:else}
      <div class="space-y-4">
        {#each traces as trace}
          <Link
            to="/projects/{projectId}/traces/{trace.trace_id}"
            class="block"
          >
            <div class="pulse-card p-6 hover:bg-white/10 transition-all group">
              <div class="flex items-start justify-between gap-4">
                <div class="flex-1 min-w-0">
                  <div class="flex items-center gap-3 mb-3">
                    <Activity size={18} class="text-pulse-400 shrink-0" />
                    <span class="font-mono text-sm text-pulse-400 truncate"
                      >{trace.trace_id}</span
                    >
                  </div>

                  {#if trace.name}
                    <h3
                      class="text-lg font-semibold text-white mb-2 group-hover:text-pulse-400 transition-colors"
                    >
                      {trace.name}
                    </h3>
                  {/if}

                  {#if trace.description}
                    <p class="text-sm text-slate-400 mb-4">
                      {trace.description}
                    </p>
                  {/if}

                  <div
                    class="flex flex-wrap items-center gap-4 text-xs text-slate-500"
                  >
                    {#if trace.op}
                      <div class="flex items-center gap-1.5">
                        <span class="font-semibold">Operation:</span>
                        <span class="text-slate-300">{trace.op}</span>
                      </div>
                    {/if}

                    {#if trace.start_timestamp && trace.timestamp}
                      <div class="flex items-center gap-1.5">
                        <Clock size={12} />
                        <span
                          >{formatDuration(
                            trace.start_timestamp,
                            trace.timestamp,
                          )}</span
                        >
                      </div>
                    {/if}

                    {#if trace.status}
                      <span
                        class="rounded px-2 py-0.5 bg-white/10 text-slate-300 uppercase text-[10px] font-bold"
                      >
                        {trace.status}
                      </span>
                    {/if}
                  </div>
                </div>

                <ChevronRight
                  size={20}
                  class="text-slate-600 group-hover:text-pulse-400 transition-colors shrink-0"
                />
              </div>
            </div>
          </Link>
        {/each}
      </div>
    {/if}
  </div>
</div>

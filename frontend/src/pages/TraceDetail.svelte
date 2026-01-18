<script>
  import { onMount } from "svelte";
  import { navigate } from "../lib/router";
  import Link from "../components/Link.svelte";
  import { Activity, Clock, ArrowRight, ChevronLeft, AlertCircle, ChevronRight } from "lucide-svelte";
  import { apiGet } from "../lib/api.js";
  import { toast } from "../stores/toast.js";
  import { getErrorLevelColor } from "../lib/statusColors";

  let spans = [];
  let loading = true;
  let projectId = "";
  let traceId = "";
  let linkedErrors = [];
  let loadingErrors = false;

  onMount(async () => {
    // Get project ID and trace ID from URL
    const path = window.location.pathname;
    const match = path.match(/\/projects\/([^\/]+)\/traces\/([^\/]+)/);
    if (match) {
      projectId = match[1];
      traceId = match[2];
      await loadTraceDetails();
    } else {
      loading = false;
    }
  });

  async function loadTraceDetails() {
    try {
      const data = await apiGet(`/projects/${projectId}/traces/${traceId}`);
      spans = data || [];

      // Load linked errors
      try {
        loadingErrors = true;
        linkedErrors = await apiGet(`/traces/${traceId}/errors`);
      } catch (e) {
        console.error("Failed to load linked errors:", e);
        linkedErrors = [];
      } finally {
        loadingErrors = false;
      }
    } catch (error) {
      console.error("Failed to load trace details:", error);
      toast.error("Failed to load trace details");
    } finally {
      loading = false;
    }
  }

  function formatDuration(start, end) {
    if (!start || !end) return "N/A";
    const startTime = new Date(start).getTime();
    const endTime = new Date(end).getTime();
    return `${(endTime - startTime).toFixed(2)}ms`;
  }

  function formatDate(dateString) {
    if (!dateString) return "N/A";
    return new Date(dateString).toLocaleString();
  }

  function getStatusColor(status) {
    const colors = {
      ok: "text-emerald-400",
      cancelled: "text-amber-400",
      invalid_argument: "text-red-400",
      deadline_exceeded: "text-red-400",
      not_found: "text-red-400",
      already_exists: "text-amber-400",
      permission_denied: "text-red-400",
      resource_exhausted: "text-red-400",
      failed_precondition: "text-amber-400",
      aborted: "text-amber-400",
      out_of_range: "text-red-400",
      unimplemented: "text-amber-400",
      internal: "text-red-400",
      unavailable: "text-red-400",
      data_loss: "text-red-400",
      unauthenticated: "text-red-400",
    };
    return colors[status?.toLowerCase()] || "text-slate-400";
  }

  // Build tree structure from spans
  $: spanTree = buildSpanTree(spans);

  function buildSpanTree(spans) {
    if (!spans || spans.length === 0) return [];

    const spanMap = new Map();
    const roots = [];

    // First pass: create map
    spans.forEach((span) => {
      spanMap.set(span.span_id, { ...span, children: [] });
    });

    // Second pass: build tree
    spans.forEach((span) => {
      const node = spanMap.get(span.span_id);
      if (!span.parent_span_id || span.parent_span_id === "") {
        roots.push(node);
      } else {
        const parent = spanMap.get(span.parent_span_id);
        if (parent) {
          parent.children.push(node);
        } else {
          roots.push(node);
        }
      }
    });

    return roots;
  }
</script>

<div
  class="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 p-4 sm:p-6 lg:p-8"
>
  <div class="mx-auto max-w-7xl">
    <div class="mb-8">
      <Link
        to="/projects/{projectId}/traces"
        class="inline-flex items-center gap-2 text-sm text-slate-400 hover:text-pulse-400 mb-4 transition-colors"
      >
        <ChevronLeft size={16} />
        Back to Traces
      </Link>
      <h1 class="text-3xl font-bold text-white mb-2">Trace Details</h1>
      <p class="font-mono text-sm text-pulse-400">{traceId}</p>
    </div>

    {#if loading}
      <div class="pulse-card p-12 text-center">
        <div
          class="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-pulse-400"
        ></div>
        <p class="mt-4 text-slate-400">Loading trace details...</p>
      </div>
    {:else if spans.length === 0}
      <div class="pulse-card p-12 text-center">
        <Activity size={48} class="mx-auto mb-4 text-slate-600" />
        <p class="text-slate-400">No spans found for this trace</p>
      </div>
    {:else}
      <div class="space-y-6">
        <!-- Linked Errors Section -->
        {#if linkedErrors.length > 0 || loadingErrors}
          <div class="pulse-card p-6">
            <h2 class="text-lg font-semibold text-white mb-4 flex items-center gap-2">
              <AlertCircle size={18} class="text-red-400" />
              Linked Errors ({linkedErrors.length})
            </h2>

            {#if loadingErrors}
              <div class="flex items-center justify-center py-4">
                <div
                  class="h-6 w-6 animate-spin rounded-full border-2 border-pulse-500 border-t-transparent"
                ></div>
              </div>
            {:else if linkedErrors.length > 0}
              <div class="space-y-3">
                {#each linkedErrors as error}
                  <Link
                    to="/errors/{error.id}"
                    class="group block rounded-lg border border-white/10 bg-white/5 p-4 transition-all hover:bg-white/10 hover:border-red-500/30"
                  >
                    <div class="flex items-start justify-between gap-3">
                      <div class="min-w-0 flex-1">
                        <div class="mb-2 flex items-center gap-2">
                          {#if error.level}
                            {@const levelColors = getErrorLevelColor(error.level)}
                            <span
                              class="rounded px-2 py-0.5 text-[10px] font-bold uppercase {levelColors.bg} {levelColors.text} {levelColors.border} border"
                            >
                              {error.level}
                            </span>
                          {/if}
                          {#if error.status}
                            <span
                              class="rounded px-2 py-0.5 text-[10px] font-bold uppercase {error.status === 'resolved' ? 'bg-green-500/20 text-green-400' : error.status === 'ignored' ? 'bg-yellow-500/20 text-yellow-400' : 'bg-red-500/20 text-red-400'}"
                            >
                              {error.status}
                            </span>
                          {/if}
                        </div>
                        <div class="mb-1 truncate text-sm font-semibold text-white">
                          {error.message || "No message"}
                        </div>
                        <div class="flex items-center gap-3 text-xs text-slate-400">
                          {#if error.environment}
                            <span>{error.environment}</span>
                          {/if}
                          {#if error.timestamp}
                            <span>{formatDate(error.timestamp)}</span>
                          {/if}
                        </div>
                      </div>
                      <ChevronRight
                        size={18}
                        class="text-slate-600 transition-transform group-hover:translate-x-1 group-hover:text-red-400 flex-shrink-0"
                      />
                    </div>
                  </Link>
                {/each}
              </div>
            {/if}
          </div>
        {/if}

        <!-- Trace Spans -->
        <div class="space-y-4">
        {#each spanTree as rootSpan}
          <div class="pulse-card p-6">
            <div class="mb-4">
              <div class="flex items-center gap-3 mb-2">
                <Activity size={18} class="text-pulse-400" />
                <span class="font-mono text-sm text-pulse-400"
                  >{rootSpan.span_id}</span
                >
                {#if rootSpan.status}
                  <span
                    class="rounded px-2 py-0.5 bg-white/10 text-xs font-bold uppercase {getStatusColor(
                      rootSpan.status,
                    )}"
                  >
                    {rootSpan.status}
                  </span>
                {/if}
              </div>

              {#if rootSpan.name}
                <h3 class="text-lg font-semibold text-white">
                  {rootSpan.name}
                </h3>
              {/if}

              {#if rootSpan.description}
                <p class="text-sm text-slate-400 mt-1">
                  {rootSpan.description}
                </p>
              {/if}
            </div>

            <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4 text-xs">
              {#if rootSpan.op}
                <div>
                  <span class="text-slate-500">Operation</span>
                  <p class="text-slate-300 font-mono">{rootSpan.op}</p>
                </div>
              {/if}

              {#if rootSpan.start_timestamp && rootSpan.timestamp}
                <div>
                  <span class="text-slate-500">Duration</span>
                  <p class="text-slate-300">
                    {formatDuration(
                      rootSpan.start_timestamp,
                      rootSpan.timestamp,
                    )}
                  </p>
                </div>
              {/if}

              {#if rootSpan.start_timestamp}
                <div>
                  <span class="text-slate-500">Started</span>
                  <p class="text-slate-300">
                    {formatDate(rootSpan.start_timestamp)}
                  </p>
                </div>
              {/if}
            </div>

            {#if rootSpan.data}
              <div
                class="mt-4 p-4 bg-slate-900/50 rounded-lg border border-white/5"
              >
                <pre
                  class="text-xs text-slate-300 overflow-x-auto">{JSON.stringify(
                    JSON.parse(rootSpan.data || "{}"),
                    null,
                    2,
                  )}</pre>
              </div>
            {/if}

            {#if rootSpan.children && rootSpan.children.length > 0}
              <div class="mt-6 ml-4 space-y-3 border-l-2 border-white/10 pl-4">
                {#each rootSpan.children as child}
                  <div class="pulse-card p-4 bg-white/5">
                    <div class="flex items-center gap-2 mb-2">
                      <ArrowRight size={14} class="text-slate-500" />
                      <span class="font-mono text-xs text-pulse-400"
                        >{child.span_id}</span
                      >
                      {#if child.status}
                        <span
                          class="rounded px-1.5 py-0.5 bg-white/10 text-[10px] font-bold uppercase {getStatusColor(
                            child.status,
                          )}"
                        >
                          {child.status}
                        </span>
                      {/if}
                    </div>

                    {#if child.name}
                      <p class="text-sm font-semibold text-white">
                        {child.name}
                      </p>
                    {/if}

                    {#if child.op}
                      <p class="text-xs text-slate-400 mt-1">{child.op}</p>
                    {/if}

                    {#if child.start_timestamp && child.timestamp}
                      <p class="text-xs text-slate-500 mt-2">
                        {formatDuration(child.start_timestamp, child.timestamp)}
                      </p>
                    {/if}
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {/each}
        </div>
      </div>
    {/if}
  </div>
</div>

<script>
  import { onMount, onDestroy } from "svelte";
  import { api } from "../lib/api";
  import {
    Activity,
    TrendingUp,
    Clock,
    AlertCircle,
    BarChart3,
    Zap,
    PieChart,
  } from "lucide-svelte";

  let loading = true;
  let stats = null;
  let timeSeries = [];
  let operationStats = [];
  let selectedHours = 24;
  let selectedProject = "";
  let projects = [];
  let refreshInterval = null;

  onMount(async () => {
    await loadProjects();
    await loadData();
    // Refresh every 30 seconds
    refreshInterval = setInterval(loadData, 30000);
  });

  onDestroy(() => {
    if (refreshInterval) {
      clearInterval(refreshInterval);
    }
  });

  async function loadProjects() {
    try {
      projects = await api.get("/projects");
    } catch (err) {
      console.error("Failed to load projects:", err);
    }
  }

  async function loadData() {
    loading = true;
    try {
      const params = new URLSearchParams();
      if (selectedProject) params.append("project_id", selectedProject);
      params.append("hours", selectedHours.toString());

      const [statsData, timeSeriesData, operationData] = await Promise.all([
        api.get(`/traces/stats?${params}`),
        api.get(`/traces/timeseries?${params}`),
        api.get(`/traces/operations?${params}&limit=10`),
      ]);

      stats = statsData;
      timeSeries = timeSeriesData || [];
      operationStats = operationData || [];
    } catch (err) {
      console.error("Failed to load trace analytics:", err);
    } finally {
      loading = false;
    }
  }

  function formatDuration(ms) {
    if (!ms && ms !== 0) return "N/A";
    if (ms < 1) return `${(ms * 1000).toFixed(2)}μs`;
    if (ms < 1000) return `${ms.toFixed(2)}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
  }

  function formatNumber(num) {
    if (!num && num !== 0) return "0";
    return new Intl.NumberFormat().format(num);
  }

  function formatPercent(num) {
    if (!num && num !== 0) return "0%";
    return `${num.toFixed(2)}%`;
  }

  $: chartData = {
    labels: timeSeries.map((t) => {
      const date = new Date(t.hour);
      return date.toLocaleTimeString("en-US", { hour: "2-digit", minute: "2-digit" });
    }),
    counts: timeSeries.map((t) => t.count),
    avgMs: timeSeries.map((t) => t.avg_ms),
    p95Ms: timeSeries.map((t) => t.p95_ms),
    errors: timeSeries.map((t) => t.error_count),
  };

  // Chart calculation helpers
  $: maxCount = chartData.counts.length > 0 ? Math.max(...chartData.counts, 1) : 1;
  $: maxMs = chartData.avgMs.length > 0 || chartData.p95Ms.length > 0
    ? Math.max(...chartData.avgMs, ...chartData.p95Ms, 1)
    : 1;
  $: stepX = chartData.labels.length > 1 ? 800 / (chartData.labels.length - 1) : 800;
</script>

<div class="min-h-screen bg-[#050505] text-white p-4 md:p-8">
  <div class="mx-auto w-full max-w-7xl">
    <!-- Header -->
    <div class="mb-8 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h1 class="text-3xl font-bold text-white mb-2">Trace Analytics</h1>
        <p class="text-slate-400">Performance metrics and statistics for your traces</p>
      </div>

      <div class="flex flex-wrap items-center gap-3">
        <select
          bind:value={selectedProject}
          on:change={loadData}
          class="rounded-lg border border-white/10 bg-white/5 px-4 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-pulse-500"
        >
          <option value="">All Projects</option>
          {#each projects as project}
            <option value={project.id}>{project.name}</option>
          {/each}
        </select>

        <select
          bind:value={selectedHours}
          on:change={loadData}
          class="rounded-lg border border-white/10 bg-white/5 px-4 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-pulse-500"
        >
          <option value={1}>Last Hour</option>
          <option value={6}>Last 6 Hours</option>
          <option value={24}>Last 24 Hours</option>
          <option value={168}>Last 7 Days</option>
        </select>
      </div>
    </div>

    {#if loading}
      <div class="flex h-96 items-center justify-center">
        <div
          class="h-10 w-10 animate-spin rounded-full border-2 border-pulse-500 border-t-transparent"
        ></div>
      </div>
    {:else if stats}
      <!-- Summary Stats -->
      <div class="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div
          class="rounded-xl border border-white/10 bg-gradient-to-br from-white/[0.03] to-white/[0.01] p-6 backdrop-blur-xl"
        >
          <div class="mb-2 flex items-center gap-2">
            <Activity size={20} class="text-pulse-400" />
            <span class="text-xs font-bold uppercase tracking-widest text-slate-400"
              >Total Traces</span
            >
          </div>
          <div class="text-3xl font-bold text-white">{formatNumber(stats.total_traces)}</div>
        </div>

        <div
          class="rounded-xl border border-white/10 bg-gradient-to-br from-white/[0.03] to-white/[0.01] p-6 backdrop-blur-xl"
        >
          <div class="mb-2 flex items-center gap-2">
            <Clock size={20} class="text-blue-400" />
            <span class="text-xs font-bold uppercase tracking-widest text-slate-400"
              >Avg Duration</span
            >
          </div>
          <div class="text-3xl font-bold text-white">{formatDuration(stats.avg_duration_ms)}</div>
        </div>

        <div
          class="rounded-xl border border-white/10 bg-gradient-to-br from-white/[0.03] to-white/[0.01] p-6 backdrop-blur-xl"
        >
          <div class="mb-2 flex items-center gap-2">
            <TrendingUp size={20} class="text-green-400" />
            <span class="text-xs font-bold uppercase tracking-widest text-slate-400"
              >P95 Duration</span
            >
          </div>
          <div class="text-3xl font-bold text-white">{formatDuration(stats.p95_duration_ms)}</div>
        </div>

        <div
          class="rounded-xl border border-white/10 bg-gradient-to-br from-white/[0.03] to-white/[0.01] p-6 backdrop-blur-xl"
        >
          <div class="mb-2 flex items-center gap-2">
            <AlertCircle size={20} class="text-red-400" />
            <span class="text-xs font-bold uppercase tracking-widest text-slate-400"
              >Error Rate</span
            >
          </div>
          <div class="text-3xl font-bold text-white">{formatPercent(stats.error_rate)}</div>
          <div class="mt-1 text-xs text-slate-500">
            {formatNumber(stats.error_count)} errors
          </div>
        </div>
      </div>

      <!-- Duration Percentiles -->
      <div
        class="mb-8 rounded-xl border border-white/10 bg-gradient-to-br from-white/[0.03] to-white/[0.01] p-6 backdrop-blur-xl"
      >
        <h2
          class="mb-6 flex items-center gap-2 text-sm font-bold uppercase tracking-widest text-slate-400"
        >
          <BarChart3 size={16} />
          Duration Percentiles
        </h2>
        <div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <div>
            <div class="text-xs text-slate-500 mb-1">P50 (Median)</div>
            <div class="text-xl font-bold text-white">{formatDuration(stats.p50_duration_ms)}</div>
          </div>
          <div>
            <div class="text-xs text-slate-500 mb-1">P75</div>
            <div class="text-xl font-bold text-white">{formatDuration(stats.p75_duration_ms)}</div>
          </div>
          <div>
            <div class="text-xs text-slate-500 mb-1">P95</div>
            <div class="text-xl font-bold text-white">{formatDuration(stats.p95_duration_ms)}</div>
          </div>
          <div>
            <div class="text-xs text-slate-500 mb-1">P99</div>
            <div class="text-xl font-bold text-white">{formatDuration(stats.p99_duration_ms)}</div>
          </div>
        </div>
        <div class="mt-4 grid grid-cols-2 gap-4">
          <div>
            <div class="text-xs text-slate-500 mb-1">Min</div>
            <div class="text-lg font-semibold text-white">{formatDuration(stats.min_duration_ms)}</div>
          </div>
          <div>
            <div class="text-xs text-slate-500 mb-1">Max</div>
            <div class="text-lg font-semibold text-white">{formatDuration(stats.max_duration_ms)}</div>
          </div>
        </div>
      </div>

      <!-- Time Series Chart -->
      {#if chartData.labels.length > 0}
        <div
          class="mb-8 rounded-xl border border-white/10 bg-gradient-to-br from-white/[0.03] to-white/[0.01] p-6 backdrop-blur-xl"
        >
          <h2
            class="mb-6 flex items-center gap-2 text-sm font-bold uppercase tracking-widest text-slate-400"
          >
            <TrendingUp size={16} />
            Traces Over Time
          </h2>
          <div class="h-64">
            <svg viewBox="0 0 800 200" class="w-full h-full">
              <!-- Grid lines -->
              {#each Array(5) as _, i}
                {@const y = (i * 200) / 4}
                <line x1="0" y1={y} x2="800" y2={y} stroke="rgba(255,255,255,0.05)" stroke-width="1" />
              {/each}

              <!-- P95 line -->
              {#if chartData.p95Ms.length > 0}
                <polyline
                  points={chartData.p95Ms.map((ms, i) => `${i * stepX},${200 - (ms / maxMs) * 200}`).join(" ")}
                  fill="none"
                  stroke="rgb(251, 146, 60)"
                  stroke-width="2"
                  opacity="0.7"
                />
              {/if}

              <!-- Average line -->
              {#if chartData.avgMs.length > 0}
                <polyline
                  points={chartData.avgMs.map((ms, i) => `${i * stepX},${200 - (ms / maxMs) * 200}`).join(" ")}
                  fill="none"
                  stroke="rgb(59, 130, 246)"
                  stroke-width="2"
                />
              {/if}

              <!-- Count bars -->
              {#each chartData.counts as count, i}
                {@const height = (count / maxCount) * 200}
                {@const x = i * stepX}
                <rect
                  x={x - stepX * 0.3}
                  y={200 - height}
                  width={stepX * 0.6}
                  height={height}
                  fill="rgba(139, 92, 246, 0.3)"
                />
              {/each}
            </svg>
          </div>
          <div class="mt-4 flex items-center gap-4 text-xs text-slate-400">
            <div class="flex items-center gap-2">
              <div class="h-3 w-3 rounded bg-blue-500"></div>
              <span>Average Duration</span>
            </div>
            <div class="flex items-center gap-2">
              <div class="h-3 w-3 rounded bg-orange-500"></div>
              <span>P95 Duration</span>
            </div>
            <div class="flex items-center gap-2">
              <div class="h-3 w-3 rounded bg-purple-500/30"></div>
              <span>Trace Count</span>
            </div>
          </div>
        </div>
      {/if}

      <!-- Operation Stats -->
      {#if operationStats.length > 0}
        <div
          class="mb-8 rounded-xl border border-white/10 bg-gradient-to-br from-white/[0.03] to-white/[0.01] p-6 backdrop-blur-xl"
        >
          <h2
            class="mb-6 flex items-center gap-2 text-sm font-bold uppercase tracking-widest text-slate-400"
          >
            <PieChart size={16} />
            Top Operations
          </h2>
          <div class="space-y-3">
            {#each operationStats as op}
              <div class="flex items-center justify-between rounded-lg border border-white/10 bg-white/5 p-4">
                <div class="flex-1">
                  <div class="mb-1 flex items-center gap-2">
                    <Zap size={14} class="text-pulse-400" />
                    <span class="font-semibold text-white">{op.operation || "unknown"}</span>
                  </div>
                  <div class="flex items-center gap-4 text-xs text-slate-400">
                    <span>{formatNumber(op.count)} traces</span>
                    <span>Avg: {formatDuration(op.avg_ms)}</span>
                    <span>P95: {formatDuration(op.p95_ms)}</span>
                  </div>
                </div>
                <div class="text-right">
                  <div class="text-sm font-bold text-red-400">{formatPercent(op.error_rate)}</div>
                  <div class="text-xs text-slate-500">error rate</div>
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/if}

      <!-- Status Distribution -->
      <div
        class="rounded-xl border border-white/10 bg-gradient-to-br from-white/[0.03] to-white/[0.01] p-6 backdrop-blur-xl"
      >
        <h2
          class="mb-6 flex items-center gap-2 text-sm font-bold uppercase tracking-widest text-slate-400"
        >
          <PieChart size={16} />
          Status Distribution
        </h2>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div class="rounded-lg border border-green-500/20 bg-green-500/10 p-4">
            <div class="text-xs text-green-400 mb-1">OK</div>
            <div class="text-2xl font-bold text-white">{formatNumber(stats.ok_count)}</div>
            <div class="text-xs text-slate-400 mt-1">
              {formatPercent((stats.ok_count / stats.total_traces) * 100)}
            </div>
          </div>
          <div class="rounded-lg border border-red-500/20 bg-red-500/10 p-4">
            <div class="text-xs text-red-400 mb-1">ERROR</div>
            <div class="text-2xl font-bold text-white">{formatNumber(stats.error_count)}</div>
            <div class="text-xs text-slate-400 mt-1">
              {formatPercent((stats.error_count / stats.total_traces) * 100)}
            </div>
          </div>
          <div class="rounded-lg border border-slate-500/20 bg-slate-500/10 p-4">
            <div class="text-xs text-slate-400 mb-1">UNKNOWN</div>
            <div class="text-2xl font-bold text-white">{formatNumber(stats.unknown_count)}</div>
            <div class="text-xs text-slate-400 mt-1">
              {formatPercent((stats.unknown_count / stats.total_traces) * 100)}
            </div>
          </div>
        </div>
      </div>
    {:else}
      <div class="flex h-96 items-center justify-center">
        <div class="text-center">
          <Activity size={48} class="mx-auto mb-4 text-slate-600" />
          <p class="text-slate-400">No trace data available</p>
        </div>
      </div>
    {/if}
  </div>
</div>

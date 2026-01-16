<script>
  import { onMount, onDestroy } from 'svelte';
  import { Link } from 'svelte-routing';
  import { api } from '../lib/api';
  import { getErrorLevelColor, getMonitorStatusColor } from '../lib/statusColors';
  import {
    Activity,
    AlertCircle,
    TrendingUp,
    PieChart,
    Clock,
    Zap,
    Shield,
    BarChart3,
    Eye,
    ArrowUpRight,
    ArrowDownRight,
    Minus
  } from 'lucide-svelte';

  let insights = null;
  let loading = true;
  let selectedProject = '';
  let projects = [];
  let timeRange = '7d';
  let refreshInterval = null;

  onMount(async () => {
    await Promise.all([loadProjects(), loadInsights()]);

    // Set up real-time polling every 30 seconds
    refreshInterval = setInterval(() => {
      if (!document.hidden) {
        loadInsights(false);
      }
    }, 30000);
  });

  onDestroy(() => {
    if (refreshInterval) {
      clearInterval(refreshInterval);
    }
  });

  async function loadProjects() {
    try {
      projects = await api.get('/projects') || [];
    } catch (error) {
      console.error('Failed to load projects:', error);
      projects = [];
    }
  }

  async function loadInsights(showLoading = true) {
    if (showLoading) {
      loading = true;
    }
    try {
      let url = `/insights?range=${timeRange}`;
      if (selectedProject) {
        url += `&projectId=${selectedProject}`;
      }
      insights = await api.get(url);
    } catch (error) {
      console.error('Failed to load insights:', error);
    } finally {
      if (showLoading) {
        loading = false;
      }
    }
  }

  function handleProjectChange() {
    loadInsights();
  }

  function handleTimeRangeChange() {
    loadInsights();
  }

  function formatNumber(num) {
    if (num == null) return '0';
    return new Intl.NumberFormat().format(num);
  }

  function formatPercentage(num) {
    if (num == null || isNaN(num)) return '0%';
    return `${num.toFixed(2)}%`;
  }

  function formatDuration(ms) {
    if (!ms) return '0ms';
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
  }

  function formatDate(dateString) {
    if (!dateString) return 'N/A';
    return new Date(dateString).toLocaleString();
  }

  function getTrendIcon(direction) {
    if (direction === 'up') return ArrowUpRight;
    if (direction === 'down') return ArrowDownRight;
    return Minus;
  }

  function getTrendColor(direction) {
    if (direction === 'up') return 'text-emerald-500';
    if (direction === 'down') return 'text-red-500';
    return 'text-slate-500';
  }
</script>

<div class="animate-in fade-in slide-in-from-bottom-4 duration-500">
  {#if loading}
    <div class="flex h-96 items-center justify-center">
      <div class="h-10 w-10 animate-spin rounded-full border-2 border-pulse-500 border-t-transparent"></div>
    </div>
  {:else if insights}
    <!-- Header -->
    <div class="mb-6 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h1 class="text-xl font-semibold tracking-tight text-white mb-0.5">Insights</h1>
        <p class="text-xs text-slate-400">Comprehensive overview of errors, traces, uptime, and coverage</p>
      </div>

      <div class="flex items-center gap-3">
        <!-- Project Filter -->
        <select
          bind:value={selectedProject}
          on:change={handleProjectChange}
          class="h-9 rounded-lg border border-white/[0.08] bg-white/[0.04] px-3 text-xs font-medium text-white outline-none focus:border-pulse-500/50 focus:ring-1 focus:ring-pulse-500/20 focus:bg-white/[0.06] transition-all"
        >
          <option value="">All Projects</option>
          {#each projects as project}
            <option value={project.id}>{project.name}</option>
          {/each}
        </select>

        <!-- Time Range Filter -->
        <div class="flex items-center gap-1 rounded-lg border border-white/[0.08] bg-white/[0.04] p-1">
          <button
            on:click={() => { timeRange = '24h'; handleTimeRangeChange(); }}
            class="px-3 py-1.5 text-xs font-medium rounded transition-all {timeRange === '24h' ? 'bg-pulse-500 text-white' : 'text-slate-400 hover:text-white'}"
          >
            24h
          </button>
          <button
            on:click={() => { timeRange = '7d'; handleTimeRangeChange(); }}
            class="px-3 py-1.5 text-xs font-medium rounded transition-all {timeRange === '7d' ? 'bg-pulse-500 text-white' : 'text-slate-400 hover:text-white'}"
          >
            7d
          </button>
          <button
            on:click={() => { timeRange = '30d'; handleTimeRangeChange(); }}
            class="px-3 py-1.5 text-xs font-medium rounded transition-all {timeRange === '30d' ? 'bg-pulse-500 text-white' : 'text-slate-400 hover:text-white'}"
          >
            30d
          </button>
        </div>
      </div>
    </div>

    <!-- Summary Cards -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
      <!-- Errors Card -->
      <div class="pulse-card p-6">
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-3">
            <div class="p-2 rounded-lg bg-red-500/10 text-red-400">
              <AlertCircle size={20} />
            </div>
            <div>
              <div class="text-[10px] uppercase text-slate-500 font-bold tracking-wider">Errors</div>
              <div class="text-2xl font-bold text-white">{formatNumber(insights.errors?.total_errors || 0)}</div>
            </div>
          </div>
        </div>
        <div class="space-y-2">
          <div class="flex items-center justify-between text-xs">
            <span class="text-slate-500">Unresolved</span>
            <span class="font-bold text-white">{formatNumber(insights.errors?.by_status?.unresolved || 0)}</span>
          </div>
          <div class="flex items-center justify-between text-xs">
            <span class="text-slate-500">Resolved</span>
            <span class="font-bold text-emerald-400">{formatNumber(insights.errors?.by_status?.resolved || 0)}</span>
          </div>
        </div>
      </div>

      <!-- Traces Card -->
      <div class="pulse-card p-6">
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-3">
            <div class="p-2 rounded-lg bg-indigo-500/10 text-indigo-400">
              <Zap size={20} />
            </div>
            <div>
              <div class="text-[10px] uppercase text-slate-500 font-bold tracking-wider">Traces</div>
              <div class="text-2xl font-bold text-white">{formatNumber(insights.traces?.total_traces || 0)}</div>
            </div>
          </div>
        </div>
        <div class="space-y-2">
          <div class="flex items-center justify-between text-xs">
            <span class="text-slate-500">Avg Duration</span>
            <span class="font-bold text-white font-mono">{formatDuration(insights.traces?.avg_duration_ms || 0)}</span>
          </div>
        </div>
      </div>

      <!-- Uptime Card -->
      <div class="pulse-card p-6">
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-3">
            <div class="p-2 rounded-lg bg-emerald-500/10 text-emerald-400">
              <Activity size={20} />
            </div>
            <div>
              <div class="text-[10px] uppercase text-slate-500 font-bold tracking-wider">Uptime</div>
              <div class="text-2xl font-bold text-white">{formatPercentage(insights.uptime?.avg_uptime_7d || 0)}</div>
            </div>
          </div>
        </div>
        <div class="space-y-2">
          <div class="flex items-center justify-between text-xs">
            <span class="text-slate-500">Monitors</span>
            <span class="font-bold text-white">{formatNumber(insights.uptime?.total_monitors || 0)}</span>
          </div>
          <div class="flex items-center justify-between text-xs">
            <span class="text-slate-500">24h Uptime</span>
            <span class="font-bold text-emerald-400">{formatPercentage(insights.uptime?.avg_uptime_24h || 0)}</span>
          </div>
        </div>
      </div>

      <!-- Coverage Card -->
      <div class="pulse-card p-6">
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-3">
            <div class="p-2 rounded-lg bg-amber-500/10 text-amber-400">
              <PieChart size={20} />
            </div>
            <div>
              <div class="text-[10px] uppercase text-slate-500 font-bold tracking-wider">Coverage</div>
              <div class="text-2xl font-bold text-white">
                {selectedProject
                  ? formatPercentage(insights.coverage?.current_coverage || 0)
                  : formatPercentage(insights.coverage?.avg_coverage || 0)}
              </div>
            </div>
          </div>
        </div>
        <div class="space-y-2">
          {#if insights.coverage?.trend}
            {@const TrendIcon = getTrendIcon(insights.coverage.trend.direction)}
            {@const trendColor = getTrendColor(insights.coverage.trend.direction)}
            <div class="flex items-center justify-between text-xs">
              <span class="text-slate-500">Trend</span>
              <div class="flex items-center gap-1">
                <TrendIcon size={12} class={trendColor} />
                <span class="font-bold {trendColor}">
                  {Math.abs(insights.coverage.trend.delta).toFixed(1)}%
                </span>
              </div>
            </div>
          {/if}
          <div class="flex items-center justify-between text-xs">
            <span class="text-slate-500">History Points</span>
            <span class="font-bold text-white">{formatNumber(insights.coverage?.history_count || 0)}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Detailed Sections -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
      <!-- Error Breakdown -->
      <div class="pulse-card p-6">
        <h2 class="text-sm font-semibold text-white mb-4 flex items-center gap-2">
          <AlertCircle size={16} class="text-red-400" />
          Error Breakdown
        </h2>

        {#if insights.errors?.by_level}
          <div class="space-y-3">
            {#each Object.keys(insights.errors.by_level || {}) as level}
              {@const count = insights.errors.by_level[level] || 0}
              {@const percentage = insights.errors.total_errors > 0 ? (count / insights.errors.total_errors) * 100 : 0}
              <div>
                <div class="flex items-center justify-between mb-1.5">
                  <span class="text-xs font-medium text-slate-400 uppercase">{level}</span>
                  <span class="text-xs font-bold text-white">{formatNumber(count)} ({percentage.toFixed(1)}%)</span>
                </div>
                <div class="h-1.5 w-full rounded-full bg-white/5 overflow-hidden">
                  <div
                    class="h-full rounded-full transition-all duration-500 {level === 'error' || level === 'fatal' ? 'bg-red-500' : level === 'warning' ? 'bg-amber-500' : 'bg-blue-500'}"
                    style="width: {percentage}%"
                  ></div>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Uptime Breakdown -->
      <div class="pulse-card p-6">
        <h2 class="text-sm font-semibold text-white mb-4 flex items-center gap-2">
          <Activity size={16} class="text-emerald-400" />
          Uptime Overview
        </h2>

        <div class="space-y-4">
          <div>
            <div class="flex items-center justify-between mb-1.5">
              <span class="text-xs font-medium text-slate-400">24 Hours</span>
              <span class="text-sm font-bold text-white">{formatPercentage(insights.uptime?.avg_uptime_24h || 0)}</span>
            </div>
            <div class="h-2 w-full rounded-full bg-white/5 overflow-hidden">
              <div
                class="h-full rounded-full bg-emerald-500 transition-all duration-500"
                style="width: {insights.uptime?.avg_uptime_24h || 0}%"
              ></div>
            </div>
          </div>

          <div>
            <div class="flex items-center justify-between mb-1.5">
              <span class="text-xs font-medium text-slate-400">7 Days</span>
              <span class="text-sm font-bold text-white">{formatPercentage(insights.uptime?.avg_uptime_7d || 0)}</span>
            </div>
            <div class="h-2 w-full rounded-full bg-white/5 overflow-hidden">
              <div
                class="h-full rounded-full bg-emerald-500 transition-all duration-500"
                style="width: {insights.uptime?.avg_uptime_7d || 0}%"
              ></div>
            </div>
          </div>

          <div>
            <div class="flex items-center justify-between mb-1.5">
              <span class="text-xs font-medium text-slate-400">30 Days</span>
              <span class="text-sm font-bold text-white">{formatPercentage(insights.uptime?.avg_uptime_30d || 0)}</span>
            </div>
            <div class="h-2 w-full rounded-full bg-white/5 overflow-hidden">
              <div
                class="h-full rounded-full bg-emerald-500 transition-all duration-500"
                style="width: {insights.uptime?.avg_uptime_30d || 0}%"
              ></div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Recent Activity -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Recent Errors -->
      <div class="pulse-card p-6">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-sm font-semibold text-white flex items-center gap-2">
            <AlertCircle size={16} class="text-red-400" />
            Recent Errors
          </h2>
          <Link to="/issues" class="text-xs font-bold text-pulse-400 hover:text-white transition-colors">
            View All →
          </Link>
        </div>

        <div class="space-y-3">
          {#if insights.errors?.recent && insights.errors.recent.length > 0}
            {#each insights.errors.recent.slice(0, 5) as error}
              <Link
                to="/errors/{error.id}"
                class="group flex items-center gap-3 p-3 rounded-lg border border-white/[0.05] bg-white/[0.02] hover:bg-white/[0.05] transition-all"
              >
                <div class="h-2 w-2 rounded-full {error.level === 'error' || error.level === 'fatal' ? 'bg-red-500' : error.level === 'warning' ? 'bg-amber-500' : 'bg-blue-500'}"></div>
                <div class="flex-1 min-w-0">
                  <div class="text-xs font-semibold text-white truncate group-hover:text-pulse-400 transition-colors">
                    {error.message || 'No message'}
                  </div>
                  <div class="text-[10px] text-slate-500 mt-0.5">
                    {formatDate(error.created_at)} • {error.environment || 'N/A'}
                  </div>
                </div>
                <div class="text-xs font-bold text-slate-500">
                  {error.event_count || 1}x
                </div>
              </Link>
            {/each}
          {:else}
            <div class="text-center py-8 text-slate-500 text-sm">No recent errors</div>
          {/if}
        </div>
      </div>

      <!-- Recent Traces -->
      <div class="pulse-card p-6">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-sm font-semibold text-white flex items-center gap-2">
            <Zap size={16} class="text-indigo-400" />
            Recent Traces
          </h2>
          {#if selectedProject}
            <Link to="/projects/{selectedProject}/traces" class="text-xs font-bold text-pulse-400 hover:text-white transition-colors">
              View All →
            </Link>
          {/if}
        </div>

        <div class="space-y-3">
          {#if insights.traces?.recent && insights.traces.recent.length > 0}
            {#each insights.traces.recent.slice(0, 5) as trace}
              {@const duration = trace.start_timestamp && trace.timestamp
                ? new Date(trace.timestamp).getTime() - new Date(trace.start_timestamp).getTime()
                : 0}
              <div class="flex items-center gap-3 p-3 rounded-lg border border-white/[0.05] bg-white/[0.02]">
                <div class="h-2 w-2 rounded-full bg-indigo-500"></div>
                <div class="flex-1 min-w-0">
                  <div class="text-xs font-semibold text-white truncate">
                    {trace.description || trace.name || 'Unknown Transaction'}
                  </div>
                  <div class="text-[10px] text-slate-500 mt-0.5">
                    {trace.op || 'default'} • {formatDate(trace.timestamp)}
                  </div>
                </div>
                <div class="text-xs font-bold text-indigo-400 font-mono">
                  {formatDuration(duration)}
                </div>
              </div>
            {/each}
          {:else}
            <div class="text-center py-8 text-slate-500 text-sm">No recent traces</div>
          {/if}
        </div>
      </div>
    </div>

    <!-- Coverage History Chart -->
    {#if insights.coverage?.recent_history && insights.coverage.recent_history.length > 0}
      <div class="pulse-card p-6 mt-6">
        <h2 class="text-sm font-semibold text-white mb-4 flex items-center gap-2">
          <PieChart size={16} class="text-amber-400" />
          Coverage Trend
        </h2>

        <div class="h-32 w-full rounded-lg bg-black/40 p-4 border border-white/5">
          <svg viewBox="0 0 400 100" class="h-full w-full overflow-visible">
            {#if insights.coverage.recent_history && insights.coverage.recent_history.length > 0}
              {@const points = insights.coverage.recent_history.map((h, i) => {
                const x = (i / (insights.coverage.recent_history.length - 1)) * 380 + 10;
                const y = 90 - ((h.percentage / 100) * 70 + 10);
                return `${x},${y}`;
              }).join(' ')}
              <polyline
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              class="text-amber-500"
              points={points}
            />
            <polyline
              fill="none"
              stroke="currentColor"
              stroke-width="4"
              stroke-linecap="round"
              stroke-linejoin="round"
              class="text-amber-500 blur-sm opacity-20"
              points={points}
            />
            {/if}
          </svg>
        </div>

        <div class="mt-4 grid grid-cols-5 gap-2 text-center">
          {#each insights.coverage.recent_history.slice(-5) as entry}
            <div>
              <div class="text-xs font-bold text-white">{entry.percentage.toFixed(0)}%</div>
              <div class="text-[10px] text-slate-500">{new Date(entry.created_at).toLocaleDateString()}</div>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Monitors List -->
    {#if insights.uptime?.monitors && insights.uptime.monitors.length > 0}
      <div class="pulse-card p-6 mt-6">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-sm font-semibold text-white flex items-center gap-2">
            <Activity size={16} class="text-emerald-400" />
            Monitors
          </h2>
          {#if selectedProject}
            <Link to="/projects/{selectedProject}" class="text-xs font-bold text-pulse-400 hover:text-white transition-colors">
              Manage →
            </Link>
          {/if}
        </div>

        <div class="space-y-3">
          {#each insights.uptime.monitors as monitor}
            <div class="flex items-center justify-between p-3 rounded-lg border border-white/[0.05] bg-white/[0.02]">
              <div class="flex-1 min-w-0">
                <div class="text-xs font-semibold text-white truncate">{monitor.name}</div>
                <div class="text-[10px] text-slate-500 mt-0.5 font-mono truncate">{monitor.url}</div>
              </div>
              <div class="flex items-center gap-3">
                {#each [monitor] as m}
                  {@const statusColors = getMonitorStatusColor(m.status)}
                  <span class="text-xs font-bold {statusColors.text} uppercase">
                    {statusColors.icon} {m.status || 'unknown'}
                  </span>
                {/each}
                {#if monitor.last_checked_at}
                  <span class="text-[10px] text-slate-500">
                    {new Date(monitor.last_checked_at).toLocaleTimeString()}
                  </span>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}
</div>

<script>
  import { onMount, onDestroy } from "svelte";
  import { navigate } from "../lib/router";
  import Link from "../components/Link.svelte";
  import { api } from "../lib/api";
  import { toast } from "../stores/toast";
  import { getIssueStatusColor } from "../lib/statusColors";
  import {
    Activity,
    AlertCircle,
    TrendingUp,
    Shield,
    Terminal,
    PieChart,
    ArrowUpRight,
    Zap,
  } from "lucide-svelte";

  let projects = [];
  let recentErrors = [];
  let monitors = [];
  let loading = true;
  let stats = {
    totalEvents: 0,
    activeProjects: 0,
    criticalErrors: 0,
    healthScore: 100,
    totalMonitors: 0,
    upMonitors: 0,
    downMonitors: 0,
    avgUptime: 0,
  };

  let refreshInterval = null;

  // Refresh data when page becomes visible (e.g., returning from another page)
  function visibilityHandler() {
    if (!document.hidden) {
      loadData(false); // Background refresh
    }
  }

  onMount(async () => {
    await loadData(true); // Initial load
    document.addEventListener("visibilitychange", visibilityHandler);

    // Set up real-time polling every 30 seconds (optimized from 10s)
    refreshInterval = setInterval(() => {
      if (!document.hidden) {
        loadData(false); // Background refresh, no loading state
      }
    }, 30000);
  });

  onDestroy(() => {
    document.removeEventListener("visibilitychange", visibilityHandler);
    if (refreshInterval) {
      clearInterval(refreshInterval);
    }
  });

  let trends = [];
  $: maxTrendEvents = Math.max(
    50,
    ...(trends || []).map((t) => (t.traces || 0) + (t.errors || 0)),
  );

  async function loadData(showLoading = false) {
    if (showLoading) {
      loading = true;
    }
    try {
      const [projectsResponse, errorsResponse, insightsResponse] =
        await Promise.all([
          api.get("/projects", { ttl: 30000 }), // Cache for 30 seconds
          api.get("/errors?limit=20&use_cursor=true", { ttl: 10000 }), // Cache for 10 seconds
          api.get("/insights?range=24h", { ttl: 30000 }), // Cache for 30 seconds
        ]);

      projects = Array.isArray(projectsResponse)
        ? projectsResponse
        : projectsResponse?.data || [];

      // Handle both array response and object with data property
      const errorsList = Array.isArray(errorsResponse)
        ? errorsResponse
        : errorsResponse?.data || errorsResponse || [];

      recentErrors = Array.isArray(errorsList)
        ? errorsList.map((err) => {
            const project = (projects || []).find(
              (p) => p.id === err.project_id,
            );
            return { ...err, projectName: project ? project.name : "Unknown" };
          })
        : [];

      // Load monitors for all projects
      monitors = [];
      for (const project of projects) {
        try {
          const projectMonitors = await api
            .get(`/projects/${project.id}/monitors`)
            .catch(() => []);
          if (Array.isArray(projectMonitors)) {
            monitors.push(
              ...projectMonitors.map((m) => ({
                ...m,
                projectName: project.name,
              })),
            );
          }
        } catch (err) {
          // Ignore errors for individual projects
        }
      }

      // Use real insights data if available
      if (insightsResponse) {
        stats.totalEvents =
          (insightsResponse.errors?.total_errors || 0) +
          (insightsResponse.traces?.total_traces || 0);
        stats.criticalErrors =
          insightsResponse.errors?.by_level?.error ||
          insightsResponse.errors?.by_level?.fatal ||
          0;
        trends = insightsResponse.trends || [];
      } else {
        // Fallback to manual calculation
        stats.totalEvents = (projects || []).reduce(
          (sum, p) => sum + (p.current_month_events || 0),
          0,
        );
        stats.criticalErrors = (recentErrors || []).filter(
          (e) => e.level === "error" || e.level === "fatal",
        ).length;
      }

      stats.activeProjects = (projects || []).length;
      stats.totalMonitors = monitors.length;
      stats.upMonitors = monitors.filter((m) => m.status === "up").length;
      stats.downMonitors = monitors.filter((m) => m.status === "down").length;

      // Artificial health score based on error ratio and monitor status
      if (stats.totalEvents > 0) {
        const errorRatio =
          stats.criticalErrors / Math.max(1, stats.totalEvents);
        stats.healthScore = Math.max(
          0,
          Math.min(100, 100 - Math.round(errorRatio * 500)),
        ); // Scaled for visibility
      } else {
        stats.healthScore = 100;
      }

      // Adjust health score based on monitor status
      if (stats.totalMonitors > 0) {
        const monitorHealth = (stats.upMonitors / stats.totalMonitors) * 100;
        stats.healthScore = Math.round(
          stats.healthScore * 0.7 + monitorHealth * 0.3,
        );
      }
    } catch (error) {
      console.error("Failed to load data:", error);
      // Ensure arrays are never null
      projects = projects || [];
      recentErrors = recentErrors || [];
      monitors = monitors || [];
      if (showLoading) {
        toast.add("Failed to load dashboard data", "error");
      }
    } finally {
      if (showLoading) {
        loading = false;
      }
    }
  }

  function getHealthColor(score) {
    if (score >= 90) return "text-emerald-500";
    if (score >= 70) return "text-amber-500";
    return "text-red-500";
  }

  function getStatusColorClass(status) {
    const colors = getIssueStatusColor(status);
    return `${colors.bg} ${colors.text} ${colors.border} border`;
  }
</script>

<div class="space-y-6 animate-in fade-in slide-in-from-bottom-4 duration-500">
  <div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
    <div>
      <h1 class="text-xl font-semibold tracking-tight text-white mb-0.5">
        Dashboard
      </h1>
      <p class="text-xs text-slate-400">
        System-wide overview and health summary
      </p>
    </div>
    <div class="flex items-center gap-3">
      <Link
        to="/projects"
        class="pulse-button bg-white/5 text-xs text-white hover:bg-white/10 flex items-center gap-2 py-2 px-4"
      >
        <Shield size={14} />
        <span>Manage Projects</span>
      </Link>
    </div>
  </div>

  {#if loading}
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {#each Array(4) as _}
        <div class="pulse-card h-28 skeleton rounded-lg"></div>
      {/each}
    </div>
  {:else}
    <!-- Summary Stats -->
    <div
      class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-5 animate-fade-in"
    >
      <div
        role="button"
        tabindex="0"
        on:click={() => navigate("/projects")}
        on:keydown={(e) => e.key === "Enter" && navigate("/projects")}
        class="pulse-card relative overflow-hidden p-4 group cursor-pointer"
      >
        <div
          class="absolute -right-4 -top-4 text-emerald-500/5 transition-transform duration-300 group-hover:scale-105"
        >
          <Shield size={80} />
        </div>
        <div class="relative z-10">
          <div
            class="flex items-center gap-1.5 text-[10px] font-semibold text-slate-400 mb-2 uppercase tracking-wider"
          >
            <Shield size={12} class="text-emerald-400" />
            <span>Health Score</span>
          </div>
          <div
            class="text-3xl font-bold {getHealthColor(
              stats.healthScore,
            )} mb-0.5 tracking-tight leading-none"
          >
            {stats.healthScore}%
          </div>
          <div class="text-[10px] text-slate-500">System operational</div>
        </div>
      </div>

      <div
        role="button"
        tabindex="0"
        on:click={() => navigate("/projects")}
        on:keydown={(e) => e.key === "Enter" && navigate("/projects")}
        class="pulse-card relative overflow-hidden p-4 group cursor-pointer"
      >
        <div
          class="absolute -right-4 -top-4 text-pulse-500/5 transition-transform duration-300 group-hover:scale-105"
        >
          <Zap size={80} />
        </div>
        <div class="relative z-10">
          <div
            class="flex items-center gap-1.5 text-[10px] font-semibold text-slate-400 mb-2 uppercase tracking-wider"
          >
            <Zap size={12} class="text-pulse-400" />
            <span>Total Events</span>
          </div>
          <div
            class="text-3xl font-bold text-white mb-0.5 tracking-tight leading-none"
          >
            {stats.totalEvents.toLocaleString()}
          </div>
          <div class="text-[10px] text-slate-500">This month</div>
        </div>
      </div>

      <div
        role="button"
        tabindex="0"
        on:click={() => navigate("/issues")}
        on:keydown={(e) => e.key === "Enter" && navigate("/issues")}
        class="pulse-card relative overflow-hidden p-4 group cursor-pointer"
      >
        <div
          class="absolute -right-4 -top-4 text-red-500/5 transition-transform duration-300 group-hover:scale-105"
        >
          <AlertCircle size={80} />
        </div>
        <div class="relative z-10">
          <div
            class="flex items-center gap-1.5 text-[10px] font-semibold text-slate-400 mb-2 uppercase tracking-wider"
          >
            <AlertCircle size={12} class="text-red-400" />
            <span>Recent Errors</span>
          </div>
          <div
            class="text-3xl font-bold text-red-400 mb-0.5 tracking-tight leading-none"
          >
            {stats.criticalErrors}
          </div>
          <div class="text-[10px] text-slate-500">Last 20 events</div>
        </div>
      </div>

      <div
        role="button"
        tabindex="0"
        on:click={() => navigate("/projects")}
        on:keydown={(e) => e.key === "Enter" && navigate("/projects")}
        class="pulse-card relative overflow-hidden p-4 group cursor-pointer"
      >
        <div
          class="absolute -right-4 -top-4 text-slate-500/5 transition-transform duration-300 group-hover:scale-105"
        >
          <TrendingUp size={80} />
        </div>
        <div class="relative z-10">
          <div
            class="flex items-center gap-1.5 text-[10px] font-semibold text-slate-400 mb-2 uppercase tracking-wider"
          >
            <Activity size={12} class="text-slate-300" />
            <span>Active Projects</span>
          </div>
          <div
            class="text-3xl font-bold text-white mb-0.5 tracking-tight leading-none"
          >
            {stats.activeProjects}
          </div>
          <div class="text-[10px] text-slate-500">Currently tracked</div>
        </div>
      </div>

      <div
        role="button"
        tabindex="0"
        on:click={() => navigate("/projects")}
        on:keydown={(e) => e.key === "Enter" && navigate("/projects")}
        class="pulse-card relative overflow-hidden p-4 group cursor-pointer"
      >
        <div
          class="absolute -right-4 -top-4 text-emerald-500/5 transition-transform duration-300 group-hover:scale-105"
        >
          <Activity size={80} />
        </div>
        <div class="relative z-10">
          <div
            class="flex items-center gap-1.5 text-[10px] font-semibold text-slate-400 mb-2 uppercase tracking-wider"
          >
            <Activity size={12} class="text-emerald-400" />
            <span>Uptime Monitors</span>
          </div>
          <div
            class="text-3xl font-bold text-white mb-0.5 tracking-tight leading-none"
          >
            {stats.totalMonitors}
          </div>
          <div class="text-[10px] text-slate-500">
            {stats.upMonitors} up, {stats.downMonitors} down
          </div>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 gap-4 lg:grid-cols-12">
      <!-- Trend Visualization -->
      <div class="lg:col-span-8">
        <div class="pulse-card p-5 h-full min-h-[280px]">
          <div class="flex items-center justify-between mb-5">
            <h2
              class="flex items-center gap-2 text-base font-semibold text-white"
            >
              <TrendingUp size={16} class="text-emerald-500" />
              <span>Event Throughput</span>
            </h2>
            <div
              class="flex items-center gap-4 text-xs font-medium uppercase tracking-wider text-slate-500"
            >
              <span class="flex items-center gap-1.5"
                ><span class="h-2 w-2 rounded-full bg-emerald-500"></span> Success</span
              >
              <span class="flex items-center gap-1.5"
                ><span class="h-2 w-2 rounded-full bg-red-500"></span> Error</span
              >
            </div>
          </div>

          <!-- Chart with actual data -->
          <div class="relative">
            <div
              class="flex h-48 items-end gap-1.5 sm:gap-2 px-3 overflow-x-auto pb-8"
            >
              {#if (trends || []).length > 0}
                {#each trends as stat}
                  {@const successEvents = stat.traces}
                  {@const errorEvents = stat.errors}
                  {@const maxEvents = Math.max(
                    50,
                    ...trends.map((t) => t.traces + t.errors),
                  )}
                  {@const successHeight = (successEvents / maxEvents) * 100}
                  {@const errorHeight = (errorEvents / maxEvents) * 100}
                  {@const hourPart =
                    (stat.hour || "").split(" ")[1]?.substring(0, 5) || "--:--"}

                  <div
                    class="group relative flex-1 min-w-[20px] flex flex-col justify-end gap-1 hover:z-10"
                  >
                    <div
                      class="w-full bg-red-500/30 rounded-t-sm group-hover:bg-red-500/50 transition-all duration-200 group-hover:shadow-lg group-hover:shadow-red-500/20"
                      style="height: {errorHeight}%"
                      title="{errorEvents} errors"
                    ></div>
                    <div
                      class="w-full bg-emerald-500/50 rounded-t-sm group-hover:bg-emerald-500/70 transition-all duration-200 group-hover:shadow-lg group-hover:shadow-emerald-500/20"
                      style="height: {successHeight}%"
                      title="{successEvents} events"
                    ></div>
                    <div
                      class="absolute -bottom-7 left-1/2 -translate-x-1/2 text-[9px] font-mono text-slate-500 whitespace-nowrap"
                    >
                      {hourPart}
                    </div>
                    <!-- Tooltip on hover -->
                    <div
                      class="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-2 py-1 bg-slate-900/95 border border-white/10 rounded text-[10px] text-white whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none z-20 shadow-2xl"
                    >
                      <div class="font-bold mb-1 border-b border-white/10 pb-1">
                        {stat.hour}
                      </div>
                      <div class="text-emerald-400">{successEvents} traces</div>
                      <div class="text-red-400">{errorEvents} errors</div>
                    </div>
                  </div>
                {/each}
              {:else}
                <div
                  class="flex-1 h-full flex items-center justify-center text-slate-600 text-[10px] font-mono"
                >
                  No activity in the last 24 hours
                </div>
              {/if}
            </div>
            <!-- Y-axis labels -->
            <div
              class="absolute left-0 top-0 h-48 flex flex-col justify-between text-[9px] text-slate-600 font-mono px-1 opacity-50"
            >
              <span>{Math.round(maxTrendEvents)}</span>
              <span>{Math.round(maxTrendEvents * 0.75)}</span>
              <span>{Math.round(maxTrendEvents * 0.5)}</span>
              <span>{Math.round(maxTrendEvents * 0.25)}</span>
              <span>0</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Quick Actions / Status -->
      <div class="lg:col-span-4">
        <div class="space-y-4">
          <div class="pulse-card p-4">
            <h3
              class="flex items-center gap-2 text-xs font-semibold text-white mb-3"
            >
              <Terminal size={14} class="text-pulse-400" />
              <span>System Status</span>
            </h3>
            <div class="space-y-3">
              <div class="flex items-center justify-between">
                <span class="text-xs text-slate-400">Database</span>
                <span
                  class="flex items-center gap-1 text-[10px] font-bold text-emerald-500 uppercase tracking-wider"
                >
                  <span
                    class="h-1 w-1 rounded-full bg-emerald-500 animate-pulse"
                  ></span>
                  Operational
                </span>
              </div>
              <div class="flex items-center justify-between">
                <span class="text-xs text-slate-400">Ingestion Flow</span>
                <span
                  class="flex items-center gap-1 text-[10px] font-bold text-emerald-500 uppercase tracking-wider"
                >
                  <span
                    class="h-1 w-1 rounded-full bg-emerald-500 animate-pulse"
                  ></span>
                  Stable
                </span>
              </div>
              <div class="flex items-center justify-between">
                <span class="text-xs text-slate-400">Quota Server</span>
                <span
                  class="flex items-center gap-1 text-[10px] font-bold text-emerald-500 uppercase tracking-wider"
                >
                  <span
                    class="h-1 w-1 rounded-full bg-emerald-500 animate-pulse"
                  ></span>
                  Active
                </span>
              </div>
            </div>
          </div>

          <div class="pulse-card p-4">
            <h3
              class="flex items-center gap-2 text-xs font-semibold text-white mb-3"
            >
              <PieChart size={14} class="text-amber-400" />
              <span>Quota Alerts</span>
            </h3>
            <div class="space-y-3">
              {#each (projects || []).filter((p) => p.max_events_per_month > 0 && p.current_month_events / p.max_events_per_month > 0.8) as project}
                <div
                  class="p-2.5 rounded-lg bg-amber-500/10 border border-amber-500/20"
                >
                  <div class="flex justify-between items-center mb-1.5">
                    <span
                      class="text-[10px] font-semibold text-amber-500 uppercase tracking-wider"
                      >{project.name}</span
                    >
                    <span class="text-[9px] text-amber-200 font-mono"
                      >{Math.round(
                        (project.current_month_events /
                          project.max_events_per_month) *
                          100,
                      )}%</span
                    >
                  </div>
                  <div
                    class="h-0.5 w-full bg-amber-500/20 rounded-full overflow-hidden"
                  >
                    <div
                      class="h-full bg-amber-500"
                      style="width: {(project.current_month_events /
                        project.max_events_per_month) *
                        100}%"
                    ></div>
                  </div>
                </div>
              {:else}
                <p class="text-[10px] text-slate-500 text-center py-2">
                  No quota alerts
                </p>
              {/each}
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Recent Activity -->
    <div>
      <div class="flex items-center justify-between mb-3">
        <h2 class="text-sm font-semibold text-white">Recent Activity</h2>
        <Link
          to="/issues"
          class="text-xs font-medium text-pulse-400 hover:text-pulse-300 flex items-center gap-1 transition-colors"
        >
          <span>View all</span>
          <ArrowUpRight size={12} />
        </Link>
      </div>

      <div class="pulse-card overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full text-left text-xs">
            <thead
              class="border-b border-white/[0.08] bg-white/[0.02] text-[9px] font-semibold uppercase tracking-wider text-slate-500"
            >
              <tr>
                <th class="px-3 py-2">Status</th>
                <th class="px-3 py-2">Issue</th>
                <th class="px-3 py-2 hidden md:table-cell">Project</th>
                <th class="px-3 py-2 text-right">Time</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-white/[0.06]">
              {#each recentErrors.slice(0, 5) as error}
                <tr
                  role="button"
                  tabindex="0"
                  class="group hover:bg-white/[0.03] transition-colors cursor-pointer"
                  on:click={() => navigate(`/errors/${error.id}`)}
                  on:keydown={(e) =>
                    e.key === "Enter" && navigate(`/errors/${error.id}`)}
                >
                  <td class="px-3 py-2.5">
                    {#if error.status}
                      {@const statusColors = getIssueStatusColor(error.status)}
                      <span
                        class="rounded-full px-2 py-0.5 text-[9px] font-bold uppercase tracking-tight {getStatusColorClass(
                          error.status,
                        )}"
                      >
                        {statusColors.icon}
                        {error.status}
                      </span>
                    {:else}
                      {@const statusColors = getIssueStatusColor("unresolved")}
                      <span
                        class="rounded-full px-2 py-0.5 text-[9px] font-bold uppercase tracking-tight {getStatusColorClass(
                          'unresolved',
                        )}"
                      >
                        {statusColors.icon} unresolved
                      </span>
                    {/if}
                  </td>
                  <td class="px-3 py-2.5">
                    <div class="flex flex-col">
                      <Link
                        to="/errors/{error.id}"
                        class="font-medium text-white group-hover:text-pulse-400 transition-colors text-xs line-clamp-1"
                      >
                        {error.message || "Unknown Error"}
                      </Link>
                      <span class="text-[9px] text-slate-500 md:hidden"
                        >{error.projectName}</span
                      >
                    </div>
                  </td>
                  <td
                    class="px-3 py-2.5 text-slate-400 text-xs hidden md:table-cell"
                    >{error.projectName}</td
                  >
                  <td
                    class="px-3 py-2.5 text-right text-[9px] text-slate-500 tabular-nums"
                  >
                    {new Date(error.created_at).toLocaleTimeString([], {
                      hour: "2-digit",
                      minute: "2-digit",
                    })}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  {/if}
</div>

<script>
  import { onMount } from 'svelte';
  import { CheckCircle2, XCircle, Clock, Activity } from 'lucide-svelte';
  import { getMonitorStatusColor } from '../lib/statusColors';

  let statusData = null;
  let loading = true;
  let projectId = '';

  onMount(async () => {
    const path = window.location.pathname;
    const match = path.match(/\/status\/([^\/]+)/);
    if (match) {
      projectId = match[1];
      await loadStatus();
    } else {
      loading = false;
    }
  });

  async function loadStatus() {
    try {
      const response = await fetch(`/api/status/${projectId}`);
      if (!response.ok) {
        throw new Error(`Failed to load status: ${response.status}`);
      }
      const data = await response.json();
      statusData = data;
    } catch (error) {
      console.error('Failed to load status:', error);
      statusData = null;
    } finally {
      loading = false;
    }
  }

  function getStatusColor(status) {
    const colors = getMonitorStatusColor(status);
    return colors.text;
  }

  function getStatusBg(status) {
    const colors = getMonitorStatusColor(status);
    return `${colors.bg} ${colors.border} border`;
  }

  function formatUptime(percentage) {
    if (percentage == null || isNaN(percentage) || percentage === 0) return 'N/A';
    return `${percentage.toFixed(2)}%`;
  }

  function formatDate(dateString) {
    if (!dateString) return 'N/A';
    try {
      return new Date(dateString).toLocaleString();
    } catch (e) {
      return dateString;
    }
  }

  function formatResponseTime(ms) {
    if (ms == null || ms === undefined) return 'N/A';
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
  }
</script>

<div class="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 p-4 sm:p-6 lg:p-8">
  <div class="mx-auto max-w-6xl">
    {#if loading}
      <div class="pulse-card p-12 text-center">
        <div class="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-pulse-400"></div>
        <p class="mt-4 text-slate-400">Loading status...</p>
      </div>
    {:else if !statusData || !statusData.monitors}
      <div class="pulse-card p-12 text-center">
        <XCircle size={48} class="mx-auto mb-4 text-slate-600" />
        <p class="text-slate-400">Status page not found</p>
        {#if statusData && !statusData.monitors}
          <p class="text-xs text-slate-500 mt-2">No monitors configured for this project</p>
        {/if}
      </div>
    {:else}
      <div class="mb-8">
        <h1 class="text-4xl font-bold text-white mb-2">
          {statusData.project?.name || 'Status Page'}
        </h1>
        <p class="text-slate-400">Real-time system status and uptime monitoring</p>
      </div>

      {#if !statusData.monitors || statusData.monitors.length === 0}
        <div class="pulse-card p-12 text-center">
          <Activity size={48} class="mx-auto mb-4 text-slate-500" />
          <p class="text-slate-400">No monitors configured</p>
          <p class="text-xs text-slate-500 mt-2">Configure monitors in your project settings to see status information</p>
        </div>
      {:else}
        <div class="space-y-6">
          {#each statusData.monitors as monitor}
          <div class="pulse-card p-6 hover:bg-white/[0.02] transition-colors">
            <div class="flex items-start justify-between mb-4">
              <div class="flex-1">
                <div class="flex items-center gap-3 mb-2">
                  <Activity size={20} class="text-pulse-400" />
                  <h2 class="text-xl font-bold text-white">{monitor.name}</h2>
                <span class="rounded px-2 py-0.5 text-xs font-bold {getStatusColor(monitor.status)} {getStatusBg(monitor.status)}">
                  {monitor.status}
                </span>
                </div>
                <p class="text-sm text-slate-400 font-mono">{monitor.url}</p>
              </div>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
              <div class="p-4 bg-white/5 rounded-lg border border-white/10">
                <div class="text-xs text-slate-500 mb-1">Uptime (24h)</div>
                <div class="text-2xl font-bold text-white">{formatUptime(monitor.uptime_24h)}</div>
              </div>
              <div class="p-4 bg-white/5 rounded-lg border border-white/10">
                <div class="text-xs text-slate-500 mb-1">Uptime (7d)</div>
                <div class="text-2xl font-bold text-white">{formatUptime(monitor.uptime_7d)}</div>
              </div>
              <div class="p-4 bg-white/5 rounded-lg border border-white/10">
                <div class="text-xs text-slate-500 mb-1">Uptime (30d)</div>
                <div class="text-2xl font-bold text-white">{formatUptime(monitor.uptime_30d)}</div>
              </div>
            </div>

            {#if monitor.recent_checks && monitor.recent_checks.length > 0}
              <div class="mt-6 pt-6 border-t border-white/10">
                <h3 class="text-sm font-bold text-slate-400 mb-3 uppercase tracking-wider">Recent Checks</h3>
                <div class="space-y-2 max-h-64 overflow-y-auto">
                  {#each monitor.recent_checks.slice(0, 20) as check}
                    <div class="flex items-center justify-between p-3 bg-white/5 rounded-lg border border-white/5 hover:bg-white/[0.08] transition-colors">
                      <div class="flex items-center gap-3 flex-1 min-w-0">
                        {#if check.status === 'up'}
                          <CheckCircle2 size={16} class="text-emerald-400 flex-shrink-0" />
                        {:else}
                          <XCircle size={16} class="text-red-400 flex-shrink-0" />
                        {/if}
                        <span class="text-sm text-slate-300 whitespace-nowrap">{formatDate(check.created_at)}</span>
                        {#if check.status_code}
                          <span class="text-xs text-slate-500 font-mono px-1.5 py-0.5 rounded bg-white/5">{check.status_code}</span>
                        {/if}
                      </div>
                      <div class="flex items-center gap-4 text-xs text-slate-400 flex-shrink-0">
                        {#if check.response_time != null}
                          <span class="flex items-center gap-1 whitespace-nowrap">
                            <Clock size={12} />
                            {formatResponseTime(check.response_time)}
                          </span>
                        {/if}
                        {#if check.error_message}
                          <span class="text-red-400 truncate max-w-xs">{check.error_message}</span>
                        {/if}
                      </div>
                    </div>
                  {/each}
                </div>
              </div>
            {/if}
          </div>
        {/each}
        </div>
      {/if}
    {/if}
  </div>
</div>

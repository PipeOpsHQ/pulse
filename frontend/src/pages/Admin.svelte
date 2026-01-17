<script>
  import { onMount } from "svelte";
  import { api } from "../lib/api";
  import {
    Database,
    Layers,
    Activity,
    Monitor,
    PieChart,
    Users,
    FileCode,
    Server,
    ShieldCheck,
  } from "lucide-svelte";
  import { toast } from "../stores/toast";

  let stats = $state(null);
  let loading = $state(true);

  async function loadStats() {
    loading = true;
    try {
      const response = await api.get("/admin/stats");
      stats = response;
    } catch (err) {
      console.error("Failed to load admin stats:", err);
      toast.add("Failed to load system statistics", "error");
    } finally {
      loading = false;
    }
  }

  onMount(loadStats);

  function formatBytes(bytes, decimals = 2) {
    if (bytes === 0) return "0 Bytes";
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = [
      "Bytes",
      "KiB",
      "MiB",
      "GiB",
      "TiB",
      "PiB",
      "EiB",
      "ZiB",
      "YiB",
    ];
    const i = Math.floor(Math.log(bytes) / Math.log(k));

    // Prevent index out of bounds if bytes is huge or calculation error
    const index = Math.min(i, sizes.length - 1);

    return `${parseFloat((bytes / Math.pow(k, index)).toFixed(dm))} ${sizes[index]}`;
  }
</script>

<div class="animate-in fade-in slide-in-from-bottom-4 duration-500">
  <div class="mb-8 flex items-center justify-between">
    <div>
      <h1 class="text-2xl font-bold tracking-tight text-white mb-1">
        System Administration
      </h1>
      <p class="text-sm text-slate-400">
        Resource monitoring and database statistics
      </p>
    </div>
    <button
      onclick={loadStats}
      class="flex items-center gap-2 rounded-lg bg-white/5 border border-white/10 px-4 py-2 text-xs font-semibold text-white hover:bg-white/10 transition-all"
    >
      <Activity size={14} class={loading ? "animate-pulse" : ""} />
      Refresh Data
    </button>
  </div>

  {#if loading && !stats}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      {#each Array(8) as _}
        <div class="h-32 pulse-card skeleton rounded-2xl"></div>
      {/each}
    </div>
  {:else if stats}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
      <!-- Database Size -->
      <div class="pulse-card p-6 flex items-center gap-4">
        <div
          class="h-12 w-12 rounded-xl bg-blue-500/10 flex items-center justify-center text-blue-400"
        >
          <Database size={24} />
        </div>
        <div>
          <div
            class="text-[10px] font-bold text-slate-500 uppercase tracking-widest mb-1"
          >
            DB Size
          </div>
          <div class="text-xl font-bold text-white">
            {formatBytes(stats.database_size)}
          </div>
        </div>
      </div>

      <!-- Projects -->
      <div class="pulse-card p-6 flex items-center gap-4">
        <div
          class="h-12 w-12 rounded-xl bg-purple-500/10 flex items-center justify-center text-purple-400"
        >
          <Layers size={24} />
        </div>
        <div>
          <div
            class="text-[10px] font-bold text-slate-500 uppercase tracking-widest mb-1"
          >
            Projects
          </div>
          <div class="text-xl font-bold text-white">{stats.projects}</div>
        </div>
      </div>

      <!-- Errors -->
      <div class="pulse-card p-6 flex items-center gap-4">
        <div
          class="h-12 w-12 rounded-xl bg-red-500/10 flex items-center justify-center text-red-400"
        >
          <Activity size={24} />
        </div>
        <div>
          <div
            class="text-[10px] font-bold text-slate-500 uppercase tracking-widest mb-1"
          >
            Total Errors
          </div>
          <div class="text-xl font-bold text-white">{stats.errors}</div>
        </div>
      </div>

      <!-- Trace Spans -->
      <div class="pulse-card p-6 flex items-center gap-4">
        <div
          class="h-12 w-12 rounded-xl bg-green-500/10 flex items-center justify-center text-green-400"
        >
          <PieChart size={24} />
        </div>
        <div>
          <div
            class="text-[10px] font-bold text-slate-500 uppercase tracking-widest mb-1"
          >
            Total Spans
          </div>
          <div class="text-xl font-bold text-white">{stats.spans}</div>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
      <!-- Detailed Stats -->
      <div class="pulse-card p-6">
        <h3
          class="flex items-center gap-2 text-sm font-bold text-white uppercase tracking-widest mb-6 border-b border-white/5 pb-4"
        >
          <Server size={16} class="text-pulse-400" />
          Component Breakdown
        </h3>

        <div class="space-y-4">
          <div
            class="flex items-center justify-between p-3 rounded-xl bg-white/[0.02] border border-white/5"
          >
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-lg bg-amber-500/10 text-amber-500">
                <Monitor size={18} />
              </div>
              <span class="text-sm text-slate-300 font-medium"
                >Uptime Monitors</span
              >
            </div>
            <span class="text-lg font-bold text-white">{stats.monitors}</span>
          </div>

          <div
            class="flex items-center justify-between p-3 rounded-xl bg-white/[0.02] border border-white/5"
          >
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-lg bg-pink-500/10 text-pink-500">
                <FileCode size={18} />
              </div>
              <span class="text-sm text-slate-300 font-medium"
                >Coverage Snapshots</span
              >
            </div>
            <span class="text-lg font-bold text-white"
              >{stats.coverage_snapshots}</span
            >
          </div>

          <div
            class="flex items-center justify-between p-3 rounded-xl bg-white/[0.02] border border-white/5"
          >
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-lg bg-cyan-500/10 text-cyan-500">
                <Users size={18} />
              </div>
              <span class="text-sm text-slate-300 font-medium"
                >Registered Users</span
              >
            </div>
            <span class="text-lg font-bold text-white">{stats.users}</span>
          </div>
        </div>
      </div>

      <!-- System Health / Path -->
      <div class="pulse-card p-6">
        <h3
          class="flex items-center gap-2 text-sm font-bold text-white uppercase tracking-widest mb-6 border-b border-white/5 pb-4"
        >
          <ShieldCheck size={16} class="text-emerald-400" />
          Engine Status
        </h3>

        <div class="space-y-6">
          <div>
            <div
              class="text-[10px] text-slate-500 uppercase tracking-wider mb-2"
            >
              Database Path
            </div>
            <div
              class="p-3 rounded-xl bg-black border border-white/10 font-mono text-xs text-pulse-400 break-all leading-relaxed"
            >
              {stats.database_path}
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div
              class="p-4 rounded-2xl bg-emerald-500/5 border border-emerald-500/10 text-center"
            >
              <div
                class="text-[10px] text-emerald-500/50 uppercase tracking-widest mb-1"
              >
                DB Connection
              </div>
              <div class="text-emerald-400 font-bold uppercase text-xs">
                Healthy
              </div>
            </div>
            <div
              class="p-4 rounded-2xl bg-blue-500/5 border border-blue-500/10 text-center"
            >
              <div
                class="text-[10px] text-blue-500/50 uppercase tracking-widest mb-1"
              >
                Storage Mode
              </div>
              <div class="text-blue-400 font-bold uppercase text-xs">
                Persistent
              </div>
            </div>
          </div>

          <p class="text-[10px] text-slate-500 italic text-center">
            Last updated: {new Date().toLocaleTimeString()}
          </p>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .skeleton {
    background: linear-gradient(
      90deg,
      rgba(255, 255, 255, 0.02) 25%,
      rgba(255, 255, 255, 0.05) 50%,
      rgba(255, 255, 255, 0.02) 75%
    );
    background-size: 200% 100%;
    animation: loading 1.5s infinite;
  }

  @keyframes loading {
    to {
      background-position-x: -200%;
    }
  }
</style>

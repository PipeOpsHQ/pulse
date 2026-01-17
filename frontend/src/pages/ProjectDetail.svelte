<script>
  import { onMount, onDestroy } from "svelte";
  import { navigate } from "../lib/router";
  import Link from "../components/Link.svelte";
  import { api } from "../lib/api";
  import { toast } from "../stores/toast";
  import { ensureHttps } from "../lib/utils";
  import {
    getErrorLevelColor,
    getMonitorStatusColor,
  } from "../lib/statusColors";
  import {
    Key,
    Link as LinkIcon,
    Copy,
    Eye,
    EyeOff,
    BookOpen,
    Terminal,
    Activity,
    Info,
    Calendar,
    ChevronRight,
    Search,
    Puzzle,
    Shield,
    PieChart,
    Timer,
    LayoutList,
    AlertCircle,
    Trash2,
  } from "lucide-svelte";

  let project = null;
  let errors = [];
  let coverageHistory = [];
  let activeTab = "overview";
  let traces = [];
  let loadingTraces = false;
  let detailedFiles = [];
  let selectedSnapshot = null;
  let showDetailedReport = false;
  let loadingDetails = false;
  let loading = true;
  let showApiKey = false;
  let showDSN = false;
  let refreshInterval = null;
  let monitors = [];
  let loadingMonitors = false;
  let showMonitorModal = false;
  let selectedMonitor = null;
  let newMonitor = {
    name: "",
    type: "http",
    url: "",
    interval: 60,
    timeout: 30,
  };

  // Check if project has any data
  $: hasData =
    project &&
    ((errors && errors.length > 0) ||
      (coverageHistory && coverageHistory.length > 0) ||
      (traces && traces.length > 0) ||
      (monitors && monitors.length > 0) ||
      (project.coverage && project.coverage > 0));

  $: dsn = project
    ? `https://${project.api_key}@${window.location.host}/${project.id}`
    : "";

  let projectId = "";

  async function loadProjectData(showLoading = false) {
    if (!projectId) {
      if (showLoading) {
        loading = false;
      }
      return;
    }

    if (showLoading) {
      loading = true;
    }
    try {
      const [projectData, errorsData, historyData, monitorsData] =
        await Promise.all([
          api.get(`/projects/${projectId}`),
          api.get(`/projects/${projectId}/errors`),
          api.get(`/projects/${projectId}/coverage/history`),
          api.get(`/projects/${projectId}/monitors`).catch(() => []),
        ]);

      project = projectData;
      errors = errorsData?.data || [];
      coverageHistory = historyData || [];
      monitors = Array.isArray(monitorsData) ? monitorsData : [];
    } catch (err) {
      console.error("Failed to load project:", err);
      if (showLoading) {
        toast.error("Failed to load project details");
      }
    } finally {
      if (showLoading) {
        loading = false;
      }
    }
  }

  onMount(async () => {
    const pathParts = window.location.pathname.split("/projects/");
    projectId = pathParts.length > 1 ? pathParts[1] : "";

    // Read tab from URL query params
    const urlParams = new URLSearchParams(window.location.search);
    const tabParam = urlParams.get("tab");
    if (
      tabParam &&
      ["overview", "coverage", "monitors", "traces"].includes(tabParam)
    ) {
      activeTab = tabParam;
    }

    await loadProjectData(true); // Initial load

    // Set up real-time polling every 10 seconds
    refreshInterval = setInterval(async () => {
      if (!document.hidden && projectId) {
        await loadProjectData(false); // Background refresh, no loading state
      }
    }, 10000);
  });

  onDestroy(() => {
    if (refreshInterval) {
      clearInterval(refreshInterval);
    }
  });

  function formatDate(dateString) {
    return new Date(dateString).toLocaleString();
  }

  function getLevelColorClass(level) {
    const colors = getErrorLevelColor(level);
    return `${colors.bg} ${colors.text} ${colors.border} border`;
  }

  function getMonitorStatusClass(status) {
    const colors = getMonitorStatusColor(status);
    return `${colors.bg} ${colors.text} ${colors.border} border`;
  }

  function copyText(text, label) {
    navigator.clipboard.writeText(text);
    toast.success(`${label} copied!`);
  }

  async function fetchDetailedReport(snapshot) {
    selectedSnapshot = snapshot;
    showDetailedReport = true;
    loadingDetails = true;
    try {
      const response = await api.get(
        `/projects/${projectId}/coverage/snapshots/${snapshot.id}/files`,
      );
      detailedFiles = response || [];
    } catch (err) {
      toast.error("Failed to load detailed report");
    } finally {
      loadingDetails = false;
    }
  }

  // Update URL when tab changes
  function setActiveTab(tab) {
    activeTab = tab;
    const url = new URL(window.location);
    url.searchParams.set("tab", tab);
    window.history.pushState({}, "", url);
  }

  async function loadTraces() {
    setActiveTab("traces");
    if (traces.length > 0) return;
    loadingTraces = true;
    try {
      const res = await api.get(`/projects/${project.id}/traces`);
      traces = res || [];
    } catch (err) {
      toast.fromHttpError(err);
    } finally {
      loadingTraces = false;
    }
  }

  async function loadMonitors(force = false) {
    setActiveTab("monitors");
    if (!force && monitors.length > 0 && !loadingMonitors) return;
    loadingMonitors = true;
    try {
      const res = await api.get(`/projects/${projectId}/monitors`);
      monitors = Array.isArray(res) ? res : [];
    } catch (err) {
      toast.fromHttpError(err);
    } finally {
      loadingMonitors = false;
    }
  }

  function openEditMonitor(monitor) {
    selectedMonitor = monitor;
    newMonitor = {
      name: monitor.name,
      type: monitor.type,
      url: monitor.url,
      interval: monitor.interval,
      timeout: monitor.timeout || 30,
    };
    showMonitorModal = true;
  }

  function openCreateMonitor() {
    selectedMonitor = null;
    newMonitor = { name: "", type: "http", url: "", interval: 60, timeout: 30 };
    showMonitorModal = true;
  }

  async function saveMonitor() {
    if (!newMonitor.name || !newMonitor.url) {
      toast.warning("Name and URL are required");
      return;
    }
    try {
      if (selectedMonitor) {
        // Update existing monitor
        await api.put(`/projects/${projectId}/monitors/${selectedMonitor.id}`, {
          name: newMonitor.name,
          type: newMonitor.type,
          url: ensureHttps(newMonitor.url),
          interval: newMonitor.interval,
          timeout: newMonitor.timeout,
        });
        toast.success("Monitor updated successfully");
      } else {
        // Create new monitor
        await api.post(`/projects/${projectId}/monitors`, {
          name: newMonitor.name,
          type: newMonitor.type,
          url: ensureHttps(newMonitor.url),
          interval: newMonitor.interval,
          timeout: newMonitor.timeout,
        });
        toast.success("Monitor created successfully");
      }
      showMonitorModal = false;
      selectedMonitor = null;
      newMonitor = {
        name: "",
        type: "http",
        url: "",
        interval: 60,
        timeout: 30,
      };
      await loadMonitors(true); // Force reload to show new monitor
    } catch (err) {
      toast.fromHttpError(err);
    }
  }

  async function deleteMonitor(monitorId) {
    if (!confirm("Are you sure you want to delete this monitor?")) return;
    try {
      await api.delete(`/projects/${projectId}/monitors/${monitorId}`);
      toast.success("Monitor deleted successfully");
      await loadMonitors(true); // Force reload to reflect deletion
    } catch (err) {
      toast.fromHttpError(err);
    }
  }

  function getStatusPageUrl() {
    if (!project) return "";
    return `https://${window.location.host}/status/${projectId}`;
  }

  let selectedTrace = null;
  let traceSpans = [];
  let loadingTraceDetails = false;

  async function openTrace(trace) {
    selectedTrace = trace;
    loadingTraceDetails = true;
    try {
      const res = await api.get(
        `/projects/${project.id}/traces/${trace.trace_id}`,
      );
      traceSpans = res || [];
    } catch (err) {
      toast.error("Failed to load trace details");
    } finally {
      loadingTraceDetails = false;
    }
  }

  $: curlCoverageExample = project
    ? `curl -X POST "https://${window.location.host}/api/${project.id}/coverage" \\
  -H "X-Pulse-Auth: ${project.api_key}" \\
  -H "Content-Type: application/json" \\
  -d '{"coverage": 84.5}'`
    : "";

  $: curlFileExample = project
    ? `curl -X POST "https://${window.location.host}/api/${project.id}/coverage" \\
  -H "X-Pulse-Auth: ${project.api_key}" \\
  -F "file=@coverage.out"`
    : "";

  $: sentryExample = project
    ? `import * as Sentry from "@sentry/browser";

Sentry.init({
  dsn: "${dsn}",
  environment: "production",
  tracesSampleRate: 1.0,
});`
    : "";

  $: curlErrorExample = project
    ? `curl -X POST "https://${window.location.host}/api/${project.id}/store/" \\
  -H "X-Sentry-Auth: Sentry sentry_key=${project.api_key}, sentry_version=7" \\
  -H "Content-Type: application/json" \\
  -d '{
  "message": "Test error message",
  "level": "error",
  "environment": "production"
}'`
    : "";

  $: curlSimpleExample = project
    ? `curl -X POST "https://${window.location.host}/api/${project.id}/store/" \\
  -H "X-Pulse-Auth: ${project.api_key}" \\
  -H "Content-Type: application/json" \\
  -d '{"message": "Test error", "level": "error"}'`
    : "";

  $: coverageDelta =
    coverageHistory.length >= 2
      ? (coverageHistory[0].percentage - coverageHistory[1].percentage).toFixed(
          1,
        )
      : null;

  $: chartPoints =
    coverageHistory.length < 2
      ? ""
      : (() => {
          // Create reversed array without mutation - iterate backwards
          const points = [];
          for (let i = coverageHistory.length - 1; i >= 0; i--) {
            points.push(coverageHistory[i]);
          }
          const width = 300;
          const height = 60;
          const padding = 5;

          return points
            .map((p, i) => {
              const x =
                (i / (points.length - 1)) * (width - 2 * padding) + padding;
              const y =
                height -
                ((p.percentage / 100) * (height - 2 * padding) + padding);
              return `${x},${y}`;
            })
            .join(" ");
        })();

  $: badgeMarkdown = project
    ? `[![Coverage State](https://${window.location.host}/api/projects/${project.id}/coverage/badge)](#)`
    : "";
</script>

<div class="animate-in fade-in slide-in-from-bottom-4 duration-500">
  {#if loading}
    <div class="flex h-96 items-center justify-center">
      <div
        class="h-10 w-10 animate-spin rounded-full border-2 border-pulse-500 border-t-transparent"
      ></div>
    </div>
  {:else if project}
    <div
      class="mb-6 flex flex-col sm:flex-row sm:items-center justify-between gap-3"
    >
      <div>
        <nav
          class="mb-2 flex items-center gap-2 text-xs font-medium text-slate-500"
        >
          <Link to="/" class="hover:text-white transition-colors"
            >Dashboard</Link
          >
          <ChevronRight size={12} />
          <Link to="/projects" class="hover:text-white transition-colors"
            >Projects</Link
          >
          <ChevronRight size={12} />
          <span class="text-pulse-400 font-bold">{project.name}</span>
        </nav>
        <h1 class="text-xl font-semibold tracking-tight text-white">
          {project.name}
        </h1>
      </div>

      <div
        class="flex items-center gap-2 rounded-xl bg-white/5 p-1 border border-white/5"
      >
        <button
          on:click={() => copyText(dsn, "DSN")}
          class="flex items-center gap-2 px-3 py-1.5 text-xs font-bold text-slate-300 hover:text-white hover:bg-white/5 rounded-lg transition-all"
        >
          <Copy size={14} />
          <span>Copy DSN</span>
        </button>
      </div>
    </div>

    <!-- Quick DSN Header -->
    <div
      class="mb-6 pulse-card border-pulse-500/20 bg-gradient-to-r from-pulse-600/10 to-transparent p-4 relative overflow-hidden group"
    >
      <div
        class="absolute -right-6 -top-6 text-pulse-500/10 transition-transform group-hover:scale-105"
      >
        <Terminal size={100} />
      </div>
      <div
        class="relative z-10 flex flex-col md:flex-row md:items-center justify-between gap-4"
      >
        <div class="space-y-0.5">
          <h3
            class="text-sm font-semibold text-white flex items-center gap-1.5"
          >
            <LinkIcon size={14} class="text-pulse-400" />
            Project DSN
          </h3>
          <p class="text-xs text-slate-400 max-w-xl">
            Use this DSN to connect Pulse to your application using
            Sentry-compatible SDKs.
          </p>
        </div>
        <div class="flex items-center gap-2 w-full md:w-auto">
          <div class="flex-1 md:w-80 group/dsn relative">
            <input
              type="text"
              readonly
              value={showDSN ? dsn : "••••••••••••••••••••••••••••••••"}
              class="w-full bg-black/60 border border-white/10 rounded-lg px-4 py-3 font-mono text-xs text-pulse-400 outline-none focus:border-pulse-500/50"
            />
            <button
              on:click={() => (showDSN = !showDSN)}
              class="absolute right-3 top-1/2 -translate-y-1/2 text-slate-500 hover:text-white transition-colors"
            >
              {#if showDSN}<EyeOff size={16} />{:else}<Eye size={16} />{/if}
            </button>
          </div>
          <button
            on:click={() => copyText(dsn, "DSN")}
            class="pulse-button-primary h-[42px] px-6 whitespace-nowrap"
          >
            Copy DSN
          </button>
        </div>
      </div>
    </div>

    <!-- Tabs -->
    <div class="mb-5 flex items-center gap-4 border-b border-white/[0.08]">
      <button
        class="pb-3 text-xs font-semibold uppercase tracking-wider transition-all {activeTab ===
        'overview'
          ? 'border-b-2 border-pulse-500 text-white'
          : 'text-slate-500 hover:text-white'}"
        on:click={() => setActiveTab("overview")}
      >
        Overview
      </button>
      <button
        class="pb-3 text-xs font-semibold uppercase tracking-wider transition-all {activeTab ===
        'coverage'
          ? 'border-b-2 border-pulse-500 text-white'
          : 'text-slate-500 hover:text-white'}"
        on:click={() => setActiveTab("coverage")}
      >
        Coverage
      </button>
      <button
        class="pb-3 text-xs font-semibold uppercase tracking-wider transition-all {activeTab ===
        'traces'
          ? 'border-b-2 border-pulse-500 text-white'
          : 'text-slate-500 hover:text-white'}"
        on:click={() => {
          setActiveTab("traces");
          loadTraces();
        }}
      >
        Performance
      </button>
      <button
        class="pb-3 text-xs font-semibold uppercase tracking-wider transition-all {activeTab ===
        'monitors'
          ? 'border-b-2 border-pulse-500 text-white'
          : 'text-slate-500 hover:text-white'}"
        on:click={loadMonitors}
      >
        Uptime
      </button>
    </div>

    {#if activeTab === "overview"}
      <div
        class="grid grid-cols-1 gap-8 lg:grid-cols-12 animate-in fade-in duration-300"
      >
        <!-- Integration Guide (Left) - Show when no data -->
        {#if !hasData}
          <div class="lg:col-span-8 space-y-8">
            <div
              class="rounded-xl border border-white/10 bg-white/5 backdrop-blur-xl overflow-hidden"
            >
              <div
                class="border-b border-white/10 bg-white/5 p-6 flex items-center justify-between"
              >
                <h2
                  class="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-white"
                >
                  <BookOpen size={16} class="text-pulse-400" />
                  <span>Getting Started - Integration Guide</span>
                </h2>
                <div class="flex items-center gap-2">
                  <div
                    class="h-2 w-2 rounded-full bg-amber-500 animate-pulse"
                  ></div>
                  <span
                    class="text-[10px] font-bold text-amber-400 uppercase tracking-widest"
                    >Waiting for data</span
                  >
                </div>
              </div>

              <div class="p-6 space-y-8 text-sm">
                <!-- DSN -->
                <div>
                  <h4 class="mb-2 font-semibold text-white">
                    Sentry DSN (Recommended)
                  </h4>
                  <p class="mb-4 text-slate-400 leading-relaxed">
                    Use this DSN with any Sentry SDK (Svelte, React, Go, etc.)
                    for automatic error capturing.
                  </p>

                  <div class="flex items-center gap-2">
                    <div
                      class="relative flex-1 overflow-hidden rounded-lg border border-white/10 bg-black/40 px-4 py-2.5 font-mono text-xs text-pulse-400"
                    >
                      {#if showDSN}
                        {dsn}
                      {:else}
                        ••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••
                      {/if}
                    </div>
                    <button
                      class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-white/10 bg-white/5 text-slate-400 transition-all hover:bg-white/10 hover:text-white"
                      on:click={() => (showDSN = !showDSN)}
                    >
                      {#if showDSN}<EyeOff size={16} />{:else}<Eye
                          size={16}
                        />{/if}
                    </button>
                    <button
                      class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-pulse-600 text-white transition-all hover:bg-pulse-500"
                      on:click={() => copyText(dsn, "DSN")}
                    >
                      <Copy size={16} />
                    </button>
                  </div>
                </div>

                <!-- Code Example -->
                <div>
                  <h4 class="mb-2 font-semibold text-white">
                    JavaScript/TypeScript SDK:
                  </h4>
                  <div
                    class="rounded-lg border border-white/10 bg-black/60 p-4"
                  >
                    <pre class="font-mono text-xs text-slate-300"><code
                        >{sentryExample}</code
                      ></pre>
                  </div>
                </div>

                <!-- cURL Examples -->
                <div class="space-y-4">
                  <div>
                    <h4 class="mb-2 font-semibold text-white text-sm">
                      Test with cURL (Sentry format):
                    </h4>
                    <div
                      class="rounded-lg border border-white/10 bg-black/60 p-4"
                    >
                      <pre
                        class="font-mono text-xs text-slate-300 overflow-x-auto"><code
                          >{curlErrorExample}</code
                        ></pre>
                    </div>
                  </div>
                  <div>
                    <h4 class="mb-2 font-semibold text-white text-sm">
                      Or use simple format:
                    </h4>
                    <div
                      class="rounded-lg border border-white/10 bg-black/60 p-4"
                    >
                      <pre
                        class="font-mono text-xs text-slate-300 overflow-x-auto"><code
                          >{curlSimpleExample}</code
                        ></pre>
                    </div>
                  </div>
                  <p class="text-[11px] text-slate-500">
                    Use either format to test your DSN connection. Check server
                    logs for debugging if it doesn't work.
                  </p>
                </div>

                <!-- Coverage Ingestion -->
                <div class="border-t border-white/5 pt-8">
                  <h4 class="mb-2 font-semibold text-white">
                    Project Coverage Ingestion
                  </h4>
                  <p class="mb-4 text-slate-400">
                    Push your test coverage percentage from your CI/CD pipeline.
                  </p>
                  <div
                    class="rounded-lg border border-white/10 bg-black/60 p-4"
                  >
                    <pre class="font-mono text-xs text-slate-300"><code
                        >{curlCoverageExample}</code
                      ></pre>
                  </div>
                </div>

                <div class="mt-4">
                  <h4 class="mb-2 font-semibold text-white text-sm">
                    Upload Raw Results (Go / LCOV):
                  </h4>
                  <p class="mb-3 text-[11px] text-slate-500">
                    Push raw `coverage.out` or `lcov.info` files for a detailed
                    file-by-file breakdown.
                  </p>
                  <div
                    class="rounded-lg border border-white/10 bg-black/60 p-4"
                  >
                    <pre class="font-mono text-xs text-slate-300"><code
                        >{curlFileExample}</code
                      ></pre>
                  </div>
                </div>
              </div>
            </div>
          </div>
        {:else}
          <!-- Integration Guide (Left) - Compact when data exists -->
          <div class="lg:col-span-8 space-y-8">
            <div
              class="rounded-xl border border-white/10 bg-white/5 backdrop-blur-xl overflow-hidden"
            >
              <div
                class="border-b border-white/10 bg-white/5 p-6 flex items-center justify-between"
              >
                <h2
                  class="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-white"
                >
                  <BookOpen size={16} class="text-pulse-400" />
                  <span>Integration Guide</span>
                </h2>
                <Link
                  to="/docs"
                  class="text-[10px] font-bold text-slate-500 hover:text-white transition-colors uppercase tracking-widest"
                  >Full Docs &rarr;</Link
                >
              </div>

              <div class="p-6 space-y-6 text-sm">
                <!-- DSN -->
                <div>
                  <h4 class="mb-2 font-semibold text-white">Sentry DSN</h4>
                  <div class="flex items-center gap-2">
                    <div
                      class="relative flex-1 overflow-hidden rounded-lg border border-white/10 bg-black/40 px-4 py-2.5 font-mono text-xs text-pulse-400"
                    >
                      {#if showDSN}
                        {dsn}
                      {:else}
                        ••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••
                      {/if}
                    </div>
                    <button
                      class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-white/10 bg-white/5 text-slate-400 transition-all hover:bg-white/10 hover:text-white"
                      on:click={() => (showDSN = !showDSN)}
                    >
                      {#if showDSN}<EyeOff size={16} />{:else}<Eye
                          size={16}
                        />{/if}
                    </button>
                    <button
                      class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-pulse-600 text-white transition-all hover:bg-pulse-500"
                      on:click={() => copyText(dsn, "DSN")}
                    >
                      <Copy size={16} />
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <!-- Recent Errors Table -->
            <div
              class="rounded-xl border border-white/10 bg-white/5 backdrop-blur-xl"
            >
              <div
                class="flex items-center justify-between border-b border-white/10 p-6"
              >
                <h2
                  class="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-white"
                >
                  <Activity size={16} class="text-red-400" />
                  <span>Recent Issues</span>
                </h2>
                <span
                  class="rounded-full bg-white/10 px-2.5 py-1 text-[10px] font-bold text-white tracking-widest uppercase"
                  >{errors.length} detected</span
                >
              </div>

              <div class="divide-y divide-white/5">
                {#if errors.length === 0}
                  <div class="flex flex-col items-center justify-center py-12">
                    <div class="mb-3 text-pulse-900"><Shield size={48} /></div>
                    <p class="text-sm text-slate-500">
                      Your perimeter is secure. No errors found.
                    </p>
                  </div>
                {:else}
                  {#each errors as error}
                    <Link
                      to="/errors/{error.id}"
                      class="group flex items-center gap-4 p-4 transition-all hover:bg-white/5"
                    >
                      {@const levelColors = getErrorLevelColor(error.level)}
                      <div
                        class="h-10 w-1 shrink-0 rounded-full {levelColors.dot}"
                      ></div>
                      <div class="min-w-0 flex-1">
                        <div
                          class="truncate text-sm font-semibold text-white group-hover:text-pulse-400 transition-colors uppercase tracking-tight"
                        >
                          {error.message || "No message"}
                        </div>
                        <div
                          class="mt-1 flex items-center gap-2 text-[10px] text-slate-500"
                        >
                          <span class="text-slate-400 font-bold"
                            >{error.environment || "PROD"}</span
                          >
                          <span>•</span>
                          <span>{formatDate(error.created_at)}</span>
                        </div>
                      </div>
                      <ChevronRight
                        size={14}
                        class="text-slate-600 opacity-0 transition-all group-hover:opacity-100 group-hover:translate-x-1"
                      />
                    </Link>
                  {/each}
                {/if}
              </div>
            </div>
          </div>
        {/if}

        <!-- Info Column (Right) -->
        <div class="lg:col-span-4 space-y-6">
          <div
            class="rounded-xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm"
          >
            <h3
              class="mb-6 flex items-center gap-2 text-xs font-bold uppercase tracking-widest text-slate-400"
            >
              <Info size={14} />
              <span>Project Settings</span>
            </h3>

            <div class="space-y-6">
              <div>
                <div class="mb-1 text-[10px] uppercase text-slate-500">
                  Project ID
                </div>
                <div class="font-mono text-sm text-white">{project.id}</div>
              </div>
              <div>
                <div class="mb-1 text-[10px] uppercase text-slate-500">
                  Created At
                </div>
                <div class="text-sm text-white">
                  {formatDate(project.created_at)}
                </div>
              </div>

              <div class="border-t border-white/5 pt-6">
                <div class="mb-3 flex items-center justify-between">
                  <span class="text-[10px] uppercase text-slate-500"
                    >Secret API Key</span
                  >
                  <span
                    class="flex h-5 w-5 items-center justify-center rounded-full bg-red-500/10 text-red-500"
                    ><Shield size={10} /></span
                  >
                </div>
                <div class="flex items-center gap-2">
                  <div
                    class="flex-1 overflow-hidden rounded-md border border-white/5 bg-black px-3 py-2 font-mono text-[10px] text-red-400"
                  >
                    {#if showApiKey}{project.api_key}{:else}••••••••••••••••••••••••{/if}
                  </div>
                  <button
                    class="flex h-8 w-8 shrink-0 items-center justify-center rounded border border-white/10 text-slate-500 hover:text-white"
                    on:click={() => (showApiKey = !showApiKey)}
                  >
                    {#if showApiKey}<EyeOff size={14} />{:else}<Eye
                        size={14}
                      />{/if}
                  </button>
                  <button
                    class="flex h-8 w-8 shrink-0 items-center justify-center rounded border border-white/10 text-slate-500 hover:text-white"
                    on:click={() => copyText(project.api_key, "API Key")}
                  >
                    <Copy size={14} />
                  </button>
                </div>
                <p class="mt-2 text-[10px] text-slate-600 leading-tight">
                  Keep this key secret. If compromised, regenerate it
                  immediately.
                </p>
              </div>
            </div>
          </div>

          <!-- System Stats -->
          <div
            class="rounded-xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm"
          >
            <div class="mb-4 flex items-center justify-between">
              <h3
                class="flex items-center gap-2 text-xs font-bold uppercase tracking-widest text-slate-400"
              >
                <Activity size={14} />
                <span>Performance</span>
              </h3>
              <span class="text-[10px] font-bold text-green-500 uppercase"
                >Live</span
              >
            </div>

            <div class="space-y-4">
              <div>
                <div
                  class="mb-1 flex justify-between text-[10px] text-slate-500"
                >
                  <span>Usage (Monthly)</span>
                  <span
                    >{Math.round(
                      (project.current_month_events /
                        project.max_events_per_month) *
                        100,
                    )}%</span
                  >
                </div>
                <div class="h-1.5 w-full rounded-full bg-white/5">
                  <div
                    class="h-full rounded-full {project.current_month_events >=
                    project.max_events_per_month
                      ? 'bg-red-500'
                      : 'bg-pulse-500'} shadow-[0_0_8px_rgba(139,92,246,0.3)]"
                    style="width: {Math.min(
                      100,
                      (project.current_month_events /
                        project.max_events_per_month) *
                        100,
                    )}%"
                  ></div>
                </div>
              </div>
              <div class="flex justify-between text-[10px]">
                <span class="text-slate-500">Events Recorded</span>
                <span class="text-white font-mono"
                  >{project.current_month_events} / {project.max_events_per_month}</span
                >
              </div>
            </div>
          </div>

          <!-- Coverage Stats -->
          <div
            class="rounded-xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm"
          >
            <div class="mb-4 flex items-center justify-between">
              <h3
                class="flex items-center gap-2 text-xs font-bold uppercase tracking-widest text-slate-400"
              >
                <PieChart size={14} />
                <span>Test Coverage</span>
              </h3>
              {#if project.coverage_updated_at}
                <span
                  class="text-[9px] font-medium text-slate-500 truncate max-w-[100px]"
                  >Last: {new Date(
                    project.coverage_updated_at,
                  ).toLocaleDateString()}</span
                >
              {/if}
            </div>

            <div class="space-y-4">
              <div>
                <div
                  class="mb-1 flex justify-between text-[10px] text-slate-500"
                >
                  <div class="flex items-center gap-2">
                    <span>Code Coverage</span>
                    {#if coverageDelta !== null}
                      <span
                        class="text-[10px] font-bold {parseFloat(
                          coverageDelta,
                        ) >= 0
                          ? 'text-emerald-500'
                          : 'text-red-500'}"
                      >
                        {parseFloat(coverageDelta) >= 0 ? "↑" : "↓"}
                        {Math.abs(coverageDelta)}%
                      </span>
                    {/if}
                  </div>
                  <span
                    class="{project.coverage >= 80
                      ? 'text-emerald-500'
                      : project.coverage >= 50
                        ? 'text-amber-500'
                        : 'text-red-500'} font-bold"
                    >{project.coverage || 0}%</span
                  >
                </div>
                <div class="h-1.5 w-full rounded-full bg-white/5">
                  <div
                    class="h-full rounded-full transition-all duration-1000 {project.coverage >=
                    80
                      ? 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.3)]'
                      : project.coverage >= 50
                        ? 'bg-amber-500 shadow-[0_0_8px_rgba(245,158,11,0.3)]'
                        : 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.3)]'}"
                    style="width: {project.coverage || 0}%"
                  ></div>
                </div>
              </div>

              <!-- Trend Chart -->
              {#if coverageHistory.length >= 1}
                <div class="space-y-3">
                  <div class="flex items-center justify-between">
                    <span class="text-[10px] uppercase text-slate-500"
                      >Trend (Last 30 runs)</span
                    >
                    <button
                      on:click={() => fetchDetailedReport(coverageHistory[0])}
                      class="text-[10px] font-bold text-pulse-400 hover:text-white transition-colors"
                    >
                      View Breakdown
                    </button>
                  </div>
                  {#if coverageHistory.length >= 2}
                    <div
                      class="h-16 w-full rounded-lg bg-black/40 p-1 border border-white/5"
                    >
                      <svg
                        viewBox="0 0 300 60"
                        class="h-full w-full overflow-visible"
                      >
                        <polyline
                          fill="none"
                          stroke="currentColor"
                          stroke-width="2"
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          class="text-pulse-500 opacity-50"
                          points={chartPoints}
                        />
                        <!-- Glow effect -->
                        <polyline
                          fill="none"
                          stroke="currentColor"
                          stroke-width="4"
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          class="text-pulse-500 blur-sm opacity-20"
                          points={chartPoints}
                        />
                      </svg>
                    </div>
                  {/if}
                </div>
              {/if}

              <div class="border-t border-white/5 pt-4">
                <div class="mb-3 flex items-center justify-between">
                  <span class="text-[10px] uppercase text-slate-500"
                    >Coverage Badge</span
                  >
                </div>
                <div class="flex items-center gap-2">
                  <img
                    src="/api/projects/{project.id}/coverage/badge"
                    alt="Coverage Badge"
                    class="h-5"
                    on:error={(e) => {
                      e.target.style.display = "none";
                    }}
                  />
                  <button
                    on:click={() => copyText(badgeMarkdown, "Badge Markdown")}
                    class="ml-auto flex items-center gap-2 text-[10px] font-bold text-pulse-400 hover:text-white transition-colors"
                  >
                    <Copy size={12} />
                    Copy Markdown
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    {/if}

    {#if activeTab === "coverage"}
      <div class="animate-in fade-in duration-300">
        <div
          class="rounded-xl border border-white/10 bg-white/5 backdrop-blur-xl overflow-hidden"
        >
          <div
            class="border-b border-white/10 bg-white/5 p-6 flex items-center justify-between"
          >
            <h2
              class="flex items-center gap-2 text-sm font-bold uppercase tracking-widest text-white"
            >
              <PieChart size={16} class="text-pulse-400" />
              <span>Test Coverage Reports</span>
            </h2>
          </div>

          <div class="p-6">
            {#if coverageHistory.length === 0}
              <div
                class="flex flex-col items-center justify-center py-12 text-center"
              >
                <div
                  class="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-pulse-500/10 text-pulse-400"
                >
                  <PieChart size={32} />
                </div>
                <h3 class="mb-2 text-base font-semibold text-white">
                  No Coverage Data Yet
                </h3>
                <p class="mb-6 text-sm text-slate-500 max-w-md">
                  Upload coverage reports from your CI/CD pipeline to track test
                  coverage over time.
                </p>
                <div
                  class="w-full max-w-2xl rounded-lg border border-white/10 bg-black/60 p-4"
                >
                  <p class="mb-2 text-xs font-semibold text-white">
                    Upload Coverage:
                  </p>
                  <pre
                    class="font-mono text-xs text-slate-300 overflow-x-auto"><code
                      >{curlCoverageExample}</code
                    ></pre>
                </div>
              </div>
            {:else}
              <div class="space-y-6">
                <!-- Current Coverage -->
                <div class="rounded-lg border border-white/10 bg-black/40 p-6">
                  <div class="flex items-center justify-between mb-4">
                    <div>
                      <span class="text-[10px] uppercase text-slate-500"
                        >Current Coverage</span
                      >
                      <div class="mt-1 flex items-baseline gap-2">
                        <span
                          class="{project.coverage >= 80
                            ? 'text-emerald-500'
                            : project.coverage >= 50
                              ? 'text-amber-500'
                              : 'text-red-500'} text-2xl font-bold"
                          >{project.coverage || 0}%</span
                        >
                        {#if coverageDelta !== null}
                          <span
                            class="text-sm font-bold {parseFloat(
                              coverageDelta,
                            ) >= 0
                              ? 'text-emerald-500'
                              : 'text-red-500'}"
                          >
                            {parseFloat(coverageDelta) >= 0 ? "↑" : "↓"}
                            {Math.abs(coverageDelta)}%
                          </span>
                        {/if}
                      </div>
                    </div>
                    {#if project.coverage_updated_at}
                      <span class="text-[10px] text-slate-500"
                        >Last updated: {new Date(
                          project.coverage_updated_at,
                        ).toLocaleDateString()}</span
                      >
                    {/if}
                  </div>
                  <div
                    class="h-3 w-full rounded-full bg-white/10 overflow-hidden"
                  >
                    <div
                      class="h-full rounded-full transition-all duration-1000 {project.coverage >=
                      80
                        ? 'bg-emerald-500'
                        : project.coverage >= 50
                          ? 'bg-amber-500'
                          : 'bg-red-500'}"
                      style="width: {project.coverage || 0}%"
                    ></div>
                  </div>
                </div>

                <!-- Coverage History -->
                <div>
                  <h3
                    class="mb-4 text-sm font-bold uppercase tracking-widest text-white"
                  >
                    Coverage History
                  </h3>
                  <div class="space-y-2">
                    {#each coverageHistory.slice(0, 20) as entry}
                      <div
                        class="flex items-center justify-between rounded-lg border border-white/10 bg-black/40 p-4 hover:bg-white/5 transition-colors"
                      >
                        <div class="flex items-center gap-4">
                          <div
                            class="flex h-10 w-10 items-center justify-center rounded-lg bg-pulse-500/10 text-pulse-400"
                          >
                            <PieChart size={18} />
                          </div>
                          <div>
                            <div class="text-sm font-semibold text-white">
                              {entry.percentage.toFixed(1)}%
                            </div>
                            <div class="text-[10px] text-slate-500">
                              {new Date(entry.created_at).toLocaleString()}
                            </div>
                          </div>
                        </div>
                        <button
                          on:click={() => fetchDetailedReport(entry)}
                          class="rounded-lg border border-white/10 bg-white/5 px-3 py-1.5 text-xs font-bold text-white hover:bg-white/10 transition-colors"
                        >
                          View Details
                        </button>
                      </div>
                    {/each}
                  </div>
                </div>

                <!-- Coverage Badge -->
                <div class="rounded-lg border border-white/10 bg-black/40 p-6">
                  <h3
                    class="mb-4 text-sm font-bold uppercase tracking-widest text-white"
                  >
                    Coverage Badge
                  </h3>
                  <div class="flex items-center gap-4">
                    <img
                      src="/api/projects/{project.id}/coverage/badge"
                      alt="Coverage Badge"
                      class="h-6"
                      on:error={(e) => {
                        e.target.style.display = "none";
                      }}
                    />
                    <button
                      on:click={() => copyText(badgeMarkdown, "Badge Markdown")}
                      class="ml-auto flex items-center gap-2 rounded-lg border border-white/10 bg-white/5 px-3 py-1.5 text-xs font-bold text-white hover:bg-white/10 transition-colors"
                    >
                      <Copy size={12} />
                      Copy Markdown
                    </button>
                  </div>
                </div>
              </div>
            {/if}
          </div>
        </div>
      </div>
    {/if}

    {#if activeTab === "traces"}
      <div class="animate-in fade-in duration-300">
        <div
          class="rounded-xl border border-white/10 bg-white/5 backdrop-blur-xl"
        >
          <div
            class="border-b border-white/10 p-6 flex items-center justify-between"
          >
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-lg bg-indigo-500/10 text-indigo-400">
                <Activity size={20} />
              </div>
              <div>
                <h2
                  class="text-xs font-semibold uppercase tracking-wider text-white"
                >
                  Transaction Traces
                </h2>
                <p class="text-xs text-slate-500">
                  Distributed tracing performance data
                </p>
              </div>
            </div>
            <button
              on:click={() => {
                traces = [];
                loadTraces();
              }}
              class="text-xs font-bold text-pulse-400 hover:text-white transition-colors"
            >
              Refresh
            </button>
          </div>

          {#if loadingTraces}
            <div class="flex flex-col items-center justify-center py-20">
              <div
                class="h-8 w-8 animate-spin rounded-full border-4 border-indigo-500/20 border-t-indigo-500"
              ></div>
            </div>
          {:else if traces.length === 0}
            <div
              class="flex flex-col items-center justify-center py-20 text-center"
            >
              <div class="mb-4 rounded-full bg-white/5 p-4 text-slate-500">
                <Timer size={32} />
              </div>
              <p class="text-slate-400 font-medium">No traces captured yet.</p>
              <p class="mt-2 text-xs text-slate-600 max-w-md">
                Configure your Sentry SDK to send performance data
                (transactions) effectively. Ensure <code>tracesSampleRate</code>
                is set to <code>1.0</code>.
              </p>
            </div>
          {:else}
            <div class="overflow-x-auto">
              <table class="w-full text-left">
                <thead>
                  <tr
                    class="border-b border-white/5 text-[10px] font-bold uppercase tracking-wider text-slate-500"
                  >
                    <th class="px-6 py-3">Transaction</th>
                    <th class="px-6 py-3">Op</th>
                    <th class="px-6 py-3">Duration</th>
                    <th class="px-6 py-3">Time</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-white/5">
                  {#each traces as trace}
                    <tr
                      class="group hover:bg-white/5 transition-colors cursor-pointer"
                      on:click={() => openTrace(trace)}
                    >
                      <td class="px-6 py-4">
                        <div
                          class="text-sm font-bold text-white group-hover:text-indigo-400 transition-colors"
                        >
                          {trace.description ||
                            trace.name ||
                            "Unknown Transaction"}
                        </div>
                        <div class="text-xs text-slate-500 font-mono mt-0.5">
                          {trace.trace_id.substring(0, 8)}...
                        </div>
                      </td>
                      <td class="px-6 py-4">
                        <span
                          class="inline-flex items-center rounded-md bg-white/10 px-2 py-1 text-xs font-medium text-slate-300 ring-1 ring-inset ring-white/10"
                          >{trace.op || "default"}</span
                        >
                      </td>
                      <td class="px-6 py-4">
                        <span class="text-sm font-mono text-emerald-400">
                          {(
                            new Date(trace.timestamp).getTime() -
                            new Date(trace.start_timestamp).getTime()
                          ).toFixed(0)}ms
                        </span>
                      </td>
                      <td
                        class="px-6 py-4 text-xs text-slate-400 whitespace-nowrap"
                      >
                        {new Date(trace.timestamp).toLocaleString()}
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        </div>
      </div>
    {/if}

    {#if activeTab === "monitors"}
      <div class="animate-in fade-in duration-300">
        <div
          class="rounded-xl border border-white/10 bg-white/5 backdrop-blur-xl"
        >
          <div
            class="border-b border-white/10 p-6 flex items-center justify-between"
          >
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-lg bg-emerald-500/10 text-emerald-400">
                <Timer size={20} />
              </div>
              <div>
                <h2
                  class="text-xs font-semibold uppercase tracking-wider text-white"
                >
                  Uptime Monitors
                </h2>
                <p class="text-xs text-slate-500">
                  Track service availability and response times
                </p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              {#if monitors.length > 0}
                <a
                  href={getStatusPageUrl()}
                  target="_blank"
                  class="px-4 py-2 text-xs rounded-lg border border-white/10 text-slate-300 hover:bg-white/5 hover:text-white transition-all flex items-center gap-2"
                >
                  <Eye size={14} />
                  View Status Page
                </a>
              {/if}
              <button
                on:click={openCreateMonitor}
                class="pulse-button-primary px-4 py-2 text-xs"
              >
                + Create Monitor
              </button>
            </div>
          </div>

          {#if monitors.length > 0}
            <div
              class="p-6 border-b border-white/10 bg-gradient-to-r from-emerald-500/5 to-transparent"
            >
              <div class="flex items-center justify-between">
                <div>
                  <h3 class="text-sm font-semibold text-white mb-1">
                    Status Page
                  </h3>
                  <p class="text-xs text-slate-400">
                    Public status page is automatically generated from your
                    monitors
                  </p>
                </div>
                <div class="flex items-center gap-2">
                  <input
                    type="text"
                    readonly
                    value={getStatusPageUrl()}
                    class="px-3 py-2 text-xs rounded-lg border border-white/10 bg-white/5 text-slate-300 font-mono w-64"
                  />
                  <button
                    on:click={() =>
                      copyText(getStatusPageUrl(), "Status page URL")}
                    class="p-2 rounded-lg border border-white/10 text-slate-400 hover:bg-white/5 hover:text-white transition-all"
                    title="Copy status page URL"
                  >
                    <Copy size={16} />
                  </button>
                </div>
              </div>
            </div>
          {/if}

          {#if loadingMonitors}
            <div class="flex flex-col items-center justify-center py-20">
              <div
                class="h-8 w-8 animate-spin rounded-full border-4 border-emerald-500/20 border-t-emerald-500"
              ></div>
            </div>
          {:else if monitors.length === 0}
            <div
              class="flex flex-col items-center justify-center py-20 text-center"
            >
              <div class="mb-4 rounded-full bg-white/5 p-4 text-slate-500">
                <Timer size={32} />
              </div>
              <p class="text-slate-400 font-medium">
                No monitors configured yet.
              </p>
              <p class="mt-2 text-xs text-slate-600 max-w-md">
                Create an uptime monitor to track the availability and response
                times of your services.
              </p>
              <button
                on:click={openCreateMonitor}
                class="mt-6 pulse-button-primary px-6 py-2.5 text-sm"
              >
                Create Your First Monitor
              </button>
            </div>
          {:else}
            <div class="p-6 space-y-4">
              {#each monitors as monitor}
                <div
                  class="rounded-lg border border-white/10 bg-black/40 p-5 hover:bg-white/[0.02] transition-colors"
                >
                  <div class="flex items-start justify-between mb-4">
                    <div class="flex-1 min-w-0">
                      <div class="flex items-center gap-3 mb-2">
                        <h3 class="text-sm font-semibold text-white truncate">
                          {monitor.name}
                        </h3>
                        <span
                          class="rounded-full px-2.5 py-0.5 text-[10px] font-bold uppercase {getMonitorStatusClass(
                            monitor.status,
                          )}"
                        >
                          {#if monitor.status === "up"}
                            ✓ {monitor.status}
                          {:else if monitor.status === "down"}
                            ✕ {monitor.status}
                          {:else}
                            {monitor.status || "unknown"}
                          {/if}
                        </span>
                      </div>
                      <p class="text-xs text-slate-400 font-mono truncate">
                        {monitor.url}
                      </p>
                    </div>
                    <div class="ml-4 flex items-center gap-2">
                      <button
                        on:click={() => openEditMonitor(monitor)}
                        class="rounded-lg p-2 text-slate-500 hover:bg-blue-500/10 hover:text-blue-400 transition-colors"
                        title="Edit monitor"
                      >
                        <Puzzle size={16} />
                      </button>
                      <button
                        on:click={() => deleteMonitor(monitor.id)}
                        class="rounded-lg p-2 text-slate-500 hover:bg-red-500/10 hover:text-red-400 transition-colors"
                        title="Delete monitor"
                      >
                        <Trash2 size={16} />
                      </button>
                    </div>
                  </div>

                  <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
                    <div
                      class="p-3 rounded-lg bg-white/5 border border-white/10"
                    >
                      <div
                        class="text-[10px] text-slate-500 mb-1 uppercase tracking-wider"
                      >
                        Type
                      </div>
                      <div class="text-xs font-bold text-pulse-400 uppercase">
                        {monitor.type || "http"}
                      </div>
                    </div>
                    <div
                      class="p-3 rounded-lg bg-white/5 border border-white/10"
                    >
                      <div
                        class="text-[10px] text-slate-500 mb-1 uppercase tracking-wider"
                      >
                        Interval
                      </div>
                      <div class="text-sm font-bold text-white">
                        {monitor.interval || 60}s
                      </div>
                    </div>
                    <div
                      class="p-3 rounded-lg bg-white/5 border border-white/10"
                    >
                      <div
                        class="text-[10px] text-slate-500 mb-1 uppercase tracking-wider"
                      >
                        Timeout
                      </div>
                      <div class="text-sm font-bold text-white">
                        {monitor.timeout || 30}s
                      </div>
                    </div>
                    <div
                      class="p-3 rounded-lg bg-white/5 border border-white/10"
                    >
                      <div
                        class="text-[10px] text-slate-500 mb-1 uppercase tracking-wider"
                      >
                        Last Checked
                      </div>
                      <div class="text-xs font-medium text-slate-300">
                        {#if monitor.last_checked_at}
                          {formatDate(monitor.last_checked_at)}
                        {:else}
                          Never
                        {/if}
                      </div>
                    </div>
                  </div>

                  {#if monitor.status === "down"}
                    <div
                      class="mt-4 p-3 rounded-lg bg-red-500/10 border border-red-500/20"
                    >
                      <div class="flex items-center gap-2 text-xs text-red-400">
                        <AlertCircle size={14} />
                        <span class="font-semibold"
                          >Service is currently down</span
                        >
                      </div>
                    </div>
                  {/if}
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    {/if}
  {:else}
    <div class="flex flex-col items-center justify-center py-20 text-center">
      <h2 class="text-lg font-semibold text-white">Project Not Found</h2>
      <Link to="/" class="mt-4 text-pulse-400 hover:underline"
        >Back to Dashboard</Link
      >
    </div>
  {/if}

  {#if showDetailedReport}
    <div
      class="fixed inset-0 z-[100] flex items-center justify-center p-4 sm:p-6"
    >
      <div
        class="absolute inset-0 bg-black/80 backdrop-blur-md"
        on:click={() => (showDetailedReport = false)}
        role="button"
        tabindex="0"
        on:keydown={(e) => e.key === "Escape" && (showDetailedReport = false)}
        aria-label="Close background"
      ></div>

      <div
        class="relative w-full max-w-4xl max-h-[85vh] overflow-hidden rounded-2xl border border-white/10 bg-[#0a0a0b] shadow-2xl flex flex-col transition-all duration-300"
      >
        <div
          class="flex items-center justify-between border-b border-white/10 p-6 bg-white/5"
        >
          <div class="flex items-center gap-4">
            <div
              class="flex h-12 w-12 items-center justify-center rounded-xl bg-pulse-500/10 text-pulse-400"
            >
              <PieChart size={24} />
            </div>
            <div>
              <h3 class="text-base font-semibold text-white">
                Coverage Breakdown
              </h3>
              <p class="text-xs text-slate-500">
                Detailed file analysis for snapshot from {new Date(
                  selectedSnapshot?.created_at,
                ).toLocaleString()}
              </p>
            </div>
          </div>
          <button
            on:click={() => (showDetailedReport = false)}
            class="rounded-lg p-2 text-slate-400 hover:bg-white/5 hover:text-white transition-all focus:outline-none"
            aria-label="Close modal"
          >
            <Search size={20} class="rotate-45" />
          </button>
        </div>

        <div
          class="flex-1 overflow-y-auto p-6 scrollbar-thin scrollbar-thumb-white/10"
        >
          {#if loadingDetails}
            <div
              class="flex flex-col items-center justify-center py-20 text-center"
            >
              <div
                class="h-12 w-12 animate-spin rounded-full border-4 border-pulse-500/20 border-t-pulse-500"
              ></div>
              <span class="mt-4 text-slate-400 font-medium"
                >Analysing snapshots...</span
              >
            </div>
          {:else if detailedFiles.length === 0}
            <div
              class="flex flex-col items-center justify-center py-20 text-center rounded-xl bg-white/[0.02] border border-dashed border-white/10"
            >
              <div
                class="flex h-12 w-12 items-center justify-center rounded-full bg-white/5 text-slate-500 mb-4"
              >
                <Search size={24} />
              </div>
              <span class="text-slate-500 italic"
                >No per-file mapping available.</span
              >
              <p class="mt-2 text-xs text-slate-600 max-w-sm">
                Detailed reports are generated automatically when uploading raw
                coverage files (coverage.out, lcov.info). Percentage-only
                uploads do not support drill-down view.
              </p>
            </div>
          {:else}
            <div class="overflow-x-auto">
              <table class="w-full text-left">
                <thead>
                  <tr
                    class="border-b border-white/5 text-[10px] font-bold uppercase tracking-wider text-slate-500"
                  >
                    <th class="pb-3 pr-4">File Path</th>
                    <th class="pb-3 pr-4 text-right">Coverage</th>
                    <th class="pb-3 w-32">Health</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-white/[0.02]">
                  {#each detailedFiles as file}
                    <tr class="group hover:bg-white/[0.02] transition-colors">
                      <td class="py-4 pr-4">
                        <span
                          class="text-sm font-medium text-slate-300 group-hover:text-white transition-colors block truncate max-w-md"
                          title={file.file_path}
                        >
                          {file.file_path}
                        </span>
                      </td>
                      <td class="py-4 pr-4 text-right">
                        <span
                          class="text-sm font-mono font-bold {file.percentage >=
                          80
                            ? 'text-emerald-500'
                            : file.percentage >= 50
                              ? 'text-amber-500'
                              : 'text-red-500'}"
                        >
                          {file.percentage.toFixed(1)}%
                        </span>
                      </td>
                      <td class="py-4">
                        <div class="h-1.5 w-full rounded-full bg-white/5">
                          <div
                            class="h-full rounded-full transition-all duration-1000 {file.percentage >=
                            80
                              ? 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.3)]'
                              : file.percentage >= 50
                                ? 'bg-amber-500 shadow-[0_0_8px_rgba(245,158,11,0.3)]'
                                : 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.3)]'}"
                            style="width: {file.percentage}%"
                          ></div>
                        </div>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        </div>
        <div class="bg-white/5 p-4 border-t border-white/10 flex justify-end">
          <button
            on:click={() => (showDetailedReport = false)}
            class="px-5 py-2 rounded-lg bg-pulse-600 text-white text-xs font-bold hover:bg-pulse-500 transition-all shadow-lg"
          >
            Close Breakdown
          </button>
        </div>
      </div>
    </div>
  {/if}

  {#if selectedTrace}
    <div
      class="fixed inset-0 z-[100] flex items-center justify-center p-4 sm:p-6"
    >
      <div
        class="absolute inset-0 bg-black/80 backdrop-blur-md"
        on:click={() => (selectedTrace = null)}
        role="button"
        tabindex="0"
        on:keydown={(e) => e.key === "Escape" && (selectedTrace = null)}
        aria-label="Close background"
      ></div>

      <div
        class="relative w-full max-w-5xl max-h-[90vh] overflow-hidden rounded-2xl border border-white/10 bg-[#0a0a0b] shadow-2xl flex flex-col"
      >
        <div
          class="flex items-center justify-between border-b border-white/10 p-6 bg-white/5"
        >
          <div class="flex items-center gap-4">
            <div
              class="flex h-12 w-12 items-center justify-center rounded-xl bg-indigo-500/10 text-indigo-400"
            >
              <Activity size={24} />
            </div>
            <div>
              <h3 class="text-base font-semibold text-white">
                {selectedTrace.description || selectedTrace.name}
              </h3>
              <div class="flex items-center gap-2 mt-1">
                <span class="text-xs font-mono text-slate-500"
                  >{selectedTrace.trace_id}</span
                >
                <span
                  class="rounded bg-white/10 px-1.5 py-0.5 text-[10px] font-medium text-slate-300"
                  >{selectedTrace.op}</span
                >
              </div>
            </div>
          </div>
          <button
            on:click={() => (selectedTrace = null)}
            class="rounded-lg p-2 text-slate-400 hover:bg-white/5 hover:text-white transition-all focus:outline-none"
            aria-label="Close modal"
          >
            <Search size={20} class="rotate-45" />
          </button>
        </div>

        <div class="flex-1 overflow-y-auto p-6">
          {#if loadingTraceDetails}
            <div class="flex flex-col items-center justify-center py-20">
              <div
                class="h-10 w-10 animate-spin rounded-full border-4 border-indigo-500/20 border-t-indigo-500"
              ></div>
            </div>
          {:else}
            <!-- Timeline Visualization -->
            <div class="relative space-y-2">
              {#each traceSpans as span}
                {@const duration =
                  new Date(span.timestamp).getTime() -
                  new Date(span.start_timestamp).getTime()}
                {@const totalDuration =
                  new Date(selectedTrace.timestamp).getTime() -
                    new Date(selectedTrace.start_timestamp).getTime() || 1}
                {@const startOffset =
                  new Date(span.start_timestamp).getTime() -
                  new Date(selectedTrace.start_timestamp).getTime()}
                {@const widthPercent = Math.max(
                  0.5,
                  (duration / totalDuration) * 100,
                )}
                {@const leftPercent = (startOffset / totalDuration) * 100}

                <div
                  class="group relative flex items-center gap-4 rounded-lg p-2 hover:bg-white/5 transition-colors"
                >
                  <div class="w-1/3 min-w-[200px] text-xs">
                    <div
                      class="font-bold text-slate-300 truncate"
                      title={span.description || span.name}
                    >
                      {span.description || span.name}
                    </div>
                    <div class="text-[10px] text-slate-500 font-mono mt-0.5">
                      {span.op}
                    </div>
                  </div>
                  <div class="flex-1 relative h-6">
                    <!-- Timeline Bar -->
                    <div
                      class="absolute top-1 h-4 rounded-full {span.span_id ===
                      selectedTrace.span_id
                        ? 'bg-indigo-500'
                        : 'bg-slate-600'} opacity-80 group-hover:opacity-100 transition-all border border-white/10"
                      style="left: {leftPercent}%; width: {widthPercent}%;"
                    ></div>
                    <span
                      class="absolute top-1 text-[10px] text-slate-400 ml-2 whitespace-nowrap"
                      style="left: {leftPercent + widthPercent}%;"
                    >
                      {duration.toFixed(1)}ms
                    </span>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    </div>
  {/if}

  <!-- Monitor Creation Modal -->
  {#if showMonitorModal}
    <div
      class="fixed inset-0 z-[100] flex items-center justify-center p-4 sm:p-6"
    >
      <div
        class="absolute inset-0 bg-black/80 backdrop-blur-md"
        on:click={() => (showMonitorModal = false)}
        role="button"
        tabindex="0"
        on:keydown={(e) => e.key === "Escape" && (showMonitorModal = false)}
        aria-label="Close background"
      ></div>

      <div
        class="relative w-full max-w-md overflow-hidden rounded-2xl border border-white/10 bg-[#0a0a0b] shadow-2xl"
      >
        <div
          class="flex items-center justify-between border-b border-white/10 p-6 bg-white/5"
        >
          <h3 class="text-base font-semibold text-white">
            {selectedMonitor ? "Edit Monitor" : "Create Uptime Monitor"}
          </h3>
          <button
            on:click={() => (showMonitorModal = false)}
            class="rounded-lg p-2 text-slate-400 hover:bg-white/5 hover:text-white transition-all"
          >
            <Search size={20} class="rotate-45" />
          </button>
        </div>

        <div class="p-6 space-y-4">
          <div>
            <label class="block text-xs font-medium text-slate-400 mb-2"
              >Monitor Name</label
            >
            <input
              type="text"
              bind:value={newMonitor.name}
              placeholder="e.g., Production API"
              class="pulse-input w-full"
            />
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-400 mb-2"
              >Monitor Type</label
            >
            <select bind:value={newMonitor.type} class="pulse-input w-full">
              <option value="http">HTTP/HTTPS</option>
              <option value="tcp">TCP</option>
              <option value="icmp">ICMP (Ping)</option>
              <option value="dns">DNS</option>
            </select>
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-400 mb-2">
              {#if newMonitor.type === "http" || newMonitor.type === "https"}
                URL
              {:else if newMonitor.type === "tcp"}
                Host:Port (e.g., example.com:3306)
              {:else if newMonitor.type === "icmp"}
                Hostname/IP (e.g., example.com or 8.8.8.8)
              {:else if newMonitor.type === "dns"}
                Hostname (e.g., example.com)
              {/if}
            </label>
            <input
              type="text"
              bind:value={newMonitor.url}
              placeholder={newMonitor.type === "http" ||
              newMonitor.type === "https"
                ? "https://api.example.com/health"
                : newMonitor.type === "tcp"
                  ? "example.com:3306"
                  : newMonitor.type === "icmp"
                    ? "example.com or 8.8.8.8"
                    : "example.com"}
              class="pulse-input w-full"
            />
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-xs font-medium text-slate-400 mb-2"
                >Interval (seconds)</label
              >
              <input
                type="number"
                bind:value={newMonitor.interval}
                min="30"
                max="3600"
                class="pulse-input w-full"
              />
            </div>
            <div>
              <label class="block text-xs font-medium text-slate-400 mb-2"
                >Timeout (seconds)</label
              >
              <input
                type="number"
                bind:value={newMonitor.timeout}
                min="5"
                max="300"
                class="pulse-input w-full"
              />
            </div>
          </div>
        </div>

        <div
          class="bg-white/5 p-4 border-t border-white/10 flex justify-end gap-3"
        >
          <button
            on:click={() => (showMonitorModal = false)}
            class="px-5 py-2 rounded-lg border border-white/10 text-xs font-bold text-slate-400 hover:text-white hover:bg-white/5 transition-all"
          >
            Cancel
          </button>
          <button on:click={saveMonitor} class="pulse-button-primary px-5 py-2">
            {selectedMonitor ? "Update Monitor" : "Create Monitor"}
          </button>
        </div>
      </div>
    </div>
  {/if}
</div>

<script>
  import { onMount } from "svelte";
  import { navigate } from "../lib/router";
  import Link from "../components/Link.svelte";
  import { api } from "../lib/api";
  import { toast } from "../stores/toast";
  import {
    CheckCircle2,
    EyeOff,
    Trash2,
    RotateCcw,
    ChevronLeft,
    Terminal,
    Info,
    User as UserIcon,
    Tag,
    Calendar,
    Hash,
    Layers,
    Activity,
    Code,
    Cpu,
    Globe,
    List,
    FileCode,
    GitBranch,
    Copy,
  } from "lucide-svelte";

  let error = null;
  let project = null;
  let loading = true;
  let activeTab = "details";
  let stacktrace = null;
  let context = null;
  let user = null;
  let tags = null;
  let stackTraceView = "frames"; // 'raw', 'frames', 'code', 'tree'

  onMount(async () => {
    const path = window.location.pathname;
    let errorId = "";

    if (path.includes("/issues/")) {
      const parts = path.split("/issues/");
      if (parts.length > 1) errorId = parts[1];
    } else if (path.includes("/errors/")) {
      const parts = path.split("/errors/");
      if (parts.length > 1) errorId = parts[1];
    }

    if (!errorId) {
      loading = false;
      return;
    }

    await loadError(errorId);
  });

  async function loadError(errorId) {
    try {
      error = await api.get(`/errors/${errorId}`);

      if (error) {
        if (error.project_id) {
          try {
            project = await api.get(`/projects/${error.project_id}`);
          } catch (e) {
            console.error("Failed to load project:", e);
          }
        }

        try {
          stacktrace = error.stacktrace
            ? JSON.parse(error.stacktrace)
            : { frames: [] };
        } catch (e) {
          stacktrace = { frames: [] };
        }
        try {
          context = error.context ? JSON.parse(error.context) : {};
        } catch (e) {
          context = {};
        }
        try {
          user = error.user ? JSON.parse(error.user) : {};
        } catch (e) {
          user = {};
        }
        try {
          tags = error.tags ? JSON.parse(error.tags) : {};
        } catch (e) {
          tags = {};
        }
      }
    } catch (err) {
      console.error("Failed to load error:", err);
      toast.fromHttpError(err);
    } finally {
      loading = false;
    }
  }

  async function updateStatus(newStatus) {
    if (!error) return;
    try {
      await api.patch(`/errors/${error.id}`, { status: newStatus });
      // Reload error data to ensure we have the latest state
      await loadError(error.id);
      toast.success(`Error marked as ${newStatus}`);
    } catch (err) {
      console.error("Failed to update status:", err);
      toast.fromHttpError(err);
    }
  }

  async function deleteError() {
    if (!error) return;
    if (!confirm("Are you sure you want to delete this error permanently?"))
      return;

    try {
      await api.delete(`/errors/${error.id}`);
      toast.success("Error deleted successfully");
      navigate("/issues");
    } catch (err) {
      console.error("Failed to delete error:", err);
      toast.fromHttpError(err);
    }
  }

  function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleString();
  }

  import { getErrorLevelColor, getIssueStatusColor } from "../lib/statusColors";

  function getLevelColorClass(level) {
    const colors = getErrorLevelColor(level);
    return `${colors.bg} ${colors.text} ${colors.border} border`;
  }

  function getStatusColorClass(status) {
    const colors = getIssueStatusColor(status);
    return `${colors.bg} ${colors.text} ${colors.border} border`;
  }

  function getFrames() {
    if (!stacktrace) return [];
    // Handle different stacktrace formats
    if (stacktrace.frames && Array.isArray(stacktrace.frames)) {
      return stacktrace.frames;
    }
    if (Array.isArray(stacktrace)) {
      return stacktrace;
    }
    return [];
  }

  function copyToClipboard(text) {
    navigator.clipboard.writeText(text);
    toast.success("Copied to clipboard");
  }

  function formatFilename(filename) {
    if (!filename) return "Unknown";
    // Extract just the filename from path
    const parts = filename.split("/");
    return parts[parts.length - 1];
  }

  function getInAppClass(inApp) {
    return inApp
      ? "bg-pulse-500/10 border-pulse-500/30"
      : "bg-slate-500/5 border-slate-500/10";
  }
</script>

<div class="animate-in fade-in slide-in-from-bottom-4 duration-500">
  {#if loading}
    <div class="flex h-96 items-center justify-center">
      <div
        class="h-10 w-10 animate-spin rounded-full border-2 border-pulse-500 border-t-transparent"
      ></div>
    </div>
  {:else if error}
    <!-- Detail Header -->
    <div
      class="mb-6 overflow-hidden rounded-lg border border-white/[0.08] bg-gradient-to-br from-white/[0.03] to-white/[0.01] backdrop-blur-xl"
    >
      <div class="p-4">
        <div class="mb-4 flex flex-wrap items-start justify-between gap-3">
          <div class="min-w-0 flex-1">
            <div class="mb-1.5 flex items-center gap-2">
              <span
                class="rounded px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wider {getLevelColorClass(
                  error.level,
                )}"
              >
                {error.level}
              </span>
              <span class="text-[9px] text-slate-500 font-mono"
                ># {error.id.substring(0, 8)}</span
              >
            </div>
            <h1
              class="truncate text-lg font-semibold tracking-tight text-white leading-tight"
            >
              {error.message || "No message"}
            </h1>
          </div>

          <div class="flex flex-wrap items-center gap-1.5">
            {#if error.status === "unresolved"}
              {@const statusColors = getIssueStatusColor(error.status)}
              <button
                class="inline-flex h-8 items-center gap-1.5 rounded-lg bg-green-500 px-3 text-xs font-semibold text-black transition-all hover:bg-green-400"
                on:click={() => updateStatus("resolved")}
              >
                <CheckCircle2 size={12} /> Resolve
              </button>
              <button
                class="inline-flex h-8 items-center gap-1.5 rounded-lg border border-white/[0.08] bg-white/[0.04] px-3 text-xs font-semibold text-slate-300 transition-all hover:bg-white/[0.08] hover:text-white"
                on:click={() => updateStatus("ignored")}
              >
                <EyeOff size={12} /> Ignore
              </button>
            {:else}
              <button
                class="inline-flex h-8 items-center gap-1.5 rounded-lg border border-white/[0.08] bg-white/[0.04] px-3 text-xs font-semibold text-slate-300 transition-all hover:bg-white/[0.08] hover:text-white"
                on:click={() => updateStatus("unresolved")}
              >
                <RotateCcw size={12} /> Unresolve
              </button>
              {#if error.status !== "resolved"}
                <button
                  class="inline-flex h-8 items-center gap-1.5 rounded-lg bg-green-500 px-3 text-xs font-semibold text-black transition-all hover:bg-green-400"
                  on:click={() => updateStatus("resolved")}
                >
                  <CheckCircle2 size={12} /> Resolve
                </button>
              {/if}
            {/if}
            <button
              class="inline-flex h-8 items-center gap-1.5 rounded-lg border border-red-500/20 bg-red-500/10 px-3 text-xs font-semibold text-red-500 transition-all hover:bg-red-500/20"
              on:click={deleteError}
            >
              <Trash2 size={12} /> Delete
            </button>
          </div>
        </div>

        <div class="flex items-center gap-4 border-t border-white/[0.08] pt-3">
          <button
            class="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider transition-all {activeTab ===
            'details'
              ? 'text-pulse-400'
              : 'text-slate-500 hover:text-white'}"
            on:click={() => (activeTab = "details")}
          >
            <Terminal size={12} /> Details
          </button>
          <button
            class="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider transition-all {activeTab ===
            'tags'
              ? 'text-pulse-400'
              : 'text-slate-500 hover:text-white'}"
            on:click={() => (activeTab = "tags")}
          >
            <Tag size={12} /> Tags
          </button>
          <button
            class="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider transition-all {activeTab ===
            'events'
              ? 'text-pulse-400'
              : 'text-slate-500 hover:text-white'}"
            on:click={() => (activeTab = "events")}
          >
            <Activity size={12} /> Events
          </button>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 gap-8 lg:grid-cols-12">
      <!-- Main Content -->
      <div class="lg:col-span-8 space-y-8">
        {#if activeTab === "details"}
          <!-- Stack Trace -->
          <div
            class="rounded-xl border border-white/10 bg-black/40 overflow-hidden"
          >
            <div
              class="flex items-center justify-between border-b border-white/10 bg-white/5 px-4 py-3"
            >
              <h2
                class="flex items-center gap-2 text-xs font-bold uppercase tracking-widest text-slate-400"
              >
                <Code size={14} class="text-pulse-400" />
                <span>Stack Trace</span>
              </h2>
              <div class="flex items-center gap-2">
                <!-- View Toggle Buttons -->
                <div
                  class="flex items-center gap-1 rounded-lg border border-white/10 bg-black/40 p-1"
                >
                  <button
                    class="px-2 py-1 text-[10px] font-bold uppercase transition-all {stackTraceView ===
                    'frames'
                      ? 'bg-pulse-500 text-white'
                      : 'text-slate-500 hover:text-white'}"
                    on:click={() => (stackTraceView = "frames")}
                    title="Frames View"
                  >
                    <List size={12} class="inline" />
                  </button>
                  <button
                    class="px-2 py-1 text-[10px] font-bold uppercase transition-all {stackTraceView ===
                    'code'
                      ? 'bg-pulse-500 text-white'
                      : 'text-slate-500 hover:text-white'}"
                    on:click={() => (stackTraceView = "code")}
                    title="Code View"
                  >
                    <FileCode size={12} class="inline" />
                  </button>
                  <button
                    class="px-2 py-1 text-[10px] font-bold uppercase transition-all {stackTraceView ===
                    'tree'
                      ? 'bg-pulse-500 text-white'
                      : 'text-slate-500 hover:text-white'}"
                    on:click={() => (stackTraceView = "tree")}
                    title="Tree View"
                  >
                    <GitBranch size={12} class="inline" />
                  </button>
                  <button
                    class="px-2 py-1 text-[10px] font-bold uppercase transition-all {stackTraceView ===
                    'raw'
                      ? 'bg-pulse-500 text-white'
                      : 'text-slate-500 hover:text-white'}"
                    on:click={() => (stackTraceView = "raw")}
                    title="Raw JSON"
                  >
                    <Code size={12} class="inline" />
                  </button>
                </div>
                <button
                  class="flex items-center gap-1 rounded-lg border border-white/10 bg-black/40 px-2 py-1 text-[10px] font-bold text-slate-400 hover:text-white transition-colors"
                  on:click={() =>
                    copyToClipboard(JSON.stringify(stacktrace, null, 2))}
                  title="Copy JSON"
                >
                  <Copy size={12} />
                </button>
              </div>
            </div>

            <div class="overflow-x-auto">
              {#if stackTraceView === "raw"}
                <!-- Raw JSON View -->
                <div class="p-6">
                  <pre
                    class="font-mono text-xs leading-relaxed text-slate-300"><code
                      >{JSON.stringify(stacktrace, null, 2)}</code
                    ></pre>
                </div>
              {:else if stackTraceView === "frames"}
                <!-- Frames List View -->
                <div class="divide-y divide-white/5">
                  {#each getFrames() as frame, index}
                    {@const isInApp = frame.in_app !== false}
                    <div
                      class="group p-4 transition-colors hover:bg-white/5 {getInAppClass(
                        isInApp,
                      )} border-l-4 {isInApp
                        ? 'border-pulse-500'
                        : 'border-slate-500/30'}"
                    >
                      <div class="flex items-start justify-between gap-4">
                        <div class="flex-1 min-w-0">
                          <div class="mb-1 flex items-center gap-2">
                            <span
                              class="font-mono text-xs font-bold text-pulse-400"
                              >#{getFrames().length - index}</span
                            >
                            {#if frame.function}
                              <span
                                class="font-mono text-sm font-semibold text-white"
                                >{frame.function}</span
                              >
                            {/if}
                            {#if isInApp}
                              <span
                                class="rounded bg-pulse-500/20 px-1.5 py-0.5 text-[9px] font-bold text-pulse-300"
                                >IN-APP</span
                              >
                            {/if}
                          </div>
                          {#if frame.filename}
                            <div
                              class="mb-2 flex items-center gap-2 text-xs text-slate-400"
                            >
                              <FileCode size={12} />
                              <span class="font-mono">{frame.filename}</span>
                              {#if frame.lineno}
                                <span class="text-pulse-400"
                                  >:{frame.lineno}</span
                                >
                              {/if}
                            </div>
                          {/if}
                          {#if frame.context_line}
                            <div
                              class="mt-2 rounded-lg border border-white/10 bg-black/60 p-3"
                            >
                              <pre
                                class="font-mono text-xs text-slate-300">{frame.context_line}</pre>
                            </div>
                          {/if}
                          {#if frame.vars && Object.keys(frame.vars).length > 0}
                            <div class="mt-2 space-y-1">
                              {#each Object.entries(frame.vars) as [key, value]}
                                <div class="flex items-start gap-2 text-xs">
                                  <span class="font-bold text-slate-500"
                                    >{key}:</span
                                  >
                                  <span
                                    class="font-mono text-slate-300 break-all"
                                    >{String(value)}</span
                                  >
                                </div>
                              {/each}
                            </div>
                          {/if}
                        </div>
                      </div>
                    </div>
                  {:else}
                    <div class="p-8 text-center">
                      <p class="text-sm text-slate-500">
                        No stack frames available
                      </p>
                    </div>
                  {/each}
                </div>
              {:else if stackTraceView === "code"}
                <!-- Code Context View -->
                <div class="divide-y divide-white/5">
                  {#each getFrames() as frame, index}
                    {@const isInApp = frame.in_app !== false}
                    <div
                      class="p-6 {getInAppClass(isInApp)} border-l-4 {isInApp
                        ? 'border-pulse-500'
                        : 'border-slate-500/30'}"
                    >
                      <div class="mb-4 flex items-center justify-between">
                        <div>
                          <div class="flex items-center gap-2 mb-1">
                            <span
                              class="font-mono text-xs font-bold text-pulse-400"
                              >Frame #{getFrames().length - index}</span
                            >
                            {#if frame.function}
                              <span
                                class="font-mono text-sm font-semibold text-white"
                                >{frame.function}</span
                              >
                            {/if}
                          </div>
                          {#if frame.filename}
                            <div class="text-xs text-slate-400 font-mono">
                              {frame.filename}{#if frame.lineno}:{frame.lineno}{/if}
                            </div>
                          {/if}
                        </div>
                        {#if isInApp}
                          <span
                            class="rounded bg-pulse-500/20 px-2 py-1 text-[9px] font-bold text-pulse-300"
                            >IN-APP</span
                          >
                        {/if}
                      </div>

                      {#if frame.pre_context || frame.context_line || frame.post_context}
                        <div
                          class="rounded-lg border border-white/10 bg-black/60 overflow-hidden"
                        >
                          <div class="overflow-x-auto">
                            <table class="w-full font-mono text-xs">
                              <tbody>
                                {#if frame.pre_context}
                                  {#each frame.pre_context as line, i}
                                    <tr class="text-slate-500">
                                      <td
                                        class="w-12 px-3 py-1 text-right select-none"
                                        >{frame.lineno
                                          ? frame.lineno -
                                            frame.pre_context.length +
                                            i
                                          : ""}</td
                                      >
                                      <td class="px-3 py-1">{line}</td>
                                    </tr>
                                  {/each}
                                {/if}
                                {#if frame.context_line}
                                  <tr
                                    class="bg-pulse-500/10 border-l-2 border-pulse-500"
                                  >
                                    <td
                                      class="w-12 px-3 py-1 text-right text-pulse-400 font-bold select-none"
                                      >{frame.lineno || ""}</td
                                    >
                                    <td
                                      class="px-3 py-1 text-white font-semibold"
                                      >{frame.context_line}</td
                                    >
                                  </tr>
                                {/if}
                                {#if frame.post_context}
                                  {#each frame.post_context as line, i}
                                    <tr class="text-slate-500">
                                      <td
                                        class="w-12 px-3 py-1 text-right select-none"
                                        >{frame.lineno
                                          ? frame.lineno + i + 1
                                          : ""}</td
                                      >
                                      <td class="px-3 py-1">{line}</td>
                                    </tr>
                                  {/each}
                                {/if}
                              </tbody>
                            </table>
                          </div>
                        </div>
                      {:else if frame.context_line}
                        <div
                          class="rounded-lg border border-white/10 bg-black/60 p-3"
                        >
                          <pre
                            class="font-mono text-xs text-slate-300">{frame.context_line}</pre>
                        </div>
                      {:else}
                        <div class="text-xs text-slate-500 italic">
                          No code context available
                        </div>
                      {/if}
                    </div>
                  {:else}
                    <div class="p-8 text-center">
                      <p class="text-sm text-slate-500">
                        No stack frames available
                      </p>
                    </div>
                  {/each}
                </div>
              {:else if stackTraceView === "tree"}
                <!-- Tree/Hierarchical View -->
                <div class="p-6">
                  <div class="space-y-2">
                    {#each getFrames() as frame, index}
                      {@const isInApp = frame.in_app !== false}
                      {@const isLast = index === getFrames().length - 1}
                      <div class="flex items-start gap-3">
                        <div class="flex flex-col items-center pt-1">
                          {#if !isLast}
                            <div class="h-6 w-0.5 bg-white/10"></div>
                          {/if}
                          <div
                            class="flex h-6 w-6 items-center justify-center rounded-full {isInApp
                              ? 'bg-pulse-500 text-white'
                              : 'bg-slate-500/30 text-slate-400'} text-[10px] font-bold"
                          >
                            {getFrames().length - index}
                          </div>
                          {#if !isLast}
                            <div class="h-6 w-0.5 bg-white/10"></div>
                          {/if}
                        </div>
                        <div class="flex-1 min-w-0 pb-4">
                          <div
                            class="rounded-lg border border-white/10 bg-white/5 p-4 {getInAppClass(
                              isInApp,
                            )}"
                          >
                            <div class="mb-2 flex items-center gap-2">
                              {#if frame.function}
                                <span
                                  class="font-mono text-sm font-semibold text-white"
                                  >{frame.function}</span
                                >
                              {/if}
                              {#if isInApp}
                                <span
                                  class="rounded bg-pulse-500/20 px-1.5 py-0.5 text-[9px] font-bold text-pulse-300"
                                  >IN-APP</span
                                >
                              {/if}
                            </div>
                            {#if frame.filename}
                              <div
                                class="mb-2 flex items-center gap-2 text-xs text-slate-400"
                              >
                                <FileCode size={12} />
                                <span class="font-mono"
                                  >{formatFilename(frame.filename)}</span
                                >
                                {#if frame.lineno}
                                  <span class="text-pulse-400"
                                    >:{frame.lineno}</span
                                  >
                                {/if}
                              </div>
                            {/if}
                            {#if frame.context_line}
                              <div
                                class="mt-2 rounded border border-white/10 bg-black/60 p-2"
                              >
                                <pre
                                  class="font-mono text-xs text-slate-300">{frame.context_line}</pre>
                              </div>
                            {/if}
                          </div>
                        </div>
                      </div>
                    {:else}
                      <div class="p-8 text-center">
                        <p class="text-sm text-slate-500">
                          No stack frames available
                        </p>
                      </div>
                    {/each}
                  </div>
                </div>
              {/if}
            </div>
          </div>

          <!-- Context -->
          {#if context && Object.keys(context).length > 0}
            <div
              class="rounded-xl border border-white/10 bg-black/40 overflow-hidden"
            >
              <div
                class="flex items-center justify-between border-b border-white/10 bg-white/5 px-4 py-3"
              >
                <h2
                  class="flex items-center gap-2 text-xs font-bold uppercase tracking-widest text-slate-400"
                >
                  <Hash size={14} class="text-pulse-400" />
                  <span>Context</span>
                </h2>
              </div>
              <div class="p-6">
                <pre
                  class="font-mono text-xs leading-relaxed text-slate-300"><code
                    >{JSON.stringify(context, null, 2)}</code
                  ></pre>
              </div>
            </div>
          {/if}

          <!-- User Information -->
          {#if user && Object.keys(user).length > 0}
            <div
              class="rounded-xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm"
            >
              <h2
                class="mb-4 flex items-center gap-2 text-xs font-bold uppercase tracking-widest text-slate-400"
              >
                <UserIcon size={14} class="text-pulse-400" />
                <span>User Impact</span>
              </h2>
              <div class="flex items-center gap-4">
                <div
                  class="flex h-12 w-12 items-center justify-center rounded-full bg-pulse-500 text-black font-bold text-lg"
                >
                  {user.email?.[0]?.toUpperCase() || "U"}
                </div>
                <div>
                  <div class="text-sm font-semibold text-white">
                    {user.email || "Anonymous User"}
                  </div>
                  <div class="font-mono text-[10px] text-slate-500">
                    ID: {user.id || "N/A"}
                  </div>
                </div>
              </div>
            </div>
          {/if}
        {:else if activeTab === "tags"}
          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            {#if tags && Object.keys(tags).length > 0}
              {#each Object.entries(tags) as [key, value]}
                <div
                  class="rounded-lg border border-white/10 bg-white/5 p-4 transition-colors hover:bg-white/10"
                >
                  <div
                    class="mb-1 text-[10px] font-bold uppercase tracking-widest text-slate-500"
                  >
                    {key}
                  </div>
                  <div class="font-mono text-sm text-pulse-400">{value}</div>
                </div>
              {/each}
            {:else}
              <div class="col-span-2 py-20 text-center">
                <p class="text-sm text-slate-500">
                  No tags available for this event.
                </p>
              </div>
            {/if}
          </div>
        {:else if activeTab === "events"}
          <div
            class="rounded-xl border border-white/10 bg-white/5 overflow-hidden"
          >
            <div class="border-b border-white/10 bg-white/5 px-4 py-3">
              <h2
                class="flex items-center gap-2 text-xs font-bold uppercase tracking-widest text-slate-400"
              >
                <Calendar size={14} class="text-pulse-400" />
                <span>Timeline</span>
              </h2>
            </div>
            <div class="p-6">
              <div class="relative border-l border-white/10 pl-6 space-y-8">
                <div class="relative">
                  <div
                    class="absolute -left-[29px] top-1 h-3 w-3 rounded-full border-2 border-pulse-500 bg-black"
                  ></div>
                  <div class="text-xs font-bold text-white">
                    {formatDate(error.created_at)}
                  </div>
                  <div class="mt-1 text-xs text-slate-500">
                    Issue first tracked in <span class="text-pulse-400"
                      >{error.environment || "production"}</span
                    > environment.
                  </div>
                </div>
              </div>
            </div>
          </div>
        {/if}
      </div>

      <!-- Sidebar -->
      <div class="lg:col-span-4 space-y-6">
        <div
          class="rounded-xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm"
        >
          <h3
            class="mb-6 flex items-center gap-2 text-xs font-bold uppercase tracking-widest text-slate-400"
          >
            <Info size={14} />
            <span>Details</span>
          </h3>

          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <span class="text-xs text-slate-500">Project</span>
              {#if project}
                <Link
                  to="/projects/{project.id}"
                  class="text-xs font-bold text-pulse-400 hover:underline"
                  >{project.name}</Link
                >
              {:else}
                <span class="text-xs text-slate-400">Deleted Project</span>
              {/if}
            </div>
            <div class="flex items-center justify-between font-mono">
              <span class="text-xs text-slate-500">Environment</span>
              <span
                class="rounded bg-pulse-500/10 px-2 py-0.5 text-[10px] font-bold text-pulse-400"
                >{error.environment || "Production"}</span
              >
            </div>
            <div class="flex items-center justify-between">
              <span class="text-xs text-slate-500">Status</span>
              {#if error.status}
                {@const statusColors = getIssueStatusColor(error.status)}
                <span
                  class="rounded-full px-2 py-0.5 text-xs font-bold uppercase {getStatusColorClass(
                    error.status,
                  )}"
                >
                  {statusColors.icon}
                  {error.status}
                </span>
              {:else}
                <span
                  class="rounded-full px-2 py-0.5 text-xs font-bold uppercase {getStatusColorClass(
                    'unresolved',
                  )}"
                >
                  ● unresolved
                </span>
              {/if}
            </div>
            <div class="flex items-center justify-between">
              <span class="text-xs text-slate-500">Release</span>
              <span class="text-xs font-bold text-slate-300"
                >{error.release || "v1.0.0"}</span
              >
            </div>
            <div class="flex items-center justify-between">
              <span class="text-xs text-slate-500">First Seen</span>
              <span class="text-xs text-slate-300"
                >{formatDate(error.created_at)}</span
              >
            </div>
          </div>
        </div>

        <!-- Infrastructure Meta -->
        <div
          class="rounded-xl border border-white/10 bg-white/5 p-6 backdrop-blur-sm"
        >
          <h3
            class="mb-6 flex items-center gap-2 text-xs font-bold uppercase tracking-widest text-slate-400"
          >
            <Layers size={14} />
            <span>Infrastructure</span>
          </h3>
          <div class="space-y-4">
            <div class="flex items-center justify-between text-xs">
              <span class="flex items-center gap-2 text-slate-500"
                ><Globe size={12} /> Region</span
              >
              <span class="text-slate-300">us-east-1</span>
            </div>
            <div class="flex items-center justify-between text-xs">
              <span class="flex items-center gap-2 text-slate-500"
                ><Cpu size={12} /> Runtime</span
              >
              <span class="text-slate-300">Node.js 18.x</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  {:else}
    <div class="flex flex-col items-center justify-center py-20 text-center">
      <div class="mb-4 text-slate-700">
        <Info size={64} />
      </div>
      <h2 class="text-xl font-bold text-white">Issue Not Found</h2>
      <p class="mb-6 text-sm text-slate-500">
        The issue you're looking for doesn't exist or has been deleted.
      </p>
      <Link
        to="/issues"
        class="inline-flex items-center gap-2 rounded-lg bg-pulse-600 px-6 py-2.5 text-sm font-bold text-white transition-all hover:bg-pulse-500"
      >
        <ChevronLeft size={16} /> Back to Issues
      </Link>
    </div>
  {/if}
</div>

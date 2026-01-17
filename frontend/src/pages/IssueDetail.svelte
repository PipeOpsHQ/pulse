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
    ChevronDown,
    ChevronRight as ChevronRightIcon,
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
  let expandedFrames = new Set();
  let showFullStacktrace = false;
  let occurrences = [];

  function toggleFrame(index) {
    if (expandedFrames.has(index)) {
      expandedFrames.delete(index);
    } else {
      expandedFrames.add(index);
    }
    expandedFrames = expandedFrames; // trigger reactivity
  }

  function frameIsInApp(frame) {
    return frame.in_app !== false;
  }

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

        try {
          occurrences = await api.get(`/errors/${errorId}/occurrences`);
        } catch (e) {
          console.error("Failed to load occurrences:", e);
          occurrences = [error]; // fallback to the main error if fetch fails
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

  function getPrimaryFrame() {
    const frames = getFrames();
    if (!frames.length) return null;
    // Find first in-app frame (usually the most relevant)
    const inAppFrame = frames.find((f) => f.in_app !== false);
    return inAppFrame || frames[0];
  }

  $: primaryFrame = getPrimaryFrame();
  $: request = context?.request || null;
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
            {#if primaryFrame}
              <div class="mt-2 flex items-center gap-2 text-xs text-slate-400">
                <span class="font-bold text-pulse-400"
                  >{primaryFrame.function || "unknown"}</span
                >
                <span>in</span>
                <span class="font-mono"
                  >{formatFilename(primaryFrame.filename)}</span
                >
                {#if primaryFrame.lineno}
                  <span>at line</span>
                  <span class="text-white font-bold">{primaryFrame.lineno}</span
                  >
                {/if}
              </div>
            {/if}
          </div>

          <div
            class="flex flex-wrap items-center gap-6 mr-4 px-4 py-2 rounded-xl bg-white/5 border border-white/10"
          >
            <div class="flex flex-col items-center">
              <span class="text-lg font-bold text-white leading-tight"
                >{error.event_count || 1}</span
              >
              <span
                class="text-[9px] font-black uppercase tracking-widest text-slate-500"
                >Events</span
              >
            </div>
            <div class="w-px h-8 bg-white/10"></div>
            <div class="flex flex-col items-center">
              <span class="text-lg font-bold text-white leading-tight"
                >{error.user_count || 1}</span
              >
              <span
                class="text-[9px] font-black uppercase tracking-widest text-slate-500"
                >Users</span
              >
            </div>
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
          {#if request || (context && context.request)}
            <button
              class="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider transition-all {activeTab ===
              'request'
                ? 'text-pulse-400'
                : 'text-slate-500 hover:text-white'}"
              on:click={() => (activeTab = "request")}
            >
              <Globe size={12} /> Request
            </button>
          {/if}
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
          <!-- Timeline View (Prominent) -->
          <div
            class="rounded-xl border border-white/10 bg-white/5 overflow-hidden shadow-2xl"
          >
            <div
              class="flex items-center justify-between border-b border-white/10 bg-white/5 px-4 py-3"
            >
              <h2
                class="flex items-center gap-2 text-xs font-bold uppercase tracking-widest text-slate-400"
              >
                <Calendar size={14} class="text-pulse-400" />
                <span>Event Timeline ({occurrences.length})</span>
              </h2>
              <div class="text-[10px] text-slate-500 font-medium">
                Showing last 50 occurrences
              </div>
            </div>
            <div class="p-6">
              <div class="relative border-l border-white/10 pl-6 space-y-6">
                {#each occurrences.length > 0 ? occurrences : [error] as occurrence, i}
                  <div class="relative group">
                    <div
                      class="absolute -left-[29.5px] top-1.5 h-2.5 w-2.5 rounded-full border-2 {i ===
                      0
                        ? 'border-pulse-500 bg-pulse-500 shadow-[0_0_8px_rgba(168,85,247,0.5)]'
                        : 'border-slate-600 bg-black'} transition-all group-hover:scale-125"
                    ></div>
                    <div class="flex items-center justify-between">
                      <div class="text-xs font-bold text-white">
                        {formatDate(
                          occurrence.timestamp || occurrence.created_at,
                        )}
                      </div>
                      <div
                        class="px-2 py-0.5 rounded bg-white/5 border border-white/10 text-[9px] font-mono text-slate-400"
                      >
                        {occurrence.id.substring(0, 8)}
                      </div>
                    </div>
                    <div class="mt-1 text-xs text-slate-500">
                      {#if i === occurrences.length - 1 && occurrences.length > 1}
                        Issue first tracked in
                      {:else if i === 0}
                        Latest occurrence in
                      {:else}
                        Occurrence detected in
                      {/if}
                      <span class="text-pulse-400 font-semibold"
                        >{occurrence.environment || "production"}</span
                      >
                      environment.
                      {#if occurrence.release}
                        <span class="ml-2 text-slate-600 font-mono text-[10px]"
                          >Release: {occurrence.release}</span
                        >
                      {/if}
                    </div>
                  </div>
                {/each}
              </div>
            </div>
          </div>

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
              <div class="flex items-center gap-4">
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

                <!-- Full/In-App Toggle -->
                <div class="flex items-center gap-2">
                  <span
                    class="text-[9px] font-bold text-slate-500 uppercase tracking-widest"
                    >Full</span
                  >
                  <button
                    class="relative inline-flex h-4 w-8 shrink-0 cursor-pointer items-center rounded-full transition-colors focus:outline-none {showFullStacktrace
                      ? 'bg-pulse-600'
                      : 'bg-white/10'}"
                    on:click={() => (showFullStacktrace = !showFullStacktrace)}
                    aria-label="Toggle full stacktrace"
                  >
                    <span
                      class="pointer-events-none inline-block h-2.5 w-2.5 transform rounded-full bg-white shadow-lg transition duration-200 {showFullStacktrace
                        ? 'translate-x-4.5'
                        : 'translate-x-1'}"
                    ></span>
                  </button>
                  <span
                    class="text-[9px] font-bold text-slate-300 uppercase tracking-widest"
                    >In-App</span
                  >
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

            <div class="overflow-hidden">
              {#if stackTraceView === "raw"}
                <div class="p-6">
                  <pre
                    class="font-mono text-xs leading-relaxed text-slate-300"><code
                      >{JSON.stringify(stacktrace, null, 2)}</code
                    ></pre>
                </div>
              {:else if stackTraceView === "frames"}
                {@const filteredFrames = (getFrames() || []).filter(
                  (f) => showFullStacktrace || f.in_app !== false,
                )}
                <div class="divide-y divide-white/[0.03]">
                  {#each filteredFrames as frame, index}
                    {@const isExpanded = expandedFrames.has(index)}
                    {@const isInApp = frame.in_app !== false}
                    <div
                      class="group transition-all duration-200 {isInApp
                        ? 'bg-pulse-500/[0.02]'
                        : 'bg-transparent opacity-60'}"
                    >
                      <div
                        role="button"
                        tabindex="0"
                        class="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-white/[0.05]"
                        on:click={() => toggleFrame(index)}
                        on:keydown={(e) =>
                          e.key === "Enter" && toggleFrame(index)}
                      >
                        <div
                          class="flex h-5 w-5 items-center justify-center rounded text-slate-500 transition-transform {isExpanded
                            ? 'rotate-90'
                            : ''}"
                        >
                          <ChevronRightIcon size={14} />
                        </div>
                        <div
                          class="flex-1 min-w-0 flex flex-wrap items-center gap-x-3 gap-y-1"
                        >
                          <span
                            class="font-mono text-[13px] font-bold {isInApp
                              ? 'text-white'
                              : 'text-slate-400'}"
                            >{frame.function || "<anonymous>"}</span
                          >
                          <div
                            class="flex items-center gap-1.5 text-xs text-slate-500 font-mono italic"
                          >
                            <span>{formatFilename(frame.filename)}</span>
                            {#if frame.lineno}<span
                                class="text-pulse-500/70 font-bold"
                                >:{frame.lineno}</span
                              >{/if}
                          </div>
                          {#if isInApp}
                            <span
                              class="rounded-full bg-pulse-500/10 border border-pulse-500/20 px-2 py-0.5 text-[8px] font-black uppercase tracking-widest text-pulse-400"
                              >IN-APP</span
                            >
                          {/if}
                        </div>
                        <div
                          class="text-[10px] font-mono text-slate-700 font-bold tabular-nums"
                        >
                          #{filteredFrames.length - index}
                        </div>
                      </div>

                      {#if isExpanded}
                        <div
                          class="px-4 pb-4 animate-in slide-in-from-top-2 duration-200"
                        >
                          <div
                            class="ml-8 overflow-hidden rounded-xl border border-white/10 bg-black/60"
                          >
                            {#if frame.pre_context || frame.context_line || frame.post_context}
                              <div class="overflow-x-auto">
                                <table
                                  class="w-full font-mono text-[11px] leading-relaxed"
                                >
                                  <tbody>
                                    {#if frame.pre_context}
                                      {#each frame.pre_context as line, i}
                                        <tr
                                          class="text-slate-500 hover:bg-white/5 group/line"
                                        >
                                          <td
                                            class="w-10 px-3 py-0.5 text-right select-none opacity-40 group-hover/line:opacity-100"
                                            >{frame.lineno
                                              ? frame.lineno -
                                                frame.pre_context.length +
                                                i
                                              : ""}</td
                                          >
                                          <td class="px-3 py-0.5 whitespace-pre"
                                            >{line}</td
                                          >
                                        </tr>
                                      {/each}
                                    {/if}
                                    {#if frame.context_line}
                                      <tr
                                        class="bg-pulse-500/20 text-white font-medium border-l-2 border-pulse-500"
                                      >
                                        <td
                                          class="w-10 px-3 py-1 text-right select-none text-pulse-400 font-bold"
                                          >{frame.lineno || ""}</td
                                        >
                                        <td class="px-3 py-1 whitespace-pre"
                                          >{frame.context_line}</td
                                        >
                                      </tr>
                                    {/if}
                                    {#if frame.post_context}
                                      {#each frame.post_context as line, i}
                                        <tr
                                          class="text-slate-500 hover:bg-white/5 group/line"
                                        >
                                          <td
                                            class="w-10 px-3 py-0.5 text-right select-none opacity-40 group-hover/line:opacity-100"
                                            >{frame.lineno
                                              ? frame.lineno + i + 1
                                              : ""}</td
                                          >
                                          <td class="px-3 py-0.5 whitespace-pre"
                                            >{line}</td
                                          >
                                        </tr>
                                      {/each}
                                    {/if}
                                  </tbody>
                                </table>
                              </div>
                            {:else if frame.context_line}
                              <div class="p-4 bg-pulse-500/10">
                                <pre
                                  class="font-mono text-xs text-white whitespace-pre-wrap">{frame.context_line}</pre>
                              </div>
                            {:else}
                              <div
                                class="p-4 text-center text-[10px] text-slate-600 uppercase tracking-widest font-bold"
                              >
                                No source code context available
                              </div>
                            {/if}
                            {#if frame.vars && Object.keys(frame.vars).length > 0}
                              <div
                                class="border-t border-white/5 bg-white/[0.02] p-4"
                              >
                                <div
                                  class="text-[9px] font-bold text-slate-500 uppercase tracking-widest mb-2"
                                >
                                  Local Variables
                                </div>
                                <div
                                  class="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-1"
                                >
                                  {#each Object.entries(frame.vars) as [key, value]}
                                    <div
                                      class="flex items-start gap-2 text-[11px] font-mono"
                                    >
                                      <span class="text-pulse-400/80 shrink-0"
                                        >{key}</span
                                      >
                                      <span class="text-slate-400 font-bold"
                                        >=</span
                                      >
                                      <span class="text-slate-200 break-all"
                                        >{JSON.stringify(value)}</span
                                      >
                                    </div>
                                  {/each}
                                </div>
                              </div>
                            {/if}
                          </div>
                        </div>
                      {/if}
                    </div>
                  {:else}
                    <div class="p-16 text-center">
                      <div
                        class="inline-flex h-12 w-12 items-center justify-center rounded-full bg-white/5 text-slate-600 mb-4"
                      >
                        <List size={24} />
                      </div>
                      <p class="text-sm font-medium text-slate-400">
                        No frames match current filter
                      </p>
                      <button
                        class="mt-4 text-xs font-bold text-pulse-400 hover:text-pulse-300 transition-colors"
                        on:click={() => (showFullStacktrace = true)}
                        >Show hidden system frames</button
                      >
                    </div>
                  {/each}
                </div>
              {:else if stackTraceView === "tree"}
                <div class="p-6">
                  <div class="space-y-2">
                    {#each getFrames() || [] as frame, index}
                      {@const isInApp = frame.in_app !== false}
                      {@const isLast = index === getFrames().length - 1}
                      <div class="flex items-start gap-3">
                        <div class="flex flex-col items-center pt-1">
                          <div
                            class="flex h-6 w-6 items-center justify-center rounded-full {isInApp
                              ? 'bg-pulse-500 text-white'
                              : 'bg-slate-500/30 text-slate-400'} text-[10px] font-bold shadow-sm"
                          >
                            {getFrames().length - index}
                          </div>
                          {#if !isLast}<div
                              class="h-full w-0.5 bg-white/10 my-1"
                            ></div>{/if}
                        </div>
                        <div class="flex-1 min-w-0 pb-6">
                          <div
                            class="rounded-xl border border-white/5 bg-white/[0.03] p-4 group transition-colors hover:border-white/10"
                          >
                            <div class="mb-2 flex items-center gap-2">
                              <span
                                class="font-mono text-sm font-bold {isInApp
                                  ? 'text-white'
                                  : 'text-slate-400'}"
                                >{frame.function || "<anonymous>"}</span
                              >
                              {#if isInApp}<span
                                  class="rounded bg-pulse-500/20 px-1.5 py-0.5 text-[8px] font-bold text-pulse-300 uppercase tracking-widest"
                                  >IN-APP</span
                                >{/if}
                            </div>
                            <div
                              class="flex items-center gap-2 text-xs text-slate-500 font-mono"
                            >
                              <FileCode size={12} />
                              <span>{formatFilename(frame.filename)}</span>
                              {#if frame.lineno}<span class="text-pulse-400"
                                  >:{frame.lineno}</span
                                >{/if}
                            </div>
                          </div>
                        </div>
                      </div>
                    {:else}
                      <div class="p-16 text-center text-slate-500 italic">
                        No frames available
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
        {:else if activeTab === "request"}
          {@const req = request || context?.request}
          <div class="space-y-6">
            <div
              class="rounded-xl border border-white/10 bg-black/40 overflow-hidden"
            >
              <div class="border-b border-white/10 bg-white/5 px-4 py-3">
                <h2
                  class="flex items-center gap-2 text-xs font-bold uppercase tracking-widest text-slate-400"
                >
                  <Globe size={14} class="text-pulse-400" />
                  <span>Request Body / Params</span>
                </h2>
              </div>
              <div class="p-4 space-y-4">
                <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div class="p-3 rounded-lg bg-white/5 border border-white/10">
                    <div
                      class="text-[10px] text-slate-500 mb-1 uppercase tracking-wider"
                    >
                      Method
                    </div>
                    <div class="text-sm font-bold text-pulse-400">
                      {req.method || "GET"}
                    </div>
                  </div>
                  <div class="p-3 rounded-lg bg-white/5 border border-white/10">
                    <div
                      class="text-[10px] text-slate-500 mb-1 uppercase tracking-wider"
                    >
                      URL
                    </div>
                    <div class="text-sm font-mono text-white break-all">
                      {req.url || "Unknown"}
                    </div>
                  </div>
                </div>

                {#if req.query_string}
                  <div>
                    <div
                      class="text-[10px] text-slate-500 mb-1 uppercase tracking-wider"
                    >
                      Query String
                    </div>
                    <div
                      class="p-3 rounded-lg bg-black/60 font-mono text-xs text-white"
                    >
                      {req.query_string}
                    </div>
                  </div>
                {/if}

                {#if req.headers && Object.keys(req.headers).length > 0}
                  <div>
                    <div
                      class="text-[10px] text-slate-500 mb-2 uppercase tracking-wider"
                    >
                      Headers
                    </div>
                    <div
                      class="rounded-lg border border-white/10 bg-black/60 overflow-hidden"
                    >
                      <table class="w-full text-left text-xs">
                        <tbody class="divide-y divide-white/5">
                          {#each Object.entries(req.headers) as [key, value]}
                            <tr>
                              <td
                                class="px-3 py-2 font-bold text-slate-500 border-r border-white/5 w-1/3"
                                >{key}</td
                              >
                              <td
                                class="px-3 py-2 font-mono text-slate-300 break-all"
                                >{value}</td
                              >
                            </tr>
                          {/each}
                        </tbody>
                      </table>
                    </div>
                  </div>
                {/if}
              </div>
            </div>
          </div>
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
          <div class="py-20 text-center">
            <Activity
              size={48}
              class="mx-auto text-slate-600 mb-4 opacity-20"
            />
            <p class="text-slate-500">
              Detailed event history moved to the main details tab.
            </p>
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
              <span class="text-xs text-slate-500">Events</span>
              <span class="text-xs font-bold text-white tabular-nums"
                >{error.event_count || 1}</span
              >
            </div>
            <div class="flex items-center justify-between">
              <span class="text-xs text-slate-500">Users</span>
              <span class="text-xs font-bold text-white tabular-nums"
                >{error.user_count || 1}</span
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
                ><Globe size={12} /> OS</span
              >
              <span class="text-slate-300"
                >{context?.os?.name || tags?.os || "Unknown"}
                {context?.os?.version || ""}</span
              >
            </div>
            <div class="flex items-center justify-between text-xs">
              <span class="flex items-center gap-2 text-slate-500"
                ><Cpu size={12} /> Runtime</span
              >
              <span class="text-slate-300"
                >{context?.runtime?.name || tags?.runtime || "Unknown"}
                {context?.runtime?.version || ""}</span
              >
            </div>
            <div class="flex items-center justify-between text-xs">
              <span class="flex items-center gap-2 text-slate-500"
                ><Terminal size={12} /> SDK</span
              >
              <span class="text-slate-300 font-mono text-[10px]"
                >{error.platform || "Unknown"}</span
              >
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

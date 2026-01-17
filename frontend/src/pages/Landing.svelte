<script>
  import { navigate } from "../lib/router";
  import Link from "../components/Link.svelte";
  import {
    Zap,
    Shield,
    Layers,
    Feather,
    Activity,
    Code,
    Lock,
    Github,
    Twitter,
    Layout,
    ArrowRight,
    Search,
  } from "lucide-svelte";

  function handleGetStarted() {
    navigate("/login");
  }
</script>

<div class="min-h-screen bg-[#050505] text-white selection:bg-pulse-500/30">
  <!-- Navigation -->
  <nav
    class="sticky top-0 z-50 border-b border-white/5 bg-[#050505]/80 backdrop-blur-xl"
  >
    <div class="mx-auto flex max-w-7xl items-center justify-between px-6 py-4">
      <div class="flex items-center gap-2">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 100 100"
          class="h-8 w-8"
        >
          <defs>
            <linearGradient id="nav-grad" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" style="stop-color:#6366f1;stop-opacity:1" />
              <stop offset="100%" style="stop-color:#8b5cf6;stop-opacity:1" />
            </linearGradient>
          </defs>
          <rect width="100" height="100" rx="20" fill="url(#nav-grad)" />
          <path
            d="M20 50 L35 50 L45 20 L55 80 L65 50 L80 50"
            fill="none"
            stroke="white"
            stroke-width="8"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
        <span class="text-xl font-bold tracking-tight">Pulse</span>
      </div>
      <div class="flex items-center gap-6">
        <a
          href="#features"
          class="text-sm font-medium text-slate-400 hover:text-white transition-colors"
          >Features</a
        >
        <a
          href="https://github.com/PipeOpsHQ/pulse"
          target="_blank"
          class="text-sm font-medium text-slate-400 hover:text-white transition-colors"
          >GitHub</a
        >
        <button
          onclick={handleGetStarted}
          class="rounded-full bg-white px-5 py-2 text-sm font-semibold text-black hover:bg-slate-200 transition-all transform active:scale-95"
        >
          Sign In
        </button>
      </div>
    </div>
  </nav>

  <!-- Hero Section -->
  <header class="relative overflow-hidden pt-24 pb-32">
    <!-- Background Glow -->
    <div
      class="absolute top-0 left-1/2 -translate-x-1/2 h-[500px] w-full max-w-4xl bg-pulse-600/20 blur-[120px] rounded-full -z-10"
    ></div>

    <div class="mx-auto max-w-7xl px-6 text-center">
      <div
        class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs font-medium text-pulse-400 mb-8"
      >
        <span class="relative flex h-2 w-2">
          <span
            class="animate-ping absolute inline-flex h-full w-full rounded-full bg-pulse-400 opacity-75"
          ></span>
          <span class="relative inline-flex rounded-full h-2 w-2 bg-pulse-500"
          ></span>
        </span>
        Project Pulse is now Open Source
      </div>

      <h1
        class="mx-auto max-w-3xl text-5xl font-bold tracking-tight md:text-7xl"
      >
        Lightweight Observability for <span
          class="bg-gradient-to-r from-pulse-400 to-indigo-400 bg-clip-text text-transparent"
          >Modern Teams</span
        >
      </h1>

      <p class="mx-auto mt-8 max-w-2xl text-lg text-slate-400 leading-relaxed">
        Trace errors, monitor performance, track coverage, and manage uptime. A
        drop-in, self-hosted replacement for complex tracking stacks.
      </p>

      <div class="mt-12 flex flex-wrap items-center justify-center gap-4">
        <button
          onclick={handleGetStarted}
          class="group relative overflow-hidden rounded-full bg-pulse-600 px-8 py-4 font-bold text-white transition-all hover:scale-105 active:scale-95"
        >
          <span class="relative z-10">Join Pulse</span>
          <div
            class="absolute inset-0 bg-gradient-to-r from-pulse-600 to-pulse-400 opacity-0 group-hover:opacity-100 transition-opacity"
          ></div>
        </button>
        <a
          href="https://github.com/PipeOpsHQ/pulse"
          target="_blank"
          class="rounded-full border border-white/10 bg-white/5 px-8 py-4 font-bold text-white hover:bg-white/10 transition-all"
        >
          View Documentation
        </a>
      </div>

      <!-- Dashboard Preview -->
      <div
        class="mt-20 rounded-2xl border border-white/10 bg-slate-900/50 p-2 shadow-2xl backdrop-blur-sm"
      >
        <div
          class="overflow-hidden rounded-xl border border-white/5 bg-[#0a0a0a]"
        >
          <div
            class="flex items-center justify-between border-b border-white/5 bg-white/5 px-4 py-2"
          >
            <div class="flex gap-1.5">
              <div class="h-3 w-3 rounded-full bg-red-500/50"></div>
              <div class="h-3 w-3 rounded-full bg-yellow-500/50"></div>
              <div class="h-3 w-3 rounded-full bg-green-500/50"></div>
            </div>
            <div class="text-[10px] text-slate-500 font-mono">
              pulse.hq/dashboard
            </div>
            <div class="w-12"></div>
          </div>
          <div
            class="aspect-video bg-[#050505] p-8 flex items-center justify-center relative"
          >
            <!-- Placeholder for a real screenshot if available, otherwise a nice graphic -->
            <div
              class="grid grid-cols-3 gap-6 w-[80%] opacity-20 group-hover:opacity-40 transition-opacity"
            >
              <div
                class="h-32 rounded-lg bg-white/5 border border-white/5 animate-pulse flex items-center justify-center"
              >
                <Zap size={24} class="text-orange-500" />
              </div>
              <div
                class="h-32 rounded-lg bg-white/5 border border-white/5 animate-pulse flex items-center justify-center"
                style="animation-delay: 0.2s"
              >
                <Activity size={24} class="text-pulse-500" />
              </div>
              <div
                class="h-32 rounded-lg bg-white/5 border border-white/5 animate-pulse flex items-center justify-center"
                style="animation-delay: 0.4s"
              >
                <Shield size={24} class="text-blue-500" />
              </div>
              <div
                class="col-span-2 h-44 rounded-lg bg-white/5 border border-white/5 animate-pulse p-4"
                style="animation-delay: 0.1s"
              >
                <div class="flex gap-2 mb-4">
                  <div class="h-2 w-1/4 bg-white/10 rounded"></div>
                  <div class="h-2 w-1/2 bg-white/10 rounded"></div>
                </div>
                <div class="space-y-3">
                  <div class="h-2 w-full bg-white/5 rounded"></div>
                  <div class="h-2 w-[90%] bg-white/5 rounded"></div>
                  <div class="h-2 w-[95%] bg-white/5 rounded"></div>
                </div>
              </div>
              <div
                class="h-44 rounded-lg bg-white/5 border border-white/5 animate-pulse flex items-center justify-center"
                style="animation-delay: 0.3s"
              >
                <Code size={24} class="text-purple-500" />
              </div>
            </div>
            <div
              class="absolute inset-0 flex flex-col items-center justify-center"
            >
              <div
                class="h-16 w-16 mb-4 rounded-full bg-pulse-600/20 flex items-center justify-center border border-pulse-500/30"
              >
                <Activity size={32} class="text-pulse-400" />
              </div>
              <p class="text-slate-200 font-medium">
                Beautiful Performance Metrics
              </p>
              <p class="text-slate-500 text-sm mt-1">
                Ready to deploy in seconds
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </header>

  <!-- Features Grid -->
  <section id="features" class="py-24 bg-white/[0.02]">
    <div class="mx-auto max-w-7xl px-6">
      <div class="mb-16 text-center">
        <h2 class="text-3xl font-bold md:text-4xl">Everything you need</h2>
        <p class="mt-4 text-slate-400">
          One tool to rule your entire observability stack.
        </p>
      </div>

      <div class="grid gap-8 md:grid-cols-2 lg:grid-cols-3">
        <!-- Sentry Compatible -->
        <div
          class="group rounded-2xl border border-white/5 bg-white/[0.03] p-8 hover:border-pulse-500/30 transition-all"
        >
          <div
            class="mb-5 flex h-12 w-12 items-center justify-center rounded-xl bg-orange-500/10 text-orange-400"
          >
            <Zap size={24} />
          </div>
          <h3 class="text-xl font-bold mb-3">Sentry Compatible</h3>
          <p class="text-slate-400 leading-relaxed">
            Drop-in replacement for your Sentry SDKs. Just change the DSN and
            start receiving events instantly. No code changes required.
          </p>
        </div>

        <!-- Privacy First -->
        <div
          class="group rounded-2xl border border-white/5 bg-white/[0.03] p-8 hover:border-pulse-500/30 transition-all"
        >
          <div
            class="mb-5 flex h-12 w-12 items-center justify-center rounded-xl bg-blue-500/10 text-blue-400"
          >
            <Shield size={24} />
          </div>
          <h3 class="text-xl font-bold mb-3">Privacy First</h3>
          <p class="text-slate-400 leading-relaxed">
            Self-host Pulse on your own infrastructure. Your sensitive error
            data and source code never leave your network.
          </p>
        </div>

        <!-- Full Stack -->
        <div
          class="group rounded-2xl border border-white/5 bg-white/[0.03] p-8 hover:border-pulse-500/30 transition-all"
        >
          <div
            class="mb-5 flex h-12 w-12 items-center justify-center rounded-xl bg-pulse-500/10 text-pulse-400"
          >
            <Layers size={24} />
          </div>
          <h3 class="text-xl font-bold mb-3">Full-Stack Context</h3>
          <p class="text-slate-400 leading-relaxed">
            Integrated error tracking, performance spans, code coverage, and
            uptime monitoring in a single unified view.
          </p>
        </div>

        <!-- Lightweight -->
        <div
          class="group rounded-2xl border border-white/5 bg-white/[0.03] p-8 hover:border-pulse-500/30 transition-all"
        >
          <div
            class="mb-5 flex h-12 w-12 items-center justify-center rounded-xl bg-green-500/10 text-green-400"
          >
            <Feather size={24} />
          </div>
          <h3 class="text-xl font-bold mb-3">Feather Light</h3>
          <p class="text-slate-400 leading-relaxed">
            The entire backend is a single Go binary. Uses SQLite for
            zero-config persistence. Deploy on a 512MB RAM VPS.
          </p>
        </div>

        <!-- Real-time -->
        <div
          class="group rounded-2xl border border-white/5 bg-white/[0.03] p-8 hover:border-pulse-500/30 transition-all"
        >
          <div
            class="mb-5 flex h-12 w-12 items-center justify-center rounded-xl bg-red-500/10 text-red-400"
          >
            <Activity size={24} />
          </div>
          <h3 class="text-xl font-bold mb-3">Real-time Insights</h3>
          <p class="text-slate-400 leading-relaxed">
            Get instant notifications and dashboard updates as errors happen.
            Stay ahead of production issues with low latency.
          </p>
        </div>

        <!-- Open Source -->
        <div
          class="group rounded-2xl border border-white/5 bg-white/[0.03] p-8 hover:border-pulse-500/30 transition-all"
        >
          <div
            class="mb-5 flex h-12 w-12 items-center justify-center rounded-xl bg-purple-500/10 text-purple-400"
          >
            <Code size={24} />
          </div>
          <h3 class="text-xl font-bold mb-3">100% Open Source</h3>
          <p class="text-slate-400 leading-relaxed">
            Fully transparent and community-driven. Built for developers who
            want control over their monitoring tools.
          </p>
        </div>
      </div>
    </div>
  </section>

  <!-- CTA Section -->
  <section class="py-24">
    <div class="mx-auto max-w-5xl px-6">
      <div
        class="rounded-3xl bg-gradient-to-br from-pulse-600 to-indigo-700 px-8 py-16 text-center shadow-2xl relative overflow-hidden"
      >
        <div
          class="absolute top-0 right-0 -translate-y-1/2 translate-x-1/2 h-64 w-64 bg-white/10 blur-3xl rounded-full"
        ></div>
        <h2 class="text-3xl font-bold md:text-5xl relative z-10">
          Start tracing in minutes
        </h2>
        <p class="mt-6 text-pulse-100 text-lg relative z-10">
          Join teams who value lightweight, focused, and secure observability.
        </p>
        <div class="mt-10 flex flex-wrap justify-center gap-4 relative z-10">
          <button
            onclick={handleGetStarted}
            class="rounded-full bg-white px-10 py-4 font-bold text-black hover:bg-slate-100 transition-all hover:shadow-lg"
          >
            Create Account
          </button>
          <a
            href="https://github.com/PipeOpsHQ/pulse"
            target="_blank"
            class="rounded-full border border-white/20 bg-black/10 px-10 py-4 font-bold text-white hover:bg-black/20 transition-all"
          >
            Star on GitHub
          </a>
        </div>
      </div>
    </div>
  </section>

  <!-- Footer -->
  <footer class="border-t border-white/5 py-12">
    <div
      class="mx-auto max-w-7xl px-6 flex flex-col md:flex-row items-center justify-between gap-6"
    >
      <div class="flex items-center gap-2 border-none opacity-40">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 100 100"
          class="h-6 w-6"
        >
          <defs>
            <linearGradient
              id="footer-grad"
              x1="0%"
              y1="0%"
              x2="100%"
              y2="100%"
            >
              <stop offset="0%" style="stop-color:#6366f1;stop-opacity:1" />
              <stop offset="100%" style="stop-color:#8b5cf6;stop-opacity:1" />
            </linearGradient>
          </defs>
          <rect width="100" height="100" rx="20" fill="url(#footer-grad)" />
          <path
            d="M20 50 L35 50 L45 20 L55 80 L65 50 L80 50"
            fill="none"
            stroke="white"
            stroke-width="12"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
        <span class="text-lg font-bold tracking-tight">Pulse</span>
      </div>
      <p class="text-sm text-slate-500">© 2026 PipeOps. Licensed under MIT.</p>
      <div class="flex gap-6">
        <a
          href="https://twitter.com/PipeOpsHQ"
          target="_blank"
          aria-label="Follow us on Twitter"
          class="text-slate-500 hover:text-white transition-colors"
          ><Twitter size={20} /></a
        >
        <a
          href="https://github.com/PipeOpsHQ/pulse"
          target="_blank"
          aria-label="View on GitHub"
          class="text-slate-500 hover:text-white transition-colors"
          ><Github size={20} /></a
        >
      </div>
    </div>
  </footer>
</div>

<style>
  /* Pulse animations and effects */
  @keyframes gradient {
    0% {
      background-position: 0% 50%;
    }
    50% {
      background-position: 100% 50%;
    }
    100% {
      background-position: 0% 50%;
    }
  }
</style>

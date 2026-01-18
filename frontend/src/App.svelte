<script>
  import Sidebar from "./components/Sidebar.svelte";
  import Header from "./components/Header.svelte";
  import Toast from "./components/Toast.svelte";
  import Dashboard from "./pages/Dashboard.svelte";
  import IssuesList from "./pages/IssuesList.svelte";
  import IssueDetail from "./pages/IssueDetail.svelte";
  import ProjectDetail from "./pages/ProjectDetail.svelte";
  import Projects from "./pages/Projects.svelte";
  import Settings from "./pages/Settings.svelte";
  import Login from "./pages/Login.svelte";
  import SecurityVault from "./pages/SecurityVault.svelte";
  import Traces from "./pages/Traces.svelte";
  import AllTraces from "./pages/AllTraces.svelte";
  import TraceDetail from "./pages/TraceDetail.svelte";
  import TraceAnalytics from "./pages/TraceAnalytics.svelte";
  import StatusPage from "./pages/StatusPage.svelte";
  import Insights from "./pages/Insights.svelte";
  import Admin from "./pages/Admin.svelte";
  import Landing from "./pages/Landing.svelte";
  import { onMount } from "svelte";
  import { isAuthenticated } from "./stores/auth";

  let currentPath = $state(
    typeof window !== "undefined" ? window.location.pathname : "/",
  );
  let authenticated = $state(false);
  let loadingAuth = $state(true);

  // Subscribe to auth store
  const unsubscribe = isAuthenticated.subscribe((value) => {
    authenticated = value;
    loadingAuth = false;
  });

  onMount(() => {
    const handleLocationChange = () => {
      currentPath = window.location.pathname;
    };

    window.addEventListener("popstate", handleLocationChange);
    return () => {
      window.removeEventListener("popstate", handleLocationChange);
      unsubscribe();
    };
  });

  function getBreadcrumbs(path) {
    const crumbs = [{ label: "Pulse", path: "/" }];

    if (path === "/") return crumbs;
    if (path === "/login") return crumbs;

    if (path.startsWith("/projects/")) {
      const parts = path.split("/").filter((p) => p);
      if (parts.length >= 2) {
        crumbs.push({ label: "Projects", path: "/projects" });
        if (parts.length === 2) {
          crumbs.push({ label: parts[1], path: path }); // In real app, we'd fetch project name
        } else if (parts[2] === "issues") {
          crumbs.push({
            label: "Project issues",
            path: `/projects/${parts[1]}`,
          });
        } else if (parts[2] === "traces") {
          crumbs.push({
            label: "Project traces",
            path: `/projects/${parts[1]}`,
          });
        }
      }
    } else if (path === "/projects") {
      crumbs.push({ label: "Projects", path: "/projects" });
    } else if (path.startsWith("/errors/")) {
      crumbs.push({ label: "Issues", path: "/issues" });
      crumbs.push({ label: "Error Details", path: path });
    } else if (path.startsWith("/issues")) {
      crumbs.push({ label: "Issues", path: path });
    } else if (path === "/settings") {
      crumbs.push({ label: "Settings", path: "/settings" });
    } else if (path === "/insights") {
      crumbs.push({ label: "Insights", path: "/insights" });
    } else if (path === "/security-vault") {
      crumbs.push({ label: "Security Vault", path: "/security-vault" });
    } else if (path === "/admin") {
      crumbs.push({ label: "Admin", path: "/admin" });
    }

    return crumbs;
  }

  function isPublicRoute(path) {
    return path === "/" || path === "/login" || path.startsWith("/status/");
  }
</script>

<Toast />

{#if loadingAuth}
  <div class="flex h-screen w-screen items-center justify-center bg-[#050505]">
    <div
      class="h-10 w-10 animate-spin rounded-full border-2 border-pulse-500 border-t-transparent"
    ></div>
  </div>
{:else if !authenticated && !isPublicRoute(currentPath)}
  <Login />
{:else if currentPath === "/login"}
  <Login />
{:else if currentPath.startsWith("/status/")}
  <StatusPage />
{:else if !authenticated && currentPath === "/"}
  <Landing />
{:else if !authenticated && !isPublicRoute(currentPath)}
  <Login />
{:else if currentPath === "/login"}
  <Login />
{:else}
  <div class="flex min-h-screen bg-[#050505] text-white">
    <Sidebar {currentPath} />

    <div class="flex flex-1 flex-col transition-all duration-300 min-w-0">
      <Header breadcrumbs={getBreadcrumbs(currentPath)} />

      <main class="flex-1 overflow-y-auto p-4 md:p-8">
        <div class="mx-auto w-full max-w-7xl">
          {#if (currentPath === "/" && authenticated) || currentPath === "/dashboard"}
            <Dashboard />
          {:else if currentPath === "/projects"}
            <Projects />
          {:else if currentPath === "/issues"}
            <IssuesList />
          {:else if currentPath === "/settings"}
            <Settings />
          {:else if currentPath === "/insights"}
            <Insights />
          {:else if currentPath === "/security-vault"}
            <SecurityVault />
          {:else if currentPath === "/admin"}
            <Admin />
          {:else if currentPath === "/traces"}
            <AllTraces />
          {:else if currentPath === "/trace-analytics"}
            <TraceAnalytics />
          {:else if currentPath.startsWith("/errors/")}
            <IssueDetail />
          {:else if currentPath.startsWith("/projects/")}
            {#if currentPath.includes("/traces/")}
              <TraceDetail />
            {:else if currentPath.includes("/traces")}
              <Traces />
            {:else if currentPath.includes("/issues/")}
              <IssueDetail />
            {:else}
              <ProjectDetail />
            {/if}
          {:else}
            <div class="text-white text-center p-20">
              <h1 class="text-4xl font-bold mb-4">404</h1>
              <p class="text-slate-400">Page not found: {currentPath}</p>
            </div>
          {/if}
        </div>
      </main>
    </div>
  </div>
{/if}

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    background-color: #050505;
  }
</style>

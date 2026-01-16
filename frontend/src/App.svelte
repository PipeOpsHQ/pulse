<script>
  import { Router, Link, Route } from 'svelte-routing';
  import Sidebar from './components/Sidebar.svelte';
  import Header from './components/Header.svelte';
  import AuthGuard from './components/AuthGuard.svelte';
  import { sidebarCollapsed } from './stores/ui';
  import Toast from './components/Toast.svelte';
  import Dashboard from './pages/Dashboard.svelte';
  import IssuesList from './pages/IssuesList.svelte';
  import IssueDetail from './pages/IssueDetail.svelte';
  import ProjectDetail from './pages/ProjectDetail.svelte';
  import Projects from './pages/Projects.svelte';
  import Settings from './pages/Settings.svelte';
  import Login from './pages/Login.svelte';
  import SecurityVault from './pages/SecurityVault.svelte';
  import Traces from './pages/Traces.svelte';
  import TraceDetail from './pages/TraceDetail.svelte';
  import StatusPage from './pages/StatusPage.svelte';
  import Insights from './pages/Insights.svelte';
  import Redirector from './components/Redirector.svelte';
  import { onMount } from 'svelte';

  let currentPath = typeof window !== 'undefined' ? window.location.pathname : '/';

  onMount(() => {
    const originalPushState = history.pushState;
    const originalReplaceState = history.replaceState;

    const handleLocationChange = () => {
      currentPath = window.location.pathname;
    };

    // Monkey-patch history to catch svelte-routing navigations
    history.pushState = function() {
      originalPushState.apply(this, arguments);
      handleLocationChange();
    };

    history.replaceState = function() {
      originalReplaceState.apply(this, arguments);
      handleLocationChange();
    };

    window.addEventListener('popstate', handleLocationChange);

    return () => {
      history.pushState = originalPushState;
      history.replaceState = originalReplaceState;
      window.removeEventListener('popstate', handleLocationChange);
    };
  });

  function getBreadcrumbs(path) {
    const crumbs = [{ label: 'Pulse', path: '/' }];

    if (path === '/') return crumbs;

    if (path.startsWith('/projects/')) {
      const parts = path.split('/').filter(p => p);
      if (parts.length >= 2) {
        crumbs.push({ label: 'Projects', path: '/projects' });
        if (parts.length === 2) {
          crumbs.push({ label: parts[1], path: path }); // In real app, we'd fetch project name
        } else if (parts[2] === 'issues') {
          crumbs.push({ label: parts[1], path: `/projects/${parts[1]}` });
          crumbs.push({ label: 'Issues', path: path });
        }
      }
    } else if (path === '/projects') {
      crumbs.push({ label: 'Projects', path: '/projects' });
    } else if (path.startsWith('/errors/')) {
      crumbs.push({ label: 'Issues', path: '/issues' });
      crumbs.push({ label: 'Error Details', path: path });
    } else if (path.startsWith('/issues')) {
      crumbs.push({ label: 'Issues', path: path });
    } else if (path === '/settings') {
      crumbs.push({ label: 'Settings', path: '/settings' });
    } else if (path === '/insights') {
      crumbs.push({ label: 'Insights', path: '/insights' });
    } else if (path === '/security-vault') {
      crumbs.push({ label: 'Security Vault', path: '/security-vault' });
    }

    return crumbs;
  }
</script>

<Toast />

<Router>
  <!-- Public routes (no auth required) -->
  <Route path="/login" component={Login} />
  <Route path="/status/:projectId" component={StatusPage} />

  <!-- Protected routes (require auth) -->
  <AuthGuard>
    <div class="flex min-h-screen bg-[#050505]">
      <Sidebar {currentPath} />

      <div
        class="flex flex-1 flex-col transition-all duration-300 min-w-0"
      >
        <Header breadcrumbs={getBreadcrumbs(currentPath)} />

        <main class="flex-1 overflow-y-auto p-4 md:p-8">
          <div class="mx-auto w-full max-w-7xl">
            <Route path="/" component={Dashboard} />
            <Route path="/projects" component={Projects} />
            <Route path="/issues" component={IssuesList} />
            <Route path="/settings" component={Settings} />
            <Route path="/insights" component={Insights} />
            <Route path="/security-vault" component={SecurityVault} />
            <Route path="/errors/:id" component={IssueDetail} />
            <Route path="/projects/:id" component={ProjectDetail} />
            <Route path="/projects/:projectId/traces" component={Traces} />
            <Route path="/projects/:projectId/traces/:traceId" component={TraceDetail} />
            <Route path="/projects/:projectId/issues/:issueId" component={IssueDetail} />
            <Route path="/:id" let:params>
              <Redirector id={params.id} />
            </Route>
          </div>
        </main>
      </div>
    </div>
  </AuthGuard>
</Router>
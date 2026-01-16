<script>
  import { onMount } from 'svelte';
  import { navigate } from 'svelte-routing';
  import { Activity, Plus, Search, ExternalLink, Shield, Edit3, Trash2, Settings2, Hash } from 'lucide-svelte';
  import { api } from '../lib/api';
  import { toast } from '../stores/toast';

  let projects = [];
  let loading = true;
  let searchQuery = '';
  let showCreateModal = false;
  let showQuotaModal = false;
  let selectedProject = null;
  let newProjectName = '';
  let editingQuota = 1000;

  async function loadProjects() {
    loading = true;
    try {
      const response = await api.get('/projects');
      projects = Array.isArray(response) ? response : (response?.data || []);
    } catch (err) {
      toast.add('Failed to load projects', 'error');
      projects = [];
    } finally {
      loading = false;
    }
  }

  async function handleCreateProject() {
    if (!newProjectName.trim()) return;
    try {
      await api.post('/projects', { name: newProjectName });
      newProjectName = '';
      showCreateModal = false;
      toast.add('Project created successfully', 'success');
      // Force immediate refresh
      await loadProjects();
    } catch (err) {
      toast.add('Failed to create project', 'error');
    }
  }

  async function handleUpdateQuota() {
    if (!selectedProject) return;
    try {
      await api.patch(`/projects/${selectedProject.id}/quota`, { max_events_per_month: editingQuota });
      toast.add('Quota updated successfully', 'success');
      showQuotaModal = false;
      // Force immediate refresh
      await loadProjects();
    } catch (err) {
      toast.add('Failed to update quota', 'error');
    }
  }

  async function handleDeleteProject(id) {
    if (!confirm('Are you sure you want to delete this project? This action cannot be undone.')) return;
    try {
      await api.delete(`/projects/${id}`);
      toast.add('Project deleted successfully', 'success');
      // Force immediate refresh - remove from list immediately for better UX
      projects = (projects || []).filter(p => p.id !== id);
      // Then refresh to ensure consistency
      await loadProjects();
    } catch (err) {
      toast.add('Failed to delete project', 'error');
      // Reload on error to ensure state is correct
      await loadProjects();
    }
  }

  function openQuotaModal(project) {
    selectedProject = project;
    editingQuota = project.max_events_per_month;
    showQuotaModal = true;
  }

  onMount(loadProjects);

  $: filteredProjects = (projects || []).filter(p =>
    p.name?.toLowerCase().includes(searchQuery.toLowerCase())
  );
</script>

<div class="space-y-6 sm:space-y-8">
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 mb-5">
    <div>
      <h1 class="text-xl font-semibold tracking-tight text-white mb-0.5">Projects</h1>
      <p class="text-sm text-slate-400">Manage your monitoring targets and quotas</p>
    </div>
    <button
      on:click={() => showCreateModal = true}
      class="pulse-button-primary flex items-center gap-2 w-full sm:w-auto justify-center"
    >
      <Plus size={18} />
      <span>New Project</span>
    </button>
  </div>

  <!-- Search and Actions -->
  <div class="flex items-center gap-4">
    <div class="relative flex-1">
      <Search class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" size={18} />
      <input
        type="text"
        placeholder="Search projects..."
        bind:value={searchQuery}
        class="pulse-input w-full pl-10 font-medium"
      />
    </div>
  </div>

  {#if loading}
    <div class="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {#each Array(4) as _}
        <div class="pulse-card h-[200px] skeleton rounded-2xl"></div>
      {/each}
    </div>
  {:else if filteredProjects.length === 0}
    <div class="pulse-card flex flex-col items-center justify-center py-20 text-center">
      <div class="mb-4 rounded-full bg-slate-800/50 p-4 text-slate-500">
        <Activity size={48} />
      </div>
      <h3 class="text-xl font-bold text-white">No projects found</h3>
      <p class="mt-2 text-slate-400">Create your first project to start tracking errors</p>
    </div>
  {:else}
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {#each filteredProjects as project}
        <div
          role="button"
          tabindex="0"
          on:click={() => navigate(`/projects/${project.id}`)}
          on:keydown={(e) => e.key === 'Enter' && navigate(`/projects/${project.id}`)}
          class="pulse-card group relative p-6 transition-all duration-300 hover:bg-white/[0.06] cursor-pointer min-h-[200px] flex flex-col justify-between"
        >
          <div class="mb-3 flex items-start justify-between">
            <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-pulse-500/10 text-pulse-400">
              <Activity size={18} />
            </div>
            <div class="flex items-center gap-1">
              <button
                on:click|stopPropagation={() => openQuotaModal(project)}
                class="rounded-md p-1 text-slate-500 hover:bg-white/10 hover:text-white transition-colors"
                title="Manage Quota"
              >
                <Settings2 size={14} />
              </button>
              <button
                on:click|stopPropagation={() => handleDeleteProject(project.id)}
                class="rounded-md p-1 text-slate-500 hover:bg-red-500/10 hover:text-red-500 transition-colors"
                title="Delete Project"
              >
                <Trash2 size={14} />
              </button>
            </div>
          </div>

          <div class="flex items-center justify-between mb-2.5">
            <h3 class="text-xs font-semibold text-white group-hover:text-pulse-400 transition-colors truncate pr-2">
              {project.name}
            </h3>
            <span class="text-[8px] font-mono text-slate-600 bg-white/5 px-1.5 py-0.5 rounded uppercase">
              {project.id.split('-')[0]}
            </span>
          </div>

          <div class="space-y-3">
            <!-- Quota usage -->
            <div>
              <div class="flex items-center justify-between text-[10px] mb-1">
                <span class="text-slate-500">Usage</span>
                <span class="font-medium {project.current_month_events >= project.max_events_per_month ? 'text-red-400' : 'text-slate-300'}">
                  {project.current_month_events} / {project.max_events_per_month}
                </span>
              </div>
              <div class="h-1 w-full overflow-hidden rounded-full bg-white/5">
                <div
                  class="h-full rounded-full transition-all duration-500 {project.current_month_events >= project.max_events_per_month ? 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.4)]' : 'bg-pulse-500 shadow-[0_0_8px_rgba(139,92,246,0.4)]'}"
                  style="width: {Math.min(100, (project.current_month_events / project.max_events_per_month) * 100)}%"
                ></div>
              </div>
            </div>

            <div class="flex items-center justify-between text-[9px] text-slate-600 pt-2 border-t border-white/5">
              <div class="flex items-center gap-1">
                <Hash size={10} />
                <span class="font-mono">{project.api_key.slice(0, 8)}...</span>
              </div>
              <div class="flex items-center gap-1 text-pulse-500 opacity-0 group-hover:opacity-100 transition-opacity">
                <span>DETAILS</span>
                <ExternalLink size={10} />
              </div>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Create Project Modal -->
{#if showCreateModal}
  <div class="fixed inset-0 z-[2000] flex items-center justify-center p-4">
    <button
      type="button"
      class="absolute inset-0 bg-black/60 backdrop-blur-sm cursor-default"
      on:click={() => showCreateModal = false}
      aria-label="Close modal"
    ></button>
    <div class="pulse-card relative w-full max-w-md border-white/10 p-6 sm:p-8 shadow-2xl">
      <h2 class="text-xl sm:text-2xl font-bold text-white mb-2">Create New Project</h2>
      <p class="text-xs sm:text-sm text-slate-400 mb-6">Enter a name for your monitoring target</p>

      <div class="space-y-4">
        <div>
          <label for="name" class="block text-sm font-medium text-slate-400 mb-1.5">Project Name</label>
          <input
            id="name"
            type="text"
            bind:value={newProjectName}
            placeholder="e.g. Production API"
            class="pulse-input w-full"
            on:keydown={(e) => e.key === 'Enter' && handleCreateProject()}
          />
        </div>

        <div class="flex gap-3 pt-4">
          <button
            on:click={() => showCreateModal = false}
            class="pulse-button flex-1 bg-white/5 text-white hover:bg-white/10"
          >
            Cancel
          </button>
          <button
            on:click={handleCreateProject}
            class="pulse-button-primary flex-1"
            disabled={!newProjectName.trim()}
          >
            Create Project
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- Edit Quota Modal -->
{#if showQuotaModal && selectedProject}
  <div class="fixed inset-0 z-[2000] flex items-center justify-center p-4">
    <button
      type="button"
      class="absolute inset-0 bg-black/60 backdrop-blur-sm cursor-default"
      on:click={() => showQuotaModal = false}
      aria-label="Close modal"
    ></button>
    <div class="pulse-card relative w-full max-w-sm border-white/10 p-6 sm:p-8 shadow-2xl">
      <h2 class="text-xl sm:text-2xl font-bold text-white mb-2">Adjust Quota</h2>
      <p class="text-xs sm:text-sm text-slate-400 mb-6">Set the monthly event limit for <span class="text-pulse-400 font-bold">{selectedProject.name}</span></p>

      <div class="space-y-4">
        <div>
          <label for="quota" class="block text-sm font-medium text-slate-400 mb-1.5">Monthly Limit (events)</label>
          <div class="relative">
            <input
              id="quota"
              type="number"
              bind:value={editingQuota}
              class="pulse-input w-full pr-12"
            />
            <span class="absolute right-3 top-1/2 -translate-y-1/2 text-[10px] font-bold text-slate-600 uppercase">MONTH</span>
          </div>
          <p class="mt-2 text-[10px] text-slate-500 leading-tight">
            Once this limit is reached, incoming events will be rejected with a 429 status code.
          </p>
        </div>

        <div class="flex gap-3 pt-4">
          <button
            on:click={() => showQuotaModal = false}
            class="pulse-button flex-1 bg-white/5 text-white hover:bg-white/10"
          >
            Cancel
          </button>
          <button
            on:click={handleUpdateQuota}
            class="pulse-button-primary flex-1"
          >
            Update Quota
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}

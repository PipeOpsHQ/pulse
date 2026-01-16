<script>
  import { onMount } from 'svelte';
  import { ShieldCheck, Lock, Key, RefreshCw, Globe, ShieldAlert, CheckCircle, Info, Copy, Eye, EyeOff, Activity } from 'lucide-svelte';
  import { api } from '../lib/api';
  import { toast } from '../stores/toast';
  import { user } from '../stores/auth';

  let projects = [];
  let selectedProjectId = '';
  let selectedProject = null;
  let loading = true;
  let rotationLoading = false;
  let policiesLoading = false;
  let mfaSetupLoading = false;

  // MFA State
  let showMFAModal = false;
  let mfaSecret = '';
  let mfaUrl = '';
  let mfaVerificationCode = '';

  // Policy State
  let policies = {
    ip_whitelist: '',
    allowed_domains: '',
    enforced: false
  };
  let keyHistory = [];

  onMount(async () => {
    try {
      projects = await api.get('/projects') || [];
      if (projects.length > 0) {
        selectedProjectId = projects[0].id;
        await loadProjectSecurity();
      }
    } catch (err) {
      toast.add('Failed to load project details', 'error');
    } finally {
      loading = false;
    }
  });

  async function loadProjectSecurity() {
    if (!selectedProjectId) return;
    policiesLoading = true;
    try {
      selectedProject = projects.find(p => p.id === selectedProjectId);
      policies = await api.get(`/projects/${selectedProjectId}/security-policies`) || { ip_whitelist: '', allowed_domains: '', enforced: false };
      keyHistory = await api.get(`/projects/${selectedProjectId}/key-history`) || [];
    } catch (err) {
      toast.add('Failed to load security policies', 'error');
    } finally {
      policiesLoading = false;
    }
  }

  async function handleRotateKey() {
    if (!confirm('Are you sure you want to rotate the API key? The current key will be invalidated immediately.')) return;
    rotationLoading = true;
    try {
      const resp = await api.post(`/projects/${selectedProjectId}/rotate-key`);
      toast.add('API Key rotated successfully', 'success');
      // Update local storage/state if needed, but usually just refresh history
      await loadProjectSecurity();
    } catch (err) {
      toast.add('Failed to rotate API key', 'error');
    } finally {
      rotationLoading = false;
    }
  }

  async function handleUpdatePolicies() {
    policiesLoading = true;
    try {
      await api.post(`/projects/${selectedProjectId}/security-policies`, policies);
      toast.add('Security policies updated', 'success');
      // Reload policies to ensure UI reflects server state
      await loadProjectSecurity();
    } catch (err) {
      toast.add('Failed to update policies', 'error');
    } finally {
      policiesLoading = false;
    }
  }

  async function startMFASetup() {
    mfaSetupLoading = true;
    try {
      const resp = await api.post('/security/mfa/setup');
      mfaSecret = resp.secret;
      mfaUrl = resp.url;
      showMFAModal = true;
    } catch (err) {
      toast.add('Failed to start MFA setup', 'error');
    } finally {
      mfaSetupLoading = false;
    }
  }

  async function verifyAndEnableMFA() {
    if (!mfaVerificationCode) return;
    try {
      await api.post('/security/mfa/enable', {
        secret: mfaSecret,
        code: mfaVerificationCode
      });
      toast.add('MFA enabled successfully', 'success');
      showMFAModal = false;
      // Refresh user state
      const updatedUser = await api.get('/auth/me');
      user.set(updatedUser);
    } catch (err) {
      toast.add('Invalid verification code', 'error');
    }
  }

  function copyToClipboard(text, message = 'Copied to clipboard') {
    navigator.clipboard.writeText(text);
    toast.add(message, 'success');
  }
</script>

<div class="space-y-6 sm:space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
  <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
    <div>
      <h1 class="text-2xl sm:text-3xl font-bold text-white font-display">Security Vault</h1>
      <p class="text-sm text-slate-400">Manage infrastructure security, API policies, and multi-factor authentication.</p>
    </div>
  </div>

  <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
    <!-- Left Column: MFA and Global Status -->
    <div class="space-y-6">
      <div class="pulse-card border-white/10 p-6 relative overflow-hidden group">
        <div class="absolute -right-4 -top-4 text-emerald-500/5 transition-transform group-hover:scale-110">
          <ShieldCheck size={120} />
        </div>

        <div class="flex items-center gap-3 mb-6">
          <div class="h-10 w-10 rounded-lg bg-emerald-500/10 flex items-center justify-center text-emerald-500">
            <Lock size={20} />
          </div>
          <div>
            <h2 class="text-lg font-bold text-white">Identity Protection</h2>
            <p class="text-xs text-slate-500">Secure your administrative access</p>
          </div>
        </div>

        <div class="space-y-4">
          <div class="flex items-center justify-between p-3 rounded-lg bg-white/5 border border-white/10">
            <div class="flex items-center gap-2">
              <div class="h-2 w-2 rounded-full {$user?.mfa_enabled ? 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]' : 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.5)]'}"></div>
              <span class="text-sm font-medium text-white">2FA / MFA</span>
            </div>
            <span class="text-[10px] font-bold uppercase tracking-widest {$user?.mfa_enabled ? 'text-emerald-500' : 'text-red-500'}">
              {$user?.mfa_enabled ? 'Active' : 'Missing'}
            </span>
          </div>

          {#if !$user?.mfa_enabled}
            <div class="bg-amber-500/10 border border-amber-500/20 rounded-lg p-3">
              <div class="flex items-start gap-2">
                <ShieldAlert size={14} class="text-amber-500 mt-0.5 shrink-0" />
                <p class="text-[10px] text-amber-200/80 leading-relaxed">
                  Highly Recommended: Enable MFA to protect your administrator account from unauthorized access.
                </p>
              </div>
            </div>
          {/if}

          <button
            on:click={startMFASetup}
            disabled={$user?.mfa_enabled || mfaSetupLoading}
            class="pulse-button w-full flex items-center justify-center gap-2 py-3 {$user?.mfa_enabled ? 'bg-white/5 text-slate-500' : 'pulse-button-primary'}"
          >
            {#if mfaSetupLoading}
              <RefreshCw size={16} class="animate-spin" />
            {:else}
              <ShieldCheck size={18} />
            {/if}
            <span>{$user?.mfa_enabled ? 'MFA Protected' : 'Setup Multi-Factor'}</span>
          </button>
        </div>
      </div>

      <div class="pulse-card border-white/10 p-6">
        <h3 class="text-sm font-bold text-white mb-4 flex items-center gap-2">
          <Info size={14} class="text-pulse-400" />
          Security Audit
        </h3>
        <div class="space-y-3">
          <div class="flex items-center justify-between text-xs">
            <span class="text-slate-500">Last Login IP</span>
            <span class="text-slate-300 font-mono">127.0.0.1</span>
          </div>
          <div class="flex items-center justify-between text-xs">
            <span class="text-slate-500">Active Sessions</span>
            <span class="text-slate-300">1</span>
          </div>
          <div class="flex items-center justify-between text-xs">
            <span class="text-slate-500">System Visibility</span>
            <span class="text-emerald-500 font-bold">Public</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Right Column: Project Security Policies & Key Rotation -->
    <div class="lg:col-span-2 space-y-6">
      <div class="pulse-card border-white/10 p-6">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-8">
          <div class="flex items-center gap-3">
            <div class="h-10 w-10 rounded-lg bg-pulse-500/10 flex items-center justify-center text-pulse-400">
              <Globe size={20} />
            </div>
            <div>
              <h2 class="text-lg font-bold text-white">Project Ingress Policies</h2>
              <p class="text-xs text-slate-500">Restrict how errors are ingested</p>
            </div>
          </div>

          <select
            bind:value={selectedProjectId}
            on:change={loadProjectSecurity}
            class="pulse-input sm:w-64"
          >
            {#each projects as project}
              <option value={project.id}>{project.name}</option>
            {/each}
          </select>
        </div>

        {#if policiesLoading}
          <div class="space-y-6 animate-pulse">
            <div class="h-12 bg-white/5 rounded-lg"></div>
            <div class="h-32 bg-white/5 rounded-lg"></div>
          </div>
        {:else}
          <div class="space-y-6">
            <div class="flex items-center justify-between p-4 rounded-xl bg-white/5 border border-white/10 transition-all {policies.enforced ? 'border-pulse-500/30' : ''}">
              <div class="flex items-center gap-3">
                <div class="h-8 w-8 rounded-lg flex items-center justify-center {policies.enforced ? 'bg-pulse-500/20 text-pulse-400' : 'bg-slate-500/10 text-slate-500'}">
                  <ShieldCheck size={18} />
                </div>
                <div>
                  <div class="text-sm font-bold text-white">Enforce Policies</div>
                  <div class="text-[10px] text-slate-500">Reject events failing security checks</div>
                </div>
              </div>
              <label class="relative inline-flex cursor-pointer items-center">
                <input type="checkbox" bind:checked={policies.enforced} class="peer sr-only">
                <div class="h-6 w-11 rounded-full bg-slate-800 transition-all after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all peer-checked:bg-pulse-600 peer-checked:after:translate-x-full peer-checked:after:border-white"></div>
              </label>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div class="space-y-2">
                <label for="ip-whitelist" class="text-xs font-bold text-slate-500 uppercase tracking-widest flex items-center gap-2">
                  <Activity size={12} />
                  IP Whitelist
                </label>
                <textarea
                  id="ip-whitelist"
                  bind:value={policies.ip_whitelist}
                  placeholder="e.g. 192.168.1.1, 10.0.0.0/24"
                  class="pulse-input w-full h-24 text-sm font-mono"
                ></textarea>
                <p class="text-[10px] text-slate-600">Comma-separated IPv4/v6 addresses or CIDR ranges.</p>
              </div>

              <div class="space-y-2">
                <label for="allowed-domains" class="text-xs font-bold text-slate-500 uppercase tracking-widest flex items-center gap-2">
                  <Globe size={12} />
                  Allowed Domains
                </label>
                <textarea
                  id="allowed-domains"
                  bind:value={policies.allowed_domains}
                  placeholder="e.g. app.example.com, localhost"
                  class="pulse-input w-full h-24 text-sm font-mono"
                ></textarea>
                <p class="text-[10px] text-slate-600">Restrict ingestion to specific Origin/Referer headers.</p>
              </div>
            </div>

            <div class="flex justify-end">
              <button
                on:click={handleUpdatePolicies}
                class="pulse-button pulse-button-primary px-8"
              >
                Save Policies
              </button>
            </div>
          </div>
        {/if}
      </div>

      <!-- Key Rotation Section -->
      <div class="pulse-card border-white/10 p-6">
        <div class="flex items-center justify-between mb-8">
          <div class="flex items-center gap-3">
            <div class="h-10 w-10 rounded-lg bg-amber-500/10 flex items-center justify-center text-amber-500">
              <Key size={20} />
            </div>
            <div>
              <h2 class="text-lg font-bold text-white">API Key Rotation</h2>
              <p class="text-xs text-slate-500">Regenerate project credentials regularly</p>
            </div>
          </div>

          <button
            on:click={handleRotateKey}
            disabled={rotationLoading}
            class="pulse-button flex items-center gap-2 px-4 py-2 border border-red-500/30 text-red-500 hover:bg-red-500/10"
          >
            {#if rotationLoading}
              <RefreshCw size={14} class="animate-spin" />
            {:else}
              <RefreshCw size={14} />
            {/if}
            <span>Rotate Key</span>
          </button>
        </div>

        <div class="space-y-4">
          <div>
            <label for="active-api-key" class="text-[10px] font-bold text-slate-500 uppercase tracking-widest mb-2 block">Active API Key</label>
            <div class="flex gap-2">
              <div id="active-api-key" class="pulse-input flex-1 font-mono text-sm bg-black/40 border-white/5 flex items-center px-4 overflow-hidden truncate italic text-slate-400">
                {selectedProject?.api_key || 'Loading...'}
              </div>
              <button
                on:click={() => copyToClipboard(selectedProject?.api_key)}
                class="h-10 w-10 shrink-0 border border-white/10 rounded-lg flex items-center justify-center text-slate-500 hover:text-white hover:bg-white/5 transition-all"
              >
                <Copy size={16} />
              </button>
            </div>
          </div>

          <div class="pt-4 border-t border-white/5">
            <h3 class="text-[10px] font-bold text-slate-500 uppercase tracking-widest mb-4 block">Rotation History</h3>
            {#if keyHistory.length === 0}
              <p class="text-xs text-slate-600 italic">No previous keys found for this project.</p>
            {:else}
              <div class="space-y-2">
                {#each keyHistory as entry}
                  <div class="flex items-center justify-between p-3 rounded-lg bg-white/[0.02] border border-white/5 text-xs">
                    <div class="flex items-center gap-3">
                      <Lock size={12} class="text-slate-700" />
                      <span class="font-mono text-slate-500">{entry.api_key.slice(0, 16)}...</span>
                    </div>
                    <span class="text-slate-600">{new Date(entry.created_at).toLocaleDateString()}</span>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        </div>
      </div>
    </div>
  </div>
</div>

<!-- MFA Setup Modal -->
{#if showMFAModal}
  <div class="fixed inset-0 z-[2000] flex items-center justify-center p-4 animate-in fade-in duration-300">
    <div
      role="button"
      tabindex="0"
      class="absolute inset-0 bg-black/80 backdrop-blur-xl"
      on:click={() => showMFAModal = false}
      on:keydown={(e) => e.key === 'Escape' && (showMFAModal = false)}
    ></div>
    <div class="pulse-card relative w-full max-w-md border-white/10 p-8 shadow-2xl bg-[#0a0a0a]">
      <div class="flex flex-col items-center text-center">
        <div class="h-16 w-16 rounded-2xl bg-pulse-500/10 flex items-center justify-center text-pulse-400 mb-6">
          <ShieldCheck size={32} />
        </div>
        <h2 class="text-2xl font-bold text-white mb-2">Enable Multi-Factor Auth</h2>
        <p class="text-sm text-slate-400 mb-8">Scan the secret below in your authenticator app (Google Authenticator, Authy, etc.)</p>

        <div class="w-full space-y-6">
          <div class="p-4 bg-white rounded-xl mb-4 flex items-center justify-center">
            <!-- For demo, we just show a placeholder QR because we don't have a QR lib yet,
                 but we'd normally render one here using the mfaUrl -->
            <div class="text-black text-center">
              <div class="text-[10px] font-bold uppercase mb-2">Authenticator QR</div>
              <div class="bg-black/5 p-4 rounded-lg">
                <Lock size={48} class="mx-auto mb-2 opacity-20" />
                <p class="text-[8px] max-w-[120px] font-mono break-all">{mfaUrl}</p>
              </div>
            </div>
          </div>

          <div class="space-y-4">
            <div>
              <label for="mfa-verification-code" class="block text-left text-xs font-bold text-slate-500 uppercase tracking-widest mb-2">Verification Code</label>
              <input
                type="text"
                id="mfa-verification-code"
                bind:value={mfaVerificationCode}
                placeholder="000 000"
                class="pulse-input w-full text-center text-xl tracking-[0.5em] font-bold"
                maxlength="6"
              />
            </div>

            <div class="flex gap-3 pt-4">
              <button
                on:click={() => showMFAModal = false}
                class="pulse-button flex-1 bg-white/5 text-white hover:bg-white/10"
              >
                Cancel
              </button>
              <button
                on:click={verifyAndEnableMFA}
                disabled={!mfaVerificationCode || mfaVerificationCode.length < 6}
                class="pulse-button-primary flex-1 py-3"
              >
                Verify & Enable
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  /* No special styles needed for now */
</style>

<script>
  import { onMount } from "svelte";
  import {
    Shield,
    User,
    Bell,
    Database,
    Save,
    Mail,
    Heart,
    Slack,
    Globe,
    Info,
    ChevronRight,
    ExternalLink,
    Terminal,
    Lock,
  } from "lucide-svelte";
  import { api } from "../lib/api";
  import { toast } from "../stores/toast";
  import { ensureHttps } from "../lib/utils";

  let sections = [
    { id: "profile", label: "Profile", icon: User },
    { id: "notifications", label: "Notifications", icon: Bell },
    { id: "smtp", label: "Email (SMTP)", icon: Mail },
    { id: "security", label: "Security", icon: Shield },
    { id: "advanced", label: "Advanced", icon: Database },
    { id: "support", label: "Pulse Roadmap", icon: Heart },
  ];

  let activeSection = "profile";

  // State for settings
  let smtpSettings = {
    host: "",
    port: 587,
    user: "",
    pass: "",
  };

  let notificationSettings = {
    slackWebhook: "",
    genericWebhook: "",
  };

  let globalSettings = {
    retentionDays: 30,
  };

  let maintenanceLoading = false;

  async function loadSettings() {
    try {
      const data = await api.get("/settings");
      if (data) {
        notificationSettings.slackWebhook = data.slack_webhook || "";
        notificationSettings.genericWebhook = data.generic_webhook || "";
        smtpSettings.host = data.smtp_host || "";
        smtpSettings.port = parseInt(data.smtp_port) || 587;
        smtpSettings.user = data.smtp_user || "";
        if (data.smtp_host) {
          smtpSettings.pass = "••••••••••••";
        }
        globalSettings.retentionDays = parseInt(data.retention_days) || 30;
      }
    } catch (e) {
      console.error("Failed to fetch settings:", e);
    }
  }

  onMount(async () => {
    await loadSettings();
  });

  async function handleSave() {
    try {
      const payload = {
        slack_webhook: ensureHttps(notificationSettings.slackWebhook),
        generic_webhook: ensureHttps(notificationSettings.genericWebhook),
        smtp_host: smtpSettings.host,
        smtp_port: smtpSettings.port.toString(),
        smtp_user: smtpSettings.user,
        retention_days: globalSettings.retentionDays.toString(),
      };

      if (smtpSettings.pass && smtpSettings.pass !== "••••••••••••") {
        payload.smtp_pass = smtpSettings.pass;
      }

      await api.post("/settings", payload);
      toast.add("Settings saved successfully", "success");
      // Reload settings to ensure UI reflects server state
      await loadSettings();
    } catch (e) {
      toast.add("Failed to save settings", "error");
    }
  }

  async function handleManualCleanup() {
    maintenanceLoading = true;
    try {
      const result = await api.post("/system/cleanup", {});
      toast.add(
        `Cleanup complete! Deleted ${result.deleted || 0} old errors.`,
        "success",
      );
      // Refresh settings to show updated state
      await loadSettings();
    } catch (err) {
      toast.add("Failed to run system cleanup", "error");
    } finally {
      maintenanceLoading = false;
    }
  }
</script>

<div
  class="space-y-6 sm:space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500"
>
  <div>
    <h1 class="text-xl font-semibold tracking-tight text-white mb-0.5">
      Settings
    </h1>
    <p class="text-xs text-slate-400">
      Manage your account and global configurations
    </p>
  </div>

  <div class="flex flex-col gap-8 lg:flex-row">
    <!-- Sidebar -->
    <div class="w-full lg:w-72 shrink-0">
      <nav
        class="flex flex-row lg:flex-col gap-1.5 overflow-x-auto no-scrollbar lg:sticky lg:top-8 pb-4 lg:pb-0"
      >
        {#each sections as section}
          <button
            on:click={() => (activeSection = section.id)}
            class="flex items-center justify-between rounded-xl px-4 py-3 text-xs sm:text-sm font-medium transition-all group whitespace-nowrap lg:w-full {activeSection ===
            section.id
              ? 'bg-pulse-500/10 text-white shadow-[inset_0_0_0_1px_rgba(139,92,246,0.2)]'
              : 'text-slate-400 hover:bg-white/5 hover:text-slate-200'}"
          >
            <div class="flex items-center gap-2 sm:gap-3">
              <svelte:component
                this={section.icon}
                size={18}
                class={activeSection === section.id
                  ? "text-pulse-400"
                  : "group-hover:text-slate-200"}
              />
              <span>{section.label}</span>
            </div>
            {#if activeSection === section.id}
              <ChevronRight size={14} class="text-pulse-500 hidden lg:block" />
            {/if}
          </button>
        {/each}
      </nav>
    </div>

    <!-- Content -->
    <div class="flex-1 min-w-0">
      <div class="pulse-card overflow-hidden">
        <div class="p-4 sm:p-8">
          {#if activeSection === "profile"}
            <div class="space-y-6">
              <div class="flex items-center gap-3 mb-2">
                <div
                  class="h-10 w-10 rounded-full bg-pulse-500/10 border border-pulse-500/20 flex items-center justify-center text-pulse-400"
                >
                  <User size={20} />
                </div>
                <div>
                  <h2 class="text-xl font-bold text-white">Profile Settings</h2>
                  <p
                    class="text-xs text-slate-500 uppercase font-bold tracking-tighter"
                  >
                    Your personal identity in Pulse
                  </p>
                </div>
              </div>

              <div class="grid gap-6 md:grid-cols-2 pt-4">
                <div class="space-y-2">
                  <label
                    for="profile-name"
                    class="text-sm font-medium text-slate-400">Full Name</label
                  >
                  <input
                    type="text"
                    id="profile-name"
                    class="pulse-input w-full"
                    value="Administrator"
                  />
                </div>
                <div class="space-y-2">
                  <label
                    for="profile-email"
                    class="text-sm font-medium text-slate-400"
                    >Email Address</label
                  >
                  <input
                    type="email"
                    id="profile-email"
                    class="pulse-input w-full bg-white/[0.02]"
                    value="admin@example.com"
                    disabled
                  />
                  <p class="text-[10px] text-slate-600 italic">
                    Managed by system administrator
                  </p>
                </div>
              </div>
              <div class="space-y-2">
                <label
                  for="profile-bio"
                  class="text-sm font-medium text-slate-400">Bio</label
                >
                <textarea
                  id="profile-bio"
                  class="pulse-input w-full h-32"
                  placeholder="Tell us about yourself..."
                ></textarea>
              </div>
            </div>
          {:else if activeSection === "notifications"}
            <div class="space-y-8">
              <div class="flex items-center gap-3">
                <div
                  class="h-10 w-10 rounded-full bg-pulse-500/10 border border-pulse-500/20 flex items-center justify-center text-pulse-400"
                >
                  <Bell size={20} />
                </div>
                <div>
                  <h2 class="text-xl font-bold text-white">
                    Service Notifications
                  </h2>
                  <p
                    class="text-xs text-slate-500 uppercase font-bold tracking-tighter"
                  >
                    Hook Pulse into your workflow
                  </p>
                </div>
              </div>

              <div class="space-y-6 pt-4">
                <!-- Slack -->
                <div
                  class="p-5 rounded-2xl bg-white/[0.03] border border-white/5 space-y-4"
                >
                  <div class="flex items-center justify-between">
                    <div class="flex items-center gap-2">
                      <Slack size={18} class="text-emerald-400" />
                      <span class="font-bold text-white">Slack Webhook</span>
                    </div>
                    <span
                      class="text-[9px] font-bold px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-500 uppercase tracking-widest"
                      >Active</span
                    >
                  </div>
                  <div class="space-y-2">
                    <input
                      type="text"
                      class="pulse-input w-full font-mono text-xs"
                      bind:value={notificationSettings.slackWebhook}
                    />
                    <p class="text-[10px] text-slate-500">
                      Events will be posted to the channel associated with this
                      webhook.
                    </p>
                  </div>
                </div>

                <!-- Generic Webhook -->
                <div
                  class="p-5 rounded-2xl bg-white/[0.03] border border-white/5 space-y-4"
                >
                  <div class="flex items-center justify-between">
                    <div class="flex items-center gap-2">
                      <Globe size={18} class="text-blue-400" />
                      <span class="font-bold text-white">Generic Webhook</span>
                    </div>
                    <span
                      class="text-[9px] font-bold px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-500 uppercase tracking-widest"
                      >Active</span
                    >
                  </div>
                  <div class="space-y-2">
                    <input
                      type="text"
                      class="pulse-input w-full font-mono text-xs"
                      bind:value={notificationSettings.genericWebhook}
                    />
                    <p class="text-[10px] text-slate-500 text-right">
                      POST JSON payload to this endpoint on every critical
                      error.
                    </p>
                  </div>
                </div>
              </div>
            </div>
          {:else if activeSection === "smtp"}
            <div class="space-y-8">
              <div class="flex items-center gap-3">
                <div
                  class="h-10 w-10 rounded-full bg-pulse-500/10 border border-pulse-500/20 flex items-center justify-center text-pulse-400"
                >
                  <Mail size={20} />
                </div>
                <div>
                  <h2 class="text-xl font-bold text-white">
                    Email Configuration (SMTP)
                  </h2>
                  <p
                    class="text-xs text-slate-500 uppercase font-bold tracking-tighter"
                  >
                    System email delivery details
                  </p>
                </div>
              </div>

              <div class="grid gap-6 md:grid-cols-12 pt-4">
                <div class="md:col-span-9 space-y-2">
                  <label
                    for="smtp-host"
                    class="text-sm font-medium text-slate-400">Host</label
                  >
                  <input
                    type="text"
                    id="smtp-host"
                    class="pulse-input w-full"
                    bind:value={smtpSettings.host}
                  />
                </div>
                <div class="md:col-span-3 space-y-2">
                  <label
                    for="smtp-port"
                    class="text-sm font-medium text-slate-400">Port</label
                  >
                  <input
                    type="number"
                    id="smtp-port"
                    class="pulse-input w-full"
                    bind:value={smtpSettings.port}
                  />
                </div>
                <div class="md:col-span-6 space-y-2">
                  <label
                    for="smtp-user"
                    class="text-sm font-medium text-slate-400">User</label
                  >
                  <input
                    type="text"
                    id="smtp-user"
                    class="pulse-input w-full"
                    bind:value={smtpSettings.user}
                  />
                </div>
                <div class="md:col-span-6 space-y-2">
                  <label
                    for="smtp-pass"
                    class="text-sm font-medium text-slate-400">Password</label
                  >
                  <input
                    type="password"
                    id="smtp-pass"
                    class="pulse-input w-full"
                    bind:value={smtpSettings.pass}
                  />
                </div>
              </div>

              <div
                class="bg-pulse-500/5 rounded-xl p-4 border border-pulse-500/10 flex gap-3"
              >
                <Info size={16} class="text-pulse-400 shrink-0 mt-0.5" />
                <p class="text-xs text-slate-400 leading-relaxed">
                  Pulse uses SMTP for sending automated alerts, password resets,
                  and quota reminders. We recommend a dedicated transactional
                  mail provider like SendGrid or AWS SES.
                </p>
              </div>
            </div>
          {:else if activeSection === "support"}
            <div class="space-y-8">
              <div class="flex items-center gap-3">
                <div
                  class="h-10 w-10 rounded-full bg-pulse-500/10 border border-pulse-500/20 flex items-center justify-center text-pink-400"
                >
                  <Heart size={20} />
                </div>
                <div>
                  <h2 class="text-xl font-bold text-white">
                    Pulse Roadmap & Support
                  </h2>
                  <p
                    class="text-xs text-slate-500 uppercase font-bold tracking-tighter"
                  >
                    Help us shape the future of Pulse
                  </p>
                </div>
              </div>

              <div class="grid gap-6 md:grid-cols-2">
                <div
                  class="p-6 rounded-2xl bg-white/[0.03] border border-white/5 group hover:border-pulse-500/30 transition-all"
                >
                  <div class="flex items-center justify-between mb-4">
                    <span
                      class="text-xs font-bold text-slate-500 uppercase tracking-widest"
                      >Immediate Works</span
                    >
                    <Terminal size={14} class="text-slate-600" />
                  </div>
                  <ul class="space-y-3">
                    {#each ["Advanced Query Builder", "Team Workspaces", "Log Aggregation", "Uptime Monitoring"] as feature}
                      <li
                        class="flex items-center gap-3 text-sm text-slate-300"
                      >
                        <div class="h-1 w-1 rounded-full bg-pulse-500"></div>
                        {feature}
                      </li>
                    {/each}
                  </ul>
                </div>

                <div
                  class="p-6 rounded-2xl bg-pulse-600/10 border border-pulse-500/20 flex flex-col justify-between"
                >
                  <div>
                    <h4 class="text-white font-bold mb-2">
                      Support Open Source
                    </h4>
                    <p class="text-xs text-slate-400 leading-relaxed">
                      Pulse is built with love by the community. Your
                      contributions whether code or financial help us stay
                      independent.
                    </p>
                  </div>
                  <button
                    class="mt-6 flex h-10 items-center justify-center gap-2 rounded-lg bg-pulse-600 font-bold text-white hover:bg-pulse-500 transition-colors"
                  >
                    <Heart size={16} />
                    <span>Sponsor Project</span>
                  </button>
                </div>
              </div>

              <div
                class="flex justify-center gap-8 py-4 opacity-50 border-t border-white/5 pt-8"
              >
                <a
                  href="https://github.com/nitrocode"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="flex items-center gap-1.5 text-xs text-slate-400 hover:text-white transition-colors"
                >
                  <ExternalLink size={12} /> Github
                </a>
                <a
                  href="https://docs.pulse-oss.com"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="flex items-center gap-1.5 text-xs text-slate-400 hover:text-white transition-colors"
                >
                  <ExternalLink size={12} /> Documentation
                </a>
                <a
                  href="https://discord.gg/pulse"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="flex items-center gap-1.5 text-xs text-slate-400 hover:text-white transition-colors"
                >
                  <ExternalLink size={12} /> Discord
                </a>
              </div>
            </div>
          {:else if activeSection === "security"}
            <div
              class="flex flex-col items-center justify-center py-20 text-center text-slate-500"
            >
              <Lock size={48} class="mb-4 opacity-20" />
              <h3 class="text-lg font-bold text-slate-300 mb-1">
                Security Vault
              </h3>
              <p class="text-sm mb-6">
                Manage security policies, API rotation, and MFA.
              </p>
              <a
                href="/security-vault"
                class="pulse-button-primary px-6 py-2 rounded-lg font-bold transition-all hover:scale-105"
              >
                Enter Security Vault
              </a>
            </div>
          {:else if activeSection === "advanced"}
            <div class="space-y-8">
              <div class="flex items-center gap-3">
                <div
                  class="h-10 w-10 rounded-full bg-pulse-500/10 border border-pulse-500/20 flex items-center justify-center text-emerald-400"
                >
                  <Database size={20} />
                </div>
                <div>
                  <h2 class="text-xl font-bold text-white">
                    Advanced Configuration
                  </h2>
                  <p
                    class="text-xs text-slate-500 uppercase font-bold tracking-tighter"
                  >
                    Fine-tune system behavior
                  </p>
                </div>
              </div>

              <div class="space-y-6 pt-4">
                <!-- Data Retention -->
                <div
                  class="p-6 rounded-2xl bg-white/[0.03] border border-white/5 space-y-4"
                >
                  <div class="flex items-center justify-between">
                    <div class="flex items-center gap-2">
                      <Database size={18} class="text-emerald-400" />
                      <span class="font-bold text-white">Data Retention</span>
                    </div>
                  </div>
                  <div class="flex items-center justify-between">
                    <div class="space-y-1">
                      <label
                        for="retention-days"
                        class="text-sm font-medium text-slate-300"
                        >Retention Period (Days)</label
                      >
                      <p class="text-[10px] text-slate-500">
                        Errors older than this will be permanently deleted.
                      </p>
                    </div>
                    <div class="flex items-center gap-3">
                      <input
                        id="retention-days"
                        type="number"
                        class="pulse-input w-24 text-center"
                        bind:value={globalSettings.retentionDays}
                        min="1"
                        max="365"
                      />
                      <span class="text-xs text-slate-500 font-medium"
                        >days</span
                      >
                    </div>
                  </div>
                </div>

                <!-- System Maintenance -->
                <div
                  class="p-6 rounded-2xl bg-white/[0.03] border border-white/5 space-y-4"
                >
                  <div class="flex items-center justify-between">
                    <div class="flex items-center gap-2">
                      <Terminal size={18} class="text-pulse-400" />
                      <span class="font-bold text-white"
                        >System Maintenance</span
                      >
                    </div>
                  </div>
                  <div class="flex items-center justify-between">
                    <div class="space-y-1">
                      <p class="text-sm font-medium text-white">
                        Manual Database Purge
                      </p>
                      <p class="text-[10px] text-slate-500">
                        Immediately run the cleanup process based on your
                        retention policy.
                      </p>
                    </div>
                    <button
                      class="pulse-button bg-red-500/10 text-red-500 hover:bg-red-500/20 px-4 py-2 text-xs font-bold transition-all"
                      on:click={handleManualCleanup}
                      disabled={maintenanceLoading}
                    >
                      {maintenanceLoading ? "Cleaning..." : "Run Cleanup"}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          {/if}
        </div>

        {#if activeSection !== "support" && activeSection !== "security"}
          <div class="bg-white/5 p-6 flex justify-end border-t border-white/10">
            <button
              on:click={handleSave}
              class="pulse-button-primary flex items-center gap-2"
            >
              <Save size={18} />
              <span>Save Changes</span>
            </button>
          </div>
        {/if}
      </div>
    </div>
  </div>
</div>

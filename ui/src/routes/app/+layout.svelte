<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { logout } from '$lib/api';

  let { children, data } = $props();

  const nav = [
    { href: '/app/visits',     label: 'Visits',      icon: IconCalendar },
    { href: '/app/labs',       label: 'Labs',         icon: IconChart },
    { href: '/app/agent',      label: 'AI',           icon: IconAgent },
    { href: '/app/medications',label: 'Meds',         icon: IconPill },
    { href: '/app/profile',    label: 'Profile',      icon: IconPerson },
  ];

  function active(href: string) {
    return $page.url.pathname.startsWith(href);
  }

  async function doLogout() {
    await logout();
    goto('/');
  }
</script>

<div class="shell">
  <!-- Desktop Sidebar -->
  <aside class="sidebar">
    <div class="brand">
      <div class="brand-icon">
        <svg viewBox="0 0 32 32" fill="none"><rect width="32" height="32" rx="8" fill="#007AFF"/><path d="M16 8C11.6 8 8 11.6 8 16s3.6 8 8 8 8-3.6 8-8-3.6-8-8-8zm0 3a1 1 0 1 1 0 2 1 1 0 0 1 0-2zm-.5 4h1v6h-1v-6z" fill="white"/></svg>
      </div>
      <span class="brand-name">MedHelp</span>
    </div>

    <nav class="sidebar-nav">
      {#each nav as n}
        <a href={n.href} class="nav-item" class:active={active(n.href)}>
          <n.icon />
          <span>{n.label}</span>
        </a>
      {/each}
    </nav>

    <div class="sidebar-footer">
      {#if data?.me}
        <div class="user-row">
          <div class="avatar">{(data.me.email || 'U')[0].toUpperCase()}</div>
          <div class="user-info">
            <div class="user-name">{data.me.display_name ?? data.me.email}</div>
          </div>
        </div>
      {/if}
      <button class="logout-btn" onclick={doLogout}>Sign out</button>
    </div>
  </aside>

  <!-- Main content -->
  <main class="content">
    {@render children()}
  </main>

  <!-- Mobile Tab Bar -->
  <nav class="tabbar">
    {#each nav as n}
      <a href={n.href} class="tab" class:active={active(n.href)}>
        <n.icon />
        <span>{n.label}</span>
      </a>
    {/each}
  </nav>
</div>

<!-- Icon components -->
{#snippet IconCalendar()}
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
    <rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/>
  </svg>
{/snippet}

{#snippet IconChart()}
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
    <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
  </svg>
{/snippet}

{#snippet IconAgent()}
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
    <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
  </svg>
{/snippet}

{#snippet IconPill()}
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
    <path d="M10.5 20H4a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v7.5"/>
    <circle cx="17" cy="17" r="5"/><line x1="14" y1="17" x2="20" y2="17"/>
  </svg>
{/snippet}

{#snippet IconPerson()}
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
    <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>
  </svg>
{/snippet}

<style>
  :global(*, *::before, *::after) { box-sizing: border-box; margin: 0; padding: 0; }
  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Display', 'Segoe UI', Helvetica, Arial, sans-serif;
    -webkit-font-smoothing: antialiased;
    background: #F2F2F7;
    color: #1C1C1E;
  }
  :global(h1, h2, h3) { letter-spacing: -0.3px; }

  /* CSS vars */
  :global(:root) {
    --blue: #007AFF;
    --red: #FF3B30;
    --green: #34C759;
    --orange: #FF9500;
    --yellow: #FFCC00;
    --bg: #F2F2F7;
    --bg2: #FFFFFF;
    --separator: #E5E5EA;
    --text: #1C1C1E;
    --text2: #636366;
    --text3: #AEAEB2;
    --sidebar-w: 240px;
    --tabbar-h: 60px;
    --radius: 14px;
  }

  .shell {
    display: flex;
    min-height: 100svh;
  }

  /* ── Sidebar (desktop) ── */
  .sidebar {
    width: var(--sidebar-w);
    background: rgba(255,255,255,0.8);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    border-right: 1px solid var(--separator);
    display: flex;
    flex-direction: column;
    padding: 20px 12px;
    position: fixed;
    top: 0;
    left: 0;
    height: 100svh;
    z-index: 100;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 4px 8px 20px;
  }

  .brand-icon svg { width: 32px; height: 32px; }

  .brand-name {
    font-size: 20px;
    font-weight: 700;
    letter-spacing: -0.5px;
  }

  .sidebar-nav {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 12px;
    border-radius: 10px;
    text-decoration: none;
    color: var(--text2);
    font-size: 15px;
    font-weight: 500;
    transition: background 0.15s, color 0.15s;
  }

  .nav-item svg { width: 20px; height: 20px; flex-shrink: 0; }
  .nav-item:hover { background: var(--bg); color: var(--text); }
  .nav-item.active { background: rgba(0,122,255,0.1); color: var(--blue); }

  .sidebar-footer {
    border-top: 1px solid var(--separator);
    padding-top: 16px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .user-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 4px 8px;
  }

  .avatar {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    background: var(--blue);
    color: white;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 13px;
    font-weight: 600;
    flex-shrink: 0;
  }

  .user-info { flex: 1; min-width: 0; }

  .user-name {
    font-size: 13px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--text);
  }

  .logout-btn {
    all: unset;
    cursor: pointer;
    display: block;
    width: 100%;
    padding: 9px 12px;
    border-radius: 10px;
    font-size: 14px;
    color: var(--red);
    font-weight: 500;
    transition: background 0.15s;
  }

  .logout-btn:hover { background: rgba(255,59,48,0.08); }

  /* ── Content ── */
  .content {
    flex: 1;
    margin-left: var(--sidebar-w);
    min-height: 100svh;
    padding-bottom: 0;
  }

  /* ── Tab bar (mobile) ── */
  .tabbar {
    display: none;
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    height: calc(var(--tabbar-h) + env(safe-area-inset-bottom));
    padding-bottom: env(safe-area-inset-bottom);
    background: rgba(255,255,255,0.92);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    border-top: 1px solid var(--separator);
    z-index: 100;
  }

  .tab {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 3px;
    text-decoration: none;
    color: var(--text3);
    font-size: 10px;
    font-weight: 500;
    transition: color 0.15s;
  }

  .tab svg { width: 22px; height: 22px; }
  .tab.active { color: var(--blue); }

  @media (max-width: 768px) {
    .sidebar { display: none; }
    .content { margin-left: 0; padding-bottom: calc(var(--tabbar-h) + env(safe-area-inset-bottom)); }
    .tabbar { display: flex; }
  }
</style>

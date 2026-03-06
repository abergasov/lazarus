<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { logout } from '$lib/api';

  let { children, data } = $props();

  const nav = [
    { href: '/app',         label: 'Home',    exact: true },
    { href: '/app/records', label: 'Records', exact: false },
    { href: '/app/profile', label: 'Profile', exact: false },
  ];

  function active(n: typeof nav[0]) {
    if (n.exact) return $page.url.pathname === n.href;
    return $page.url.pathname.startsWith(n.href);
  }

  // Hide chrome on full-screen routes (onboarding, visit detail)
  const hideChrome = $derived(
    $page.url.pathname.startsWith('/app/onboarding') ||
    $page.url.pathname.match(/\/app\/visits\/[^/]+$/)
  );

  async function doLogout() {
    await logout();
    goto('/');
  }
</script>

{#if hideChrome}
  <div class="fullscreen">
    {@render children()}
  </div>
{:else}
  <div class="shell">
    <!-- Desktop Sidebar -->
    <aside class="sidebar">
      <a href="/app" class="brand">
        <div class="brand-icon">
          <svg viewBox="0 0 32 32" fill="none" width="32" height="32"><rect width="32" height="32" rx="8" fill="#007AFF"/><path d="M16 8C11.6 8 8 11.6 8 16s3.6 8 8 8 8-3.6 8-8-3.6-8-8-8zm0 3a1 1 0 1 1 0 2 1 1 0 0 1 0-2zm-.5 4h1v6h-1v-6z" fill="white"/></svg>
        </div>
        <span class="brand-name">MedHelp</span>
      </a>

      <nav class="sidebar-nav">
        {#each nav as n}
          <a href={n.href} class="nav-item" class:active={active(n)}>
            {#if n.label === 'Home'}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>
            {:else if n.label === 'Records'}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>
            {:else}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
            {/if}
            <span>{n.label}</span>
          </a>
        {/each}
      </nav>

      <div class="sidebar-footer">
        {#if data?.me}
          <div class="user-row">
            <div class="avatar">{(data.me.display_name || data.me.email || 'U')[0].toUpperCase()}</div>
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
        <a href={n.href} class="tab" class:active={active(n)}>
          {#if n.label === 'Home'}
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>
          {:else if n.label === 'Records'}
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
          {:else}
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
          {/if}
          <span>{n.label}</span>
        </a>
      {/each}
    </nav>
  </div>
{/if}

<style>
  :global(*, *::before, *::after) { box-sizing: border-box; margin: 0; padding: 0; }
  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Display', 'Segoe UI', Helvetica, Arial, sans-serif;
    -webkit-font-smoothing: antialiased; background: #F2F2F7; color: #1C1C1E;
  }
  :global(h1, h2, h3) { letter-spacing: -0.3px; }
  :global(:root) {
    --blue: #007AFF; --red: #FF3B30; --green: #34C759; --orange: #FF9500; --yellow: #FFCC00;
    --bg: #F2F2F7; --bg2: #FFFFFF; --separator: #E5E5EA;
    --text: #1C1C1E; --text2: #636366; --text3: #AEAEB2;
    --sidebar-w: 220px; --tabbar-h: 56px; --radius: 14px;
  }

  .fullscreen { min-height: 100svh; }

  .shell { display: flex; min-height: 100svh; }

  .sidebar {
    width: var(--sidebar-w); background: rgba(255,255,255,0.85);
    backdrop-filter: blur(20px); -webkit-backdrop-filter: blur(20px);
    border-right: 1px solid var(--separator); display: flex; flex-direction: column;
    padding: 20px 12px; position: fixed; top: 0; left: 0; height: 100svh; z-index: 100;
  }

  .brand { display: flex; align-items: center; gap: 10px; padding: 4px 8px 24px; text-decoration: none; color: var(--text); }
  .brand-name { font-size: 20px; font-weight: 700; letter-spacing: -0.5px; }

  .sidebar-nav { flex: 1; display: flex; flex-direction: column; gap: 2px; }

  .nav-item {
    display: flex; align-items: center; gap: 10px; padding: 10px 12px; border-radius: 10px;
    text-decoration: none; color: var(--text2); font-size: 15px; font-weight: 500;
    transition: background 0.15s, color 0.15s;
  }
  .nav-item:hover { background: var(--bg); color: var(--text); }
  .nav-item.active { background: rgba(0,122,255,0.1); color: var(--blue); }

  .sidebar-footer { border-top: 1px solid var(--separator); padding-top: 16px; display: flex; flex-direction: column; gap: 8px; }
  .user-row { display: flex; align-items: center; gap: 10px; padding: 4px 8px; }
  .avatar {
    width: 34px; height: 34px; border-radius: 50%; background: var(--blue); color: white;
    display: flex; align-items: center; justify-content: center; font-size: 14px; font-weight: 600;
  }
  .user-info { flex: 1; min-width: 0; }
  .user-name { font-size: 13px; font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .logout-btn {
    all: unset; cursor: pointer; display: block; width: 100%; padding: 9px 12px; border-radius: 10px;
    font-size: 14px; color: var(--red); font-weight: 500; transition: background 0.15s;
  }
  .logout-btn:hover { background: rgba(255,59,48,0.08); }

  .content { flex: 1; margin-left: var(--sidebar-w); min-height: 100svh; }

  .tabbar {
    display: none; position: fixed; bottom: 0; left: 0; right: 0;
    height: calc(var(--tabbar-h) + env(safe-area-inset-bottom));
    padding-bottom: env(safe-area-inset-bottom);
    background: rgba(255,255,255,0.92); backdrop-filter: blur(20px); -webkit-backdrop-filter: blur(20px);
    border-top: 1px solid var(--separator); z-index: 100;
  }
  .tab {
    flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 3px;
    text-decoration: none; color: var(--text3); font-size: 10px; font-weight: 500; transition: color 0.15s;
  }
  .tab.active { color: var(--blue); }

  @media (max-width: 768px) {
    .sidebar { display: none; }
    .content { margin-left: 0; padding-bottom: calc(var(--tabbar-h) + env(safe-area-inset-bottom)); }
    .tabbar { display: flex; }
  }
</style>

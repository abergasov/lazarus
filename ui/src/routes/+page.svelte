<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';

  let error = $state('');
  let loading = $state(true);

  onMount(async () => {
    const url = new URL(window.location.href);
    const code = url.searchParams.get('code');

    if (code) {
      const r = await fetch('/api/v1/auth/exchange', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code }),
      });
      if (r.ok) {
        window.history.replaceState({}, '', '/');
        await goto('/app');
        return;
      }
      error = 'Sign-in failed. Please try again.';
    }

    // Check if already logged in
    const me = await fetch('/api/v1/user/me', { credentials: 'include' });
    if (me.ok) {
      await goto('/app');
      return;
    }

    loading = false;
  });
</script>

<div class="splash">
  <div class="hero">
    <div class="logo">
      <svg viewBox="0 0 64 64" fill="none" xmlns="http://www.w3.org/2000/svg">
        <rect width="64" height="64" rx="14" fill="#0D9488"/>
        <path d="M32 16C23.163 16 16 23.163 16 32s7.163 16 16 16 16-7.163 16-16S40.837 16 32 16zm0 6a2 2 0 1 1 0 4 2 2 0 0 1 0-4zm-1 8h2v12h-2V30z" fill="white"/>
      </svg>
    </div>
    <h1>MedHelp</h1>
    <p class="tagline">Your intelligent medical advocate</p>

    {#if loading}
      <div class="spinner"></div>
    {:else}
      {#if error}
        <div class="error">{error}</div>
      {/if}
      <a class="btn-primary" href="/api/auth/google/login">
        <svg viewBox="0 0 24 24" width="20" height="20" xmlns="http://www.w3.org/2000/svg">
          <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/>
          <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
          <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
          <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
        </svg>
        Continue with Google
      </a>

      <div class="features">
        <div class="feature">
          <span class="icon">🩺</span>
          <span>Prep smarter before visits</span>
        </div>
        <div class="feature">
          <span class="icon">📊</span>
          <span>Track labs &amp; trends</span>
        </div>
        <div class="feature">
          <span class="icon">🤖</span>
          <span>AI-powered guidance</span>
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  :global(*, *::before, *::after) { box-sizing: border-box; margin: 0; padding: 0; }
  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Display', 'Segoe UI', Helvetica, Arial, sans-serif;
    -webkit-font-smoothing: antialiased;
    background: linear-gradient(135deg, #0D9488 0%, #115E59 100%);
    min-height: 100svh;
  }

  .splash {
    min-height: 100svh;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
  }

  .hero {
    background: rgba(255,255,255,0.95);
    backdrop-filter: blur(20px);
    border-radius: 28px;
    padding: 48px 40px;
    max-width: 400px;
    width: 100%;
    text-align: center;
    box-shadow: 0 32px 80px rgba(0,0,0,0.2);
  }

  .logo svg {
    width: 72px;
    height: 72px;
    margin-bottom: 16px;
    filter: drop-shadow(0 8px 16px rgba(13,148,136,0.4));
  }

  h1 {
    font-size: 32px;
    font-weight: 700;
    letter-spacing: -0.5px;
    color: #1C1C1E;
    margin-bottom: 8px;
  }

  .tagline {
    font-size: 16px;
    color: #636366;
    margin-bottom: 36px;
  }

  .btn-primary {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    width: 100%;
    padding: 14px 20px;
    background: #1C1C1E;
    color: white;
    border-radius: 14px;
    text-decoration: none;
    font-size: 16px;
    font-weight: 600;
    transition: opacity 0.15s, transform 0.15s;
    margin-bottom: 32px;
  }

  .btn-primary:hover { opacity: 0.85; transform: translateY(-1px); }
  .btn-primary:active { opacity: 1; transform: translateY(0); }

  .error {
    background: #FFF2F0;
    border: 1px solid #FFCCC7;
    color: #DC2626;
    padding: 12px 16px;
    border-radius: 10px;
    font-size: 14px;
    margin-bottom: 16px;
  }

  .features {
    display: flex;
    flex-direction: column;
    gap: 12px;
    text-align: left;
  }

  .feature {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 15px;
    color: #3A3A3C;
  }

  .icon {
    font-size: 20px;
    width: 36px;
    height: 36px;
    background: #F2F2F7;
    border-radius: 10px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .spinner {
    width: 28px;
    height: 28px;
    border: 3px solid #E5E5EA;
    border-top-color: #0D9488;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    margin: 0 auto 24px;
  }

  @keyframes spin { to { transform: rotate(360deg); } }
</style>

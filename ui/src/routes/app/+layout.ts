export const ssr = false;

export async function load({ fetch, url }) {
  const r = await fetch('/api/v1/user/me', { credentials: 'include' });
  if (!r.ok) {
    if (typeof window !== 'undefined') window.location.href = '/';
    return { me: null };
  }
  const me = await r.json();

  // Check onboarding — redirect if not completed (unless already on onboarding)
  if (!url.pathname.startsWith('/app/onboarding')) {
    try {
      const homeR = await fetch('/api/v1/home', { credentials: 'include' });
      if (homeR.ok) {
        const homeData = await homeR.json();
        if (!homeData.onboarding_completed && typeof window !== 'undefined') {
          window.location.href = '/app/onboarding';
          return { me };
        }
      }
    } catch {}
  }

  return { me };
}

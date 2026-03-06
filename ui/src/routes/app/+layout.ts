export const ssr = false;

export async function load({ fetch }) {
  const r = await fetch('/api/v1/user/me', { credentials: 'include' });
  if (!r.ok) return { me: null };
  return { me: await r.json() };
}

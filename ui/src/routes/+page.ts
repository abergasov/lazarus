import { redirect } from "@sveltejs/kit";

export const ssr = false;

export async function load({ url, fetch }) {
    const code = url.searchParams.get("code");
    if (!code) return {};

    const r = await fetch("/api/v1/auth/exchange", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ code }),
    });

    if (!r.ok) throw redirect(302, "/?exchange_error=1");

    throw redirect(302, "/app");
}
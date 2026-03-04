import { PUBLIC_API_BASE } from "$env/static/public";

const API = PUBLIC_API_BASE;

export async function apiGet<T>(path: string): Promise<T> {
    const r = await fetch(API + path, { credentials: "include" });
    if (!r.ok) throw new Error(await r.text());
    return (await r.json()) as T;
}

export async function apiPost<T>(path: string, body?: any): Promise<T> {
    const r = await fetch(API + path, {
        method: "POST",
        credentials: "include",
        headers: body ? { "Content-Type": "application/json" } : undefined,
        body: body ? JSON.stringify(body) : undefined,
    });
    if (!r.ok) throw new Error(await r.text());
    return (await r.json()) as T;
}

export function loginURL(provider: "github" | "google" | "apple", next = "/app"): string {
    const u = new URL(API + `/auth/login/${provider}`);
    u.searchParams.set("next", next);
    return u.toString();
}


export function googleLoginURL(next = "/app"): string {
    const u = new URL("/api/auth/google/login", "http://local");
    // if you later support next param on backend:
    u.searchParams.set("next", next);
    return u.pathname + "?" + u.searchParams.toString();
}

export async function exchangeCode(code: string): Promise<string> {
    const r = await fetch("/api/v1/auth/exchange", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ code }),
    });

    if (!r.ok) throw new Error(await r.text());

    const j = await r.json();
    return j.code;
}
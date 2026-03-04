<script lang="ts">
    import { onMount } from "svelte";
    import { goto } from "$app/navigation";

    onMount(async () => {
        const url = new URL(window.location.href);
        const code = url.searchParams.get("code");
        if (!code) return;

        const r = await fetch("/api/v1/auth/exchange", {
            method: "POST",
            credentials: "include",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ code }),
        });

        if (r.ok) {
            // remove code from URL
            window.history.replaceState({}, "", "/");
            await goto("/app");
            return;
        }

        window.history.replaceState({}, "", "/?exchange_error=1");
    });
</script>

<h1>Lazarus</h1>
<a class="btn" href="/api/auth/google/login">Login with Google</a>

<style>
    .btn{ display:inline-block; padding:10px 14px; border:1px solid #444; border-radius:10px; text-decoration:none; color:inherit; margin-top:16px; }
</style>
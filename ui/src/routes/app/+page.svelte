<script lang="ts">
    import { onMount } from "svelte";
    import { goto } from "$app/navigation";

    let email = "";

    onMount(async () => {
        const r = await fetch("/api/v1/user/me", { credentials: "include" });
        if (!r.ok) {
            await goto("/");
            return;
        }
        const me = await r.json();
        email = me.email ?? "";
    });
</script>

<h1>Account</h1>
<p>{email}</p>

<form method="POST" action="/api/v1/auth/logout">
    <button>Logout</button>
</form>
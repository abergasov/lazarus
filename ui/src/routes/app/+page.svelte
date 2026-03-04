<script lang="ts">
    export let data: { me: { id: string; email?: string; display_name?: string } };
</script>

<h1>Dashboard</h1>
<p>user: {data.me.id}</p>
{#if data.me.email}<p>email: {data.me.email}</p>{/if}
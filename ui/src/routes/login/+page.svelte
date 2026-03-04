<script lang="ts">
  import { loginURL } from "$lib/api";
</script>

<h1>Login</h1>

<div style="display:flex; gap:12px; flex-wrap:wrap">
  <a href={loginURL("github")} class="btn">Continue with GitHub</a>
  <a href={loginURL("google")} class="btn">Continue with Google (be later)</a>
</div>

<style>
  .btn{
    display:inline-block;
    padding:10px 14px;
    border:1px solid #444;
    border-radius:10px;
    text-decoration:none;
    color:inherit;
  }
</style>
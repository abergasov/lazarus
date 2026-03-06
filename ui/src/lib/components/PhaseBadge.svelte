<script lang="ts">
  let { phase }: { phase: string } = $props();
  const color = $derived(
    phase === 'preparing' ? 'var(--blue, #007AFF)' :
    phase === 'during' ? 'var(--green, #34C759)' :
    phase === 'completed' ? 'var(--text3, #AEAEB2)' : 'var(--text3, #AEAEB2)'
  );
  const label = $derived(
    phase === 'preparing' ? 'Preparing' :
    phase === 'during' ? 'In Progress' :
    phase === 'completed' ? 'Completed' : phase
  );
</script>

<span class="badge" style="background: {color}15; color: {color}">{label}</span>

<style>
  .badge {
    display: inline-block;
    padding: 3px 10px;
    border-radius: 12px;
    font-size: 11px;
    font-weight: 600;
    text-transform: capitalize;
    letter-spacing: 0.2px;
  }
</style>

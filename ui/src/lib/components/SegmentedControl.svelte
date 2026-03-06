<script lang="ts">
  let { segments, selected = 0, onchange }: {
    segments: string[];
    selected?: number;
    onchange: (index: number) => void;
  } = $props();
</script>

<div class="segmented">
  {#each segments as seg, i}
    <button class="seg" class:active={selected === i} onclick={() => onchange(i)}>
      {seg}
    </button>
  {/each}
</div>

<style>
  .segmented {
    display: flex;
    background: rgba(118, 118, 128, 0.12);
    border-radius: 9px;
    padding: 2px;
  }
  .seg {
    all: unset;
    cursor: pointer;
    flex: 1;
    text-align: center;
    padding: 7px 12px;
    border-radius: 7px;
    font-size: 13px;
    font-weight: 500;
    color: var(--text2, #636366);
    transition: all 0.2s;
  }
  .seg.active {
    background: white;
    color: var(--text, #1C1C1E);
    font-weight: 600;
    box-shadow: 0 1px 3px rgba(0,0,0,0.08), 0 1px 2px rgba(0,0,0,0.06);
  }
</style>

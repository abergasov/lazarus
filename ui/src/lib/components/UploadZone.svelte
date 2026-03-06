<script lang="ts">
  let { onupload, label = 'Drop a file here or tap to upload' }: {
    onupload: (file: File) => Promise<void>;
    label?: string;
  } = $props();

  let dragging = $state(false);
  let uploading = $state(false);
  let fileInput: HTMLInputElement;

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    dragging = false;
    const file = e.dataTransfer?.files[0];
    if (file) doUpload(file);
  }

  function handleInput(e: Event) {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (file) doUpload(file);
  }

  async function doUpload(file: File) {
    uploading = true;
    try { await onupload(file); }
    finally { uploading = false; }
  }
</script>

<div
  class="zone"
  class:dragging
  class:uploading
  role="button"
  tabindex="0"
  ondragover={(e) => { e.preventDefault(); dragging = true; }}
  ondragleave={() => { dragging = false; }}
  ondrop={handleDrop}
  onclick={() => fileInput?.click()}
  onkeydown={(e) => { if (e.key === 'Enter') fileInput?.click(); }}
>
  {#if uploading}
    <div class="spinner"></div>
    <span class="zone-label">Processing...</span>
  {:else}
    <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" style="opacity:0.4">
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/>
    </svg>
    <span class="zone-label">{label}</span>
  {/if}
</div>
<input bind:this={fileInput} type="file" accept="image/*,.pdf,.jpg,.jpeg,.png" style="display:none" onchange={handleInput} />

<style>
  .zone {
    border: 2px dashed var(--separator, #E5E5EA);
    border-radius: var(--radius, 14px);
    padding: 32px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    cursor: pointer;
    transition: border-color 0.2s, background 0.2s;
    background: transparent;
  }
  .zone:hover, .zone.dragging {
    border-color: var(--blue, #007AFF);
    background: rgba(0, 122, 255, 0.04);
  }
  .zone.uploading {
    border-color: var(--blue, #007AFF);
    pointer-events: none;
  }
  .zone-label {
    font-size: 14px;
    color: var(--text3, #AEAEB2);
    text-align: center;
  }
  .spinner {
    width: 24px;
    height: 24px;
    border: 3px solid var(--separator, #E5E5EA);
    border-top-color: var(--blue, #007AFF);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>

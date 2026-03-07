<script lang="ts">
  import { goto } from '$app/navigation';
  import { onboarding, profile } from '$lib/api';
  import type { PatientModel, Demographics } from '$lib/types';
  import UploadZone from '$lib/components/UploadZone.svelte';

  let step = $state<'welcome' | 'processing' | 'confirm' | 'done'>('welcome');
  let processingSteps = $state<{ step: string; status: string; label: string }[]>([]);
  let extractedModel = $state<PatientModel | null>(null);
  let editDemo = $state<Demographics>({});

  function numVal(field: keyof Demographics): string {
    const v = editDemo[field];
    return v != null && v !== 0 ? String(v) : '';
  }

  function setNum(field: keyof Demographics, e: Event) {
    const v = (e.target as HTMLInputElement).value;
    (editDemo as any)[field] = v === '' ? undefined : Number(v);
  }

  async function handleUpload(files: File[]) {
    step = 'processing';
    processingSteps = [{ step: 'upload', status: 'running', label: `Uploading ${files.length} document(s)...` }];

    try {
      const res = await onboarding.upload(files);
      if (!res.ok || !res.body) {
        processingSteps = [{ step: 'error', status: 'error', label: 'Upload failed' }];
        return;
      }

      const reader = res.body.getReader();
      const dec = new TextDecoder();
      let buf = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buf += dec.decode(value, { stream: true });
        const parts = buf.split('\n\n');
        buf = parts.pop() ?? '';

        for (const part of parts) {
          if (!part.includes('data:')) continue;
          const dataLine = part.split('\n').find(l => l.startsWith('data:'));
          if (!dataLine) continue;
          const raw = dataLine.slice(5).trim();
          if (raw === '[DONE]') continue;
          try {
            const ev = JSON.parse(raw);
            if (ev.type === 'processing_step') {
              const p = ev.payload;
              const existing = processingSteps.find(s => s.step === p.step);
              if (existing) {
                existing.status = p.status;
                existing.label = p.label;
                processingSteps = [...processingSteps];
              } else {
                processingSteps = [...processingSteps, p];
              }
            } else if (ev.type === 'profile_extracted') {
              extractedModel = typeof ev.payload === 'string' ? JSON.parse(ev.payload) : ev.payload;
              editDemo = extractedModel?.demographics ?? {};
            }
          } catch {}
        }
      }

      step = 'confirm';
    } catch (e) {
      processingSteps = [...processingSteps, { step: 'error', status: 'error', label: String(e) }];
    }
  }

  async function confirmProfile() {
    await onboarding.confirm(editDemo);
    step = 'done';
    setTimeout(() => goto('/app'), 1000);
  }
</script>

<div class="onboarding">
  {#if step === 'welcome'}
    <div class="welcome">
      <div class="logo">
        <svg viewBox="0 0 64 64" fill="none" width="64" height="64"><rect width="64" height="64" rx="16" fill="#0D9488"/><path d="M32 16C22.1 16 14 24.1 14 34s8.1 18 18 18 18-8.1 18-18-8.1-18-18-18zm0 7a2.5 2.5 0 1 1 0 5 2.5 2.5 0 0 1 0-5zm-1.5 9h3v13h-3V32z" fill="white"/></svg>
      </div>
      <h1>Welcome to MedHelp</h1>
      <p class="desc">Upload your medical documents — lab results, discharge summaries, prescriptions — and we'll build your health profile automatically.</p>
      <div class="upload-area">
        <UploadZone onupload={handleUpload} label="Drop your medical documents here (select multiple)" />
      </div>
      <p class="hint">We support PDFs, photos of lab results, and scanned documents. Select as many as you need.</p>
    </div>

  {:else if step === 'processing'}
    <div class="processing">
      <h1>Building your profile...</h1>
      <div class="steps">
        {#each processingSteps as s}
          <div class="proc-step" class:done={s.status === 'done'} class:running={s.status === 'running'} class:error={s.status === 'error'}>
            {#if s.status === 'running'}
              <div class="proc-spinner"></div>
            {:else if s.status === 'done'}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--green)" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
            {:else}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--red)" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>
            {/if}
            <span>{s.label}</span>
          </div>
        {/each}
      </div>
    </div>

  {:else if step === 'confirm'}
    <div class="confirm">
      <h1>Confirm your profile</h1>
      <p class="desc">We extracted the following from your documents. Edit anything that doesn't look right.</p>

      <div class="form-section">
        <h2>Demographics</h2>
        <div class="form-grid">
          <label>
            <span>Sex</span>
            <select bind:value={editDemo.sex}>
              <option value="">—</option>
              <option value="M">Male</option>
              <option value="F">Female</option>
            </select>
          </label>
          <label>
            <span>Age</span>
            <input type="text" inputmode="numeric" value={numVal('age')} oninput={(e) => setNum('age', e)} placeholder="—" />
          </label>
          <label>
            <span>Ethnicity</span>
            <select bind:value={editDemo.ethnicity}>
              <option value="">—</option>
              <option value="white">White</option>
              <option value="african_american">African American</option>
              <option value="hispanic">Hispanic</option>
              <option value="asian">Asian</option>
              <option value="other">Other</option>
            </select>
          </label>
          <label>
            <span>Height (cm)</span>
            <input type="text" inputmode="numeric" value={numVal('height_cm')} oninput={(e) => setNum('height_cm', e)} placeholder="—" />
          </label>
          <label>
            <span>Weight (kg)</span>
            <input type="text" inputmode="numeric" value={numVal('weight_kg')} oninput={(e) => setNum('weight_kg', e)} placeholder="—" />
          </label>
        </div>

        <div class="bp-field">
          <span class="field-label">Blood Pressure</span>
          <div class="bp-row">
            <input type="text" inputmode="numeric" value={numVal('blood_pressure_systolic')} oninput={(e) => setNum('blood_pressure_systolic', e)} placeholder="Systolic" />
            <span class="bp-slash">/</span>
            <input type="text" inputmode="numeric" value={numVal('blood_pressure_diastolic')} oninput={(e) => setNum('blood_pressure_diastolic', e)} placeholder="Diastolic" />
          </div>
        </div>

        <label class="toggle-row">
          <span>Smoker</span>
          <input type="checkbox" bind:checked={editDemo.smoker} class="toggle" />
        </label>
        <label class="toggle-row">
          <span>Diabetes</span>
          <input type="checkbox" bind:checked={editDemo.diabetes} class="toggle" />
        </label>
      </div>

      {#if extractedModel?.active_conditions && extractedModel.active_conditions.length > 0}
        <div class="form-section">
          <h2>Conditions Found</h2>
          {#each extractedModel.active_conditions as cond}
            <div class="cond-row">
              <span class="cond-name">{cond.name}</span>
              <span class="cond-status">{cond.status}</span>
            </div>
          {/each}
        </div>
      {/if}

      <button class="confirm-btn" onclick={confirmProfile}>Looks right — let's go</button>
    </div>

  {:else}
    <div class="done-state">
      <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="var(--green)" stroke-width="1.5"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
      <h1>You're all set!</h1>
      <p>Redirecting to your home screen...</p>
    </div>
  {/if}
</div>

<style>
  .onboarding {
    min-height: 100svh; display: flex; flex-direction: column; align-items: center;
    justify-content: center; padding: 40px 20px; max-width: 600px; margin: 0 auto;
    box-sizing: border-box;
  }
  .welcome, .processing, .confirm, .done-state { width: 100%; text-align: center; }
  .logo { margin-bottom: 24px; }
  h1 { font-size: 28px; font-weight: 700; margin-bottom: 12px; }
  .desc { font-size: 16px; color: var(--text2); line-height: 1.5; margin-bottom: 32px; }
  .upload-area { margin-bottom: 16px; }
  .hint { font-size: 12px; color: var(--text3); }

  @media (max-width: 480px) {
    .onboarding { justify-content: flex-start; padding: 24px 16px; }
    h1 { font-size: 24px; }
    .desc { font-size: 14px; margin-bottom: 24px; }
  }

  .steps { display: flex; flex-direction: column; gap: 16px; margin-top: 32px; text-align: left; }
  .proc-step { display: flex; align-items: center; gap: 12px; font-size: 15px; color: var(--text2); }
  .proc-step.done { color: var(--text); }
  .proc-step.running { color: var(--blue); }
  .proc-spinner {
    width: 20px; height: 20px; border: 2px solid var(--separator); border-top-color: var(--blue);
    border-radius: 50%; animation: spin 0.8s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  .confirm { text-align: left; }
  .confirm h1 { text-align: center; }
  .confirm .desc { text-align: center; }
  .form-section { background: var(--bg2); border-radius: var(--radius); padding: 20px; margin-bottom: 16px; }
  .form-section h2 { font-size: 13px; font-weight: 600; margin-bottom: 16px; color: var(--text3); text-transform: uppercase; letter-spacing: 0.5px; }

  .form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 16px; }
  .form-grid label { display: flex; flex-direction: column; gap: 6px; }
  .form-grid label span { font-size: 13px; color: var(--text2); font-weight: 500; }
  .form-grid input, .form-grid select {
    padding: 12px 14px; border-radius: 10px; border: 1px solid var(--separator);
    font-size: 16px; background: var(--bg); outline: none; color: var(--text);
    transition: border-color 0.2s;
    -webkit-appearance: none; appearance: none;
    box-sizing: border-box; width: 100%; min-width: 0;
  }
  .form-grid select {
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%23999' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E");
    background-repeat: no-repeat; background-position: right 12px center;
    padding-right: 32px;
  }
  .form-grid input:focus, .form-grid select:focus { border-color: var(--blue); }
  .form-grid input::placeholder { color: var(--text3); }

  @media (max-width: 480px) {
    .form-grid { grid-template-columns: 1fr; gap: 12px; }
    .form-section { padding: 16px; }
  }

  .bp-field { margin-bottom: 16px; }
  .field-label { font-size: 13px; color: var(--text2); font-weight: 500; display: block; margin-bottom: 6px; }
  .bp-row { display: flex; align-items: center; gap: 8px; }
  .bp-row input {
    flex: 1; padding: 12px 14px; border-radius: 10px; border: 1px solid var(--separator);
    font-size: 16px; background: var(--bg); outline: none; color: var(--text);
    text-align: center; transition: border-color 0.2s;
    box-sizing: border-box; min-width: 0;
  }
  .bp-row input:focus { border-color: var(--blue); }
  .bp-row input::placeholder { color: var(--text3); }
  .bp-slash { font-size: 20px; color: var(--text3); font-weight: 300; }

  .toggle-row { display: flex; align-items: center; justify-content: space-between; padding: 10px 0; }
  .toggle-row span { font-size: 15px; color: var(--text); font-weight: 500; }
  .toggle { width: 48px; height: 28px; appearance: none; background: var(--separator); border-radius: 14px; position: relative; cursor: pointer; transition: background 0.2s; flex-shrink: 0; }
  .toggle:checked { background: var(--green); }
  .toggle::after { content: ''; position: absolute; width: 24px; height: 24px; background: white; border-radius: 50%; top: 2px; left: 2px; transition: transform 0.2s; box-shadow: 0 1px 3px rgba(0,0,0,0.15); }
  .toggle:checked::after { transform: translateX(20px); }

  .cond-row { display: flex; justify-content: space-between; padding: 10px 0; border-bottom: 1px solid var(--separator); }
  .cond-row:last-child { border-bottom: none; }
  .cond-name { font-size: 15px; font-weight: 500; }
  .cond-status { font-size: 13px; color: var(--text3); }

  .confirm-btn {
    all: unset; cursor: pointer; display: block; width: 100%; padding: 16px; border-radius: 14px;
    background: var(--blue); color: white; font-size: 17px; font-weight: 600; text-align: center;
    margin-top: 24px; transition: opacity 0.15s; box-sizing: border-box;
    -webkit-tap-highlight-color: transparent;
  }
  .confirm-btn:hover { opacity: 0.9; }
  .confirm-btn:active { opacity: 0.8; }

  .done-state { display: flex; flex-direction: column; align-items: center; gap: 16px; }
  .done-state p { color: var(--text2); font-size: 15px; }

  @media (max-width: 480px) {
    .confirm-btn { position: sticky; bottom: 16px; margin-top: 16px; }
  }
</style>

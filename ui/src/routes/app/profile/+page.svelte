<script lang="ts">
  import { onMount } from 'svelte';
  import { profile, logout } from '$lib/api';
  import type { PatientModel, Demographics } from '$lib/types';
  import { goto } from '$app/navigation';

  let model: PatientModel | null = $state(null);
  let loading = $state(true);
  let saving = $state(false);
  let error = $state('');
  let saved = $state(false);
  let editMode = $state(false);

  let demo: Demographics = $state({});

  onMount(async () => {
    try {
      model = await profile.get();
      demo = { ...(model?.demographics ?? {}) };
    } catch (e) {
      error = String(e);
    } finally {
      loading = false;
    }
  });

  async function save() {
    saving = true;
    error = '';
    try {
      model = await profile.updateDemographics(demo);
      saved = true;
      editMode = false;
      setTimeout(() => saved = false, 2500);
    } catch (e) {
      error = String(e);
    } finally {
      saving = false;
    }
  }

  async function doLogout() {
    await logout();
    goto('/');
  }

  function riskColor(label: string) {
    if (label === 'high') return 'var(--red)';
    if (label === 'intermediate') return 'var(--orange)';
    return 'var(--green)';
  }

  function bmi(h?: number, w?: number) {
    if (!h || !w) return null;
    return (w / ((h / 100) ** 2)).toFixed(1);
  }
</script>

<div class="page">
  <header class="page-header">
    <div>
      <h1>Health Profile</h1>
      <p class="subtitle">Your personal health data</p>
    </div>
    <div class="header-actions">
      {#if saved}<span class="saved-badge">✓ Saved</span>{/if}
      {#if !editMode}
        <button class="btn-outline" onclick={() => editMode = true}>Edit</button>
      {/if}
    </div>
  </header>

  {#if error}
    <div class="alert-error">{error}</div>
  {/if}

  {#if loading}
    <div class="loading-state"><div class="spinner"></div></div>
  {:else}
    <!-- Risk scores -->
    {#if model?.risk_scores?.ascvd_10yr}
      {@const rs = model.risk_scores.ascvd_10yr}
      <section class="risk-banner" style="border-color: {riskColor(rs.label)}">
        <div class="risk-content">
          <div class="risk-label">10-Year ASCVD Risk</div>
          <div class="risk-value" style="color: {riskColor(rs.label)}">{(rs.value * 100).toFixed(1)}%</div>
          <div class="risk-category" style="color: {riskColor(rs.label)}">{rs.label} risk</div>
        </div>
        <div class="risk-gauge">
          <svg viewBox="0 0 100 60" fill="none">
            <path d="M10 55 A45 45 0 0 1 90 55" stroke="#E5E5EA" stroke-width="10" stroke-linecap="round"/>
            <path d="M10 55 A45 45 0 0 1 90 55" stroke={riskColor(rs.label)} stroke-width="10" stroke-linecap="round"
              stroke-dasharray="{rs.value * 141} 141"/>
          </svg>
        </div>
      </section>
    {/if}

    <!-- Demographics form -->
    <div class="card">
      <div class="card-header">
        <h3>Demographics</h3>
      </div>

      {#if editMode}
        <form class="demo-form" onsubmit={(e) => { e.preventDefault(); save(); }}>
          <div class="form-row">
            <label>
              <span>Age</span>
              <input type="number" bind:value={demo.age} min="0" max="120" placeholder="–" />
            </label>
            <label>
              <span>Sex</span>
              <select bind:value={demo.sex}>
                <option value="">Select…</option>
                <option value="male">Male</option>
                <option value="female">Female</option>
                <option value="other">Other</option>
              </select>
            </label>
          </div>
          <div class="form-row">
            <label>
              <span>Height (cm)</span>
              <input type="number" bind:value={demo.height_cm} min="50" max="250" placeholder="–" />
            </label>
            <label>
              <span>Weight (kg)</span>
              <input type="number" bind:value={demo.weight_kg} min="10" max="400" placeholder="–" />
            </label>
          </div>
          <div class="form-row">
            <label>
              <span>SBP (mmHg)</span>
              <input type="number" bind:value={demo.blood_pressure_systolic} min="60" max="250" placeholder="–" />
            </label>
            <label>
              <span>DBP (mmHg)</span>
              <input type="number" bind:value={demo.blood_pressure_diastolic} min="40" max="150" placeholder="–" />
            </label>
          </div>
          <div class="form-row">
            <label class="check-label">
              <input type="checkbox" bind:checked={demo.smoker} />
              <span>Current smoker</span>
            </label>
            <label class="check-label">
              <input type="checkbox" bind:checked={demo.diabetes} />
              <span>Diabetes</span>
            </label>
          </div>
          <div class="form-actions">
            <button type="button" class="btn-outline" onclick={() => editMode = false}>Cancel</button>
            <button type="submit" class="btn-primary" disabled={saving}>
              {saving ? 'Saving…' : 'Save Changes'}
            </button>
          </div>
        </form>
      {:else}
        <div class="demo-grid">
          {#each [
            ['Age', demo.age ? `${demo.age} years` : '–'],
            ['Sex', demo.sex ?? '–'],
            ['Height', demo.height_cm ? `${demo.height_cm} cm` : '–'],
            ['Weight', demo.weight_kg ? `${demo.weight_kg} kg` : '–'],
            ['BMI', bmi(demo.height_cm, demo.weight_kg) ?? '–'],
            ['Blood Pressure', demo.blood_pressure_systolic ? `${demo.blood_pressure_systolic}/${demo.blood_pressure_diastolic} mmHg` : '–'],
            ['Smoker', demo.smoker ? 'Yes' : 'No'],
            ['Diabetes', demo.diabetes ? 'Yes' : 'No'],
          ] as [label, val]}
            <div class="demo-item">
              <span class="demo-label">{label}</span>
              <span class="demo-val">{val}</span>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Conditions -->
    {#if model?.conditions?.length}
      <div class="card">
        <div class="card-header"><h3>Conditions</h3></div>
        <div class="tag-list">
          {#each model.conditions as c}
            <div class="condition-tag">
              <span class="cond-icd">{c.icd10}</span>
              <span>{c.description}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Sign out -->
    <div class="card danger-zone">
      <button class="btn-danger" onclick={doLogout}>Sign Out</button>
    </div>
  {/if}
</div>

<style>
  .page { padding: 32px 28px; max-width: 680px; }

  .page-header {
    display: flex; align-items: flex-start; justify-content: space-between;
    margin-bottom: 24px; gap: 16px;
  }

  h1 { font-size: 28px; font-weight: 700; }
  .subtitle { font-size: 14px; color: var(--text2); margin-top: 2px; }

  .header-actions { display: flex; align-items: center; gap: 10px; }

  .saved-badge {
    font-size: 13px; color: var(--green); font-weight: 600;
    animation: fadeIn 0.2s;
  }
  @keyframes fadeIn { from { opacity: 0; } }

  .btn-outline {
    all: unset; cursor: pointer;
    border: 1.5px solid var(--separator); border-radius: 10px;
    padding: 8px 16px; font-size: 14px; font-weight: 500; color: var(--text);
    transition: background 0.15s;
  }
  .btn-outline:hover { background: var(--bg); }

  .btn-primary {
    all: unset; cursor: pointer;
    background: var(--blue); color: white;
    padding: 10px 20px; border-radius: 10px;
    font-size: 14px; font-weight: 600;
    transition: opacity 0.15s;
  }
  .btn-primary:hover { opacity: 0.85; }
  .btn-primary:disabled { opacity: 0.5; cursor: default; }

  .alert-error {
    background: rgba(255,59,48,0.08); border-radius: 10px;
    color: var(--red); padding: 12px 16px; font-size: 14px; margin-bottom: 20px;
  }

  .loading-state {
    display: flex; align-items: center; justify-content: center; padding: 80px;
  }

  .spinner {
    width: 28px; height: 28px;
    border: 3px solid var(--separator); border-top-color: var(--blue);
    border-radius: 50%; animation: spin 0.8s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  /* Risk */
  .risk-banner {
    background: white; border-radius: var(--radius);
    border: 2px solid var(--separator);
    padding: 20px; margin-bottom: 16px;
    display: flex; align-items: center; justify-content: space-between;
    box-shadow: 0 1px 4px rgba(0,0,0,0.06);
  }

  .risk-label { font-size: 12px; font-weight: 600; color: var(--text2); text-transform: uppercase; letter-spacing: 0.5px; }
  .risk-value { font-size: 40px; font-weight: 800; line-height: 1; margin: 4px 0; }
  .risk-category { font-size: 14px; font-weight: 600; text-transform: capitalize; }

  .risk-gauge svg { width: 100px; height: 60px; }

  /* Cards */
  .card {
    background: white; border-radius: var(--radius);
    box-shadow: 0 1px 3px rgba(0,0,0,0.06);
    margin-bottom: 16px; overflow: hidden;
  }

  .card-header {
    padding: 16px 20px 0;
  }

  .card-header h3 {
    font-size: 13px; font-weight: 700; color: var(--text2);
    text-transform: uppercase; letter-spacing: 0.5px;
    margin-bottom: 12px;
  }

  /* Demo grid */
  .demo-grid {
    display: grid; grid-template-columns: 1fr 1fr;
    gap: 0; padding: 4px 0 8px;
  }

  .demo-item {
    display: flex; flex-direction: column; gap: 2px;
    padding: 10px 20px;
    border-bottom: 1px solid var(--separator);
  }

  .demo-item:nth-last-child(-n+2) { border-bottom: none; }

  .demo-label { font-size: 12px; color: var(--text3); font-weight: 500; }
  .demo-val { font-size: 15px; font-weight: 500; color: var(--text); }

  /* Form */
  .demo-form { padding: 4px 20px 20px; display: flex; flex-direction: column; gap: 14px; }

  .form-row { display: flex; gap: 12px; }
  .form-row label, .check-label {
    flex: 1; display: flex; flex-direction: column; gap: 5px;
    font-size: 13px; font-weight: 500; color: var(--text2);
  }

  .check-label {
    flex-direction: row; align-items: center;
    background: var(--bg); border-radius: 10px;
    padding: 12px 14px; cursor: pointer;
  }
  .check-label input[type=checkbox] { accent-color: var(--blue); width: 16px; height: 16px; }
  .check-label span { font-size: 15px; color: var(--text); }

  .form-row input, .form-row select {
    all: unset; background: var(--bg); border-radius: 10px;
    padding: 11px 13px; font-size: 15px; color: var(--text);
    border: 1.5px solid transparent; transition: border-color 0.15s;
    font-family: inherit; width: 100%;
  }
  .form-row input:focus, .form-row select:focus { border-color: var(--blue); }

  .form-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 4px; }

  /* Conditions */
  .tag-list { display: flex; flex-wrap: wrap; gap: 8px; padding: 4px 20px 20px; }

  .condition-tag {
    display: flex; align-items: center; gap: 6px;
    background: var(--bg); border-radius: 20px;
    padding: 6px 12px; font-size: 13px;
  }
  .cond-icd {
    font-size: 11px; font-weight: 700;
    background: rgba(0,122,255,0.1); color: var(--blue);
    padding: 2px 6px; border-radius: 4px;
  }

  /* Danger zone */
  .danger-zone { padding: 16px 20px; display: flex; }

  .btn-danger {
    all: unset; cursor: pointer;
    color: var(--red); font-size: 15px; font-weight: 500;
    padding: 4px 0; transition: opacity 0.15s;
  }
  .btn-danger:hover { opacity: 0.7; }

  @media (max-width: 768px) {
    .page { padding: 20px 16px; }
    .demo-grid { grid-template-columns: 1fr; }
    .demo-item:nth-last-child(-n+2) { border-bottom: 1px solid var(--separator); }
    .demo-item:last-child { border-bottom: none; }
  }
</style>

<script lang="ts">
  import { profile } from '$lib/api';
  import type { PatientModel, Demographics } from '$lib/types';

  let model = $state<PatientModel | null>(null);
  let demo = $state<Demographics>({});
  let conditions = $state<{ name: string; status: string }[]>([]);
  let loading = $state(true);
  let saving = $state(false);
  let newCondition = $state('');

  async function load() {
    try {
      model = await profile.get();
      demo = model?.demographics ?? {};
      conditions = (model?.active_conditions ?? []).map(c => ({ name: c.name, status: c.status }));
    } catch {}
    loading = false;
  }

  async function saveDemographics() {
    saving = true;
    try { await profile.updateDemographics(demo); }
    finally { saving = false; }
  }

  async function saveConditions() {
    saving = true;
    try { await profile.updateConditions(conditions); }
    finally { saving = false; }
  }

  function addCondition() {
    if (!newCondition.trim()) return;
    conditions = [...conditions, { name: newCondition.trim(), status: 'active' }];
    newCondition = '';
    saveConditions();
  }

  function removeCondition(i: number) {
    conditions = conditions.filter((_, idx) => idx !== i);
    saveConditions();
  }

  function numVal(v: number | undefined): string { return v ? String(v) : ''; }
  function setNum(field: keyof Demographics, e: Event) {
    const val = parseFloat((e.target as HTMLInputElement).value);
    (demo as any)[field] = isNaN(val) ? undefined : val;
    saveDemographics();
  }

  $effect(() => { load(); });
</script>

<div class="page">
  <h1>Profile</h1>

  {#if loading}
    <div class="center"><div class="spinner"></div></div>
  {:else}
    <div class="section">
      <h2>Demographics</h2>
      <div class="card">
        <div class="row">
          <span class="label">Sex</span>
          <select class="value-select" bind:value={demo.sex} onchange={saveDemographics}>
            <option value="">—</option>
            <option value="M">Male</option>
            <option value="F">Female</option>
          </select>
        </div>
        <div class="row">
          <span class="label">Age</span>
          <div class="input-wrap">
            <input type="number" value={numVal(demo.age)} onblur={(e) => setNum('age', e)} placeholder="—" />
            <span class="unit">years</span>
          </div>
        </div>
        <div class="row">
          <span class="label">Height</span>
          <div class="input-wrap">
            <input type="number" value={numVal(demo.height_cm)} onblur={(e) => setNum('height_cm', e)} placeholder="—" />
            <span class="unit">cm</span>
          </div>
        </div>
        <div class="row">
          <span class="label">Weight</span>
          <div class="input-wrap">
            <input type="number" value={numVal(demo.weight_kg)} onblur={(e) => setNum('weight_kg', e)} placeholder="—" />
            <span class="unit">kg</span>
          </div>
        </div>
        <div class="row">
          <span class="label">Smoker</span>
          <input type="checkbox" class="toggle" bind:checked={demo.smoker} onchange={saveDemographics} />
        </div>
        <div class="row">
          <span class="label">Systolic BP</span>
          <div class="input-wrap">
            <input type="number" value={numVal(demo.blood_pressure_systolic)} onblur={(e) => setNum('blood_pressure_systolic', e)} placeholder="—" />
            <span class="unit">mmHg</span>
          </div>
        </div>
        <div class="row">
          <span class="label">Diastolic BP</span>
          <div class="input-wrap">
            <input type="number" value={numVal(demo.blood_pressure_diastolic)} onblur={(e) => setNum('blood_pressure_diastolic', e)} placeholder="—" />
            <span class="unit">mmHg</span>
          </div>
        </div>
      </div>
    </div>

    <div class="section">
      <h2>Active Conditions</h2>
      <div class="card">
        {#each conditions as cond, i}
          <div class="row">
            <span class="cond-name">{cond.name}</span>
            <button class="remove-btn" onclick={() => removeCondition(i)}>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--red)" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
          </div>
        {/each}
        <form class="add-row" onsubmit={(e) => { e.preventDefault(); addCondition(); }}>
          <input bind:value={newCondition} placeholder="Add condition..." />
          <button type="submit" disabled={!newCondition.trim()}>Add</button>
        </form>
      </div>
    </div>

    {#if model?.risk_scores?.ascvd_10yr}
      <div class="section">
        <h2>Risk Scores</h2>
        <div class="card">
          <div class="risk-row">
            <span>ASCVD 10-Year Risk</span>
            <span class="risk-value">{model.risk_scores.ascvd_10yr.value.toFixed(1)}%</span>
          </div>
        </div>
      </div>
    {/if}
  {/if}
</div>

<style>
  .page { max-width: 680px; margin: 0 auto; padding: 24px 20px 40px; }
  h1 { font-size: 28px; font-weight: 700; margin-bottom: 24px; }
  .center { display: flex; justify-content: center; padding: 60px 0; }
  .spinner { width: 32px; height: 32px; border: 3px solid var(--separator); border-top-color: var(--blue); border-radius: 50%; animation: spin 0.8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }

  .section { margin-bottom: 24px; }
  .section h2 { font-size: 13px; font-weight: 600; color: var(--text3); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 8px; padding-left: 16px; }
  .card { background: var(--bg2); border-radius: var(--radius); overflow: hidden; }
  .row {
    display: flex; align-items: center; justify-content: space-between;
    padding: 12px 16px; border-bottom: 1px solid var(--separator);
  }
  .row:last-child { border-bottom: none; }
  .label { font-size: 15px; color: var(--text); }
  .input-wrap { display: flex; align-items: center; gap: 4px; }
  .input-wrap input {
    width: 80px; text-align: right; padding: 6px 8px; border-radius: 8px;
    border: 1px solid var(--separator); font-size: 15px; background: var(--bg); outline: none;
    -moz-appearance: textfield;
  }
  .input-wrap input::-webkit-outer-spin-button,
  .input-wrap input::-webkit-inner-spin-button { -webkit-appearance: none; margin: 0; }
  .input-wrap input:focus { border-color: var(--blue); }
  .unit { font-size: 13px; color: var(--text3); min-width: 32px; }
  .value-select { padding: 6px 8px; border-radius: 8px; border: 1px solid var(--separator); font-size: 15px; background: var(--bg); }

  .toggle {
    width: 48px; height: 28px; appearance: none; -webkit-appearance: none;
    background: var(--separator); border-radius: 14px; position: relative; cursor: pointer; transition: background 0.2s; border: none;
  }
  .toggle:checked { background: var(--green); }
  .toggle::after {
    content: ''; position: absolute; width: 24px; height: 24px; background: white; border-radius: 50%;
    top: 2px; left: 2px; transition: transform 0.2s; box-shadow: 0 1px 3px rgba(0,0,0,0.15);
  }
  .toggle:checked::after { transform: translateX(20px); }

  .cond-name { font-size: 15px; }
  .remove-btn { all: unset; cursor: pointer; opacity: 0.5; transition: opacity 0.15s; }
  .remove-btn:hover { opacity: 1; }

  .add-row { display: flex; gap: 8px; padding: 12px 16px; }
  .add-row input {
    flex: 1; padding: 8px 12px; border-radius: 8px; border: 1px solid var(--separator);
    font-size: 14px; background: var(--bg); outline: none;
  }
  .add-row input:focus { border-color: var(--blue); }
  .add-row button {
    all: unset; cursor: pointer; padding: 8px 16px; border-radius: 8px;
    font-size: 14px; font-weight: 500; color: var(--blue); background: rgba(13,148,136,0.1);
  }
  .add-row button:disabled { opacity: 0.3; }

  .risk-row { display: flex; justify-content: space-between; padding: 14px 16px; }
  .risk-value { font-size: 20px; font-weight: 700; color: var(--blue); }
</style>

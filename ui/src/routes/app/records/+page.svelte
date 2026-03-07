<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { labs, documents, medications, questions, visits as visitsApi } from '$lib/api';
  import type { Lab, Document, Medication, BacklogQuestion, Visit } from '$lib/types';
  import SegmentedControl from '$lib/components/SegmentedControl.svelte';
  import UploadZone from '$lib/components/UploadZone.svelte';
  import SparkLine from '$lib/components/SparkLine.svelte';
  import ConversationSheet from '$lib/components/ConversationSheet.svelte';

  // Restore tab from URL search params for state persistence
  const initTab = parseInt(new URLSearchParams(globalThis.location?.search || '').get('tab') || '0');
  let tab = $state(isNaN(initTab) ? 0 : initTab);
  let labList = $state<Lab[]>([]);
  let docList = $state<Document[]>([]);
  let allMeds = $state<Medication[]>([]);
  let loading = $state(true);

  let showConversation = $state(false);
  let convContextType = $state('');
  let convContextId = $state('');
  let conversationLabel = $state('');

  // Questions backlog
  let questionList = $state<BacklogQuestion[]>([]);
  let visitList = $state<Visit[]>([]);
  let showAddQuestion = $state(false);
  let newQuestionText = $state('');
  let newQuestionRationale = $state('');
  let selectedQuestions = $state<Set<string>>(new Set());
  let assignVisitId = $state('');
  let assigning = $state(false);

  // Medication management
  let showAddForm = $state(false);
  let addName = $state('');
  let addDose = $state('');
  let addFrequency = $state('');
  let addStartDate = $state(new Date().toISOString().split('T')[0]);
  let addSaving = $state(false);
  let showHistory = $state(false);
  let undoMed = $state<Medication | null>(null);
  let undoTimer = $state<ReturnType<typeof setTimeout> | null>(null);

  // Document view
  let docView = $state<'timeline' | 'categories'>('timeline');

  const docCatMeta: Record<string, { label: string; color: string; order: number }> = {
    lab_result:       { label: 'Lab Results',       color: '#0D9488', order: 1 },
    specialist_visit: { label: 'Specialist Visit',  color: '#7C3AED', order: 2 },
    prescription:     { label: 'Prescription',      color: '#EA580C', order: 3 },
    imaging:          { label: 'Imaging',           color: '#2563EB', order: 4 },
    discharge:        { label: 'Discharge',         color: '#DC2626', order: 5 },
    referral:         { label: 'Referral',          color: '#CA8A04', order: 6 },
    vaccination:      { label: 'Vaccination',       color: '#16A34A', order: 7 },
    other:            { label: 'Other',             color: '#6B7280', order: 99 },
  };

  const docTimeline = $derived.by(() => {
    const byDate: Record<string, Document[]> = {};
    for (const doc of docList) {
      const d = (doc.document_date || doc.created_at).split('T')[0];
      if (!byDate[d]) byDate[d] = [];
      byDate[d].push(doc);
    }
    return Object.entries(byDate)
      .sort(([a], [b]) => b.localeCompare(a))
      .map(([date, docs]) => ({ date, docs }));
  });

  const docCategories = $derived.by(() => {
    const cats: Record<string, { label: string; color: string; order: number; docs: Document[] }> = {};
    for (const doc of docList) {
      const key = doc.category || 'other';
      if (!cats[key]) {
        const meta = docCatMeta[key] ?? docCatMeta.other;
        cats[key] = { ...meta, docs: [] };
      }
      cats[key].docs.push(doc);
    }
    return Object.values(cats).sort((a, b) => a.order - b.order);
  });

  // Lab dashboard — restore view from URL
  const initView = new URLSearchParams(globalThis.location?.search || '').get('view');
  let labView = $state<'categories' | 'timeline'>(initView === 'timeline' ? 'timeline' : 'categories');
  let labSearch = $state('');
  let expandedCats = $state<Set<string>>(new Set());
  let expandedDates = $state<Set<string>>(new Set());

  const activeMeds = $derived(allMeds.filter(m => m.is_active));
  const pastMeds = $derived(allMeds.filter(m => !m.is_active));

  async function load() {
    loading = true;
    const [labRes, docRes, medRes, qRes, vRes] = await Promise.allSettled([
      labs.list(), documents.list(), medications.listAll(),
      questions.list(true), visitsApi.list(),
    ]);
    if (labRes.status === 'fulfilled') labList = labRes.value;
    if (docRes.status === 'fulfilled') docList = docRes.value;
    if (medRes.status === 'fulfilled') allMeds = medRes.value;
    if (qRes.status === 'fulfilled') questionList = qRes.value;
    if (vRes.status === 'fulfilled') visitList = vRes.value;
    loading = false;
    initLabExpanded();
  }

  // ── Lab classification ──────────────────────────────────────────────

  const categoryMeta: Record<string, { label: string; order: number }> = {
    cbc:          { label: 'Blood Counts (CBC)', order: 1 },
    metabolic:    { label: 'Metabolic Panel',    order: 2 },
    liver:        { label: 'Liver Function',     order: 3 },
    lipids:       { label: 'Lipid Panel',        order: 4 },
    kidney:       { label: 'Kidney Function',    order: 5 },
    thyroid:      { label: 'Thyroid',            order: 6 },
    iron:         { label: 'Iron Studies',       order: 7 },
    coagulation:  { label: 'Coagulation',        order: 8 },
    inflammation: { label: 'Inflammation',       order: 9 },
    vitamins:     { label: 'Vitamins',           order: 10 },
    proteins:     { label: 'Proteins',           order: 11 },
    other:        { label: 'Other',              order: 99 },
  };

  function classifyLab(name: string): string {
    const n = name.toLowerCase();
    // CBC — English, Bosnian/Croatian/Serbian, Japanese
    if (/\b(wbc|rbc|hemoglobin|hgb|hematocrit|hct|platelet|plt|mcv|mchc?|rdw|mpv)\b/.test(n)) return 'cbc';
    if (/\b(neutrophil|lymphocyte|monocyte|eosinophil|basophil|leukocyte|erythrocyte)\b/.test(n)) return 'cbc';
    if (/(white blood|red blood|blood count|reticulocyte)/.test(n)) return 'cbc';
    if (/eritrocit|hematokrit|leukocit|trombocit|limfocit|granulocit|bazo[fk]il|eozino[fk]il|mono[ck]it|neutro[fk]il|hemoglob/i.test(n)) return 'cbc';
    if (/白血球|赤血球|血小板|ヘモグロビン|ヘマトクリット|好中球|リンパ球|単球|好酸球|好塩基球/.test(name)) return 'cbc';
    // Coagulation (before liver — catches PT, aPTT)
    if (/\b(aptt|ptt|fibrinogen|antithrombin|d-?dimer|prothrombin|inr)\b/.test(n)) return 'coagulation';
    if (/^pt\b/.test(n)) return 'coagulation';
    if (/fibrinogen|protromb/i.test(n)) return 'coagulation';
    // Liver
    if (/\b(sgpt|sgot|ggt|gamma.?glutamyl)\b/.test(n)) return 'liver';
    if (/\bbilirubin\b/.test(n)) return 'liver';
    if (/\b(transaminase|fosfataza)\b/.test(n)) return 'liver';
    if (/^(alt|ast)\b/.test(n)) return 'liver';
    if (/alanine.*trans|aspartate.*trans/.test(n)) return 'liver';
    if (/alkaline.?phosph/.test(n) || /^alp\b/.test(n)) return 'liver';
    if (/ビリルビン|肝機能|AST|ALT|γ.?GTP/.test(name)) return 'liver';
    // Kidney
    if (/\b(creatinine|egfr|gfr|cystatin|cre)\b/.test(n)) return 'kidney';
    if (/\bbun\b/.test(n) || /\burea\b/.test(n)) return 'kidney';
    if (/kreatinin/i.test(n)) return 'kidney';
    if (/クレアチニン|腎機能|尿素窒素|eGFR/.test(name)) return 'kidney';
    // Lipids
    if (/\b(cholesterol|ldl|hdl|triglyceride|lipid|lipoprotein|apolipoprotein|tg)\b/.test(n)) return 'lipids';
    if (/holesterol|triglicerid|lipid/i.test(n)) return 'lipids';
    if (/コレステロール|中性脂肪|LDL|HDL|脂質/.test(name)) return 'lipids';
    // Thyroid
    if (/\b(tsh|thyroid|thyroxine|thyroglobulin)\b/.test(n) || /\bt[34]\b/.test(n)) return 'thyroid';
    if (/甲状腺/.test(name)) return 'thyroid';
    // Iron
    if (/\b(iron|ferritin|tibc|transferrin|siderophilin)\b/.test(n) || /^fe\b/.test(n)) return 'iron';
    if (/feritin|željezo|gvožđe|gvozdje/i.test(n)) return 'iron';
    if (/フェリチン|鉄/.test(name)) return 'iron';
    // Inflammation
    if (/\b(crp|esr|procalcitonin)\b/.test(n) || /c.?reactive/.test(n) || /sed.?rate/.test(n)) return 'inflammation';
    if (/sedimentacij/i.test(n)) return 'inflammation';
    if (/CRP|炎症/.test(name)) return 'inflammation';
    // Vitamins
    if (/\b(vitamin|b12|cobalamin|folate|folic)\b/.test(n) || /25.?hydroxy/.test(n)) return 'vitamins';
    if (/ビタミン/.test(name)) return 'vitamins';
    // Metabolic (broad — glucose, electrolytes, pancreatic, etc.)
    if (/\b(glucose|sugar|sodium|potassium|chloride|co2|bicarbonate|calcium|magnesium|phosph|hba1c|a1c|amylase|lipase)\b/.test(n)) return 'metabolic';
    if (/glukoz|glikemij|kalij|natrij|kalcij|magnezij|amilaz/i.test(n)) return 'metabolic';
    if (/pankreati[čc]/i.test(n)) return 'metabolic';
    if (/血糖|グルコース|HbA1c|ナトリウム|カリウム|カルシウム|アミラーゼ/.test(name)) return 'metabolic';
    // Proteins
    if (/\b(albumin|globulin|total.?protein|immunoglobulin|ig[gamde]\b)\b/.test(n)) return 'proteins';
    if (/proteini|albumin|imunoglobul/i.test(n)) return 'proteins';
    return 'other';
  }

  // ── Lab name normalization & dedup ──────────────────────────────────

  /** Canonical synonyms: maps normalized forms to a single canonical key */
  const labSynonyms: Record<string, string> = {
    'alb': 'albumin',
    'alb albumin': 'albumin',
    'hgb': 'hemoglobin',
    'hb': 'hemoglobin',
    'hct': 'hematocrit',
    'plt': 'platelet',
    'platelets': 'platelet',
    'wbc': 'white blood cells',
    'rbc': 'red blood cells',
    'tg': 'triglycerides',
    'chol': 'cholesterol',
    'cre': 'creatinine',
    'glu': 'glucose',
    'insulin na gladno': 'fasting insulin',
    'insulin na prazan stomak': 'fasting insulin',
    'insulin fastin': 'fasting insulin',
    'glukoza na taste': 'fasting glucose',
    'glukoza naste': 'fasting glucose',
    'glukoza na prazan stomak': 'fasting glucose',
    'glikemija': 'glucose',
    'secer u krvi': 'glucose',
    'šećer u krvi': 'glucose',
    'eritrociti': 'red blood cells',
    'leukociti': 'white blood cells',
    'trombociti': 'platelet',
    'hemoglobin hemoglobin': 'hemoglobin',
  };

  /**
   * Normalize a lab name for dedup grouping:
   * 1. lowercase
   * 2. strip parenthetical content like "(Albumin)" — they're aliases
   * 3. collapse whitespace
   * 4. check synonym table
   */
  function normalizeLabName(name: string): string {
    if (!name) return 'unknown';
    let n = name.toLowerCase();
    // Remove parenthetical aliases: "ALB (Albumin)" → "alb"
    n = n.replace(/\s*\([^)]*\)\s*/g, ' ');
    // Remove common prefixes/suffixes that don't add info
    n = n.replace(/^(serum|plasma|blood|total)\s+/i, '');
    // Collapse whitespace, trim
    n = n.replace(/\s+/g, ' ').trim();
    // Check synonym map
    if (labSynonyms[n]) return labSynonyms[n];
    return n;
  }

  /** Pick the best display name from a group of labs with different raw names */
  function pickBestName(labs: Lab[]): string {
    // Prefer the longest name (most descriptive), then most recent
    const names = labs.map(l => l.lab_name || l.loinc_code || 'Unknown');
    return names.sort((a, b) => b.length - a.length)[0];
  }

  // ── Derived lab data ────────────────────────────────────────────────

  const labGroups = $derived.by(() => {
    const groups: Record<string, Lab[]> = {};
    for (const l of labList) {
      const raw = l.lab_name || l.loinc_code || 'Unknown';
      const key = l.loinc_code || normalizeLabName(raw);
      if (!groups[key]) groups[key] = [];
      groups[key].push(l);
    }
    return Object.entries(groups).map(([_key, items]) => ({
      name: pickBestName(items),
      latest: items[0],
      points: items.map(i => i.value).reverse(),
      flag: items[0].flag,
      category: classifyLab(pickBestName(items)),
    }));
  });

  const abnormalGroups = $derived(labGroups.filter(g => g.flag !== 'normal' && g.flag !== ''));

  const categorizedLabs = $derived.by(() => {
    const cats: Record<string, { key: string; label: string; order: number; groups: typeof labGroups; abnormalCount: number }> = {};
    const filtered = labSearch
      ? labGroups.filter(g => g.name.toLowerCase().includes(labSearch.toLowerCase()))
      : labGroups;
    for (const g of filtered) {
      const key = g.category;
      if (!cats[key]) {
        const meta = categoryMeta[key] ?? { label: key, order: 50 };
        cats[key] = { key, label: meta.label, order: meta.order, groups: [], abnormalCount: 0 };
      }
      cats[key].groups.push(g);
      if (g.flag !== 'normal' && g.flag !== '') cats[key].abnormalCount++;
    }
    return Object.values(cats).sort((a, b) => {
      if (a.abnormalCount > 0 && b.abnormalCount === 0) return -1;
      if (a.abnormalCount === 0 && b.abnormalCount > 0) return 1;
      return a.order - b.order;
    });
  });

  const timelineGroups = $derived.by(() => {
    const byDate: Record<string, { labs: Lab[]; abnormalCount: number }> = {};
    const filtered = labSearch
      ? labList.filter(l => (l.lab_name || l.loinc_code || '').toLowerCase().includes(labSearch.toLowerCase()))
      : labList;
    for (const l of filtered) {
      const d = l.collected_at.split('T')[0];
      if (!byDate[d]) byDate[d] = { labs: [], abnormalCount: 0 };
      byDate[d].labs.push(l);
      if (l.flag !== 'normal' && l.flag !== '') byDate[d].abnormalCount++;
    }
    return Object.entries(byDate)
      .sort(([a], [b]) => b.localeCompare(a))
      .map(([date, data]) => ({ date, ...data }));
  });

  const lastDrawDate = $derived(labList.length > 0 ? labList[0].collected_at : '');

  function initLabExpanded() {
    const initial = new Set<string>();
    for (const l of labList) {
      if (l.flag !== 'normal' && l.flag !== '') {
        initial.add(classifyLab(l.lab_name || l.loinc_code || ''));
      }
    }
    expandedCats = initial;
    if (labList.length > 0) {
      expandedDates = new Set([labList[0].collected_at.split('T')[0]]);
    }
  }

  function toggleCat(key: string) {
    const next = new Set(expandedCats);
    if (next.has(key)) next.delete(key); else next.add(key);
    expandedCats = next;
  }

  function toggleDate(date: string) {
    const next = new Set(expandedDates);
    if (next.has(date)) next.delete(date); else next.add(date);
    expandedDates = next;
  }

  // ── Shared helpers ──────────────────────────────────────────────────

  async function handleDocUpload(files: File[]) {
    await documents.upload(files);
    docList = await documents.list();
  }

  async function deleteDoc(id: string) {
    await documents.remove(id);
    docList = docList.filter(d => d.id !== id);
  }

  async function reparseDoc(id: string) {
    await documents.reparse(id);
    const idx = docList.findIndex(d => d.id === id);
    if (idx >= 0) { docList[idx].parse_status = 'processing'; docList = docList; }
  }

  async function reparseAll() {
    for (const doc of docList) {
      await documents.reparse(doc.id);
    }
    docList = docList.map(d => ({ ...d, parse_status: 'processing' }));
  }

  function askAboutSection(sectionKey: string, sectionLabel: string) {
    convContextType = 'lab_category';
    convContextId = sectionKey;
    conversationLabel = sectionLabel;
    showConversation = true;
  }

  function askAboutAttention() {
    convContextType = 'lab_category';
    convContextId = 'needs_attention';
    conversationLabel = 'Needs Attention';
    showConversation = true;
  }

  function askAboutLab(lab: Lab) {
    const labKey = lab.lab_name || lab.loinc_code || lab.id;
    convContextType = 'lab';
    convContextId = labKey;
    conversationLabel = lab.lab_name || lab.loinc_code || 'Lab';
    showConversation = true;
  }

  function askAboutMed(med: Medication) {
    convContextType = 'medication';
    convContextId = med.id;
    conversationLabel = med.name;
    showConversation = true;
  }

  function askAboutMeds() {
    convContextType = 'medication';
    convContextId = 'all';
    conversationLabel = 'Medication Interactions';
    showConversation = true;
  }

  async function addMedication() {
    if (!addName.trim()) return;
    addSaving = true;
    try {
      await medications.add({
        name: addName.trim(),
        dose: addDose.trim(),
        frequency: addFrequency.trim(),
        started_at: addStartDate || undefined,
      });
      allMeds = await medications.listAll();
      addName = ''; addDose = ''; addFrequency = '';
      addStartDate = new Date().toISOString().split('T')[0];
      showAddForm = false;
    } catch (e) { console.error('Add med failed', e); }
    addSaving = false;
  }

  async function stopMedication(med: Medication) {
    await medications.stop(med.id);
    undoMed = med;
    allMeds = await medications.listAll();
    if (undoTimer) clearTimeout(undoTimer);
    undoTimer = setTimeout(() => { undoMed = null; undoTimer = null; }, 5000);
  }

  async function undoStop() {
    if (!undoMed) return;
    await medications.reactivate(undoMed.id);
    allMeds = await medications.listAll();
    undoMed = null;
    if (undoTimer) { clearTimeout(undoTimer); undoTimer = null; }
  }

  function formatDate(s: string) {
    if (!s) return '';
    return new Date(s).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: '2-digit' });
  }

  function formatDateFull(s: string) {
    if (!s) return '';
    return new Date(s).toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' });
  }

  function formatDateRange(med: Medication) {
    const start = med.started_at ? formatDate(med.started_at) : '?';
    const end = med.ended_at ? formatDate(med.ended_at) : 'now';
    return `${start} → ${end}`;
  }

  // ── Questions helpers ──────────────────────────────────────────────

  const activeVisits = $derived(visitList.filter(v => v.status === 'preparing' || v.status === 'during'));

  function toggleQuestionSelect(id: string) {
    const next = new Set(selectedQuestions);
    if (next.has(id)) next.delete(id); else next.add(id);
    selectedQuestions = next;
  }

  function selectAll() {
    if (selectedQuestions.size === questionList.length) {
      selectedQuestions = new Set();
    } else {
      selectedQuestions = new Set(questionList.map(q => q.id));
    }
  }

  async function addNewQuestion() {
    if (!newQuestionText.trim()) return;
    await questions.create({ text: newQuestionText.trim(), rationale: newQuestionRationale.trim() });
    newQuestionText = '';
    newQuestionRationale = '';
    showAddQuestion = false;
    questionList = await questions.list(true);
  }

  async function assignSelected() {
    if (!assignVisitId || selectedQuestions.size === 0) return;
    assigning = true;
    try {
      await questions.bulkLink([...selectedQuestions], assignVisitId);
      selectedQuestions = new Set();
      assignVisitId = '';
      questionList = await questions.list(true);
    } finally {
      assigning = false;
    }
  }

  async function assignSingle(qId: string, visitId: string) {
    await questions.link(qId, visitId);
    questionList = await questions.list(true);
  }

  $effect(() => { load(); });
</script>

<div class="page">
  <h1>Records</h1>
  <div class="seg-wrap">
    <SegmentedControl segments={['Labs', 'Documents', 'Medications', 'Questions']} selected={tab} onchange={(i) => { tab = i; const u = new URL(window.location.href); u.searchParams.set('tab', String(i)); history.replaceState({}, '', u.toString()); }} />
  </div>

  {#if loading}
    <div class="center"><div class="spinner"></div></div>

  {:else if tab === 0}
    <!-- ─── Labs Dashboard ─── -->
    {#if labGroups.length === 0}
      <div class="empty">
        <p>No lab results yet.</p>
        <p class="hint">Upload a lab document to see your results here.</p>
      </div>
    {:else}
      <div class="ld">
        <!-- Summary strip -->
        <div class="ld-summary">
          <div class="ld-stat">
            <span class="ld-stat-num">{labGroups.length}</span>
            <span class="ld-stat-lbl">tests</span>
          </div>
          {#if abnormalGroups.length > 0}
            <div class="ld-stat ld-stat-warn">
              <span class="ld-stat-num">{abnormalGroups.length}</span>
              <span class="ld-stat-lbl">need attention</span>
            </div>
          {:else}
            <div class="ld-stat ld-stat-ok">
              <span class="ld-stat-num">All</span>
              <span class="ld-stat-lbl">in range</span>
            </div>
          {/if}
          {#if lastDrawDate}
            <div class="ld-stat">
              <span class="ld-stat-lbl">Last draw</span>
              <span class="ld-stat-val">{formatDate(lastDrawDate)}</span>
            </div>
          {/if}
        </div>

        <!-- Controls row -->
        <div class="ld-controls">
          <div class="ld-toggle">
            <button class:active={labView === 'categories'} onclick={() => { labView = 'categories'; const u = new URL(window.location.href); u.searchParams.set('view', 'categories'); history.replaceState({}, '', u.toString()); }}>By System</button>
            <button class:active={labView === 'timeline'} onclick={() => { labView = 'timeline'; const u = new URL(window.location.href); u.searchParams.set('view', 'timeline'); history.replaceState({}, '', u.toString()); }}>Timeline</button>
          </div>
          <div class="ld-search-wrap">
            <svg class="ld-search-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
            <input class="ld-search" type="text" placeholder="Search labs..." bind:value={labSearch} />
          </div>
        </div>

        {#if labView === 'categories'}
          <!-- Attention banner -->
          {#if abnormalGroups.length > 0 && !labSearch}
            <div class="ld-attention">
              <div class="ld-att-header">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
                <span>Needs Attention</span>
                <button class="section-chat-btn" onclick={(e) => { e.stopPropagation(); askAboutAttention(); }} title="Discuss all flagged results">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
                  Ask AI
                </button>
              </div>
              {#each abnormalGroups as group}
                <button class="ld-att-row" onclick={() => askAboutLab(group.latest)}>
                  <div class="ld-att-info">
                    <span class="ld-att-name">{group.name}</span>
                    <span class="ld-att-cat">{categoryMeta[group.category]?.label ?? 'Other'} · {formatDate(group.latest.collected_at)}</span>
                  </div>
                  <div class="ld-att-val">
                    <span class="ld-att-num">{group.latest.value}</span>
                    <span class="ld-att-unit">{group.latest.unit}</span>
                    <span class="ld-att-flag">{group.flag.toUpperCase()}</span>
                  </div>
                </button>
              {/each}
            </div>
          {/if}

          <!-- Category panels -->
          {#each categorizedLabs as cat}
            <div class="ld-cat" class:has-warn={cat.abnormalCount > 0}>
              <div class="ld-cat-header-wrap">
                <button class="ld-cat-head" onclick={() => toggleCat(cat.key)}>
                  <div class="ld-cat-left">
                    <span class="ld-cat-name">{cat.label}</span>
                    <span class="ld-cat-count">{cat.groups.length} test{cat.groups.length !== 1 ? 's' : ''}</span>
                  </div>
                  <div class="ld-cat-right">
                    {#if cat.abnormalCount > 0}
                      <span class="ld-badge warn">{cat.abnormalCount} abnormal</span>
                    {:else}
                      <span class="ld-badge ok">Normal</span>
                    {/if}
                    <svg class="ld-chev" class:open={expandedCats.has(cat.key)} width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
                  </div>
                </button>
                <button class="cat-chat-btn" onclick={() => askAboutSection(cat.key, cat.label)} title="Discuss {cat.label}">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
                </button>
              </div>
              {#if expandedCats.has(cat.key)}
                <div class="ld-cat-body">
                  {#each cat.groups as group}
                    <button class="ld-row" onclick={() => askAboutLab(group.latest)}>
                      <div class="ld-row-info">
                        <span class="ld-row-name">{group.name}</span>
                        <span class="ld-row-date">{formatDate(group.latest.collected_at)}</span>
                      </div>
                      <div class="ld-row-val" class:flagged={group.flag !== 'normal' && group.flag !== ''}>
                        {group.latest.value} <span class="ld-row-unit">{group.latest.unit}</span>
                      </div>
                      {#if group.points.length >= 2}
                        <SparkLine points={group.points} color={group.flag !== 'normal' && group.flag !== '' ? 'var(--red)' : 'var(--blue)'} />
                      {/if}
                    </button>
                  {/each}
                </div>
              {/if}
            </div>
          {/each}

        {:else}
          <!-- Timeline view -->
          <div class="tl">
            {#each timelineGroups as tg, i}
              <div class="tl-entry">
                <div class="tl-rail">
                  <div class="tl-dot" class:tl-dot-warn={tg.abnormalCount > 0}></div>
                  {#if i < timelineGroups.length - 1}<div class="tl-line"></div>{/if}
                </div>
                <div class="tl-content">
                  <button class="tl-header" onclick={() => toggleDate(tg.date)}>
                    <div class="tl-date-info">
                      <span class="tl-date">{formatDateFull(tg.date)}</span>
                      <span class="tl-count">{tg.labs.length} test{tg.labs.length !== 1 ? 's' : ''}{#if tg.abnormalCount > 0} · <span class="tl-warn-text">{tg.abnormalCount} flagged</span>{/if}</span>
                    </div>
                    <svg class="ld-chev" class:open={expandedDates.has(tg.date)} width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
                  </button>
                  {#if expandedDates.has(tg.date)}
                    <div class="tl-labs">
                      {#each tg.labs as lab}
                        <button class="tl-lab-row" onclick={() => askAboutLab(lab)}>
                          <div class="tl-lab-info">
                            <span class="tl-lab-name">{lab.lab_name || lab.loinc_code || 'Unknown'}</span>
                            <span class="tl-lab-cat">{categoryMeta[classifyLab(lab.lab_name || lab.loinc_code || '')]?.label ?? ''}</span>
                          </div>
                          <div class="tl-lab-val" class:flagged={lab.flag !== 'normal' && lab.flag !== ''}>
                            {lab.value} <span class="tl-lab-unit">{lab.unit}</span>
                            {#if lab.flag !== 'normal' && lab.flag !== ''}
                              <span class="tl-flag">{lab.flag.toUpperCase()}</span>
                            {/if}
                          </div>
                        </button>
                      {/each}
                    </div>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        {/if}

        {#if labSearch && categorizedLabs.length === 0 && timelineGroups.length === 0}
          <div class="empty"><p>No labs matching "{labSearch}"</p></div>
        {/if}
      </div>
    {/if}

  {:else if tab === 1}
    <!-- Documents — Timeline + Categories -->
    <UploadZone onupload={handleDocUpload} label="Drop documents here" />
    {#if docList.length === 0}
      <div class="empty"><p>No documents uploaded yet.</p></div>
    {:else}
      <div class="doc-controls">
        <div class="ld-toggle">
          <button class:active={docView === 'timeline'} onclick={() => { docView = 'timeline'; }}>Timeline</button>
          <button class:active={docView === 'categories'} onclick={() => { docView = 'categories'; }}>By Type</button>
        </div>
        <div class="doc-controls-right">
          <button class="reparse-all-btn" onclick={reparseAll}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
            Re-process all
          </button>
        </div>
      </div>

      {#if docView === 'timeline'}
        <div class="tl">
          {#each docTimeline as group, i}
            <div class="tl-entry">
              <div class="tl-rail">
                <div class="tl-dot"></div>
                {#if i < docTimeline.length - 1}<div class="tl-line"></div>{/if}
              </div>
              <div class="tl-content">
                <div class="dtl-date">{formatDateFull(group.date)}</div>
                <div class="dtl-docs">
                  {#each group.docs as doc}
                    <div class="doc-card">
                      <button class="doc-main" onclick={() => goto(`/app/documents/${doc.id}`)}>
                        <span class="doc-cat-badge" style="background:{docCatMeta[doc.category]?.color ?? 'var(--text3)'}20;color:{docCatMeta[doc.category]?.color ?? 'var(--text3)'}">
                          {docCatMeta[doc.category]?.label ?? doc.category}
                        </span>
                        <div class="doc-info">
                          <span class="doc-name">{doc.summary || doc.file_name || 'Document'}</span>
                          <span class="doc-detail">
                            {#if doc.specialty}<span class="doc-specialty">{doc.specialty}</span>{/if}
                            {doc.file_name || ''}
                          </span>
                        </div>
                        <span class="doc-status" class:parsed={doc.parse_status === 'done'} class:pending={doc.parse_status === 'pending' || doc.parse_status === 'processing'}>
                          {doc.parse_status === 'done' ? 'Processed' : doc.parse_status === 'pending' || doc.parse_status === 'processing' ? 'Processing...' : doc.parse_status}
                        </span>
                      </button>
                      <div class="doc-actions-inline">
                        <button class="doc-action" title="Re-process" onclick={(e) => { e.stopPropagation(); reparseDoc(doc.id); }}>
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
                        </button>
                        <button class="doc-action delete" title="Delete" onclick={(e) => { e.stopPropagation(); deleteDoc(doc.id); }}>
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                        </button>
                      </div>
                    </div>
                  {/each}
                </div>
              </div>
            </div>
          {/each}
        </div>
      {:else}
        <!-- Category view -->
        {#each docCategories as cat}
          <div class="doc-cat-group">
            <div class="doc-cat-header">
              <span class="doc-cat-badge" style="background:{cat.color}20;color:{cat.color}">{cat.label}</span>
              <span class="doc-cat-count">{cat.docs.length}</span>
            </div>
            <div class="doc-list">
              {#each cat.docs as doc}
                <div class="doc-row">
                  <button class="doc-main" onclick={() => goto(`/app/documents/${doc.id}`)}>
                    <div class="doc-info">
                      <span class="doc-name">{doc.summary || doc.file_name || 'Document'}</span>
                      <span class="doc-detail">
                        {formatDate(doc.document_date || doc.created_at)}
                        {#if doc.specialty} · {doc.specialty}{/if}
                      </span>
                    </div>
                    <span class="doc-status" class:parsed={doc.parse_status === 'done'} class:pending={doc.parse_status === 'pending' || doc.parse_status === 'processing'}>
                      {doc.parse_status === 'done' ? 'Processed' : doc.parse_status === 'pending' || doc.parse_status === 'processing' ? 'Processing...' : doc.parse_status}
                    </span>
                  </button>
                  <button class="doc-action" title="Re-process" onclick={(e) => { e.stopPropagation(); reparseDoc(doc.id); }}>
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
                  </button>
                  <button class="doc-action delete" title="Delete" onclick={(e) => { e.stopPropagation(); deleteDoc(doc.id); }}>
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                  </button>
                </div>
              {/each}
            </div>
          </div>
        {/each}
      {/if}
    {/if}

  {:else if tab === 2}
    <!-- Medications -->
    <div class="med-section">
      {#if showAddForm}
        <form class="add-form" onsubmit={(e) => { e.preventDefault(); addMedication(); }}>
          <div class="add-row">
            <input class="add-field name" bind:value={addName} placeholder="Medication name" required />
            <input class="add-field dose" bind:value={addDose} placeholder="Dose" />
            <input class="add-field freq" bind:value={addFrequency} placeholder="Frequency" />
          </div>
          <div class="add-row-bottom">
            <div class="add-date">
              <label>Started</label>
              <input type="date" bind:value={addStartDate} />
            </div>
            <div class="add-actions">
              <button type="button" class="add-cancel" onclick={() => { showAddForm = false; }} disabled={addSaving}>Cancel</button>
              <button type="submit" class="add-save" disabled={addSaving || !addName.trim()}>
                {addSaving ? 'Adding...' : 'Add'}
              </button>
            </div>
          </div>
        </form>
      {:else}
        <button class="add-med-btn" onclick={() => { showAddForm = true; }}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          Add medication
        </button>
      {/if}

      {#if undoMed}
        <div class="undo-toast">
          <span>Stopped {undoMed.name}</span>
          <button onclick={undoStop}>Undo</button>
        </div>
      {/if}

      {#if activeMeds.length === 0 && !showAddForm}
        <div class="empty">
          <p>No active medications.</p>
          <p class="hint">Add medications manually or upload a prescription document.</p>
        </div>
      {:else if activeMeds.length > 0}
        <div class="med-list">
          {#each activeMeds as med}
            <div class="med-row">
              <button class="med-info" onclick={() => askAboutMed(med)}>
                <span class="med-name">{med.name}</span>
                <span class="med-detail">
                  {med.dose}{med.dose && med.frequency ? ' · ' : ''}{med.frequency}
                  {#if med.started_at}
                    <span class="med-since">since {formatDate(med.started_at)}</span>
                  {/if}
                </span>
              </button>
              <button class="stop-btn" onclick={() => stopMedication(med)} title="Stop medication">
                Stop
              </button>
            </div>
          {/each}
        </div>

        <button class="ask-interactions" onclick={askAboutMeds}>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
          Check interactions between all medications
        </button>
      {/if}

      {#if pastMeds.length > 0}
        <button class="history-toggle" onclick={() => { showHistory = !showHistory; }}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
            style="transform: rotate({showHistory ? 90 : 0}deg); transition: transform 0.2s">
            <polyline points="9 18 15 12 9 6"/>
          </svg>
          History ({pastMeds.length})
        </button>
        {#if showHistory}
          <div class="med-list history">
            {#each pastMeds as med}
              <button class="med-row past" onclick={() => askAboutMed(med)}>
                <div class="med-info">
                  <span class="med-name">{med.name}</span>
                  <span class="med-detail">
                    {med.dose}{med.dose && med.frequency ? ' · ' : ''}{med.frequency}
                    <span class="med-period">{formatDateRange(med)}</span>
                  </span>
                </div>
                <svg class="chat-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
              </button>
            {/each}
          </div>
        {/if}
      {/if}
    </div>

  {:else if tab === 3}
    <!-- Questions Backlog -->
    <div class="q-section">
      <div class="q-actions-bar">
        {#if questionList.length > 0 && activeVisits.length > 0}
          <button class="q-select-all" onclick={selectAll}>
            {selectedQuestions.size === questionList.length ? 'Deselect all' : 'Select all'}
          </button>
        {/if}
        <button class="q-add-btn" onclick={() => { showAddQuestion = !showAddQuestion; }}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          Add question
        </button>
      </div>

      {#if showAddQuestion}
        <form class="q-add-form" onsubmit={(e) => { e.preventDefault(); addNewQuestion(); }}>
          <input bind:value={newQuestionText} placeholder="What do you want to ask your doctor?" required />
          <input bind:value={newQuestionRationale} placeholder="Why is this important? (optional)" />
          <div class="q-form-actions">
            <button type="button" class="q-cancel" onclick={() => { showAddQuestion = false; }}>Cancel</button>
            <button type="submit" class="q-submit" disabled={!newQuestionText.trim()}>Add</button>
          </div>
        </form>
      {/if}

      {#if selectedQuestions.size > 0 && activeVisits.length > 0}
        <div class="q-assign-bar">
          <span class="q-assign-count">{selectedQuestions.size} selected</span>
          <select bind:value={assignVisitId} class="q-visit-select">
            <option value="">Assign to appointment...</option>
            {#each activeVisits as v}
              <option value={v.id}>{v.doctor_name || 'Doctor'}{v.visit_date ? ' · ' + formatDate(v.visit_date) : ''}</option>
            {/each}
          </select>
          <button class="q-assign-btn" onclick={assignSelected} disabled={!assignVisitId || assigning}>
            {assigning ? 'Assigning...' : 'Assign'}
          </button>
        </div>
      {/if}

      {#if questionList.length === 0}
        <div class="empty">
          <p>No unlinked questions.</p>
          <p class="hint">Questions added by AI or manually will appear here until assigned to an appointment.</p>
        </div>
      {:else}
        <div class="q-list">
          {#each questionList as q}
            <div class="q-row">
              {#if activeVisits.length > 0}
                <button class="q-checkbox" class:checked={selectedQuestions.has(q.id)} onclick={() => toggleQuestionSelect(q.id)}>
                  {#if selectedQuestions.has(q.id)}
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="var(--blue)" stroke="white" stroke-width="3"><rect x="2" y="2" width="20" height="20" rx="4" /><polyline points="6 12 10 16 18 8" /></svg>
                  {:else}
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--text3)" stroke-width="1.5"><rect x="2" y="2" width="20" height="20" rx="4" /></svg>
                  {/if}
                </button>
              {/if}
              <div class="q-content">
                <div class="q-text">{q.text}</div>
                {#if q.rationale}
                  <div class="q-rationale">{q.rationale}</div>
                {/if}
                <div class="q-meta">
                  <span class="q-source">{q.source === 'agent' ? 'AI suggested' : 'Manual'}</span>
                  {#if q.urgency && q.urgency !== 'routine'}
                    <span class="q-urgency" class:high={q.urgency === 'high'} class:critical={q.urgency === 'critical'}>{q.urgency}</span>
                  {/if}
                </div>
              </div>
              {#if activeVisits.length > 0}
                <div class="q-single-assign">
                  <select class="q-mini-select" onchange={(e) => { const val = (e.target as HTMLSelectElement).value; if (val) assignSingle(q.id, val); }}>
                    <option value="">Assign...</option>
                    {#each activeVisits as v}
                      <option value={v.id}>{v.doctor_name || 'Doctor'}</option>
                    {/each}
                  </select>
                </div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>

{#if showConversation}
  <ConversationSheet contextType={convContextType} contextId={convContextId} contextLabel={conversationLabel} onclose={() => { showConversation = false; }} />
{/if}

<style>
  .page { max-width: 680px; margin: 0 auto; padding: 24px 20px 40px; }
  h1 { font-size: 28px; font-weight: 700; margin-bottom: 16px; }
  .seg-wrap { margin-bottom: 20px; }
  .center { display: flex; justify-content: center; padding: 60px 0; }
  .spinner { width: 32px; height: 32px; border: 3px solid var(--separator); border-top-color: var(--blue); border-radius: 50%; animation: spin 0.8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .empty { text-align: center; padding: 40px 20px; color: var(--text2); }
  .hint { font-size: 13px; color: var(--text3); margin-top: 8px; }

  /* ── Labs Dashboard ── */
  .ld { display: flex; flex-direction: column; gap: 12px; }

  .ld-summary {
    display: flex; gap: 0; padding: 16px 20px;
    background: var(--bg2); border-radius: var(--radius);
  }
  .ld-stat { display: flex; flex-direction: column; gap: 2px; flex: 1; }
  .ld-stat:not(:last-child) { border-right: 1px solid var(--separator); padding-right: 16px; margin-right: 16px; }
  .ld-stat-num { font-size: 20px; font-weight: 700; font-variant-numeric: tabular-nums; }
  .ld-stat-warn .ld-stat-num { color: var(--red); }
  .ld-stat-ok .ld-stat-num { color: var(--green); font-size: 16px; }
  .ld-stat-lbl { font-size: 12px; color: var(--text3); }
  .ld-stat-val { font-size: 14px; font-weight: 500; }

  .ld-controls { display: flex; gap: 10px; align-items: center; }
  .ld-toggle {
    display: flex; background: var(--bg2); border-radius: 10px; padding: 3px; flex-shrink: 0;
  }
  .ld-toggle button {
    all: unset; cursor: pointer; font-size: 13px; font-weight: 500;
    padding: 6px 14px; border-radius: 8px; color: var(--text3); transition: all 0.2s;
  }
  .ld-toggle button.active { background: var(--bg); color: var(--text); box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
  .ld-search-wrap {
    flex: 1; position: relative;
  }
  .ld-search-icon {
    position: absolute; left: 10px; top: 50%; transform: translateY(-50%);
    color: var(--text3); pointer-events: none;
  }
  .ld-search {
    width: 100%; padding: 8px 12px 8px 30px; border-radius: 10px; border: 1px solid var(--separator);
    font-size: 14px; background: var(--bg2); outline: none; color: var(--text); box-sizing: border-box;
  }
  .ld-search:focus { border-color: var(--blue); background: var(--bg); }
  .ld-search::placeholder { color: var(--text3); }

  /* Attention banner */
  .ld-attention {
    background: rgba(255,59,48,0.04); border: 1px solid rgba(255,59,48,0.12);
    border-radius: var(--radius); padding: 14px; display: flex; flex-direction: column; gap: 8px;
  }
  .ld-att-header {
    display: flex; align-items: center; gap: 8px;
    font-size: 12px; font-weight: 700; color: var(--red);
    text-transform: uppercase; letter-spacing: 0.5px;
  }
  .ld-att-row {
    all: unset; cursor: pointer; display: flex; align-items: center; justify-content: space-between;
    padding: 10px 12px; border-radius: 10px; background: var(--bg); transition: background 0.15s;
  }
  .ld-att-row:hover { background: var(--bg2); }
  .ld-att-info { display: flex; flex-direction: column; gap: 1px; }
  .ld-att-name { font-size: 14px; font-weight: 600; }
  .ld-att-cat { font-size: 11px; color: var(--text3); }
  .ld-att-val { display: flex; align-items: baseline; gap: 4px; }
  .ld-att-num { font-size: 16px; font-weight: 700; color: var(--red); font-variant-numeric: tabular-nums; }
  .ld-att-unit { font-size: 11px; color: var(--text3); }
  .ld-att-flag {
    font-size: 10px; font-weight: 700; color: var(--red);
    background: rgba(255,59,48,0.1); padding: 2px 6px; border-radius: 4px; text-transform: uppercase;
  }

  /* Category / date panels */
  .ld-cat, .ld-date { background: var(--bg2); border-radius: var(--radius); overflow: hidden; }
  .ld-cat.has-warn { border-left: 3px solid var(--orange); }
  .ld-cat-head, .ld-date-head {
    all: unset; cursor: pointer; display: flex; align-items: center; justify-content: space-between;
    width: 100%; padding: 14px 16px; transition: background 0.15s; box-sizing: border-box;
  }
  .ld-cat-head:hover, .ld-date-head:hover { background: var(--bg); }
  .ld-cat-left { display: flex; flex-direction: column; gap: 2px; }
  .ld-cat-name { font-size: 15px; font-weight: 600; text-align: left; }
  .ld-cat-count { font-size: 12px; color: var(--text3); }
  .ld-cat-right { display: flex; align-items: center; gap: 8px; }
  .ld-badge { font-size: 11px; font-weight: 600; padding: 3px 8px; border-radius: 6px; }
  .ld-badge.ok { color: var(--green); background: rgba(52,199,89,0.1); }
  .ld-badge.warn { color: var(--orange); background: rgba(255,149,0,0.1); }
  .ld-chev { color: var(--text3); transition: transform 0.2s; }
  .ld-chev.open { transform: rotate(180deg); }

  .ld-cat-body { border-top: 1px solid var(--separator); }
  .ld-row {
    all: unset; cursor: pointer; display: flex; align-items: center; gap: 12px;
    width: 100%; padding: 12px 16px; border-bottom: 1px solid var(--separator);
    transition: background 0.15s; box-sizing: border-box;
  }
  .ld-row:last-child { border-bottom: none; }
  .ld-row:hover { background: var(--bg); }
  .ld-row-info { flex: 1; min-width: 0; }
  .ld-row-name { font-size: 14px; font-weight: 500; display: block; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .ld-row-date { font-size: 11px; color: var(--text3); }
  .ld-row-val { font-size: 15px; font-weight: 600; font-variant-numeric: tabular-nums; white-space: nowrap; }
  .ld-row-val.flagged { color: var(--red); }
  .ld-row-unit { font-size: 11px; font-weight: 400; color: var(--text3); }

  /* ── Timeline view ── */
  .tl { display: flex; flex-direction: column; }
  .tl-entry { display: flex; gap: 0; }
  .tl-rail {
    display: flex; flex-direction: column; align-items: center;
    width: 28px; flex-shrink: 0; padding-top: 18px;
  }
  .tl-dot {
    width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0;
    background: var(--blue); border: 2px solid var(--bg2);
    box-shadow: 0 0 0 2px var(--blue);
  }
  .tl-dot-warn { background: var(--orange); box-shadow: 0 0 0 2px var(--orange); }
  .tl-line { width: 2px; flex: 1; background: var(--separator); margin: 4px 0; }
  .tl-content { flex: 1; min-width: 0; padding-bottom: 8px; }
  .tl-header {
    all: unset; cursor: pointer; display: flex; align-items: center; justify-content: space-between;
    width: 100%; padding: 12px 16px; border-radius: var(--radius);
    background: var(--bg2); transition: background 0.15s; box-sizing: border-box;
  }
  .tl-header:hover { background: var(--bg); }
  .tl-date-info { display: flex; flex-direction: column; gap: 2px; }
  .tl-date { font-size: 15px; font-weight: 600; }
  .tl-count { font-size: 12px; color: var(--text3); }
  .tl-warn-text { color: var(--orange); font-weight: 600; }
  .tl-labs {
    margin-top: 6px; background: var(--bg2); border-radius: var(--radius);
    overflow: hidden; border: 1px solid var(--separator);
  }
  .tl-lab-row {
    all: unset; cursor: pointer; display: flex; align-items: center; gap: 12px;
    width: 100%; padding: 10px 16px; border-bottom: 1px solid var(--separator);
    transition: background 0.15s; box-sizing: border-box;
  }
  .tl-lab-row:last-child { border-bottom: none; }
  .tl-lab-row:hover { background: var(--bg); }
  .tl-lab-info { flex: 1; min-width: 0; }
  .tl-lab-name { font-size: 14px; font-weight: 500; display: block; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .tl-lab-cat { font-size: 11px; color: var(--text3); }
  .tl-lab-val { font-size: 14px; font-weight: 600; font-variant-numeric: tabular-nums; white-space: nowrap; display: flex; align-items: center; gap: 6px; }
  .tl-lab-val.flagged { color: var(--red); }
  .tl-lab-unit { font-size: 11px; font-weight: 400; color: var(--text3); }
  .tl-flag {
    font-size: 9px; font-weight: 700; color: var(--red);
    background: rgba(255,59,48,0.1); padding: 2px 5px; border-radius: 4px;
  }

  /* ── Documents ── */
  .doc-controls { display: flex; justify-content: space-between; align-items: center; margin: 8px 0; }
  .doc-controls-right { display: flex; gap: 8px; }
  .doc-list { background: var(--bg2); border-radius: var(--radius); overflow: hidden; }
  .doc-row { display: flex; align-items: center; border-bottom: 1px solid var(--separator); color: var(--text2); }
  .doc-row:last-child { border-bottom: none; }
  .doc-card {
    background: var(--bg2); border-radius: var(--radius); overflow: hidden;
    border: 1px solid var(--separator); margin-bottom: 6px;
  }
  .doc-card:last-child { margin-bottom: 0; }
  .doc-main {
    all: unset; cursor: pointer; flex: 1; display: flex; align-items: center; gap: 12px;
    padding: 12px 16px; transition: background 0.15s;
  }
  .doc-main:hover { background: var(--bg); }
  .doc-info { flex: 1; min-width: 0; }
  .doc-name { font-size: 14px; font-weight: 500; color: var(--text); display: block; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .doc-detail { font-size: 12px; color: var(--text3); display: block; margin-top: 2px; }
  .doc-specialty { font-weight: 500; color: var(--text2); }
  .doc-status { font-size: 12px; font-weight: 500; padding: 3px 8px; border-radius: 8px; white-space: nowrap; flex-shrink: 0; }
  .doc-status.parsed { color: var(--green); background: rgba(52,199,89,0.1); }
  .doc-status.pending { color: var(--orange); background: rgba(255,149,0,0.1); }
  .doc-actions-inline { display: flex; gap: 2px; padding: 4px 8px; justify-content: flex-end; border-top: 1px solid var(--separator); }
  .doc-cat-badge {
    font-size: 11px; font-weight: 600; padding: 3px 8px; border-radius: 6px;
    white-space: nowrap; flex-shrink: 0;
  }
  .doc-cat-group { margin-bottom: 16px; }
  .doc-cat-header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
  .doc-cat-count { font-size: 12px; color: var(--text3); }
  .dtl-date { font-size: 15px; font-weight: 600; margin-bottom: 8px; }
  .dtl-docs { display: flex; flex-direction: column; gap: 0; }
  .reparse-all-btn {
    all: unset; cursor: pointer; display: flex; align-items: center; gap: 6px;
    font-size: 13px; font-weight: 500; color: var(--blue); padding: 6px 12px;
    border-radius: 8px; transition: background 0.15s;
  }
  .reparse-all-btn:hover { background: rgba(13,148,136,0.1); }
  .doc-action {
    all: unset; cursor: pointer; padding: 6px; border-radius: 6px;
    color: var(--text3); transition: color 0.15s, background 0.15s;
  }
  .doc-action:hover { color: var(--blue); background: rgba(13,148,136,0.08); }
  .doc-action.delete:hover { color: var(--red); background: rgba(255,59,48,0.08); }

  /* ── Medications ── */
  .med-section { display: flex; flex-direction: column; gap: 16px; }
  .add-med-btn {
    all: unset; cursor: pointer; display: flex; align-items: center; gap: 8px;
    padding: 12px 16px; border-radius: 12px; font-size: 14px; font-weight: 500;
    color: var(--blue); border: 1.5px dashed rgba(13,148,136,0.3);
    transition: background 0.15s, border-color 0.15s;
  }
  .add-med-btn:hover { background: rgba(13,148,136,0.06); border-color: var(--blue); }
  .add-form {
    background: var(--bg2); border-radius: var(--radius); padding: 16px;
    display: flex; flex-direction: column; gap: 12px; border: 1px solid var(--separator);
  }
  .add-row { display: flex; gap: 8px; }
  .add-field {
    padding: 10px 12px; border-radius: 10px; border: 1px solid var(--separator);
    font-size: 14px; background: var(--bg); outline: none; transition: border-color 0.2s;
  }
  .add-field:focus { border-color: var(--blue); }
  .add-field.name { flex: 2; }
  .add-field.dose { flex: 1; }
  .add-field.freq { flex: 1; }
  .add-row-bottom { display: flex; justify-content: space-between; align-items: center; }
  .add-date { display: flex; align-items: center; gap: 8px; font-size: 13px; color: var(--text2); }
  .add-date input { padding: 6px 10px; border-radius: 8px; border: 1px solid var(--separator); font-size: 13px; background: var(--bg); }
  .add-actions { display: flex; gap: 8px; }
  .add-cancel, .add-save {
    all: unset; cursor: pointer; font-size: 13px; font-weight: 500;
    padding: 8px 16px; border-radius: 10px; transition: background 0.15s, opacity 0.15s;
  }
  .add-cancel { color: var(--text2); }
  .add-cancel:hover { background: var(--bg); }
  .add-save { color: white; background: var(--blue); }
  .add-save:hover { opacity: 0.9; }
  .add-save:disabled, .add-cancel:disabled { opacity: 0.5; cursor: default; }
  .undo-toast {
    display: flex; align-items: center; justify-content: space-between;
    padding: 10px 16px; border-radius: 10px;
    background: var(--text); color: var(--bg2); font-size: 14px; animation: slideUp 0.25s ease;
  }
  .undo-toast button {
    all: unset; cursor: pointer; font-weight: 600; color: var(--blue);
    padding: 4px 12px; border-radius: 6px; transition: background 0.15s;
  }
  .undo-toast button:hover { background: rgba(255,255,255,0.15); }
  @keyframes slideUp { from { transform: translateY(10px); opacity: 0; } to { transform: translateY(0); opacity: 1; } }
  .med-list { background: var(--bg2); border-radius: var(--radius); overflow: hidden; }
  .med-list.history { opacity: 0.7; }
  .med-row {
    display: flex; align-items: center; gap: 12px; padding: 0;
    border-bottom: 1px solid var(--separator);
  }
  .med-row:last-child { border-bottom: none; }
  .med-row .med-info {
    all: unset; cursor: pointer; flex: 1; padding: 14px 16px;
    display: flex; flex-direction: column; gap: 2px; transition: background 0.15s;
  }
  .med-row .med-info:hover { background: var(--bg); }
  .med-row.past { cursor: pointer; transition: background 0.15s; padding: 14px 16px; }
  .med-row.past:hover { background: var(--bg); }
  .med-row.past .med-info { all: unset; flex: 1; display: flex; flex-direction: column; gap: 2px; }
  .med-name { font-size: 15px; font-weight: 500; }
  .med-detail { font-size: 13px; color: var(--text2); }
  .med-since { color: var(--text3); margin-left: 4px; }
  .med-period { color: var(--text3); margin-left: 4px; }
  .stop-btn {
    all: unset; cursor: pointer; font-size: 12px; font-weight: 600;
    padding: 6px 14px; margin-right: 12px; border-radius: 8px;
    color: var(--red); background: rgba(255,59,48,0.08); transition: background 0.15s;
  }
  .stop-btn:hover { background: rgba(255,59,48,0.15); }
  .chat-icon { color: var(--text3); flex-shrink: 0; opacity: 0; transition: opacity 0.15s; }
  .med-row.past:hover .chat-icon { opacity: 1; color: var(--blue); }
  .history-toggle {
    all: unset; cursor: pointer; display: flex; align-items: center; gap: 8px;
    font-size: 13px; font-weight: 500; color: var(--text3); padding: 4px 0; transition: color 0.15s;
  }
  .history-toggle:hover { color: var(--text2); }
  .ask-interactions {
    all: unset; cursor: pointer; display: flex; align-items: center; gap: 8px;
    padding: 12px 20px; border-radius: 12px; font-size: 14px; font-weight: 500;
    color: var(--blue); background: rgba(13,148,136,0.1); transition: background 0.15s;
  }
  .ask-interactions:hover { background: rgba(13,148,136,0.18); }

  /* ── Section / Category chat buttons ── */
  .ld-cat-header-wrap {
    display: flex; align-items: stretch;
  }
  .ld-cat-header-wrap .ld-cat-head { flex: 1; }
  .cat-chat-btn {
    all: unset; cursor: pointer; display: flex; align-items: center; justify-content: center;
    width: 40px; flex-shrink: 0; color: var(--text3); border-left: 1px solid var(--separator);
    transition: color 0.15s, background 0.15s;
  }
  .cat-chat-btn:hover { color: var(--blue); background: rgba(13,148,136,0.06); }
  .section-chat-btn {
    all: unset; cursor: pointer; display: inline-flex; align-items: center; gap: 4px;
    margin-left: auto; font-size: 11px; font-weight: 600; color: var(--red);
    padding: 3px 10px; border-radius: 6px; background: rgba(255,59,48,0.08);
    transition: background 0.15s;
  }
  .section-chat-btn:hover { background: rgba(255,59,48,0.15); }

  /* ── Questions Backlog ── */
  .q-section { display: flex; flex-direction: column; gap: 12px; }
  .q-actions-bar { display: flex; justify-content: flex-end; align-items: center; gap: 8px; }
  .q-select-all {
    all: unset; cursor: pointer; font-size: 13px; font-weight: 500;
    color: var(--text2); padding: 6px 12px; border-radius: 8px; transition: background 0.15s;
  }
  .q-select-all:hover { background: var(--bg2); }
  .q-add-btn {
    all: unset; cursor: pointer; display: flex; align-items: center; gap: 6px;
    font-size: 14px; font-weight: 500; color: var(--blue); padding: 6px 12px;
    border-radius: 10px; transition: background 0.15s;
  }
  .q-add-btn:hover { background: rgba(13,148,136,0.1); }
  .q-add-form {
    background: var(--bg2); border-radius: var(--radius); padding: 16px;
    display: flex; flex-direction: column; gap: 10px; border: 1px solid var(--separator);
  }
  .q-add-form input {
    padding: 10px 14px; border-radius: 10px; border: 1px solid var(--separator);
    font-size: 14px; outline: none; background: var(--bg);
  }
  .q-add-form input:focus { border-color: var(--blue); }
  .q-form-actions { display: flex; gap: 8px; justify-content: flex-end; }
  .q-cancel {
    all: unset; cursor: pointer; padding: 8px 16px; border-radius: 10px;
    font-size: 14px; color: var(--text2); transition: background 0.15s;
  }
  .q-cancel:hover { background: var(--bg); }
  .q-submit {
    all: unset; cursor: pointer; padding: 8px 20px; border-radius: 10px;
    background: var(--blue); color: white; font-size: 14px; font-weight: 600;
    transition: opacity 0.15s;
  }
  .q-submit:hover { opacity: 0.9; }
  .q-submit:disabled { opacity: 0.4; cursor: default; }
  .q-assign-bar {
    display: flex; align-items: center; gap: 10px;
    padding: 10px 16px; background: rgba(13,148,136,0.06);
    border: 1px solid rgba(13,148,136,0.15); border-radius: var(--radius);
  }
  .q-assign-count { font-size: 13px; font-weight: 600; color: var(--blue); white-space: nowrap; }
  .q-visit-select, .q-mini-select {
    padding: 6px 10px; border-radius: 8px; border: 1px solid var(--separator);
    font-size: 13px; background: var(--bg); color: var(--text); outline: none;
  }
  .q-visit-select { flex: 1; }
  .q-assign-btn {
    all: unset; cursor: pointer; padding: 6px 16px; border-radius: 8px;
    background: var(--blue); color: white; font-size: 13px; font-weight: 600;
    transition: opacity 0.15s; white-space: nowrap;
  }
  .q-assign-btn:hover { opacity: 0.9; }
  .q-assign-btn:disabled { opacity: 0.4; cursor: default; }
  .q-list { background: var(--bg2); border-radius: var(--radius); overflow: hidden; }
  .q-row {
    display: flex; align-items: flex-start; gap: 12px; padding: 14px 16px;
    border-bottom: 1px solid var(--separator);
  }
  .q-row:last-child { border-bottom: none; }
  .q-checkbox { all: unset; cursor: pointer; flex-shrink: 0; margin-top: 2px; }
  .q-content { flex: 1; min-width: 0; }
  .q-text { font-size: 14px; font-weight: 500; line-height: 1.4; }
  .q-rationale { font-size: 12px; color: var(--text3); margin-top: 4px; line-height: 1.3; }
  .q-meta { display: flex; align-items: center; gap: 8px; margin-top: 6px; }
  .q-source { font-size: 11px; color: var(--text3); }
  .q-urgency {
    font-size: 10px; font-weight: 700; padding: 2px 6px; border-radius: 4px; text-transform: uppercase;
  }
  .q-urgency.high { color: var(--orange); background: rgba(255,149,0,0.1); }
  .q-urgency.critical { color: var(--red); background: rgba(255,59,48,0.1); }
  .q-single-assign { flex-shrink: 0; }
  .q-mini-select { max-width: 120px; font-size: 12px; }
</style>

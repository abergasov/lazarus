# Medication Lifecycle & Holistic Agent Context — Design

## Goal

Transform medications from static display-only chips into a time-bounded lifecycle system. Users manage their medications (add, stop, view history). The agent sees the full medication timeline correlated with lab changes — enabling it to reason about cause-and-effect across a person's health journey.

## Architecture

### Medication Lifecycle UI

Records → Medications tab gets three changes:

1. **Add medication** — inline expandable form (name, dose, frequency, start date) at top of list
2. **Active meds as rows** — name, dose, frequency, start date, chat icon, Stop button. Stop sets `ended_at = today`, moves to history
3. **History section** — collapsed by default, shows stopped meds with date range. Tapping opens chat (agent knows you used to take it)

### Backend Changes

- `handleAddMedication` sets `started_at` to provided date or NOW()
- `GET /api/v1/medications?include=all` returns active + inactive
- Frontend Medication type gains `ended_at` field
- `handleDeleteMedication` remains soft-delete (Deactivate)

### Holistic Agent Context

**All medications with timeline** in `buildHealthSummary()`:
```
ACTIVE MEDICATIONS (6):
- Ozempic 0.5mg weekly (since 2025-08-15)

PAST MEDICATIONS (3):
- Metformin 500mg twice daily (2024-03-01 → 2025-06-15)
```

**Medication-Lab Temporal Correlation** — new section computed by finding labs that changed >15% within 3 months of a medication start/stop:
```
MEDICATION-LAB TIMELINE:
- Started Ozempic (2025-08-15): HbA1c 7.2% → 6.1%
- Stopped Metformin (2025-06-15): No significant lab changes
```

**Every conversation context** includes full health summary with medication history and correlations, so the agent can zoom in/out naturally regardless of where the conversation started.

### Design Principles

- One human, one story: agent understands the full timeline
- Medications never truly deleted: soft-delete preserves history
- Temporal awareness: agent sees cause-effect between drugs and lab changes
- Conversations are fluid: start narrow, go wide, or vice versa

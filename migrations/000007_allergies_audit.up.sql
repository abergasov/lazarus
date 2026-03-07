-- Patient allergies
CREATE TABLE IF NOT EXISTS allergies (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL,
    substance   VARCHAR(500) NOT NULL,
    rxcui       VARCHAR(20),
    severity    VARCHAR(20)  NOT NULL DEFAULT 'moderate', -- mild | moderate | severe | life_threatening
    reaction    TEXT,
    reported_at TIMESTAMPTZ  DEFAULT NOW(),
    created_at  TIMESTAMPTZ  DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_allergies_user ON allergies(user_id);

-- Drug-condition contraindications knowledge base
CREATE TABLE IF NOT EXISTS kb_drug_condition_contraindications (
    id          SERIAL       PRIMARY KEY,
    rxcui       VARCHAR(20)  NOT NULL,
    icd10_code  VARCHAR(10)  NOT NULL,
    severity    VARCHAR(20)  NOT NULL DEFAULT 'major', -- absolute | major | moderate
    description TEXT,
    source      VARCHAR(100),
    UNIQUE(rxcui, icd10_code)
);

-- Agent decision audit log
CREATE TABLE IF NOT EXISTS agent_audit_log (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL,
    visit_id    UUID,
    session_id  VARCHAR(100),
    phase       VARCHAR(20)  NOT NULL,
    iteration   INT          NOT NULL,
    tool_name   VARCHAR(100) NOT NULL,
    tool_args   JSONB        NOT NULL DEFAULT '{}',
    tool_result JSONB,
    is_error    BOOLEAN      NOT NULL DEFAULT FALSE,
    duration_ms INT,
    created_at  TIMESTAMPTZ  DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_user ON agent_audit_log(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_visit ON agent_audit_log(visit_id) WHERE visit_id IS NOT NULL;

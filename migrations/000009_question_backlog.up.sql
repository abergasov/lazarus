-- Standalone question backlog — questions exist independently of visits
-- They get linked to a visit when one is created, but they're never lost
CREATE TABLE IF NOT EXISTS question_backlog (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL,
    visit_id    UUID         REFERENCES visits(id),
    text        TEXT         NOT NULL,
    rationale   TEXT         NOT NULL DEFAULT '',
    urgency     VARCHAR(20)  NOT NULL DEFAULT 'routine',
    source      VARCHAR(50)  NOT NULL DEFAULT 'agent',  -- agent | user | system
    asked       BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_question_backlog_user ON question_backlog(user_id, asked, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_question_backlog_visit ON question_backlog(visit_id) WHERE visit_id IS NOT NULL;

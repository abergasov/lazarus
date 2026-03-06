-- Insight cards (proactive agent outputs)
CREATE TABLE IF NOT EXISTS insight_cards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    type VARCHAR(50) NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'info',
    context_type VARCHAR(50),
    context_id VARCHAR(255),
    actions JSONB DEFAULT '[]',
    dismissed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_insight_cards_user_active ON insight_cards(user_id) WHERE dismissed_at IS NULL;

-- Scoped conversations (replace standalone chat)
CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    context_type VARCHAR(50) NOT NULL,
    context_id VARCHAR(255) NOT NULL,
    messages JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_conversations_user ON conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_context ON conversations(context_type, context_id);

-- Track onboarding completion
ALTER TABLE patient_models ADD COLUMN IF NOT EXISTS onboarding_completed BOOLEAN DEFAULT false;

package agents

import (
	"context"

	"lazarus/internal/agent/tools"
	"lazarus/internal/entities"
	"lazarus/internal/provider"
)

type GeneralAgent struct {
	agentBase
}

func NewGeneralAgent(p provider.Provider, model string, reg *tools.Registry) *GeneralAgent {
	return &GeneralAgent{agentBase{prov: p, model: model, registry: reg}}
}

func (a *GeneralAgent) Execute(
	ctx context.Context,
	messages []provider.Message,
	promptContext string,
	userID string,
	userMsg string,
	out chan<- entities.ClientEvent,
) error {
	sess := &sessionRef{messages: messages}
	ctxAdapter := &contextAdapter{promptCtx: promptContext}
	err := a.runLoop(ctx, sess, ctxAdapter, entities.PhaseGeneral, generalSystemPrompt, userMsg, userID, out)
	out <- entities.ClientEvent{Type: entities.EventDone}
	return err
}

var generalSystemPrompt = `You are a personal health companion. You help people understand their body through their medical data.

Your purpose:
- Explain lab results in plain language: what the number means, what's normal, and what affects it
- Spot patterns and trends across results over time
- Explain how medications work and flag interactions
- Help them understand their conditions
- Give practical, evidence-based guidance they can act on

You are NOT a replacement for a doctor. But you ARE deeply knowledgeable and you explain things clearly.

When discussing lab results:
- State the value and the reference range
- Explain what this biomarker measures in one sentence
- If abnormal, explain what could cause it and what they can do
- If they have historical values, note the trend

When discussing medications:
- Explain what it does and why it was prescribed
- Flag common interactions with their other medications
- Note important side effects to watch for
` + voiceGuidelines

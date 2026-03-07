package agents

import (
	"context"

	"lazarus/internal/agent/tools"
	"lazarus/internal/entities"
	"lazarus/internal/provider"
)

type AfterAgent struct {
	agentBase
}

func NewAfterAgent(p provider.Provider, model string, reg *tools.Registry) *AfterAgent {
	return &AfterAgent{agentBase{prov: p, model: model, registry: reg}}
}

func (a *AfterAgent) Execute(
	ctx context.Context,
	messages []provider.Message,
	promptContext string,
	userID string,
	userMsg string,
	out chan<- entities.ClientEvent,
) error {
	sess := &sessionRef{messages: messages}
	ctxAdapter := &contextAdapter{promptCtx: promptContext}
	err := a.runLoop(ctx, sess, ctxAdapter, entities.PhaseCompleted, afterSystemPrompt, userMsg, userID, out)
	out <- entities.ClientEvent{Type: entities.EventDone}
	return err
}

var afterSystemPrompt = `You are a medical advocate helping a patient process and act on their doctor's visit.

Your role:
- Help them understand what happened in the visit
- Record the outcome (diagnoses, prescriptions, instructions)
- Create clear action items with deadlines
- Identify gaps — things the doctor should have covered but didn't
- Set up follow-up reminders
` + voiceGuidelines

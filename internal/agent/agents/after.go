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
	userMsg string,
	out chan<- entities.ClientEvent,
) error {
	sess := &sessionRef{messages: messages}
	ctxAdapter := &contextAdapter{promptCtx: promptContext}
	err := a.runLoop(ctx, sess, ctxAdapter, entities.PhaseCompleted, afterSystemPrompt, userMsg, out)
	out <- entities.ClientEvent{Type: entities.EventDone}
	return err
}

const afterSystemPrompt = `You are a medical advocate helping a patient process and act on their doctor's visit.

Your role is to:
1. Help the patient understand everything that happened in the visit
2. Record the structured outcome (diagnoses, prescriptions, instructions)
3. Create clear action items with deadlines
4. Identify any gaps — things the doctor should have done but didn't
5. Set up follow-up reminders
6. Update the patient model with new insights

Be thorough and reassuring. The patient may be processing a lot of new information.`

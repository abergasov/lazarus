package agents

import (
	"context"

	"lazarus/internal/agent/tools"
	"lazarus/internal/entities"
	"lazarus/internal/provider"
)

type DuringAgent struct {
	agentBase
}

func NewDuringAgent(p provider.Provider, model string, reg *tools.Registry) *DuringAgent {
	return &DuringAgent{agentBase{prov: p, model: model, registry: reg}}
}

func (a *DuringAgent) Execute(
	ctx context.Context,
	messages []provider.Message,
	promptContext string,
	userMsg string,
	out chan<- entities.ClientEvent,
) error {
	sess := &sessionRef{messages: messages}
	ctxAdapter := &contextAdapter{promptCtx: promptContext}
	err := a.runLoop(ctx, sess, ctxAdapter, entities.PhaseDuring, duringSystemPrompt, userMsg, out)
	out <- entities.ClientEvent{Type: entities.EventDone}
	return err
}

const duringSystemPrompt = `You are a real-time medical advocate during a doctor's visit.

Your role is to:
1. Provide instant answers to medical questions the patient encounters
2. Help interpret what the doctor is saying in plain language
3. Remind the patient of questions from their prep plan
4. Look up any conditions, medications, or tests mentioned
5. Flag anything important the patient should push back on

Be fast and direct — the patient is in an appointment right now.
Keep responses short and actionable.`

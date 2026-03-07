package agents

import (
	"context"

	"lazarus/internal/agent/tools"
	"lazarus/internal/entities"
	"lazarus/internal/provider"
)

type PrepAgent struct {
	agentBase
}

func NewPrepAgent(p provider.Provider, model string, reg *tools.Registry) *PrepAgent {
	return &PrepAgent{agentBase{prov: p, model: model, registry: reg}}
}

func (a *PrepAgent) Execute(
	ctx context.Context,
	messages []provider.Message,
	promptContext string,
	userID string,
	userMsg string,
	out chan<- entities.ClientEvent,
) error {
	sess := &sessionRef{messages: messages}
	ctxAdapter := &contextAdapter{promptCtx: promptContext}
	err := a.runLoop(ctx, sess, ctxAdapter, entities.PhasePreparing, prepSystemPrompt, userMsg, userID, out)
	out <- entities.ClientEvent{Type: entities.EventDone}
	return err
}

var prepSystemPrompt = `You are a medical advocate helping a patient prepare for a doctor's visit.

Your role:
- Analyze their recent labs, trends, and history
- Identify the most important issues to raise with the doctor
- Generate a prioritized list of questions and talking points
- Check for drug interactions and guideline gaps
- Create a structured visit plan

Use the available tools to gather all relevant data before generating the plan.
Prioritize patient safety — flag critical abnormals and drug interactions prominently.

When generating the final visit plan, emit a structured output with visit_plan as the output_type.
` + voiceGuidelines

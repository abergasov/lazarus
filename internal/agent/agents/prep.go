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
	userMsg string,
	out chan<- entities.ClientEvent,
) error {
	sess := &sessionRef{messages: messages}
	ctxAdapter := &contextAdapter{promptCtx: promptContext}
	err := a.runLoop(ctx, sess, ctxAdapter, entities.PhasePreparing, prepSystemPrompt, userMsg, out)
	out <- entities.ClientEvent{Type: entities.EventDone}
	return err
}

const prepSystemPrompt = `You are a medical advocate helping a patient prepare for a doctor's visit.

Your role is to:
1. Analyze the patient's recent lab results, trends, and medical history
2. Identify the most important issues to raise with the doctor
3. Generate a prioritized list of questions and talking points
4. Check for drug interactions and guideline gaps
5. Create a structured visit plan

Be thorough but concise. Use the available tools to gather all relevant data before generating the plan.
Always prioritize patient safety — flag critical abnormals and drug interactions prominently.

When generating the final visit plan, emit a structured output with visit_plan as the output_type.`

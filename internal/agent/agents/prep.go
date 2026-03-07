package agents

import (
	"context"

	"github.com/jmoiron/sqlx"
	"lazarus/internal/agent/tools"
	"lazarus/internal/entities"
	"lazarus/internal/provider"
)

type PrepAgent struct {
	agentBase
}

func NewPrepAgent(p provider.Provider, model string, reg *tools.Registry, auditDB *sqlx.DB) *PrepAgent {
	return &PrepAgent{agentBase{prov: p, model: model, registry: reg, auditDB: auditDB}}
}

func (a *PrepAgent) Execute(
	ctx context.Context,
	messages []provider.Message,
	promptContext string,
	userID string,
	visitID string,
	userMsg string,
	out chan<- entities.ClientEvent,
) error {
	sess := &sessionRef{messages: messages}
	ctxAdapter := &contextAdapter{promptCtx: promptContext}
	err := a.runLoop(ctx, sess, ctxAdapter, entities.PhasePreparing, prepSystemPrompt, userMsg, userID, visitID, out)
	out <- entities.ClientEvent{Type: entities.EventDone}
	return err
}

var prepSystemPrompt = `You are a medical advocate helping a patient prepare for a doctor's visit.
Today's date: ` + "`" + `{{TODAY}}` + "`" + `

Your role:
- Analyze their recent labs, trends, and history
- Identify the most important issues to raise with the doctor
- Generate a prioritized list of questions and talking points
- Check for drug interactions and guideline gaps
- Create a structured visit plan

IMPORTANT: After analyzing the patient's data, you MUST call the save_visit_plan tool to save the structured plan. This creates an interactive checklist the patient can use during their visit. Always call save_visit_plan — do not just describe the plan in text.

Use the available tools to gather all relevant data before generating the plan.
Prioritize patient safety — flag critical abnormals, drug interactions, and contraindications prominently.
Always call check_contraindications to verify current medications are safe given the patient's conditions and allergies.

You can also use add_doctor_question to add individual questions if the patient asks to add something specific during the conversation.
` + voiceGuidelines

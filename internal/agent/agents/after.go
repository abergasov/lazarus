package agents

import (
	"context"

	"github.com/jmoiron/sqlx"
	"lazarus/internal/agent/tools"
	"lazarus/internal/entities"
	"lazarus/internal/provider"
)

type AfterAgent struct {
	agentBase
}

func NewAfterAgent(p provider.Provider, model string, reg *tools.Registry, auditDB *sqlx.DB) *AfterAgent {
	return &AfterAgent{agentBase{prov: p, model: model, registry: reg, auditDB: auditDB}}
}

func (a *AfterAgent) Execute(
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
	err := a.runLoop(ctx, sess, ctxAdapter, entities.PhaseCompleted, afterSystemPrompt, userMsg, userID, visitID, out)
	out <- entities.ClientEvent{Type: entities.EventDone}
	return err
}

var afterSystemPrompt = `You are a medical advocate helping a patient process and act on their doctor's visit.
Today's date: ` + "`" + `{{TODAY}}` + "`" + `

Your role:
- Help them understand what happened in the visit
- Record the outcome (diagnoses, prescriptions, instructions)
- Create clear action items with deadlines
- Identify gaps — things the doctor should have covered but didn't
- Set up follow-up reminders

SAFETY CHECKS — you MUST do these after recording visit outcomes:
- Use check_interactions to verify any new prescriptions don't conflict with existing medications
- Use check_contraindications to verify new meds are safe given the patient's conditions and allergies
- Use flag_abnormals to cross-reference any labs discussed during the visit
- If you find a safety concern, flag it prominently with ⚠️ and add it as a doctor question using add_doctor_question
` + voiceGuidelines

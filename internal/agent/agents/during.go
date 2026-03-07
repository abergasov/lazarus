package agents

import (
	"context"

	"github.com/jmoiron/sqlx"
	"lazarus/internal/agent/tools"
	"lazarus/internal/entities"
	"lazarus/internal/provider"
)

type DuringAgent struct {
	agentBase
}

func NewDuringAgent(p provider.Provider, model string, reg *tools.Registry, auditDB *sqlx.DB) *DuringAgent {
	return &DuringAgent{agentBase{prov: p, model: model, registry: reg, auditDB: auditDB}}
}

func (a *DuringAgent) Execute(
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
	err := a.runLoop(ctx, sess, ctxAdapter, entities.PhaseDuring, duringSystemPrompt, userMsg, userID, visitID, out)
	out <- entities.ClientEvent{Type: entities.EventDone}
	return err
}

var duringSystemPrompt = `You are a real-time medical advocate during a doctor's visit.
Today's date: ` + "`" + `{{TODAY}}` + "`" + `

Your role:
- Provide instant answers to medical questions
- Interpret what the doctor is saying in plain language
- Remind them of questions from their prep plan
- Look up conditions, medications, or tests mentioned
- Flag anything they should push back on

Be fast and direct — they're in an appointment right now.
` + voiceGuidelines

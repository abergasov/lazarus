package agent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"lazarus/internal/agent/agents"
	"lazarus/internal/agent/tools"
	"lazarus/internal/entities"
	"lazarus/internal/provider"
)

type Orchestrator struct {
	assembler *Assembler
	providers *provider.Registry
	toolReg   *tools.Registry
	sessions  *SessionStore
	patients  *PatientModelStore
}

func NewOrchestrator(
	assembler *Assembler,
	providers *provider.Registry,
	toolReg *tools.Registry,
	sessions *SessionStore,
	patients *PatientModelStore,
) *Orchestrator {
	return &Orchestrator{
		assembler: assembler,
		providers: providers,
		toolReg:   toolReg,
		sessions:  sessions,
		patients:  patients,
	}
}

func (o *Orchestrator) Run(ctx context.Context, sess *Session, userMsg string) (<-chan entities.ClientEvent, error) {
	out := make(chan entities.ClientEvent, 64)

	go func() {
		defer close(out)

		// 1. Assemble context concurrently
		ac, err := o.assembler.Build(ctx, sess)
		if err != nil {
			out <- entities.ClientEvent{Type: entities.EventError, Payload: err.Error()}
			return
		}

		// 2. Route to sub-agent
		var runErr error
		switch sess.Phase {
		case entities.PhasePreparing:
			p, model, err := o.providers.ForRole("prep")
			if err != nil {
				out <- entities.ClientEvent{Type: entities.EventError, Payload: err.Error()}
				return
			}
			a := agents.NewPrepAgent(p, model, o.toolReg)
			runErr = a.Execute(ctx, sess.Messages, ac.SystemPromptContext, userMsg, out)

		case entities.PhaseDuring:
			p, model, err := o.providers.ForRole("during")
			if err != nil {
				out <- entities.ClientEvent{Type: entities.EventError, Payload: err.Error()}
				return
			}
			a := agents.NewDuringAgent(p, model, o.toolReg)
			runErr = a.Execute(ctx, sess.Messages, ac.SystemPromptContext, userMsg, out)

		case entities.PhaseCompleted:
			p, model, err := o.providers.ForRole("after")
			if err != nil {
				out <- entities.ClientEvent{Type: entities.EventError, Payload: err.Error()}
				return
			}
			a := agents.NewAfterAgent(p, model, o.toolReg)
			runErr = a.Execute(ctx, sess.Messages, ac.SystemPromptContext, userMsg, out)

		default:
			out <- entities.ClientEvent{Type: entities.EventError, Payload: fmt.Sprintf("unknown phase: %s", sess.Phase)}
			return
		}

		if runErr != nil {
			out <- entities.ClientEvent{Type: entities.EventError, Payload: runErr.Error()}
		}

		// 3. Persist session
		if o.sessions.db != nil {
			_ = o.sessions.Save(ctx, sess)
		}
	}()

	return out, nil
}

// ProactivePrepare runs the prep agent proactively for an upcoming visit
func (o *Orchestrator) ProactivePrepare(ctx context.Context, visit *entities.Visit) {
	if o.sessions.db == nil {
		return
	}
	sess, err := o.sessions.GetOrCreate(ctx, visit.UserID, visit.ID.String(), entities.PhasePreparing)
	if err != nil {
		return
	}
	ch, err := o.Run(ctx, sess, "Proactively build my visit plan for the upcoming appointment.")
	if err != nil {
		return
	}
	for range ch {
	}
}

// GetOrCreateSession is a convenience method for routes
func (o *Orchestrator) GetOrCreateSession(ctx context.Context, userID uuid.UUID, visitIDStr string) (*Session, error) {
	// Determine phase from visit status
	phase := entities.PhasePreparing
	if o.sessions.db != nil && visitIDStr != "" {
		visitRepo := o.assembler.visitRepo
		if visitRepo != nil {
			if v, err := visitRepo.Get(ctx, visitIDStr); err == nil {
				phase = v.Status
			}
		}
	}
	return o.sessions.GetOrCreate(ctx, userID, visitIDStr, phase)
}

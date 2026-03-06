package agent

import (
	"context"
	"time"

	"lazarus/internal/entities"
	"lazarus/internal/repository"
	"lazarus/internal/service/push"
)

type Scheduler struct {
	orchestrator *Orchestrator
	visitRepo    *repository.VisitRepo
	push         *push.Service
	interval     time.Duration
}

func NewScheduler(orch *Orchestrator, visitRepo *repository.VisitRepo, push *push.Service) *Scheduler {
	return &Scheduler{orchestrator: orch, visitRepo: visitRepo, push: push, interval: time.Hour}
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	visits, err := s.visitRepo.FindUpcomingUnprepared(ctx, 24*time.Hour)
	if err != nil {
		return
	}
	for _, v := range visits {
		go func(visit entities.Visit) {
			s.orchestrator.ProactivePrepare(ctx, &visit)
			s.push.Send(ctx, visit.UserID, "Your visit plan for tomorrow is ready.")
		}(v)
	}
}

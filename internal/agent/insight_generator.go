package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"lazarus/internal/entities"
	"lazarus/internal/repository"
)

const (
	ChangeDocumentUploaded = "document_uploaded"
	ChangeProfileUpdated   = "profile_updated"
	ChangeVisitPhase       = "visit_phase_changed"
	ChangeLabsAdded        = "labs_added"

	maxProcessingRetries = 3
)

type InsightGenerator struct {
	db *sqlx.DB
}

func NewInsightGenerator(db *sqlx.DB) *InsightGenerator {
	return &InsightGenerator{db: db}
}

// ProcessDataChange is called after any data mutation. It generates insight cards
// based on the change type. This is the proactive push — agents don't wait for questions.
func (g *InsightGenerator) ProcessDataChange(ctx context.Context, userID uuid.UUID, changeType string, contextID string) {
	if g.db == nil {
		return
	}

	icRepo := repository.NewInsightCardRepo(g.db)

	switch changeType {
	case ChangeDocumentUploaded:
		g.processDocumentUpload(ctx, userID, contextID, icRepo)
	case ChangeProfileUpdated:
		g.processProfileUpdate(ctx, userID, icRepo)
	case ChangeVisitPhase:
		g.processVisitPhase(ctx, userID, contextID, icRepo)
	case ChangeLabsAdded:
		g.processNewLabs(ctx, userID, icRepo)
	}
}

func (g *InsightGenerator) processDocumentUpload(ctx context.Context, userID uuid.UUID, docID string, repo *repository.InsightCardRepo) {
	// Only create one "document processed" card per user — skip if one already exists undismissed
	existing, _ := repo.ListActive(ctx, userID)
	for _, c := range existing {
		if c.Type == entities.InsightDocProcessed && c.DismissedAt == nil {
			return // already have an active document insight
		}
	}

	card := &entities.InsightCard{
		UserID:      userID,
		Type:        entities.InsightDocProcessed,
		Title:       "Documents processed",
		Body:        "We've analyzed your uploaded documents. Check your Records for extracted lab results and medications.",
		Severity:    entities.SeverityInfo,
		ContextType: "document",
		ContextID:   docID,
		Actions: []entities.Action{
			{Label: "View Records", Endpoint: "/records", Method: "GET"},
		},
	}
	if err := repo.Create(ctx, card); err != nil {
		slog.Error("failed to create insight card", "error", err)
	}
}

func (g *InsightGenerator) processProfileUpdate(ctx context.Context, userID uuid.UUID, repo *repository.InsightCardRepo) {
	// Check if risk scores changed significantly
	pmRepo := repository.NewPatientModelRepo(g.db)
	model, err := pmRepo.Load(ctx, userID)
	if err != nil || model == nil {
		return
	}

	// Generate risk insight if ASCVD is computed
	if model.RiskScores.ASCVD10Year != nil && model.RiskScores.ASCVD10Year.ActionNeeded {
		card := &entities.InsightCard{
			UserID:      userID,
			Type:        entities.InsightRiskChange,
			Title:       "Elevated cardiovascular risk",
			Body:        "Based on your updated profile, your 10-year ASCVD risk requires attention. Consider discussing with your doctor.",
			Severity:    entities.SeverityWarning,
			ContextType: "profile",
			ContextID:   "ascvd",
			Actions: []entities.Action{
				{Label: "Learn more", Endpoint: "/profile", Method: "GET"},
			},
		}
		if err := repo.Create(ctx, card); err != nil {
			slog.Error("failed to create risk insight", "error", err)
		}
	}
}

func (g *InsightGenerator) processVisitPhase(ctx context.Context, userID uuid.UUID, visitID string, repo *repository.InsightCardRepo) {
	visitRepo := repository.NewVisitRepo(g.db)
	visit, err := visitRepo.Get(ctx, visitID)
	if err != nil {
		return
	}

	switch visit.Status {
	case entities.VisitStatusCompleted:
		card := &entities.InsightCard{
			UserID:      userID,
			Type:        entities.InsightVisitPrep,
			Title:       "Visit completed",
			Body:        "Your visit with " + derefStr(visit.DoctorName) + " has been recorded. Review the summary and action items.",
			Severity:    entities.SeverityInfo,
			ContextType: "visit",
			ContextID:   visitID,
			Actions: []entities.Action{
				{Label: "View Summary", Endpoint: "/visits/" + visitID, Method: "GET"},
			},
		}
		if err := repo.Create(ctx, card); err != nil {
			slog.Error("failed to create visit complete insight", "error", err)
		}

	case entities.VisitStatusPreparing:
		card := &entities.InsightCard{
			UserID:      userID,
			Type:        entities.InsightVisitPrep,
			Title:       "Preparing for your visit",
			Body:        "Your appointment with " + derefStr(visit.DoctorName) + " is being prepared. AI is generating questions based on your health profile.",
			Severity:    entities.SeverityInfo,
			ContextType: "visit",
			ContextID:   visitID,
			Actions: []entities.Action{
				{Label: "View Prep", Endpoint: "/visits/" + visitID, Method: "GET"},
			},
		}
		if err := repo.Create(ctx, card); err != nil {
			slog.Error("failed to create visit prep insight", "error", err)
		}
	}
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (g *InsightGenerator) processNewLabs(ctx context.Context, userID uuid.UUID, repo *repository.InsightCardRepo) {
	// Check for abnormal lab results
	labRepo := repository.NewLabRepo(g.db)
	labs, err := labRepo.ListByUser(ctx, userID)
	if err != nil || len(labs) == 0 {
		return
	}

	abnormalCount := 0
	for _, l := range labs {
		if l.Flag != "normal" && l.Flag != "" {
			abnormalCount++
		}
	}

	if abnormalCount > 0 {
		body := fmt.Sprintf("%d of your lab results are outside normal ranges. Tap to understand what they mean and what you can do.", abnormalCount)
		card := &entities.InsightCard{
			UserID:   userID,
			Type:     entities.InsightLabTrend,
			Title:    "Lab results need your attention",
			Body:     body,
			Severity: entities.SeverityWarning,
			Actions: []entities.Action{
				{Label: "Review Labs", Endpoint: "/records", Method: "GET"},
			},
		}
		if err := repo.Create(ctx, card); err != nil {
			slog.Error("failed to create lab insight", "error", err)
		}
	}
}

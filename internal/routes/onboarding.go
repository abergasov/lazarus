package routes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"lazarus/internal/agent"
	"lazarus/internal/entities"
	"lazarus/internal/repository"
)

func (s *Server) handleOnboardingUpload(c *fiber.Ctx, userID uuid.UUID) error {
	if s.docSvc == nil {
		return c.Status(503).JSON(fiber.Map{"error": "document service not configured"})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "file required"})
	}

	// Upload document
	doc, err := s.docSvc.Upload(c.Context(), userID, "", file, "onboarding")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Stream processing steps via SSE
	writer := agent.NewStreamWriter(c)
	c.Status(200)

	// Step 1: Document uploaded
	writer.Write(entities.ClientEvent{
		Type:    "processing_step",
		Payload: map[string]string{"step": "upload", "status": "done", "label": "Document uploaded"},
	})
	writer.Flush()

	// Step 2: Parse document (async, but we wait for initial extraction)
	writer.Write(entities.ClientEvent{
		Type:    "processing_step",
		Payload: map[string]string{"step": "parse", "status": "running", "label": "Reading your document..."},
	})
	writer.Flush()

	go s.docSvc.Parse(context.Background(), doc.ID)

	writer.Write(entities.ClientEvent{
		Type:    "processing_step",
		Payload: map[string]string{"step": "parse", "status": "done", "label": "Document parsed"},
	})
	writer.Flush()

	// Step 3: Extract profile
	writer.Write(entities.ClientEvent{
		Type:    "processing_step",
		Payload: map[string]string{"step": "extract", "status": "running", "label": "Extracting your health profile..."},
	})
	writer.Flush()

	// Load or create patient model
	pmRepo := repository.NewPatientModelRepo(s.db)
	model, _ := pmRepo.Load(c.Context(), userID)
	if model == nil {
		model = &entities.PatientModel{UserID: userID}
	}

	// If orchestrator is available, run extraction agent
	if s.orchestrator != nil {
		sess, err := s.orchestrator.GetOrCreateSession(c.Context(), userID, "")
		if err == nil {
			extractPrompt := "Extract demographics, conditions, and medications from the uploaded medical document. Return structured data."
			eventCh, err := s.orchestrator.Run(c.Context(), sess, extractPrompt)
			if err == nil {
				for ev := range eventCh {
					// Forward relevant events
					if ev.Type == entities.EventTextDelta || ev.Type == entities.EventStructured {
						writer.Write(ev)
						writer.Flush()
					}
				}
			}
		}
	}

	writer.Write(entities.ClientEvent{
		Type:    "processing_step",
		Payload: map[string]string{"step": "extract", "status": "done", "label": "Profile extracted"},
	})
	writer.Flush()

	// Return the model for confirmation
	modelJSON, _ := json.Marshal(model)
	writer.Write(entities.ClientEvent{
		Type:    "profile_extracted",
		Payload: json.RawMessage(modelJSON),
	})
	writer.Flush()

	writer.Write(entities.ClientEvent{
		Type: entities.EventDone,
		Payload: entities.DonePayload{},
	})
	writer.Flush()

	return nil
}

func (s *Server) handleOnboardingConfirm(c *fiber.Ctx, userID uuid.UUID) error {
	pmRepo := repository.NewPatientModelRepo(s.db)
	model, err := pmRepo.Load(c.Context(), userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if model == nil {
		model = &entities.PatientModel{UserID: userID}
	}

	// Parse body if provided (user corrections)
	if len(c.Body()) > 2 { // not empty / "{}"
		if err := c.BodyParser(&model.Demographics); err != nil {
			// Try parsing as full model
			_ = json.Unmarshal(c.Body(), model)
		}
	}

	model.OnboardingCompleted = true
	if err := pmRepo.Save(c.Context(), model); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Create welcome insight card
	icRepo := repository.NewInsightCardRepo(s.db)
	welcome := &entities.InsightCard{
		UserID:   userID,
		Type:     entities.InsightWelcome,
		Title:    "Welcome to MedHelp",
		Body:     fmt.Sprintf("Your health profile is set up. Upload more documents to get personalized insights."),
		Severity: entities.SeverityInfo,
		Actions:  []entities.Action{},
	}
	_ = icRepo.Create(c.Context(), welcome)

	return c.JSON(model)
}

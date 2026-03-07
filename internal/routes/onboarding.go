package routes

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"

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

	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "multipart form required"})
	}

	files := form.File["files"]
	if len(files) == 0 {
		files = form.File["file"]
	}
	if len(files) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "at least one file required"})
	}

	// Capture values needed inside the stream callback
	fiberCtx := c.Context()
	reqCtx := context.Background() // can't use c.Context() inside stream callback

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")
	c.Status(200)

	// Copy file headers since multipart form may be freed after handler returns
	filesCopy := make([]*multipart.FileHeader, len(files))
	copy(filesCopy, files)

	fiberCtx.SetBodyStreamWriter(func(w *bufio.Writer) {
		writer := agent.NewBufStreamWriter(w)
		totalFiles := len(filesCopy)

		writer.Write(entities.ClientEvent{
			Type:    "processing_step",
			Payload: map[string]string{"step": "upload", "status": "running", "label": fmt.Sprintf("Uploading %d document(s)...", totalFiles)},
		})

		for i, file := range filesCopy {
			doc, err := s.docSvc.Upload(reqCtx, userID, "", file, "onboarding")
			if err != nil {
				writer.Write(entities.ClientEvent{
					Type:    "processing_step",
					Payload: map[string]string{"step": "upload", "status": "error", "label": fmt.Sprintf("Failed to upload %s: %s", file.Filename, err.Error())},
				})
				continue
			}

			go s.docSvc.Parse(context.Background(), doc.ID)

			writer.Write(entities.ClientEvent{
				Type:    "processing_step",
				Payload: map[string]string{"step": "upload", "status": "running", "label": fmt.Sprintf("Uploaded %d/%d: %s", i+1, totalFiles, file.Filename)},
			})
		}

		writer.Write(entities.ClientEvent{
			Type:    "processing_step",
			Payload: map[string]string{"step": "upload", "status": "done", "label": fmt.Sprintf("%d document(s) uploaded", totalFiles)},
		})

		writer.Write(entities.ClientEvent{
			Type:    "processing_step",
			Payload: map[string]string{"step": "parse", "status": "running", "label": "Reading your documents..."},
		})

		writer.Write(entities.ClientEvent{
			Type:    "processing_step",
			Payload: map[string]string{"step": "parse", "status": "done", "label": "Documents parsed"},
		})

		writer.Write(entities.ClientEvent{
			Type:    "processing_step",
			Payload: map[string]string{"step": "extract", "status": "running", "label": "Extracting your health profile..."},
		})

		// Load or create patient model
		pmRepo := repository.NewPatientModelRepo(s.db)
		model, _ := pmRepo.Load(reqCtx, userID)
		if model == nil {
			model = &entities.PatientModel{UserID: userID}
		}

		// If orchestrator is available, run extraction agent
		if s.orchestrator != nil {
			sess, err := s.orchestrator.GetOrCreateSession(reqCtx, userID, "")
			if err == nil {
				extractPrompt := "Extract demographics, conditions, and medications from the uploaded medical documents. Return structured data."
				eventCh, err := s.orchestrator.Run(reqCtx, sess, extractPrompt)
				if err == nil {
					for ev := range eventCh {
						if ev.Type == entities.EventTextDelta || ev.Type == entities.EventStructured {
							writer.Write(ev)
						}
					}
				}
			}
		}

		writer.Write(entities.ClientEvent{
			Type:    "processing_step",
			Payload: map[string]string{"step": "extract", "status": "done", "label": "Profile extracted"},
		})

		modelJSON, _ := json.Marshal(model)
		writer.Write(entities.ClientEvent{
			Type:    "profile_extracted",
			Payload: json.RawMessage(modelJSON),
		})

		writer.Write(entities.ClientEvent{
			Type:    entities.EventDone,
			Payload: entities.DonePayload{},
		})
	})

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

	// Count documents to personalize the message
	docRepo := repository.NewDocumentRepo(s.db)
	docs, _ := docRepo.ListByUser(c.Context(), userID)
	labRepo := repository.NewLabRepo(s.db)
	labs, _ := labRepo.ListByUser(c.Context(), userID)

	parsedCount := 0
	for _, d := range docs {
		if d.ParseStatus == entities.ParseStatusDone {
			parsedCount++
		}
	}

	var welcomeBody string
	if parsedCount > 0 && len(labs) > 0 {
		welcomeBody = fmt.Sprintf("We've processed %d documents and found %d lab results. Check your Records to review them, or schedule an appointment to get AI-powered visit preparation.", parsedCount, len(labs))
	} else if len(docs) > 0 {
		welcomeBody = fmt.Sprintf("We've uploaded %d documents. They're still being processed — check back soon, or schedule an appointment to get started.", len(docs))
	} else {
		welcomeBody = "Upload your medical documents to get started. MedHelp will extract lab results, medications, and help you prepare for doctor visits."
	}

	welcomeActions := []entities.Action{
		{Label: "View Records", Endpoint: "/records", Method: "GET"},
	}
	welcome := &entities.InsightCard{
		UserID:   userID,
		Type:     entities.InsightWelcome,
		Title:    "Welcome to MedHelp",
		Body:     welcomeBody,
		Severity: entities.SeverityInfo,
		Actions:  welcomeActions,
	}
	_ = icRepo.Create(c.Context(), welcome)

	return c.JSON(model)
}

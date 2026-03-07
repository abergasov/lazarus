package routes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"lazarus/internal/agent"
	"lazarus/internal/entities"
	"lazarus/internal/provider"
	"lazarus/internal/repository"
)

type CreateConversationRequest struct {
	ContextType string `json:"context_type"`
	ContextID   string `json:"context_id"`
}

type ConversationMessageRequest struct {
	Content string `json:"content"`
}

func (s *Server) handleCreateConversation(c *fiber.Ctx, userID uuid.UUID) error {
	var req CreateConversationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if req.ContextType == "" || req.ContextID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "context_type and context_id are required"})
	}

	repo := repository.NewConversationRepo(s.db)

	// Check if conversation already exists for this context
	existing, err := repo.GetByContext(c.Context(), userID, req.ContextType, req.ContextID)
	if err == nil && existing != nil {
		return c.JSON(existing)
	}

	conv := &entities.Conversation{
		UserID:      userID,
		ContextType: req.ContextType,
		ContextID:   req.ContextID,
		Messages:    []entities.ConversationMessage{},
	}
	if err := repo.Create(c.Context(), conv); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(conv)
}

func (s *Server) handleGetConversation(c *fiber.Ctx, userID uuid.UUID) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	repo := repository.NewConversationRepo(s.db)
	conv, err := repo.Get(c.Context(), id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	if conv.UserID != userID {
		return c.Status(403).JSON(fiber.Map{"error": "forbidden"})
	}
	return c.JSON(conv)
}

func (s *Server) handleConversationMessage(c *fiber.Ctx, userID uuid.UUID) error {
	idStr := c.Params("id")
	convID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	var req ConversationMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if req.Content == "" {
		return c.Status(400).JSON(fiber.Map{"error": "content is required"})
	}

	convRepo := repository.NewConversationRepo(s.db)
	conv, err := convRepo.Get(c.Context(), convID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	if conv.UserID != userID {
		return c.Status(403).JSON(fiber.Map{"error": "forbidden"})
	}

	// Save user message
	userMsg := entities.ConversationMessage{
		Role:      "user",
		Content:   req.Content,
		Timestamp: time.Now(),
	}
	if err := convRepo.AppendMessage(c.Context(), convID, userMsg); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to save message"})
	}

	if s.orchestrator == nil {
		return c.Status(503).JSON(fiber.Map{"error": "agent not configured"})
	}

	// Look up actual data for the context being discussed
	contextData := s.buildRichContext(c.Context(), userID, conv.ContextType, conv.ContextID)

	// Pass visit ID when context is a visit so the orchestrator loads the right phase/data
	visitIDStr := ""
	if conv.ContextType == "visit" {
		visitIDStr = conv.ContextID
	}

	sess, err := s.orchestrator.GetOrCreateSession(c.Context(), userID, visitIDStr)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Inject conversation history so the agent has memory across turns
	for _, msg := range conv.Messages {
		sess.Messages = append(sess.Messages, provider.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Build user message with actual context data
	fullMsg := req.Content
	if contextData != "" {
		fullMsg = contextData + "\n\nUser question: " + req.Content
	}

	eventCh, err := s.orchestrator.Run(c.Context(), sess, fullMsg)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Stream SSE
	writer := agent.NewStreamWriter(c)
	c.Status(200)
	var fullResponse string
	for ev := range eventCh {
		if err := writer.Write(ev); err != nil {
			break
		}
		writer.Flush()
		if ev.Type == entities.EventTextDelta {
			if p, ok := ev.Payload.(entities.TextDeltaPayload); ok {
				fullResponse += p.Text
			}
		}
	}

	// Save assistant response
	if fullResponse != "" {
		assistantMsg := entities.ConversationMessage{
			Role:      "assistant",
			Content:   fullResponse,
			Timestamp: time.Now(),
		}
		_ = convRepo.AppendMessage(c.Context(), convID, assistantMsg)
	}

	return nil
}

// buildRichContext looks up actual entity data to give the agent real context
func (s *Server) buildRichContext(ctx context.Context, userID uuid.UUID, contextType, contextID string) string {
	switch contextType {
	case "insight":
		id, err := uuid.Parse(contextID)
		if err != nil {
			return ""
		}
		repo := repository.NewInsightCardRepo(s.db)
		card, err := repo.GetByID(ctx, id)
		if err != nil {
			return ""
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("The user is asking about this insight: \"%s\" — %s\n\n", card.Title, card.Body))
		// Include their actual health data so the agent can give a substantive answer
		b.WriteString(s.buildHealthSummary(ctx, userID))
		return b.String()

	case "lab":
		// contextID can be a UUID (legacy) or a lab name (new stable key)
		var labs []struct {
			LabName     *string   `db:"lab_name"`
			LoincCode   *string   `db:"loinc_code"`
			Value       float64   `db:"value"`
			Unit        *string   `db:"unit"`
			Flag        string    `db:"flag"`
			CollectedAt time.Time `db:"collected_at"`
		}
		// Try by name first, then by LOINC code, then by UUID
		_ = s.db.SelectContext(ctx, &labs,
			`SELECT lab_name, loinc_code, value, unit, flag, collected_at
			 FROM lab_results WHERE user_id = $1 AND (lab_name = $2 OR loinc_code = $2)
			 ORDER BY collected_at DESC LIMIT 5`, userID, contextID)
		if len(labs) == 0 {
			// Fall back to UUID lookup
			id, err := uuid.Parse(contextID)
			if err != nil {
				return ""
			}
			_ = s.db.SelectContext(ctx, &labs,
				`SELECT lab_name, loinc_code, value, unit, flag, collected_at
				 FROM lab_results WHERE id = $1 AND user_id = $2 LIMIT 1`, id, userID)
		}
		if len(labs) == 0 {
			return ""
		}
		latest := labs[0]
		name := "Unknown"
		if latest.LabName != nil {
			name = *latest.LabName
		}
		unit := ""
		if latest.Unit != nil {
			unit = *latest.Unit
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("The user is asking about their %s lab result.\n\n", name))
		b.WriteString(fmt.Sprintf("Latest: %.2f %s (flag: %s, date: %s)\n",
			latest.Value, unit, latest.Flag, latest.CollectedAt.Format("2006-01-02")))
		if len(labs) > 1 {
			b.WriteString("\nRecent history:\n")
			for _, l := range labs {
				d := l.CollectedAt.Format("2006-01-02")
				b.WriteString(fmt.Sprintf("- %s: %.2f %s (%s)\n", d, l.Value, unit, l.Flag))
			}
		}
		return b.String()

	case "medication":
		if contextID == "all" {
			medRepo := repository.NewMedicationRepo(s.db)
			meds, err := medRepo.ListActive(ctx, userID)
			if err != nil || len(meds) == 0 {
				return "The patient wants to discuss their medications but none are currently on file."
			}
			var b strings.Builder
			b.WriteString("The patient wants to discuss their medications and potential interactions.\n\nActive medications:\n")
			for _, m := range meds {
				b.WriteString(fmt.Sprintf("- %s %s %s\n", m.Name, m.Dose, m.Frequency))
			}
			return b.String()
		}
		id, err := uuid.Parse(contextID)
		if err != nil {
			return ""
		}
		var med struct {
			Name      string `db:"name"`
			Dose      string `db:"dose"`
			Frequency string `db:"frequency"`
		}
		err = s.db.GetContext(ctx, &med,
			`SELECT name, dose, frequency FROM medications WHERE id = $1 AND user_id = $2`, id, userID)
		if err != nil {
			return ""
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("The user is asking about this specific medication:\n\nName: %s\nDose: %s\nFrequency: %s\n\n", med.Name, med.Dose, med.Frequency))
		b.WriteString(s.buildHealthSummary(ctx, userID))
		return b.String()

	case "visit":
		// Visit context is handled by the orchestrator's context assembler via visit ID
		return ""

	default:
		return ""
	}
}

// buildHealthSummary creates a comprehensive health overview for broad questions.
// This gives the agent real data to work with — labs, meds (active + history), temporal correlations.
// One human, one story: the agent sees the full timeline of this person's health.
func (s *Server) buildHealthSummary(ctx context.Context, userID uuid.UUID) string {
	var b strings.Builder

	// Abnormal labs
	type labRow struct {
		LabName     *string   `db:"lab_name"`
		Value       float64   `db:"value"`
		Unit        *string   `db:"unit"`
		Flag        string    `db:"flag"`
		CollectedAt time.Time `db:"collected_at"`
	}
	var abnormals []labRow
	_ = s.db.SelectContext(ctx, &abnormals,
		`SELECT lab_name, value, unit, flag, collected_at
		 FROM lab_results WHERE user_id = $1 AND flag != 'normal' AND flag != ''
		 ORDER BY collected_at DESC LIMIT 20`, userID)
	if len(abnormals) > 0 {
		b.WriteString(fmt.Sprintf("ABNORMAL LAB RESULTS (%d):\n", len(abnormals)))
		for _, l := range abnormals {
			name := "Unknown"
			if l.LabName != nil {
				name = *l.LabName
			}
			unit := ""
			if l.Unit != nil {
				unit = *l.Unit
			}
			b.WriteString(fmt.Sprintf("- %s: %.2f %s [%s] (%s)\n",
				name, l.Value, unit, strings.ToUpper(l.Flag), l.CollectedAt.Format("2006-01-02")))
		}
		b.WriteString("\n")
	}

	// All latest labs (one per test name)
	var latestLabs []labRow
	_ = s.db.SelectContext(ctx, &latestLabs,
		`SELECT DISTINCT ON (lab_name) lab_name, value, unit, flag, collected_at
		 FROM lab_results WHERE user_id = $1
		 ORDER BY lab_name, collected_at DESC`, userID)
	if len(latestLabs) > 0 {
		normalCount := 0
		for _, l := range latestLabs {
			if l.Flag == "normal" || l.Flag == "" {
				normalCount++
			}
		}
		b.WriteString(fmt.Sprintf("ALL LATEST LAB VALUES (%d tests, %d normal):\n", len(latestLabs), normalCount))
		for _, l := range latestLabs {
			name := "Unknown"
			if l.LabName != nil {
				name = *l.LabName
			}
			unit := ""
			if l.Unit != nil {
				unit = *l.Unit
			}
			flag := ""
			if l.Flag != "normal" && l.Flag != "" {
				flag = " [" + strings.ToUpper(l.Flag) + "]"
			}
			b.WriteString(fmt.Sprintf("- %s: %.2f %s%s (%s)\n",
				name, l.Value, unit, flag, l.CollectedAt.Format("2006-01-02")))
		}
		b.WriteString("\n")
	}

	// ALL medications — active + historical with timelines
	medRepo := repository.NewMedicationRepo(s.db)
	allMeds, _ := medRepo.ListAll(ctx, userID)

	var activeMeds, pastMeds []entities.Medication
	for _, m := range allMeds {
		if m.IsActive {
			activeMeds = append(activeMeds, m)
		} else {
			pastMeds = append(pastMeds, m)
		}
	}

	if len(activeMeds) > 0 {
		b.WriteString(fmt.Sprintf("ACTIVE MEDICATIONS (%d):\n", len(activeMeds)))
		for _, m := range activeMeds {
			since := ""
			if m.StartedAt != nil {
				since = fmt.Sprintf(" (since %s)", m.StartedAt.Format("2006-01-02"))
			}
			b.WriteString(fmt.Sprintf("- %s %s %s%s\n", m.Name, m.Dose, m.Frequency, since))
		}
		b.WriteString("\n")
	}

	if len(pastMeds) > 0 {
		b.WriteString(fmt.Sprintf("PAST MEDICATIONS (%d):\n", len(pastMeds)))
		for _, m := range pastMeds {
			period := ""
			if m.StartedAt != nil && m.EndedAt != nil {
				period = fmt.Sprintf(" (%s → %s)", m.StartedAt.Format("2006-01-02"), m.EndedAt.Format("2006-01-02"))
			} else if m.EndedAt != nil {
				period = fmt.Sprintf(" (stopped %s)", m.EndedAt.Format("2006-01-02"))
			}
			b.WriteString(fmt.Sprintf("- %s %s %s%s\n", m.Name, m.Dose, m.Frequency, period))
		}
		b.WriteString("\n")
	}

	// Medication-Lab temporal correlations
	// For meds with known start/end dates, find labs that changed significantly nearby
	s.buildMedLabCorrelations(ctx, userID, allMeds, &b)

	if b.Len() == 0 {
		return "No health data on file yet."
	}

	return b.String()
}

// buildMedLabCorrelations finds labs that changed meaningfully around medication start/stop dates.
// This gives the agent cause-effect data: "After starting Ozempic, HbA1c dropped 7.2→6.1"
func (s *Server) buildMedLabCorrelations(ctx context.Context, userID uuid.UUID, meds []entities.Medication, b *strings.Builder) {
	type labSnap struct {
		LabName *string   `db:"lab_name"`
		Value   float64   `db:"value"`
		Unit    *string   `db:"unit"`
		Date    time.Time `db:"collected_at"`
	}

	var correlations []string

	for _, med := range meds {
		dates := []struct {
			t     time.Time
			event string
		}{}
		if med.StartedAt != nil {
			dates = append(dates, struct {
				t     time.Time
				event string
			}{*med.StartedAt, "Started"})
		}
		if med.EndedAt != nil {
			dates = append(dates, struct {
				t     time.Time
				event string
			}{*med.EndedAt, "Stopped"})
		}

		for _, d := range dates {
			// Find labs: closest before the date vs closest after (within 3 months)
			before := d.t.AddDate(0, -3, 0)
			after := d.t.AddDate(0, 3, 0)

			var labsBefore []labSnap
			_ = s.db.SelectContext(ctx, &labsBefore,
				`SELECT DISTINCT ON (lab_name) lab_name, value, unit, collected_at
				 FROM lab_results WHERE user_id = $1 AND collected_at >= $2 AND collected_at <= $3
				 ORDER BY lab_name, collected_at DESC`,
				userID, before, d.t)

			var labsAfter []labSnap
			_ = s.db.SelectContext(ctx, &labsAfter,
				`SELECT DISTINCT ON (lab_name) lab_name, value, unit, collected_at
				 FROM lab_results WHERE user_id = $1 AND collected_at > $2 AND collected_at <= $3
				 ORDER BY lab_name, collected_at DESC`,
				userID, d.t, after)

			// Match by lab name and check for >15% change
			afterMap := map[string]labSnap{}
			for _, l := range labsAfter {
				if l.LabName != nil {
					afterMap[*l.LabName] = l
				}
			}

			for _, lb := range labsBefore {
				if lb.LabName == nil || lb.Value == 0 {
					continue
				}
				la, ok := afterMap[*lb.LabName]
				if !ok {
					continue
				}
				pctChange := ((la.Value - lb.Value) / lb.Value) * 100
				if pctChange < 0 {
					pctChange = -pctChange
				}
				if pctChange >= 15 {
					unit := ""
					if lb.Unit != nil {
						unit = *lb.Unit
					}
					direction := "increased"
					if la.Value < lb.Value {
						direction = "decreased"
					}
					correlations = append(correlations,
						fmt.Sprintf("- %s %s (%s): %s %s %.2f %s → %.2f %s",
							d.event, med.Name, d.t.Format("2006-01-02"),
							*lb.LabName, direction, lb.Value, unit, la.Value, unit))
				}
			}
		}
	}

	if len(correlations) > 0 {
		b.WriteString("MEDICATION-LAB TIMELINE:\n")
		for _, c := range correlations {
			b.WriteString(c + "\n")
		}
		b.WriteString("\n")
	}
}

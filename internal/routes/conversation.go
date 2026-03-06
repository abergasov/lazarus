package routes

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"lazarus/internal/agent"
	"lazarus/internal/entities"
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
	_ = convRepo.AppendMessage(c.Context(), convID, userMsg)

	// Build context-aware prompt
	contextPrompt := buildContextPrompt(conv.ContextType, conv.ContextID, req.Content)

	// Run agent and stream response
	if s.orchestrator == nil {
		return c.Status(503).JSON(fiber.Map{"error": "agent not configured"})
	}

	// Create or get agent session for this conversation
	sess, err := s.orchestrator.GetOrCreateSession(c.Context(), userID, "")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	eventCh, err := s.orchestrator.Run(c.Context(), sess, contextPrompt)
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
		// Accumulate text for persistence
		if ev.Type == entities.EventTextDelta {
			if payload, ok := ev.Payload.(string); ok {
				fullResponse += payload
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

func buildContextPrompt(contextType, contextID, userMessage string) string {
	prefix := ""
	switch contextType {
	case "insight":
		prefix = "[Context: Discussing insight " + contextID + "] "
	case "lab":
		prefix = "[Context: Discussing lab result " + contextID + "] "
	case "visit":
		prefix = "[Context: Discussing visit " + contextID + "] "
	case "medication":
		prefix = "[Context: Discussing medication " + contextID + "] "
	}
	return prefix + userMessage
}

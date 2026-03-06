package routes

import (
	"encoding/binary"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"lazarus/internal/agent"
	"lazarus/internal/entities"
)

type AgentRequest struct {
	VisitID    string `json:"visit_id"`
	Message    string `json:"message"`
	ProviderID string `json:"provider_id,omitempty"`
	ModelID    string `json:"model_id,omitempty"`
}

func (s *Server) handleAgentStream(c *fiber.Ctx, userIDInt int64) error {
	var req AgentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if req.Message == "" {
		return c.Status(400).JSON(fiber.Map{"error": "message is required"})
	}

	userID := int64ToUUID(userIDInt)

	sess, err := s.orchestrator.GetOrCreateSession(c.Context(), userID, req.VisitID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	eventCh, err := s.orchestrator.Run(c.Context(), sess, req.Message)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	writer := agent.NewStreamWriter(c)
	c.Status(200)
	for ev := range eventCh {
		if err := writer.Write(ev); err != nil {
			break
		}
		writer.Flush()
	}
	return nil
}

// wrapAuthUUID wraps routes that need a uuid.UUID user ID
func (s *Server) wrapAuthUUID(route func(c *fiber.Ctx, userID uuid.UUID) error) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		claims, ok := token.Claims.(*entities.UserJWT)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		return route(c, int64ToUUID(claims.GetUserID()))
	}
}

// int64ToUUID converts an int64 user ID to a deterministic UUID.
func int64ToUUID(id int64) uuid.UUID {
	var u uuid.UUID
	binary.BigEndian.PutUint64(u[8:], uint64(id))
	u[6] = (u[6] & 0x0f) | 0x40 // version 4
	u[8] = (u[8] & 0x3f) | 0x80 // variant bits
	return u
}

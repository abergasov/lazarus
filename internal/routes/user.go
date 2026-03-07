package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (s *Server) handleUser(ctx *fiber.Ctx, userID uuid.UUID) error {
	user, err := s.srvUser.GetUserByID(ctx.Context(), userID)
	if err != nil {
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}
	return ctx.JSON(user)
}

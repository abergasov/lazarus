package routes

import "github.com/gofiber/fiber/v2"

func (s *Server) handleUser(ctx *fiber.Ctx, userID int64) error {
	user, err := s.srvUser.GetUserByID(ctx.Context(), userID)
	if err != nil {
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}
	return ctx.JSON(user)
}

package agent

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"lazarus/internal/entities"
)

type StreamWriter struct {
	ctx *fiber.Ctx
}

func NewStreamWriter(c *fiber.Ctx) *StreamWriter {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")
	return &StreamWriter{ctx: c}
}

func (w *StreamWriter) Write(ev entities.ClientEvent) error {
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w.ctx.Response().BodyWriter(),
		"event: %s\ndata: %s\n\n", ev.Type, data)
	return err
}

func (w *StreamWriter) Flush() {
	if f, ok := w.ctx.Response().BodyWriter().(interface{ Flush() }); ok {
		f.Flush()
	}
}

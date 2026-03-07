package agent

import (
	"bufio"
	"encoding/json"
	"fmt"

	"lazarus/internal/entities"
)

// BufStreamWriter writes SSE events to a bufio.Writer provided by
// fasthttp's SetBodyStreamWriter callback, enabling true streaming.
type BufStreamWriter struct {
	w *bufio.Writer
}

func NewBufStreamWriter(w *bufio.Writer) *BufStreamWriter {
	return &BufStreamWriter{w: w}
}

func (sw *BufStreamWriter) Write(ev entities.ClientEvent) error {
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(sw.w, "event: %s\ndata: %s\n\n", ev.Type, data)
	if err != nil {
		return err
	}
	return sw.w.Flush()
}

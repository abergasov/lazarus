package antivirus

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
	"time"
)

type Client struct {
	addr    string
	timeout time.Duration
}

func NewClient(addr string, timeout time.Duration) *Client {
	return &Client{
		addr:    addr,
		timeout: timeout,
	}
}

func (c *Client) ScanReader(ctx context.Context, r io.Reader) error {
	dialer := net.Dialer{Timeout: c.timeout}

	conn, err := dialer.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return fmt.Errorf("dial clamd: %w", err)
	}
	defer conn.Close() //nolint:errcheck

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(c.timeout))
	}

	// INSTREAM protocol
	if _, err = conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return fmt.Errorf("write clamd command: %w", err)
	}

	buf := make([]byte, 64*1024)
	lenBuf := make([]byte, 4)

	for {
		n, readErr := r.Read(buf)
		if n > 0 {
			if n > math.MaxUint32 {
				return fmt.Errorf("chunk too large: %d", n)
			}
			chunk := buf[:n]
			binary.BigEndian.PutUint32(lenBuf, uint32(n)) //nolint:gosec // overflow is checked above
			if _, err = conn.Write(lenBuf); err != nil {
				return fmt.Errorf("write chunk size: %w", err)
			}
			if _, err = conn.Write(chunk); err != nil {
				return fmt.Errorf("write chunk body: %w", err)
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read source: %w", readErr)
		}
	}

	// zero-length chunk terminates stream
	if _, err = conn.Write([]byte{0, 0, 0, 0}); err != nil {
		return fmt.Errorf("finish clamd stream: %w", err)
	}

	resp, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read clamd response: %w", err)
	}

	resp = strings.TrimSpace(resp)
	resp = strings.TrimSuffix(resp, "\x00") // some versions of clamd append null byte

	switch {
	case strings.HasSuffix(resp, "OK"):
		return nil
	case strings.Contains(resp, "FOUND"):
		return fmt.Errorf("malware detected: %s", resp)
	default:
		return fmt.Errorf("unexpected clamd response: %s", resp)
	}
}

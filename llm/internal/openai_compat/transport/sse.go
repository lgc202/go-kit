package transport

import (
	"bufio"
	"io"
	"strings"
)

// SSEDecoder decodes Server-Sent Events (SSE) and yields concatenated "data:" payloads.
type SSEDecoder struct {
	r   *bufio.Reader
	buf []string
}

func NewSSEDecoder(r io.Reader) *SSEDecoder {
	return &SSEDecoder{r: bufio.NewReader(r)}
}

// NextData returns the next SSE data payload joined by "\n".
// It returns io.EOF when the underlying reader ends.
func (d *SSEDecoder) NextData() (string, error) {
	for {
		line, err := d.r.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}

		line = strings.TrimRight(line, "\r\n")

		if line == "" {
			if len(d.buf) > 0 {
				out := strings.Join(d.buf, "\n")
				d.buf = d.buf[:0]
				return out, nil
			}
			if err == io.EOF {
				return "", io.EOF
			}
			continue
		}

		if strings.HasPrefix(line, "data:") {
			d.buf = append(d.buf, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}

		if err == io.EOF {
			if len(d.buf) > 0 {
				out := strings.Join(d.buf, "\n")
				d.buf = d.buf[:0]
				return out, nil
			}
			return "", io.EOF
		}
	}
}

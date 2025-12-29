package openai_compat

import (
	"bufio"
	"bytes"
	"io"
)

type sseDecoder struct {
	r *bufio.Reader
}

func newSSEDecoder(r io.Reader) *sseDecoder {
	return &sseDecoder{r: bufio.NewReaderSize(r, 64*1024)}
}

// Next returns the next SSE event's concatenated data payload.
//
// It concatenates multiple `data:` lines with `\n`, per the SSE spec.
func (d *sseDecoder) Next() ([]byte, error) {
	var dataLines [][]byte
	for {
		line, err := d.r.ReadBytes('\n')
		if err != nil {
			// If we accumulated data before EOF, return it.
			if len(line) > 0 {
				line = bytes.TrimRight(line, "\r\n")
				if len(line) > 0 {
					dataLines = appendDataLine(dataLines, line)
				}
			}
			if len(dataLines) > 0 {
				return bytes.Join(dataLines, []byte("\n")), nil
			}
			if err == io.EOF {
				return nil, io.EOF
			}
			return nil, err
		}

		line = bytes.TrimRight(line, "\r\n")
		if len(line) == 0 {
			if len(dataLines) == 0 {
				continue
			}
			return bytes.Join(dataLines, []byte("\n")), nil
		}

		// Comment line.
		if line[0] == ':' {
			continue
		}
		dataLines = appendDataLine(dataLines, line)
	}
}

func appendDataLine(dst [][]byte, line []byte) [][]byte {
	if !bytes.HasPrefix(line, []byte("data:")) {
		return dst
	}
	val := line[len("data:"):]
	if len(val) > 0 && val[0] == ' ' {
		val = val[1:]
	}
	return append(dst, append([]byte(nil), val...))
}

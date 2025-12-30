package openai_compat

import (
	"bufio"
	"io"
	"strings"
)

type sseDecoder struct {
	r   *bufio.Reader
	buf []string
}

func newSSEDecoder(r io.Reader) *sseDecoder {
	return &sseDecoder{r: bufio.NewReader(r)}
}

// NextData 返回下一个 SSE 数据载荷（用 "\n" 连接），当底层读取器结束时返回 io.EOF
func (d *sseDecoder) NextData() (string, error) {
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

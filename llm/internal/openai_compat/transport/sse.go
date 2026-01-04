package transport

import (
	"bufio"
	"io"
	"strings"
)

// SSEDecoder SSE 解码器，解析并拼接 "data:" 载荷
type SSEDecoder struct {
	r   *bufio.Reader
	buf []string
}

func NewSSEDecoder(r io.Reader) *SSEDecoder {
	return &SSEDecoder{r: bufio.NewReader(r)}
}

// NextData 返回下一个 SSE data 载荷，用 "\n" 拼接
// 底层 reader 结束时返回 io.EOF
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

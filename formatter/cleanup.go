package formatter

import (
	"bytes"
	"io"
)

// HeaderCleaner removes > and < from curl --verbose output.
type HeaderCleaner struct {
	Out io.Writer

	// Verbose removes the request headers part of the output as well as the lines
	// starting with * if set to false.
	Verbose bool

	// DropResponseHeaders removes response headers from verbose curl output.
	// This is useful when response headers are captured separately.
	DropResponseHeaders bool

	// Post is inserted after the request headers.
	Post []byte

	buf  []byte
	line []byte
}

var (
	capath  = []byte("  CApath:")
	ccapath = []byte("*   CApath:")
)

func (c *HeaderCleaner) Write(p []byte) (n int, err error) {
	n = len(p)
	c.buf = c.buf[:0]
	p = bytes.Replace(p, capath, ccapath, 1) // Fix curl misformatted line
	for len(p) > 0 {
		idx := bytes.IndexByte(p, '\n')
		if idx == -1 {
			c.line = append(c.line, p...)
			break
		}
		c.line = append(c.line, p[:idx+1]...)
		p = p[idx+1:]
		ignore := false
		b, i := firstVisibleChar(c.line)
		switch b {
		case '>':
			if c.Verbose {
				c.line = c.line[i+2:]
			} else {
				ignore = true
			}
		case '<':
			if c.DropResponseHeaders {
				ignore = true
			} else {
				c.line = c.line[i+2:]
			}
		case '}', '{':
			ignore = true
			if c.Verbose && len(c.Post) > 0 {
				c.buf = append(c.buf, c.formatPost()...)
				c.Post = nil
			}
		case '*':
			if !c.Verbose {
				ignore = true
			}
		}
		if !ignore {
			c.buf = append(c.buf, c.line...)
		}
		c.line = c.line[:0]
	}
	_, err = c.Out.Write(c.buf)
	return
}

func (c *HeaderCleaner) formatPost() []byte {
	post := bytes.TrimSpace(c.Post)
	if len(post) == 0 {
		return nil
	}
	return append(append([]byte{}, post...), '\n', '\n')
}

var colorEscape = []byte("\x1b[")

func firstVisibleChar(b []byte) (byte, int) {
	if len(b) == 0 {
		return 0, -1
	}
	if bytes.HasPrefix(b, colorEscape) {
		if idx := bytes.IndexByte(b, 'm'); idx != -1 {
			if idx < len(b) {
				return b[idx+1], idx + 1
			} else {
				return 0, -1
			}
		}
	}
	return b[0], 0
}

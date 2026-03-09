package main

import (
	"bytes"
	"testing"

	"github.com/rjvkn/curli/args"
	"github.com/rjvkn/curli/formatter"
)

func TestFormatResponseJSONDecodesUnicodeEscapes(t *testing.T) {
	var out bytes.Buffer
	writeResponseBody(&out, []byte("{\"message\":\"Merhaba d\\u00fcnya \\u2615\"}"), "application/json", true, false, formatter.ColorScheme{})
	got := out.Bytes()

	if !bytes.Contains(got, []byte("Merhaba dünya ☕")) {
		t.Fatalf("formatted output %q does not contain decoded Unicode characters", got)
	}
	if bytes.Contains(got, []byte(`\u00fc`)) || bytes.Contains(got, []byte(`\u2615`)) {
		t.Fatalf("formatted output %q still contains unicode escape sequences", got)
	}
}

func TestRequestPreviewFormatsJSON(t *testing.T) {
	opts := args.Opts{"-H", "Content-Type: application/json"}
	got := requestPreview([]byte("{\"message\":\"Merhaba d\\u00fcnya\"}"), opts, formatter.ColorScheme{}, false)
	if !bytes.Contains(got, []byte("Merhaba dünya")) {
		t.Fatalf("formatted request preview %q does not contain decoded Unicode characters", got)
	}
}

func TestWriteResponseBodyFormatsValidJSONWithoutContentType(t *testing.T) {
	var out bytes.Buffer
	writeResponseBody(&out, []byte("{\"message\":\"Merhaba d\\u00fcnya \\u2615\"}"), "", true, false, formatter.ColorScheme{})
	got := out.Bytes()

	if !bytes.Contains(got, []byte("Merhaba dünya ☕")) {
		t.Fatalf("formatted output %q does not contain decoded Unicode characters", got)
	}
	if bytes.Contains(got, []byte(`\u00fc`)) || bytes.Contains(got, []byte(`\u2615`)) {
		t.Fatalf("formatted output %q still contains unicode escape sequences", got)
	}
	if !bytes.Contains(got, []byte("\n    \"message\": ")) {
		t.Fatalf("formatted output %q is not pretty-printed", got)
	}
}

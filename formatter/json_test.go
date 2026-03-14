package formatter

import (
	"bytes"
	"testing"
)

func TestFormatJSONDecodesUnicodeEscapes(t *testing.T) {
	got, err := FormatJSON([]byte("{\"message\":\"Merhaba d\\u00fcnya \\u2615\"}"), ColorScheme{})
	if err != nil {
		t.Fatalf("FormatJSON returned error: %v", err)
	}

	if !bytes.Contains(got, []byte("Merhaba dünya ☕")) {
		t.Fatalf("formatted JSON %q does not contain decoded Unicode characters", got)
	}
	if bytes.Contains(got, []byte(`\u00fc`)) || bytes.Contains(got, []byte(`\u2615`)) {
		t.Fatalf("formatted JSON %q still contains Unicode escape sequences", got)
	}
}

func TestFormatJSONPreservesObjectKeyOrder(t *testing.T) {
	got, err := FormatJSON([]byte("{\"b\":1,\"a\":2}"), ColorScheme{})
	if err != nil {
		t.Fatalf("FormatJSON returned error: %v", err)
	}

	want := "{\n    \"b\": 1,\n    \"a\": 2\n}\n"
	if string(got) != want {
		t.Fatalf("FormatJSON returned %q, want %q", got, want)
	}
}

package formatter

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const jsonIndent = "    "

// FormatJSON renders valid JSON with indentation and optional ANSI colors.
func FormatJSON(src []byte, scheme ColorScheme) ([]byte, error) {
	value, err := decodeJSON(src)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	writeJSONValue(&out, value, 0, scheme)
	if !scheme.IsZero() {
		out.WriteString(scheme.Color(ResetColor))
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

func decodeJSON(src []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(src))
	decoder.UseNumber()

	value, err := decodeJSONValue(decoder)
	if err != nil {
		return nil, err
	}

	if token, err := decoder.Token(); err == nil {
		return nil, fmt.Errorf("unexpected trailing JSON token %v", token)
	}

	return value, nil
}

type objectEntry struct {
	key   string
	value any
}

type objectValue []objectEntry

func decodeJSONValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}

	switch token := token.(type) {
	case json.Delim:
		switch token {
		case '{':
			var object objectValue
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, fmt.Errorf("unexpected object key token %T", keyToken)
				}
				value, err := decodeJSONValue(decoder)
				if err != nil {
					return nil, err
				}
				object = append(object, objectEntry{key: key, value: value})
			}
			end, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			if end != json.Delim('}') {
				return nil, fmt.Errorf("unexpected object closing token %v", end)
			}
			return object, nil
		case '[':
			var array []any
			for decoder.More() {
				value, err := decodeJSONValue(decoder)
				if err != nil {
					return nil, err
				}
				array = append(array, value)
			}
			end, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			if end != json.Delim(']') {
				return nil, fmt.Errorf("unexpected array closing token %v", end)
			}
			return array, nil
		default:
			return nil, fmt.Errorf("unexpected delimiter %q", token)
		}
	default:
		return token, nil
	}
}

func writeJSONValue(out *bytes.Buffer, value any, level int, scheme ColorScheme) {
	switch value := value.(type) {
	case objectValue:
		writeJSONObject(out, value, level, scheme)
	case []any:
		writeJSONArray(out, value, level, scheme)
	case string:
		writeJSONString(out, value, scheme.Value)
	case json.Number:
		writeColored(out, scheme.Value, []byte(value.String()))
	case float64:
		writeColored(out, scheme.Value, []byte(fmt.Sprintf("%v", value)))
	case bool:
		writeColored(out, scheme.Literal, []byte(fmt.Sprintf("%t", value)))
	case nil:
		writeColored(out, scheme.Literal, []byte("null"))
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			writeColored(out, scheme.Error, []byte(`"<format error>"`))
			return
		}
		writeColored(out, scheme.Value, encoded)
	}
}

func writeJSONObject(out *bytes.Buffer, value objectValue, level int, scheme ColorScheme) {
	writeColored(out, scheme.Default, []byte("{"))
	if len(value) == 0 {
		writeColored(out, scheme.Default, []byte("}"))
		return
	}

	for i, entry := range value {
		out.WriteByte('\n')
		writeIndent(out, level+1)
		writeJSONString(out, entry.key, scheme.Field)
		writeColored(out, scheme.Default, []byte(": "))
		writeJSONValue(out, entry.value, level+1, scheme)
		if i < len(value)-1 {
			writeColored(out, scheme.Default, []byte(","))
		}
	}

	out.WriteByte('\n')
	writeIndent(out, level)
	writeColored(out, scheme.Default, []byte("}"))
}

func writeJSONArray(out *bytes.Buffer, value []any, level int, scheme ColorScheme) {
	writeColored(out, scheme.Default, []byte("["))
	if len(value) == 0 {
		writeColored(out, scheme.Default, []byte("]"))
		return
	}

	for i, item := range value {
		out.WriteByte('\n')
		writeIndent(out, level+1)
		writeJSONValue(out, item, level+1, scheme)
		if i < len(value)-1 {
			writeColored(out, scheme.Default, []byte(","))
		}
	}

	out.WriteByte('\n')
	writeIndent(out, level)
	writeColored(out, scheme.Default, []byte("]"))
}

func writeJSONString(out *bytes.Buffer, value string, color string) {
	encoded, err := json.Marshal(value)
	if err != nil {
		writeColored(out, color, []byte(`""`))
		return
	}
	writeColored(out, color, encoded)
}

func writeIndent(out *bytes.Buffer, level int) {
	for i := 0; i < level; i++ {
		out.WriteString(jsonIndent)
	}
}

func writeColored(out *bytes.Buffer, color string, value []byte) {
	if color != "" {
		out.WriteString(color)
	}
	out.Write(value)
}

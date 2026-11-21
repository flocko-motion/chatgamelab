package adapters

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Schema is the shape a structured reply takes: a JSON Schema in the
// providers' strict mode, every property required and none other allowed.
type Schema struct {
	// Name labels the format for the provider.
	Name string
	Body map[string]any
}

// ErrBadReply is a reply the provider finished but that cannot be used: a
// refusal, one cut short, or one that does not parse against its schema. It
// is never kept, so the same turn may be made again.
var ErrBadReply = errors.New("the reply cannot be used")

// The bad replies a provider reports itself.
var (
	ErrRefused   = fmt.Errorf("the model refused: %w", ErrBadReply)
	ErrTruncated = fmt.Errorf("the reply was cut short: %w", ErrBadReply)
)

// BadReply marks why a reply cannot be used.
func BadReply(call, why string) error {
	return fmt.Errorf("%s: %s: %w", call, why, ErrBadReply)
}

// JSONObject tidies a reply that should be a JSON object: a markdown fence
// around it and any text before or after it go. A reply with no object comes
// back as it is, for the schema to refuse.
func JSONObject(reply string) string {
	text := strings.TrimSpace(reply)
	if fenced, ok := strings.CutPrefix(text, "```"); ok {
		// The opening fence's line, such as ```json, goes with it.
		if _, rest, found := strings.Cut(fenced, "\n"); found {
			text = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(rest), "```"))
		}
	}
	first, last := strings.Index(text, "{"), strings.LastIndex(text, "}")
	if first < 0 || last < first {
		return text
	}
	return text[first : last+1]
}

// Has reports whether the schema's object has a property of this name.
func (s Schema) Has(name string) bool {
	properties, _ := s.Body["properties"].(map[string]any)
	_, has := properties[name]
	return has
}

// Check reads a reply against the schema and returns its fields; an error
// says where it does not fit. It knows the strict subset of JSON Schema the
// engine asks for: strings and objects, required properties, and none other
// allowed.
func (s Schema) Check(reply string) (map[string]any, error) {
	var v any
	if err := json.Unmarshal([]byte(reply), &v); err != nil {
		return nil, fmt.Errorf("not JSON: %w", err)
	}
	if err := conform(s.Body, v, "reply"); err != nil {
		return nil, err
	}
	fields, _ := v.(map[string]any)
	return fields, nil
}

func conform(schema map[string]any, v any, at string) error {
	switch schema["type"] {
	case "string":
		if _, ok := v.(string); !ok {
			return fmt.Errorf("%s: want a string", at)
		}
	case "object":
		object, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: want an object", at)
		}
		properties, _ := schema["properties"].(map[string]any)
		for _, name := range names(schema["required"]) {
			if _, has := object[name]; !has {
				return fmt.Errorf("%s: %s is missing", at, name)
			}
		}
		for name, value := range object {
			property, known := properties[name].(map[string]any)
			if !known {
				if schema["additionalProperties"] == false {
					return fmt.Errorf("%s: %s is not in the schema", at, name)
				}
				continue
			}
			if err := conform(property, value, at+"."+name); err != nil {
				return err
			}
		}
	}
	return nil
}

// names reads a schema's list of names, built in Go or read from JSON.
func names(v any) []string {
	switch list := v.(type) {
	case []string:
		return list
	case []any:
		out := make([]string, 0, len(list))
		for _, item := range list {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

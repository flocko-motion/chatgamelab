package adapters

import "testing"

// A structured reply is tidied before it is judged: a markdown fence and the
// words around the JSON object go; a reply with no object stays as it is.
func TestJSONObject(t *testing.T) {
	for in, want := range map[string]string{
		`{"a":"b"}`:                          `{"a":"b"}`,
		"  {\"a\":\"b\"}\n":                  `{"a":"b"}`,
		"```json\n{\"a\":\"b\"}\n```":        `{"a":"b"}`,
		"```\n{\"a\":{\"c\":\"d\"}}\n```":    `{"a":{"c":"d"}}`,
		"Sure! Here it is: {\"a\":\"b\"} :)": `{"a":"b"}`,
		"no object here":                     "no object here",
		"":                                   "",
	} {
		if got := JSONObject(in); got != want {
			t.Errorf("JSONObject(%q) = %q, want %q", in, got, want)
		}
	}
}

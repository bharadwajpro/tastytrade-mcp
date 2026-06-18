package mcptools

import "encoding/json"

// marshalResult renders v as pretty JSON for a tool text result.
// On marshal failure it returns a JSON error object (never panics).
func marshalResult(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		eb, _ := json.Marshal(map[string]string{"error": err.Error()})
		return string(eb)
	}
	return string(b)
}

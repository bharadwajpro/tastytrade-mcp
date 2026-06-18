package stream

import "fmt"

// FieldSpec maps an event type to its ordered field names (from FEED_SETUP).
type FieldSpec map[string][]string

// CompactEvent is one decoded event: field name -> raw value (string or float64).
type CompactEvent struct {
	Type   string
	Fields map[string]any
}

// DecodeCompact splits one FEED_DATA payload (already JSON-decoded into
// [eventType, []any]) into events using the field order in spec.
func DecodeCompact(eventType string, flat []any, spec FieldSpec) ([]CompactEvent, error) {
	fields, ok := spec[eventType]
	if !ok {
		return nil, fmt.Errorf("no field spec for event type %q", eventType)
	}
	m := len(fields)
	if m == 0 {
		return nil, fmt.Errorf("empty field spec for %q", eventType)
	}
	if len(flat)%m != 0 {
		return nil, fmt.Errorf("flat len %d not a multiple of field count %d for %q", len(flat), m, eventType)
	}
	var out []CompactEvent
	for i := 0; i < len(flat); i += m {
		ev := CompactEvent{Type: eventType, Fields: make(map[string]any, m)}
		for j, name := range fields {
			ev.Fields[name] = flat[i+j]
		}
		out = append(out, ev)
	}
	return out, nil
}

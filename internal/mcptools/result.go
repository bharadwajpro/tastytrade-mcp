package mcptools

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"time"
)

// marshalResult renders v as pretty JSON for a tool text result.
// NaN and Inf float values are converted to null so DxLink greeks/summary fields marshal cleanly.
// On marshal failure it returns a JSON error object (never panics).
func marshalResult(v any) string {
	b, err := json.MarshalIndent(sanitizeForJSON(v), "", "  ")
	if err != nil {
		eb, _ := json.Marshal(map[string]string{"error": err.Error()})
		return string(eb)
	}
	return string(b)
}

func sanitizeForJSON(v any) any {
	if v == nil {
		return nil
	}
	return valueToJSONSafe(reflect.ValueOf(v))
}

func valueToJSONSafe(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil
		}
		return f
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			return v.Interface()
		}
		if v.CanInterface() {
			if _, ok := v.Interface().(json.Marshaler); ok {
				return v.Interface()
			}
		}
		out := make(map[string]any, v.NumField())
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue
			}
			name := jsonFieldName(f)
			if name == "" || name == "-" {
				continue
			}
			out[name] = valueToJSONSafe(v.Field(i))
		}
		return out
	case reflect.Map:
		if v.IsNil() {
			return nil
		}
		out := make(map[string]any, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			out[iter.Key().String()] = valueToJSONSafe(iter.Value())
		}
		return out
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice && v.IsNil() {
			return nil
		}
		n := v.Len()
		out := make([]any, n)
		for i := 0; i < n; i++ {
			out[i] = valueToJSONSafe(v.Index(i))
		}
		return out
	default:
		return v.Interface()
	}
}

func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return f.Name
	}
	if tag == "-" {
		return "-"
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" {
		return f.Name
	}
	return name
}

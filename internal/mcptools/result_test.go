package mcptools

import (
	"math"
	"strings"
	"testing"

	"github.com/bharadwajpro/tastytrade-mcp/stream"
)

func TestMarshalResultSanitizesNaN(t *testing.T) {
	q := stream.Quote{
		Symbol: "TEST",
		Bid:    1.0,
		Delta:  math.NaN(),
		Gamma:  math.Inf(1),
		Theta:  -0.5,
	}
	out := marshalResult(map[string]any{"quote": q})
	if strings.Contains(out, "NaN") {
		t.Fatalf("output contains NaN: %s", out)
	}
	if !strings.Contains(out, `"delta": null`) {
		t.Fatalf("expected null delta: %s", out)
	}
	if !strings.Contains(out, `"gamma": null`) {
		t.Fatalf("expected null gamma: %s", out)
	}
	if !strings.Contains(out, `"theta": -0.5`) {
		t.Fatalf("expected valid theta: %s", out)
	}
}

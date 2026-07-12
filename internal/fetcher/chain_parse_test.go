package fetcher

import (
	"encoding/json"
	"testing"
)

const sampleFONested = `{
  "data": {
    "futures": [{"symbol": "/CLU6"}, {"symbol": "/CLZ6"}],
    "option-chains": [{
      "underlying-symbol": "/CL",
      "root-symbol": "/CL",
      "exercise-style": "American",
      "expirations": [
        {
          "underlying-symbol": "/CLU6",
          "option-root-symbol": "XL5",
          "option-contract-symbol": "XL5N6",
          "expiration-type": "Weekly",
          "expiration-date": "2026-07-30",
          "days-to-expiration": 19,
          "strikes": [
            {
              "strike-price": "105.0",
              "call": "./CLU6 XL5N6 260730C105",
              "call-streamer-symbol": "./XL5N26C105:XNYM",
              "put": "./CLU6 XL5N6 260730P105",
              "put-streamer-symbol": "./XL5N26P105:XNYM"
            }
          ]
        },
        {
          "underlying-symbol": "/CLU6",
          "option-root-symbol": "LO",
          "option-contract-symbol": "LOU6",
          "expiration-type": "Regular",
          "expiration-date": "2026-08-17",
          "days-to-expiration": 37,
          "strikes": [
            {
              "strike-price": 70.0,
              "call": "./CLU6 LOU6  260817C70",
              "call-streamer-symbol": "./LOU26C70:XNYM",
              "put": "./CLU6 LOU6  260817P70",
              "put-streamer-symbol": "./LOU26P70:XNYM"
            }
          ]
        },
        {
          "underlying-symbol": "/CLZ6",
          "option-root-symbol": "LO",
          "option-contract-symbol": "LOZ6",
          "expiration-type": "Regular",
          "expiration-date": "2026-11-17",
          "days-to-expiration": 129,
          "strikes": [
            {
              "strike-price": "108.0",
              "call": "./CLZ6 LOZ6  261117C108",
              "call-streamer-symbol": "./LOZ26C108:XNYM",
              "put": "./CLZ6 LOZ6  261117P108",
              "put-streamer-symbol": "./LOZ26P108:XNYM"
            }
          ]
        }
      ]
    }]
  }
}`

func TestParseNestedFutureOptionChainAll(t *testing.T) {
	oc, err := ParseNestedFutureOptionChain([]byte(sampleFONested), "/CL", "")
	if err != nil {
		t.Fatal(err)
	}
	if oc.Underlying != "/CL" {
		t.Fatalf("underlying: %q", oc.Underlying)
	}
	if len(oc.Series) != 3 {
		t.Fatalf("series: got %d want 3", len(oc.Series))
	}
	if len(oc.Expirations) != 3 {
		t.Fatalf("flat expirations: got %d", len(oc.Expirations))
	}
	// /CLU6 should have XL5 weekly + LO regular
	clu := 0
	for _, s := range oc.Series {
		if s.FutureSymbol == "/CLU6" {
			clu++
		}
	}
	if clu != 2 {
		t.Fatalf("CLU6 series: got %d want 2", clu)
	}
}

func TestParseNestedFutureOptionChainFilterDated(t *testing.T) {
	oc, err := ParseNestedFutureOptionChain([]byte(sampleFONested), "/CLZ6", "/CLZ6")
	if err != nil {
		t.Fatal(err)
	}
	if oc.Underlying != "/CLZ6" {
		t.Fatalf("underlying: %q", oc.Underlying)
	}
	if len(oc.Series) != 1 {
		t.Fatalf("series: got %d want 1", len(oc.Series))
	}
	s := oc.Series[0]
	if s.FutureSymbol != "/CLZ6" || s.OptionRootSymbol != "LO" {
		t.Fatalf("series: %+v", s)
	}
	if len(s.Expirations) != 1 || s.Expirations[0].OptionContractSymbol != "LOZ6" {
		t.Fatalf("expirations: %+v", s.Expirations)
	}
	if s.Expirations[0].Strikes[0].CallSymbol != "./CLZ6 LOZ6  261117C108" {
		t.Fatalf("call: %q", s.Expirations[0].Strikes[0].CallSymbol)
	}
}

func TestParseNestedFutureOptionChainFilterEmpty(t *testing.T) {
	_, err := ParseNestedFutureOptionChain([]byte(sampleFONested), "/CLN9", "/CLN9")
	if err == nil {
		t.Fatal("expected error for missing future")
	}
}

func TestParseNestedChainStrikeNumberOrString(t *testing.T) {
	raw := []byte(`{"data":{"items":[{"underlying-symbol":"SPY","expirations":[{"expiration-date":"2026-07-13","days-to-expiration":2,"strikes":[
		{"strike-price":"740","call":"C1","call-streamer-symbol":".C1","put":"P1","put-streamer-symbol":".P1"},
		{"strike-price":741,"call":"C2","call-streamer-symbol":".C2","put":"P2","put-streamer-symbol":".P2"}
	]}]}]}}`)
	oc, err := ParseNestedChain(raw)
	if err != nil {
		t.Fatal(err)
	}
	if oc.Expirations[0].Strikes[0].Strike != 740 || oc.Expirations[0].Strikes[1].Strike != 741 {
		t.Fatalf("strikes: %+v", oc.Expirations[0].Strikes)
	}
}

func TestStreamerSymbolForFO(t *testing.T) {
	oc, err := ParseNestedFutureOptionChain([]byte(sampleFONested), "/CLZ6", "/CLZ6")
	if err != nil {
		t.Fatal(err)
	}
	ss, ok := StreamerSymbolFor(oc, "./CLZ6 LOZ6  261117C108", "call")
	if !ok || ss != "./LOZ26C108:XNYM" {
		t.Fatalf("streamer: ok=%v ss=%q", ok, ss)
	}
}

func TestOptionChainJSONHasSeries(t *testing.T) {
	oc, _ := ParseNestedFutureOptionChain([]byte(sampleFONested), "/CL", "")
	b, err := json.Marshal(oc)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if _, ok := m["Series"]; !ok {
		t.Fatalf("missing Series in %s", string(b))
	}
}

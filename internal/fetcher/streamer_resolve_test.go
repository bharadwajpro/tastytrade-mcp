package fetcher

import "testing"

func TestIsStreamerSymbol(t *testing.T) {
	if !IsStreamerSymbol("./OGV26C4340:XCEC") {
		t.Fatal("expected streamer symbol")
	}
	if IsStreamerSymbol("./GCV6 OGV6  260924C4340") {
		t.Fatal("trade symbol should not match")
	}
}

func TestUnderlyingFromFOSymbol(t *testing.T) {
	got := underlyingFromFOSymbol("./GCV6 OGV6  260924C4340")
	if got != "/GCV6" {
		t.Fatalf("underlying: got %q", got)
	}
}

func TestOptionSideFromSymbol(t *testing.T) {
	if optionSideFromSymbol("./GCV6 OGV6  260924C4340") != "call" {
		t.Fatal("expected call")
	}
	if optionSideFromSymbol("./CLZ6 LOZ6  261117P148") != "put" {
		t.Fatal("expected put")
	}
}

func TestStreamerSymbolsFromPositions(t *testing.T) {
	m := StreamerSymbolsFromPositions([]Position{
		{Symbol: "./GCV6 OGV6  260924C4340", StreamerSymbol: "./OGV26C4340:XCEC"},
	})
	if m["./GCV6 OGV6  260924C4340"] != "./OGV26C4340:XCEC" {
		t.Fatalf("map: %v", m)
	}
}

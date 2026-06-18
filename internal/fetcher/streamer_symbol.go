package fetcher

// StreamerSymbolFor returns the DxLink streamer symbol for an OCC option symbol
// found in the chain. side is "call" or "put". ok=false if not present.
func StreamerSymbolFor(oc *OptionChain, occSymbol, side string) (string, bool) {
	if oc == nil {
		return "", false
	}
	for _, e := range oc.Expirations {
		for _, s := range e.Strikes {
			switch side {
			case "call":
				if s.CallSymbol == occSymbol {
					return s.CallStreamerSym, s.CallStreamerSym != ""
				}
			case "put":
				if s.PutSymbol == occSymbol {
					return s.PutStreamerSym, s.PutStreamerSym != ""
				}
			}
		}
	}
	return "", false
}

// EquityStreamerSymbol: for plain equities the streamer symbol equals the ticker.
func EquityStreamerSymbol(ticker string) string { return ticker }

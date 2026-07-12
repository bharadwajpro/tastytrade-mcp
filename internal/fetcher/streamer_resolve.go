package fetcher

import (
	"context"
	"regexp"
	"strings"
)

var foSideRe = regexp.MustCompile(`([CP])\d+$`)

// IsStreamerSymbol reports whether s is already a DxLink streamer symbol (contains an exchange suffix).
func IsStreamerSymbol(s string) bool {
	return strings.Contains(strings.TrimSpace(s), ":")
}

// ResolveStreamerSymbol maps a trade symbol to its DxLink streamer symbol when possible.
// Symbols that are already streamer format are returned unchanged. On lookup failure the
// original symbol is returned so callers can still attempt a subscription.
func ResolveStreamerSymbol(ctx context.Context, d Doer, symbol string) (string, error) {
	sym := strings.TrimSpace(symbol)
	if sym == "" || IsStreamerSymbol(sym) {
		return sym, nil
	}
	// Equity option streamer symbols start with "." but not "./" (e.g. .SPY260717C500).
	if strings.HasPrefix(sym, ".") && !strings.HasPrefix(sym, "./") {
		return sym, nil
	}
	if IsFutureSymbol(sym) && !strings.Contains(sym, " ") {
		fut, err := GetFuture(ctx, d, sym)
		if err != nil {
			return sym, err
		}
		if fut.StreamerSymbol != "" {
			return fut.StreamerSymbol, nil
		}
		return sym, nil
	}
	if strings.Contains(sym, " ") || strings.HasPrefix(sym, "./") {
		if ss, ok := streamerFromOptionChain(ctx, d, sym); ok {
			return ss, nil
		}
	}
	// Plain equity tickers use the symbol as-is.
	if !strings.Contains(sym, "/") && !strings.Contains(sym, " ") {
		return sym, nil
	}
	return sym, nil
}

func streamerFromOptionChain(ctx context.Context, d Doer, tradeSymbol string) (string, bool) {
	underlying := underlyingFromFOSymbol(tradeSymbol)
	side := optionSideFromSymbol(tradeSymbol)
	if underlying == "" || side == "" {
		return "", false
	}
	chain, err := GetOptionChain(ctx, d, underlying)
	if err != nil {
		return "", false
	}
	return StreamerSymbolFor(chain, tradeSymbol, side)
}

func underlyingFromFOSymbol(sym string) string {
	parts := strings.Fields(strings.TrimSpace(sym))
	if len(parts) == 0 {
		return ""
	}
	first := parts[0]
	if strings.HasPrefix(first, "./") {
		return "/" + strings.TrimPrefix(first, "./")
	}
	if strings.HasPrefix(first, "/") {
		return first
	}
	return ""
}

func optionSideFromSymbol(sym string) string {
	m := foSideRe.FindStringSubmatch(strings.TrimSpace(sym))
	if len(m) < 2 {
		return ""
	}
	if m[1] == "C" {
		return "call"
	}
	return "put"
}

// StreamerSymbolsFromPositions builds a trade-symbol → streamer-symbol map from open positions.
func StreamerSymbolsFromPositions(positions []Position) map[string]string {
	out := make(map[string]string, len(positions))
	for _, p := range positions {
		if p.Symbol != "" && p.StreamerSymbol != "" {
			out[p.Symbol] = p.StreamerSymbol
		}
	}
	return out
}

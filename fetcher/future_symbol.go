package fetcher

import (
	"fmt"
	"regexp"
	"strings"
)

// CME month codes used in futures symbols (e.g. Z in /CLZ6).
const futureMonthCodes = "FGHJKMNQUVXZ"

// futureDatedRe matches /PRODUCT + month code + 1–2 digit year (e.g. /CLZ6, /MGCQ6, /ESU6).
var futureDatedRe = regexp.MustCompile(`(?i)^/([A-Z0-9]+?)([` + futureMonthCodes + `])(\d{1,2})$`)

// IsFutureSymbol reports whether s looks like a futures root or dated future (/CL, /CLZ6).
func IsFutureSymbol(s string) bool {
	return strings.HasPrefix(strings.TrimSpace(s), "/")
}

// FutureProductRoot returns the product root for a futures symbol.
// /CL → (/CL, false); /CLZ6 → (/CL, true); /MGCQ6 → (/MGC, true).
// Non-futures symbols return ("", false).
func FutureProductRoot(symbol string) (root string, dated bool, err error) {
	sym := strings.TrimSpace(symbol)
	if !strings.HasPrefix(sym, "/") {
		return "", false, fmt.Errorf("not a futures symbol: %q", symbol)
	}
	if m := futureDatedRe.FindStringSubmatch(strings.ToUpper(sym)); m != nil {
		return "/" + m[1], true, nil
	}
	// Product root: /CL, /GC, /ES, /MGC, …
	if len(sym) < 2 {
		return "", false, fmt.Errorf("invalid futures symbol: %q", symbol)
	}
	return strings.ToUpper(sym), false, nil
}

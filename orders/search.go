package orders

import "net/url"

type OrderFilter struct {
	Status           []string // e.g. ["Live","Received"]
	UnderlyingSymbol string
	StartDate        string // YYYY-MM-DD
	EndDate          string
	PerPage          int
}

func BuildOrderSearchQuery(f OrderFilter) string {
	q := url.Values{}
	for _, s := range f.Status {
		if s != "" {
			q.Add("status[]", s)
		}
	}
	if f.UnderlyingSymbol != "" {
		q.Set("underlying-symbol", f.UnderlyingSymbol)
	}
	if f.StartDate != "" {
		q.Set("start-date", f.StartDate)
	}
	if f.EndDate != "" {
		q.Set("end-date", f.EndDate)
	}
	if f.PerPage > 0 {
		q.Set("per-page", itoa(f.PerPage))
	}
	return q.Encode()
}

func itoa(n int) string { // small local helper; or use strconv.Itoa
	if n == 0 { return "0" }
	neg := n < 0
	if neg { n = -n }
	var b [20]byte
	i := len(b)
	for n > 0 { i--; b[i] = byte('0' + n%10); n /= 10 }
	if neg { i--; b[i] = '-' }
	return string(b[i:])
}

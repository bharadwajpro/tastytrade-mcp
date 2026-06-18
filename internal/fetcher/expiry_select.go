package fetcher

import "sort"

// SelectExpiries returns indices into a date-sorted copy: the expiration nearest to
// targetDTE plus prior/next (=-1 when none). Returns sorted expirations too.
func SelectExpiries(exps []Expiration, targetDTE int) (sorted []Expiration, target, prior, next int) {
	sorted = append([]Expiration(nil), exps...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Date < sorted[j].Date })
	target, prior, next = -1, -1, -1
	bestDiff := 1 << 30
	for i, e := range sorted {
		d := e.DaysToExpiration - targetDTE
		if d < 0 {
			d = -d
		}
		if d < bestDiff {
			bestDiff, target = d, i
		}
	}
	if target > 0 {
		prior = target - 1
	}
	if target >= 0 && target < len(sorted)-1 {
		next = target + 1
	}
	return sorted, target, prior, next
}

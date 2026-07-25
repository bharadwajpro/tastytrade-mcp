package orders

// PlaceDecision is the resolved action for a place_order call.
type PlaceDecision int

const (
	DecisionRejectDisabled PlaceDecision = iota // trading not enabled
	DecisionDryRunOnly                          // enabled but confirm not set => show dry-run
	DecisionSubmit                              // enabled and confirmed => submit
)

// DecidePlace encodes the dry-run-first rule.
func DecidePlace(tradingEnabled, confirm bool) PlaceDecision {
	if !tradingEnabled {
		return DecisionRejectDisabled
	}
	if !confirm {
		return DecisionDryRunOnly
	}
	return DecisionSubmit
}

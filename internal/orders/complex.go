package orders

import "fmt"

type ComplexOrder struct {
	Type         string  `json:"type"`                    // "OTOCO" | "OCO"
	TriggerOrder *Order  `json:"trigger-order,omitempty"` // OTOCO entry leg
	Orders       []Order `json:"orders"`                  // the OCO pair
}

func BuildComplexOrder(coType string, trigger *Order, orders []Order) (ComplexOrder, error) {
	switch coType {
	case "OTOCO":
		if trigger == nil {
			return ComplexOrder{}, fmt.Errorf("OTOCO requires a trigger-order")
		}
		if len(orders) != 2 {
			return ComplexOrder{}, fmt.Errorf("OTOCO requires exactly 2 bracket orders, got %d", len(orders))
		}
	case "OCO":
		if trigger != nil {
			return ComplexOrder{}, fmt.Errorf("OCO must not have a trigger-order")
		}
		if len(orders) != 2 {
			return ComplexOrder{}, fmt.Errorf("OCO requires exactly 2 orders, got %d", len(orders))
		}
	default:
		return ComplexOrder{}, fmt.Errorf("unsupported complex order type %q", coType)
	}
	return ComplexOrder{Type: coType, TriggerOrder: trigger, Orders: orders}, nil
}

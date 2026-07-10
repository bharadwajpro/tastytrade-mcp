package mcptools

import "testing"

func TestRegisteredToolsIncludesSelectors(t *testing.T) {
	tools := registeredTools(Gates{ReadOnly: true})

	for _, name := range []string{"select_contract_by_delta", "select_contract_by_width"} {
		if !has(tools, name) {
			t.Fatalf("registeredTools() missing %q", name)
		}
	}
}

func TestParseSelectorContracts(t *testing.T) {
	contracts, err := parseSelectorContracts(`[{"symbol":"SPY   260116C00500000","strike":500,"delta":0.42}]`)
	if err != nil {
		t.Fatalf("parseSelectorContracts() error = %v", err)
	}
	if len(contracts) != 1 {
		t.Fatalf("len(contracts) = %d, want 1", len(contracts))
	}
	if contracts[0].Symbol != "SPY   260116C00500000" || contracts[0].Strike != 500 || contracts[0].Delta != 0.42 {
		t.Fatalf("contracts[0] = %+v", contracts[0])
	}
}

func TestParseSelectorContractsRejectsEmpty(t *testing.T) {
	if _, err := parseSelectorContracts(`[]`); err == nil {
		t.Fatal("parseSelectorContracts() error = nil, want error")
	}
}

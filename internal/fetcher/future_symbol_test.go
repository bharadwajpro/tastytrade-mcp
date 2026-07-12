package fetcher

import "testing"

func TestFutureProductRoot(t *testing.T) {
	cases := []struct {
		in        string
		root      string
		dated     bool
		wantErr   bool
	}{
		{"/CL", "/CL", false, false},
		{"/CLZ6", "/CL", true, false},
		{"/clz6", "/CL", true, false},
		{"/MGCQ6", "/MGC", true, false},
		{"/ESU6", "/ES", true, false},
		{"/GCV6", "/GC", true, false},
		{"/GC", "/GC", false, false},
		{"SPY", "", false, true},
		{"/", "", false, true},
	}
	for _, tc := range cases {
		root, dated, err := FutureProductRoot(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("%s: expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: %v", tc.in, err)
		}
		if root != tc.root || dated != tc.dated {
			t.Fatalf("%s: got root=%q dated=%v want root=%q dated=%v", tc.in, root, dated, tc.root, tc.dated)
		}
	}
}

func TestIsFutureSymbol(t *testing.T) {
	if !IsFutureSymbol("/CL") || IsFutureSymbol("SPY") {
		t.Fatal("IsFutureSymbol mismatch")
	}
}

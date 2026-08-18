package main

import (
	"reflect"
	"testing"
)

func TestSouthernGulfIslandTerminalCodes(t *testing.T) {
	codes := SouthernGulfIslandTerminalCodes("via Galiano Island (Sturdies Bay), Mayne Island (Village Bay), Pender Island (Otter Bay) to Salt Spring Island (Long Harbour)")
	want := []string{"PLH", "POB", "PSB", "PVB"}

	if !reflect.DeepEqual(codes, want) {
		t.Fatalf("SouthernGulfIslandTerminalCodes() = %v, want %v", codes, want)
	}
}

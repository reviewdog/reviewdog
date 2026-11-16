package parser

import (
	"fmt"
	"strings"
	"testing"
)

func TestSarifParserLogicalLocations(t *testing.T) {
	logical := `{"logicalLocations":[{"fullyQualifiedName":"example.Function","kind":"function"}]}`
	physical := `{"physicalLocation":{"artifactLocation":{"uri":"main.go"},"region":{"startLine":3}}}`
	for _, related := range []bool{false, true} {
		t.Run(fmt.Sprintf("related=%v", related), func(t *testing.T) {
			locations, relatedLocations := logical, physical
			if related {
				locations, relatedLocations = physical, logical
			}
			input := fmt.Sprintf(`{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"example"}},"results":[{"message":{"text":"first"},"locations":[%s],"relatedLocations":[%s]},{"message":{"text":"second"},"locations":[%s]}]}]}`, locations, relatedLocations, physical)
			diagnostics, err := NewSarifParser().Parse(strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			if len(diagnostics) != 2 {
				t.Fatalf("got %d diagnostics, want 2", len(diagnostics))
			}
			first := diagnostics[0]
			if first.GetMessage() != "first" || diagnostics[1].GetMessage() != "second" {
				t.Fatal("diagnostic messages were not preserved")
			}
			if len(first.GetRelatedLocations()) != 1 {
				t.Fatalf("got %d related locations, want 1", len(first.GetRelatedLocations()))
			}
			logicalLocation := first.GetLocation()
			physicalLocation := first.GetRelatedLocations()[0].GetLocation()
			if related {
				logicalLocation, physicalLocation = physicalLocation, logicalLocation
			}
			if logicalLocation != nil {
				t.Errorf("logical-only location = %v, want nil", logicalLocation)
			}
			if physicalLocation.GetPath() != "main.go" || physicalLocation.GetRange().GetStart().GetLine() != 3 {
				t.Errorf("physical location = %v, want main.go:3", physicalLocation)
			}
			if diagnostics[1].GetLocation().GetPath() != "main.go" {
				t.Error("subsequent physical diagnostic was not preserved")
			}
		})
	}
}

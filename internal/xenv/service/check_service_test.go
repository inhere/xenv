package service

import (
	"testing"

	"github.com/inhere/xenv/internal/xenv/models"
)

func TestParseToolRequirementSimpleMap(t *testing.T) {
	tests := []struct {
		raw          string
		wantVersion  string
		wantRequired bool
	}{
		{"*", "", true},
		{">=1.32", "1.32", true},
		{">=1.32,required", "1.32", true},
		{">=1.32,optional", "1.32", false},
	}
	for _, tt := range tests {
		got, err := ParseToolRequirement(tt.raw)
		if err != nil {
			t.Fatalf("ParseToolRequirement(%q): %v", tt.raw, err)
		}
		if got.MinVersion != tt.wantVersion || got.Required != tt.wantRequired {
			t.Fatalf("ParseToolRequirement(%q) = %+v", tt.raw, got)
		}
	}
}

func TestActivityStateUsesToolRequirementsField(t *testing.T) {
	state := models.NewActivityState(".xenv.toml")
	state.ToolRequirements["rg"] = "*"
	if state.IsEmpty() {
		t.Fatal("state with tool requirements must not be empty")
	}
}

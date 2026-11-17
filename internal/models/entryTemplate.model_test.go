package models

import (
	"strings"
	"testing"
)

func TestCleanEntryTemplateInput(t *testing.T) {
	tests := []struct {
		name     string
		inName   string
		inBody   string
		wantName string
		wantBody string
		wantErr  bool
	}{
		{"trims name and trailing body space", "  Gentle  ", "What happened:\n\nHow it felt:\n\n", "Gentle", "What happened:\n\nHow it felt:", false},
		{"keeps leading structure in body", "A", "  - one\n  - two", "A", "  - one\n  - two", false},
		{"empty name", "   ", "body", "", "", true},
		{"empty body", "Name", " \n\t", "", "", true},
		{"name too long", strings.Repeat("a", EntryTemplateMaxNameLength+1), "body", "", "", true},
		{"body too long", "Name", strings.Repeat("a", EntryTemplateMaxBodyLength+1), "", "", true},
		{"counts runes not bytes", strings.Repeat("é", EntryTemplateMaxNameLength), "body", strings.Repeat("é", EntryTemplateMaxNameLength), "body", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotBody, err := CleanEntryTemplateInput(tt.inName, tt.inBody)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if gotName != tt.wantName || gotBody != tt.wantBody {
				t.Errorf("got (%q, %q), want (%q, %q)", gotName, gotBody, tt.wantName, tt.wantBody)
			}
		})
	}
}

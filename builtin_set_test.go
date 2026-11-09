package themecore

import (
	"strings"
	"testing"
)

func TestBuiltinSet(t *testing.T) {
	if err := BuiltinSet(); err != nil {
		t.Fatalf("BuiltinSet: %v", err)
	}
}

func TestCheckBuiltinSet(t *testing.T) {
	tests := []struct {
		name    string
		kind    string
		set     []string
		compono []string
		wantErr []string
	}{
		{
			name:    "subset",
			kind:    "built-in",
			set:     []string{"LINK", "IMAGE"},
			compono: []string{"IMAGE", "LINK", "NAV"},
		},
		{
			name:    "equal",
			kind:    "markdown element",
			set:     []string{"p"},
			compono: []string{"p"},
		},
		{
			name:    "missing",
			kind:    "built-in",
			set:     []string{"LINK", "GONE"},
			compono: []string{"LINK"},
			wantErr: []string{"built-in", "GONE", "without a Break row"},
		},
		{
			name:    "empty set",
			kind:    "markdown element",
			set:     nil,
			compono: []string{"p"},
			wantErr: []string{"markdown element"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkBuiltinSet(tt.kind, tt.set, tt.compono)
			if len(tt.wantErr) == 0 {
				if err != nil {
					t.Fatalf("checkBuiltinSet(%q, %v, %v) = %v, want nil", tt.kind, tt.set, tt.compono, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("checkBuiltinSet(%q, %v, %v) = nil, want an error", tt.kind, tt.set, tt.compono)
			}
			for _, want := range tt.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("checkBuiltinSet(%q, %v, %v) error %q does not contain %q", tt.kind, tt.set, tt.compono, err.Error(), want)
				}
			}
		})
	}
}

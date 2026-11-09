package themecore

import (
	"regexp"
	"testing"
)

var ledgerIDPattern = regexp.MustCompile(`^(pkg|attr|assets|css|compono)(\.[a-z0-9]+(-[a-z0-9]+)*)+$`)

func TestThemeAPIVersions(t *testing.T) {
	if got := ThemeAPICurrent(); got != 2 {
		t.Errorf("ThemeAPICurrent() = %d, want 2", got)
	}
	if got := ThemeAPIMin(); got != 1 {
		t.Errorf("ThemeAPIMin() = %d, want 1", got)
	}
}

func TestThemeAPIArithmetic(t *testing.T) {
	tests := []struct {
		name       string
		introduced []int
		at         []int
		current    int
		min        int
	}{
		{"first version", []int{1, 1}, []int{1}, 1, 1},
		{"extension", []int{1, 2, 3}, []int{1}, 3, 1},
		{"break", []int{1, 2}, []int{1, 3}, 3, 3},
		{"extension after break", []int{1, 4}, []int{1, 3}, 4, 3},
		{"no features", []int{}, []int{1}, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := make([]featureRow, len(tt.introduced))
			for i, v := range tt.introduced {
				features[i] = featureRow{IntroducedIn: v}
			}
			breaks := make([]breakRow, len(tt.at))
			for i, v := range tt.at {
				breaks[i] = breakRow{At: v}
			}
			if got := themeAPICurrent(features, breaks); got != tt.current {
				t.Errorf("themeAPICurrent = %d, want %d", got, tt.current)
			}
			if got := themeAPIMin(breaks); got != tt.min {
				t.Errorf("themeAPIMin = %d, want %d", got, tt.min)
			}
		})
	}
}

func TestLedgerInvariants(t *testing.T) {
	seen := make(map[string]int)
	for i, f := range featureRows {
		if !ledgerIDPattern.MatchString(f.ID) {
			t.Errorf("featureRows[%d] ID %q does not match pattern %q", i, f.ID, ledgerIDPattern.String())
		}
		if prev, dup := seen[f.ID]; dup {
			t.Errorf("featureRows[%d] ID %q duplicates featureRows[%d]", i, f.ID, prev)
		}
		seen[f.ID] = i
		if f.Detect == nil {
			t.Errorf("featureRows[%d] ID %q has nil Detect", i, f.ID)
		}
		cur := ThemeAPICurrent()
		if f.IntroducedIn < 1 || f.IntroducedIn > cur {
			t.Errorf("featureRows[%d] ID %q IntroducedIn=%d not in [1,%d]", i, f.ID, f.IntroducedIn, cur)
		}
	}
	if len(breakRows) == 0 {
		t.Errorf("breakRows is empty; expected at least {At:1, Reason:%q}", "First Theme API version")
		return
	}
	if breakRows[0].At != 1 || breakRows[0].Reason != "First Theme API version" {
		t.Errorf("breakRows[0] = {At:%d, Reason:%q}; want {At:1, Reason:%q}",
			breakRows[0].At, breakRows[0].Reason, "First Theme API version")
	}
	for i := 1; i < len(breakRows); i++ {
		if breakRows[i].At <= breakRows[i-1].At {
			t.Errorf("breakRows[%d].At=%d not strictly greater than breakRows[%d].At=%d",
				i, breakRows[i].At, i-1, breakRows[i-1].At)
		}
	}
}

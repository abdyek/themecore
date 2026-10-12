package themecore

import (
	"regexp"
	"testing"
)

var ledgerIDPattern = regexp.MustCompile(`^(pkg|attr|assets|css|compono)(\.[a-z0-9]+(-[a-z0-9]+)*)+$`)

func TestThemeAPIV1(t *testing.T) {
	if got := ThemeAPICurrent(); got != 1 {
		t.Errorf("ThemeAPICurrent() = %d, want 1", got)
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
			features := make([]Feature, len(tt.introduced))
			for i, v := range tt.introduced {
				features[i] = Feature{IntroducedIn: v}
			}
			breaks := make([]Break, len(tt.at))
			for i, v := range tt.at {
				breaks[i] = Break{At: v}
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
	for i, f := range Features {
		if !ledgerIDPattern.MatchString(f.ID) {
			t.Errorf("Features[%d] ID %q does not match pattern %q", i, f.ID, ledgerIDPattern.String())
		}
		if prev, dup := seen[f.ID]; dup {
			t.Errorf("Features[%d] ID %q duplicates Features[%d]", i, f.ID, prev)
		}
		seen[f.ID] = i
		if f.Detect == nil {
			t.Errorf("Features[%d] ID %q has nil Detect", i, f.ID)
		}
		cur := ThemeAPICurrent()
		if f.IntroducedIn < 1 || f.IntroducedIn > cur {
			t.Errorf("Features[%d] ID %q IntroducedIn=%d not in [1,%d]", i, f.ID, f.IntroducedIn, cur)
		}
	}
	if len(Breaks) == 0 {
		t.Errorf("Breaks is empty; expected at least {At:1, Reason:%q}", "First Theme API version")
		return
	}
	if Breaks[0].At != 1 || Breaks[0].Reason != "First Theme API version" {
		t.Errorf("Breaks[0] = {At:%d, Reason:%q}; want {At:1, Reason:%q}",
			Breaks[0].At, Breaks[0].Reason, "First Theme API version")
	}
	for i := 1; i < len(Breaks); i++ {
		if Breaks[i].At <= Breaks[i-1].At {
			t.Errorf("Breaks[%d].At=%d not strictly greater than Breaks[%d].At=%d",
				i, Breaks[i].At, i-1, Breaks[i-1].At)
		}
	}
}

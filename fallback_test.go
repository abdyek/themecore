package themecore

import (
	"bytes"
	"strings"
	"testing"
)

func TestFallbackCSSVerifies(t *testing.T) {
	if err := Verify(FallbackCSS()); err != nil {
		t.Fatalf("Verify(FallbackCSS()) = %v, want nil", err)
	}
}

func TestFallbackSourceTransforms(t *testing.T) {
	out, err := Transform(FallbackSource())
	if err != nil {
		t.Fatalf("Transform(FallbackSource()) error = %v", err)
	}
	if !bytes.Equal(out, FallbackCSS()) {
		t.Fatalf("Transform(FallbackSource()) = %q, want %q", out, FallbackCSS())
	}
}

func TestFallbackHasNoAssets(t *testing.T) {
	src := string(FallbackSource())
	if strings.Contains(src, "url(") {
		t.Errorf("FallbackSource() contains %q", "url(")
	}
	if strings.Contains(src, "Inter") {
		t.Errorf("FallbackSource() contains %q", "Inter")
	}
}

func TestFallbackRoot(t *testing.T) {
	src := string(FallbackSource())
	want := []string{
		":scope {",
		"background-color: #F4F1EA;",
		"padding: 2.5rem;",
		"box-sizing: border-box;",
	}
	for _, s := range want {
		if !strings.Contains(src, s) {
			t.Errorf("FallbackSource() missing %q", s)
		}
	}
}

func TestFallbackReturnsCopies(t *testing.T) {
	first := FallbackSource()
	if len(first) == 0 {
		t.Fatalf("FallbackSource() returned empty slice")
	}
	first[0] = 'x'
	second := FallbackSource()
	if len(second) > 0 && second[0] == 'x' {
		t.Errorf("FallbackSource() did not return a copy: first-byte mutation persisted")
	}

	cssFirst := FallbackCSS()
	if len(cssFirst) == 0 {
		t.Fatalf("FallbackCSS() returned empty slice")
	}
	cssFirst[0] = 'x'
	cssSecond := FallbackCSS()
	if len(cssSecond) > 0 && cssSecond[0] == 'x' {
		t.Errorf("FallbackCSS() did not return a copy: first-byte mutation persisted")
	}
}

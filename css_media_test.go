package themecore

import (
	"strings"
	"testing"
)

func transformMedia(prelude string) error {
	_, err := Transform([]byte("@media " + prelude + "{a{color:red}}"))
	return err
}

func transformSupports(prelude string) error {
	_, err := Transform([]byte("@supports " + prelude + "{a{color:red}}"))
	return err
}

func TestMediaAccepted(t *testing.T) {
	preludes := []string{
		`print`,
		`screen`,
		`all`,
		`only screen`,
		`not print`,
		`screen and (min-width: 48rem)`,
		`(min-width:48rem)`,
		`(width >= 48rem)`,
		`(48rem <= width < 80rem)`,
		`(400px < width)`,
		`(min-aspect-ratio: 16/9)`,
		`(aspect-ratio > 1)`,
		`(orientation: portrait)`,
		`(min-resolution: 2dppx)`,
		`(resolution >= 192dpi)`,
		`(resolution: 2x)`,
		`(hover: hover)`,
		`(hover)`,
		`(any-pointer: coarse)`,
		`(prefers-color-scheme: dark)`,
		`(prefers-reduced-motion: reduce)`,
		`(prefers-contrast: more)`,
		`(forced-colors: active)`,
		`(min-width: 0)`,
		`(max-width: 100svw)`,
		`(min-height: 10dvh)`,
		`(min-width: 30ch)`,
		`print, (hover: none)`,
		`not (hover: hover)`,
		`(hover: hover) and (pointer: fine)`,
		`(hover: none) or (pointer: coarse)`,
		`((min-width: 1px) and (max-width: 2px))`,
		`SCREEN AND (MIN-WIDTH: 48REM)`,
	}
	for _, p := range preludes {
		t.Run(p, func(t *testing.T) {
			if err := transformMedia(p); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
	t.Run("", func(t *testing.T) {
		_, err := Transform([]byte("@media{a{color:red}}"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestMediaRejected(t *testing.T) {
	cases := []struct {
		prelude string
		want    string
	}{
		{`(display-mode: standalone)`, `unknown media feature display-mode`},
		{`(scripting: none)`, `unknown media feature scripting`},
		{`(min-device-width: 1px)`, `unknown media feature min-device-width`},
		{`(color)`, `unknown media feature color`},
		{`(min-orientation: portrait)`, `unknown media feature min-orientation`},
		{`tv`, `unknown media type tv`},
		{`(min-width: 48foo)`, `invalid value of min-width`},
		{`(min-width: 1cm)`, `invalid value of min-width`},
		{`(width: calc(1px + 2px))`, `invalid value of width`},
		{`(orientation: up)`, `invalid value of orientation`},
		{`(min-resolution: 2)`, `invalid value of min-resolution`},
		{`(aspect-ratio: 16/)`, `invalid value of aspect-ratio`},
		{`(hover >= hover)`, `invalid media query`},
		{`(min-width >= 1px)`, `invalid media query`},
		{`(min-width)`, `invalid media query`},
		{`screen and (hover) or (pointer)`, `invalid media query`},
		{`only (hover)`, `invalid media query`},
		{`print and`, `invalid media query`},
		{`(width < 1px > 2px)`, `invalid media query`},
		{`foo(bar)`, `invalid media query`},
		{`((hover) (pointer))`, `invalid media query`},
	}
	for _, c := range cases {
		t.Run(c.prelude, func(t *testing.T) {
			err := transformMedia(c.prelude)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("expected error containing %q, got %q", c.want, err.Error())
			}
		})
	}
}

func TestSupportsAccepted(t *testing.T) {
	preludes := []string{
		`(display: grid)`,
		`not (display: grid)`,
		`(display: grid) and (gap: 1rem)`,
		`(display: grid) or (display: flex)`,
		`selector(::details-content)`,
		`selector(:has(> p))`,
		`(--x: 1)`,
		`((display:grid))`,
	}
	for _, p := range preludes {
		t.Run(p, func(t *testing.T) {
			if err := transformSupports(p); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSupportsRejected(t *testing.T) {
	cases := []struct {
		prelude string
		want    string
	}{
		{`selector(:root)`, `:root is not allowed`},
		{`selector(#a)`, `id selectors are not allowed`},
		{`selector(:yeni)`, `unknown pseudo-class :yeni`},
		{`selector(a, b)`, `invalid @supports condition`},
		{`font-tech(color-colrv1)`, `invalid @supports condition`},
		{`(display: grid) and (gap: 1rem) or (x: y)`, `invalid @supports condition`},
		{`display: grid`, `invalid @supports condition`},
		{`(display)`, `invalid @supports condition`},
		{`(display:)`, `invalid @supports condition`},
		{`not not (a: b)`, `invalid @supports condition`},
	}
	for _, c := range cases {
		t.Run(c.prelude, func(t *testing.T) {
			err := transformSupports(c.prelude)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("expected error containing %q, got %q", c.want, err.Error())
			}
		})
	}
	t.Run("", func(t *testing.T) {
		_, err := Transform([]byte("@supports{a{color:red}}"))
		if err == nil {
			t.Fatalf(`expected error containing "invalid @supports condition", got nil`)
		}
		if !strings.Contains(err.Error(), "invalid @supports condition") {
			t.Fatalf(`expected error containing "invalid @supports condition", got %q`, err.Error())
		}
	})
}

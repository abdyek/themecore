package themecore

import "testing"

func TestCheckSizesAccepts(t *testing.T) {
	for _, v := range []string{
		"100vw",
		"0",
		"(min-width: 48rem) 33vw, 100vw",
		"(min-width: 48rem) 33vw, (min-width: 30rem) 50vw, 100vw",
		"(width >= 48rem) and (orientation: landscape) 50vw, calc(100vw - 2rem)",
		"not (hover: hover) 20em, 1rem",
		"(prefers-reduced-motion: reduce) 10px, 1em",
		"min(100vw, 40rem)",
		"clamp(10rem, calc(50vw - 1rem), 40rem)",
		"calc(100dvw / 3)",
	} {
		t.Run(v, func(t *testing.T) {
			if err := checkSizes(v); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCheckSizesRejects(t *testing.T) {
	for _, v := range []string{
		"",
		" ",
		"auto",
		"50%",
		"calc(100% - 2rem)",
		"-10px",
		"33cm",
		"100vw 50vw",
		"100vw, 50vw",
		"100vw,",
		"(min-width: 48rem) 33vw",
		"(min-width: 48rem), 100vw",
		"screen and (min-width: 48rem) 33vw, 100vw",
		"(display-mode: standalone) 50vw, 100vw",
		"var(--x)",
		"calc(100vw - var(--x))",
		"fit-content(10px)",
	} {
		t.Run(v, func(t *testing.T) {
			if err := checkSizes(v); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

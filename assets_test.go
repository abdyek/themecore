package themecore

import (
	"strings"
	"testing"
)

func sig(prefix string, n int) []byte {
	b := make([]byte, 0, len(prefix)+n)
	b = append(b, prefix...)
	return append(b, make([]byte, n)...)
}

func TestCheckAssetsAccepts(t *testing.T) {
	woff2Data := sig("wOF2", 8)
	woffData := sig("wOFF", 8)
	pngData := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 8)...)
	jpegData := append([]byte("\xff\xd8\xff\xe0"), make([]byte, 8)...)
	webpData := []byte("RIFF\x00\x00\x00\x00WEBPVP8 ")
	avifMajorData := []byte("\x00\x00\x00\x18ftypavif\x00\x00\x00\x00mif1miaf")
	avifCompatData := []byte("\x00\x00\x00\x18ftypmif1\x00\x00\x00\x00miafavif")
	avisData := []byte("\x00\x00\x00\x14ftypavis\x00\x00\x00\x00msf1")
	gif89aData := sig("GIF89a", 4)
	gif87aData := sig("GIF87a", 4)

	p := Package{
		Files: map[string][]byte{
			"built.toml":           []byte("x"),
			"assets/theme.css":     []byte("a{}"),
			"assets/fonts/a.woff2": woff2Data,
			"assets/fonts/a.woff":  woffData,
			"assets/b.png":         pngData,
			"assets/c.jpg":         jpegData,
			"assets/c.jpeg":        jpegData,
			"assets/d.webp":        webpData,
			"assets/e.avif":        avifMajorData,
			"assets/e2.avif":       avifCompatData,
			"assets/e3.avif":       avisData,
			"assets/f.gif":         gif89aData,
			"assets/g.gif":         gif87aData,
			"assets/h.css.png":     pngData,
		},
	}

	if err := checkAssets(p); err != nil {
		t.Fatalf("checkAssets returned error: %v", err)
	}
}

func TestCheckAssetsRejects(t *testing.T) {
	pngData := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 8)...)
	jpegData := append([]byte("\xff\xd8\xff\xe0"), make([]byte, 8)...)
	woff2Data := sig("wOF2", 8)
	woffData := sig("wOFF", 8)
	avifHeicData := []byte("\x00\x00\x00\x18ftypheic\x00\x00\x00\x00mif1heic")
	avifLargeBoxData := []byte("\x00\x00\x00\x40ftypavif\x00\x00\x00\x00")
	avifShortData := []byte("\x00\x00\x00\x18ftyp")
	webpWaveData := []byte("RIFF\x00\x00\x00\x00WAVEfmt ")
	gif88aData := sig("GIF88a", 4)

	cases := []struct {
		name  string
		files map[string][]byte
		want  string
	}{
		{
			name: "theme.css missing",
			files: map[string][]byte{
				"assets/b.png": pngData,
			},
			want: "package: assets/theme.css: missing",
		},
		{
			name: "other css",
			files: map[string][]byte{
				"assets/theme.css": []byte("a{}"),
				"assets/other.css": []byte("a{}"),
			},
			want: "package: assets/other.css: not an allowed asset type",
		},
		{
			name: "nested theme.css",
			files: map[string][]byte{
				"assets/theme.css":       []byte("a{}"),
				"assets/fonts/theme.css": []byte("a{}"),
			},
			want: "package: assets/fonts/theme.css: not an allowed asset type",
		},
		{
			name: "svg",
			files: map[string][]byte{
				"assets/theme.css": []byte("a{}"),
				"assets/a.svg":     []byte("<svg/>"),
			},
			want: "assets/a.svg: not an allowed asset type",
		},
		{
			name: "ttf",
			files: map[string][]byte{
				"assets/theme.css": []byte("a{}"),
				"assets/a.ttf":     woff2Data,
			},
			want: "assets/a.ttf: not an allowed asset type",
		},
		{
			name: "no extension",
			files: map[string][]byte{
				"assets/theme.css": []byte("a{}"),
				"assets/png":       pngData,
			},
			want: "assets/png: not an allowed asset type",
		},
		{
			name: "upper case extension",
			files: map[string][]byte{
				"assets/theme.css": []byte("a{}"),
				"assets/a.PNG":     pngData,
			},
			want: "assets/a.PNG: not an allowed asset type",
		},
		{
			name: "png with jpeg bytes",
			files: map[string][]byte{
				"assets/theme.css": []byte("a{}"),
				"assets/a.png":     jpegData,
			},
			want: "package: assets/a.png: content does not match its extension",
		},
		{
			name: "woff2 with woff bytes",
			files: map[string][]byte{
				"assets/theme.css": []byte("a{}"),
				"assets/a.woff2":   woffData,
			},
			want: "assets/a.woff2: content does not match its extension",
		},
		{
			name: "webp with wave",
			files: map[string][]byte{
				"assets/theme.css": []byte("a{}"),
				"assets/a.webp":    webpWaveData,
			},
			want: "assets/a.webp: content does not match its extension",
		},
		{
			name: "avif with heic",
			files: map[string][]byte{
				"assets/theme.css": []byte("a{}"),
				"assets/a.avif":    avifHeicData,
			},
			want: "assets/a.avif: content does not match its extension",
		},
		{
			name: "avif box larger than file",
			files: map[string][]byte{
				"assets/theme.css": []byte("a{}"),
				"assets/a.avif":    avifLargeBoxData,
			},
			want: "assets/a.avif: content does not match its extension",
		},
		{
			name: "avif short",
			files: map[string][]byte{
				"assets/theme.css": []byte("a{}"),
				"assets/a.avif":    avifShortData,
			},
			want: "assets/a.avif: content does not match its extension",
		},
		{
			name: "gif88a",
			files: map[string][]byte{
				"assets/theme.css": []byte("a{}"),
				"assets/a.gif":     gif88aData,
			},
			want: "assets/a.gif: content does not match its extension",
		},
		{
			name: "empty png",
			files: map[string][]byte{
				"assets/theme.css": []byte("a{}"),
				"assets/a.png":     []byte{},
			},
			want: "assets/a.png: content does not match its extension",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := Package{Files: tc.files}
			err := checkAssets(p)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %q", tc.want, err.Error())
			}
		})
	}
}

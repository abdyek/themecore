package themecore

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"path"
	"sort"
	"strings"
)

// themeCSS is the path of the only CSS file of a package.
const themeCSS = "assets/theme.css"

// assetTypes maps each allowed extension under assets/ to a check of the
// file's magic bytes. theme.css is the only CSS file and is not in the map.
var assetTypes = map[string]func([]byte) bool{
	".woff2": isWOFF2,
	".woff":  isWOFF,
	".png":   isPNG,
	".jpeg":  isJPEG,
	".jpg":   isJPEG,
	".webp":  isWebP,
	".avif":  isAVIF,
	".gif":   isGIF,
}

// isWOFF2 reports whether b starts with the WOFF2 magic bytes.
func isWOFF2(b []byte) bool {
	return bytes.HasPrefix(b, []byte("wOF2"))
}

// isWOFF reports whether b starts with the WOFF magic bytes.
func isWOFF(b []byte) bool {
	return bytes.HasPrefix(b, []byte("wOFF"))
}

// isPNG reports whether b starts with the PNG magic bytes.
func isPNG(b []byte) bool {
	return bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n"))
}

// isJPEG reports whether b starts with the JPEG magic bytes.
func isJPEG(b []byte) bool {
	return bytes.HasPrefix(b, []byte("\xff\xd8\xff"))
}

// isWebP reports whether b is a RIFF/WEBP file.
func isWebP(b []byte) bool {
	if len(b) < 12 {
		return false
	}
	if !bytes.HasPrefix(b, []byte("RIFF")) {
		return false
	}
	return bytes.Equal(b[8:12], []byte("WEBP"))
}

// isGIF reports whether b starts with the GIF87a or GIF89a magic bytes.
func isGIF(b []byte) bool {
	return bytes.HasPrefix(b, []byte("GIF87a")) || bytes.HasPrefix(b, []byte("GIF89a"))
}

// isAVIF reports whether b starts with an ISO BMFF ftyp box whose major or
// compatible brand is "avif" or "avis".
func isAVIF(b []byte) bool {
	if len(b) < 16 {
		return false
	}
	size := binary.BigEndian.Uint32(b[0:4])
	if size < 16 {
		return false
	}
	if size%4 != 0 {
		return false
	}
	if uint32(len(b)) < size {
		return false
	}
	if !bytes.Equal(b[4:8], []byte("ftyp")) {
		return false
	}
	if bytes.Equal(b[8:12], []byte("avif")) || bytes.Equal(b[8:12], []byte("avis")) {
		return true
	}
	for i := uint32(16); i+4 <= size; i += 4 {
		if bytes.Equal(b[i:i+4], []byte("avif")) || bytes.Equal(b[i:i+4], []byte("avis")) {
			return true
		}
	}
	return false
}

// checkAssets checks the files under assets/: assets/theme.css must exist,
// every other file must have an allowed extension, and its content must start
// with that format's magic bytes. Other paths of the package are not checked
// here.
func checkAssets(p Package) error {
	if _, ok := p.Files[themeCSS]; !ok {
		return fmt.Errorf("package: %s: missing", themeCSS)
	}
	names := make([]string, 0)
	for name := range p.Files {
		if !strings.HasPrefix(name, "assets/") {
			continue
		}
		if name == themeCSS {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		ext := path.Ext(name)
		check, ok := assetTypes[ext]
		if !ok {
			return fmt.Errorf("package: %s: not an allowed asset type", name)
		}
		if !check(p.Files[name]) {
			return fmt.Errorf("package: %s: content does not match its extension", name)
		}
	}
	return nil
}

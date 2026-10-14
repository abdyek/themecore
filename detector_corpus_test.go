package themecore

import "testing"

func TestDetectorCorpus(t *testing.T) {
	if err := DetectorCorpus(); err != nil {
		t.Fatal(err)
	}
}

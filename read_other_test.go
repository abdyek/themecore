//go:build !unix

package themecore

import (
	"strings"
	"testing"
)

func TestReadDirFailsWithoutHardLinkDetection(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string][]byte{
		"built.toml": []byte("x"),
	})
	_, err := ReadDir(dir)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	wantErr := "hard links cannot be detected on this platform"
	if !strings.Contains(err.Error(), wantErr) {
		t.Errorf("error %q does not contain %q", err.Error(), wantErr)
	}
}

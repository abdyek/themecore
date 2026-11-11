package themecore

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// endless is a reader of zero bytes that never ends. n counts the bytes read.
type endless struct{ n int64 }

func (r *endless) Read(p []byte) (int, error) {
	clear(p)
	r.n += int64(len(p))
	return len(p), nil
}

// TestReadLimitedFileSize tests readLimited on its own: checkEntries rejects an
// oversized file before readLimited runs, so its own checks are never reached
// through the other tests.
func TestReadLimitedFileSize(t *testing.T) {
	cases := []struct {
		path  string
		limit int
	}{
		{"assets/theme.css", 2 * MB},
		{"built.toml", 1 * MB},
		{"LICENSE", 256 * KB},
		{"assets/a.png", 16 * MB},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			t.Run("at limit", func(t *testing.T) {
				var total int64
				data, err := readLimited(bytes.NewReader(zeros(tc.limit)), tc.path, &total)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(data) != tc.limit {
					t.Errorf("got %d bytes, want %d", len(data), tc.limit)
				}
				if total != int64(tc.limit) {
					t.Errorf("total: got %d, want %d", total, tc.limit)
				}
			})
			t.Run("over limit", func(t *testing.T) {
				var total int64
				_, err := readLimited(bytes.NewReader(zeros(tc.limit+1)), tc.path, &total)
				want := fmt.Sprintf("%s: file is larger than %d bytes", tc.path, tc.limit)
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", want)
				}
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err.Error(), want)
				}
				if total != 0 {
					t.Errorf("total: got %d, want 0", total)
				}
			})
			t.Run("endless", func(t *testing.T) {
				var total int64
				r := &endless{}
				_, err := readLimited(r, tc.path, &total)
				want := fmt.Sprintf("%s: file is larger than %d bytes", tc.path, tc.limit)
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", want)
				}
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err.Error(), want)
				}
				if total != 0 {
					t.Errorf("total: got %d, want 0", total)
				}
				if r.n != int64(tc.limit+1) {
					t.Errorf("read %d bytes, want %d", r.n, tc.limit+1)
				}
			})
		})
	}
}

// TestReadLimitedTotalSize tests readLimited against the total size limit of
// the package: the files already read must not leave room for a file that
// would push the total over 64 MB.
func TestReadLimitedTotalSize(t *testing.T) {
	path := "assets/a.png"
	t.Run("at limit", func(t *testing.T) {
		total := int64(64*MB - 10)
		data, err := readLimited(bytes.NewReader(zeros(10)), path, &total)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) != 10 {
			t.Errorf("got %d bytes, want 10", len(data))
		}
		if total != 64*MB {
			t.Errorf("total: got %d, want %d", total, 64*MB)
		}
	})
	t.Run("over limit", func(t *testing.T) {
		total := int64(64*MB - 10)
		_, err := readLimited(bytes.NewReader(zeros(11)), path, &total)
		want := fmt.Sprintf("unpacked size is larger than %d bytes", 64*MB)
		if err == nil {
			t.Fatalf("expected error containing %q, got nil", want)
		}
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err.Error(), want)
		}
		if total != 64*MB-10 {
			t.Errorf("total: got %d, want %d", total, 64*MB-10)
		}
	})
	t.Run("endless", func(t *testing.T) {
		total := int64(64*MB - 10)
		r := &endless{}
		_, err := readLimited(r, path, &total)
		want := fmt.Sprintf("unpacked size is larger than %d bytes", 64*MB)
		if err == nil {
			t.Fatalf("expected error containing %q, got nil", want)
		}
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err.Error(), want)
		}
		if total != 64*MB-10 {
			t.Errorf("total: got %d, want %d", total, 64*MB-10)
		}
		if r.n != 11 {
			t.Errorf("read %d bytes, want 11", r.n)
		}
	})
}

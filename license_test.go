package themecore

import "testing"

func TestCheckLicense(t *testing.T) {
	tests := []struct {
		name  string
		files map[string][]byte
		want  string
	}{
		{
			name:  "absent",
			files: map[string][]byte{"built.toml": []byte("x")},
			want:  "",
		},
		{
			name:  "ascii",
			files: map[string][]byte{"LICENSE": []byte("MIT License")},
			want:  "",
		},
		{
			name:  "utf-8",
			files: map[string][]byte{"LICENSE": []byte("Telif hakkı © 2026 Yunus Emre Bulut")},
			want:  "",
		},
		{
			name:  "empty",
			files: map[string][]byte{"LICENSE": []byte("")},
			want:  "",
		},
		{
			name:  "invalid utf-8",
			files: map[string][]byte{"LICENSE": append([]byte("MIT "), 0xff)},
			want:  "package: LICENSE: not valid UTF-8",
		},
		{
			name:  "truncated rune",
			files: map[string][]byte{"LICENSE": append([]byte("a"), 0xc3)},
			want:  "package: LICENSE: not valid UTF-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Package{Files: tt.files}
			err := checkLicense(p)
			if tt.want == "" {
				if err != nil {
					t.Errorf("checkLicense() unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("checkLicense() expected error %q, got nil", tt.want)
			}
			if got := err.Error(); got != tt.want {
				t.Errorf("checkLicense() error = %q, want %q", got, tt.want)
			}
		})
	}
}

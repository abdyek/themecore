package themecore

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/blake2b"
)

// newTestKey returns a fresh Ed25519 key pair together with the fixed key id
// 01 02 03 04 05 06 07 08 that the test signature files carry.
func newTestKey(t *testing.T) ([8]byte, ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	var keyID [8]byte
	for i := range keyID {
		keyID[i] = byte(i + 1)
	}
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	return keyID, pub, priv
}

// signFile builds a minisign signature file for contract that carries the
// trusted comment trusted.
func signFile(alg string, keyID [8]byte, priv ed25519.PrivateKey, contract []byte, trusted string) string {
	hash := blake2b.Sum512(contract)
	sig := ed25519.Sign(priv, hash[:])

	raw := make([]byte, 0, 2+len(keyID)+len(sig))
	raw = append(raw, alg...)
	raw = append(raw, keyID[:]...)
	raw = append(raw, sig...)

	message := make([]byte, 0, len(sig)+len(trusted))
	message = append(message, sig...)
	message = append(message, trusted...)
	global := ed25519.Sign(priv, message)

	lines := []string{
		"untrusted comment: test",
		base64.StdEncoding.EncodeToString(raw),
		"trusted comment: " + trusted,
		base64.StdEncoding.EncodeToString(global),
	}
	return strings.Join(lines, "\n") + "\n"
}

// sigLines splits a signature file into its lines.
func sigLines(file string) []string {
	return strings.Split(strings.TrimSuffix(file, "\n"), "\n")
}

// sigFile joins lines back into a signature file.
func sigFile(lines []string) string {
	return strings.Join(lines, "\n") + "\n"
}

func TestVerifyContractSignatureAccepts(t *testing.T) {
	keyID, pub, priv := newTestKey(t)
	contract := []byte("name = \"umono/page\"\nversion = \"0.1\"\n")
	const ref = "umono/page@v0.1"
	file := signFile("ED", keyID, priv, contract, "contract "+ref)

	for _, tc := range []struct {
		name string
		sig  string
	}{
		{"with trailing newline", file},
		{"without trailing newline", strings.TrimSuffix(file, "\n")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := verifyContractSignature(keyID, pub, contract, []byte(tc.sig), ref); err != nil {
				t.Fatalf("verifyContractSignature: %v", err)
			}
		})
	}
}

func TestVerifyContractSignatureRejects(t *testing.T) {
	keyID, pub, priv := newTestKey(t)
	contract := []byte("name = \"umono/page\"\nversion = \"0.1\"\n")
	const ref = "umono/page@v0.1"
	valid := signFile("ED", keyID, priv, contract, "contract "+ref)

	mutate := func(fn func([]string) []string) string {
		lines := append([]string(nil), sigLines(valid)...)
		return sigFile(fn(lines))
	}

	// another key pair that carries the same key id
	_, otherPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}

	// a key id that differs from the one of the verifying key
	var otherKeyID [8]byte
	copy(otherKeyID[:], keyID[:])
	otherKeyID[0] = 0xff

	// a contract that changed after it was signed
	changed := []byte("name = \"umono/page\"\nversion = \"0.2\"\n")

	tests := []struct {
		name     string
		file     string
		contract []byte
		ref      string
		want     string
	}{
		{
			name: "three lines",
			file: mutate(func(l []string) []string { return l[:3] }),
			want: "must have 4 lines",
		},
		{
			name: "five lines",
			file: mutate(func(l []string) []string { return append(l, "extra") }),
			want: "must have 4 lines",
		},
		{
			name: "line 1 is not an untrusted comment",
			file: mutate(func(l []string) []string {
				l[0] = "untrusted-comment: test"
				return l
			}),
			want: "line 1 must start",
		},
		{
			name: "line 3 is not a trusted comment",
			file: mutate(func(l []string) []string {
				l[2] = "trusted-comment: contract " + ref
				return l
			}),
			want: "line 3 must start",
		},
		{
			name: "line 2 is not base64",
			file: mutate(func(l []string) []string {
				l[1] = "!!!not base64!!!"
				return l
			}),
			want: "line 2: invalid base64",
		},
		{
			name: "line 2 decodes to 73 bytes",
			file: mutate(func(l []string) []string {
				l[1] = base64.StdEncoding.EncodeToString(make([]byte, 73))
				return l
			}),
			want: "line 2: must be 74 bytes",
		},
		{
			name: "algorithm is Ed",
			file: signFile("Ed", keyID, priv, contract, "contract "+ref),
			want: "algorithm must be ED",
		},
		{
			name: "key id differs from the verifying key",
			file: signFile("ED", otherKeyID, priv, contract, "contract "+ref),
			want: "key id does not match",
		},
		{
			name:     "contract changed after signing",
			file:     valid,
			contract: changed,
			want:     "signature does not match the contract",
		},
		{
			name: "signed by another key pair with the same key id",
			file: signFile("ED", keyID, otherPriv, contract, "contract "+ref),
			want: "signature does not match the contract",
		},
		{
			name: "line 4 is not base64",
			file: mutate(func(l []string) []string {
				l[3] = "!!!not base64!!!"
				return l
			}),
			want: "line 4: invalid base64",
		},
		{
			name: "line 4 decodes to 63 bytes",
			file: mutate(func(l []string) []string {
				l[3] = base64.StdEncoding.EncodeToString(make([]byte, 63))
				return l
			}),
			want: "line 4: must be 64 bytes",
		},
		{
			name: "trusted comment edited after signing",
			file: mutate(func(l []string) []string {
				l[2] = "trusted comment: contract umono/page@v0.2"
				return l
			}),
			want: "global signature does not match",
		},
		{
			name: "trusted comment names another package",
			file: signFile("ED", keyID, priv, contract, "contract umono/blog@v0.1"),
			want: "trusted comment must be",
		},
		{
			name: "trusted comment has two spaces",
			file: signFile("ED", keyID, priv, contract, "contract  umono/page@v0.1"),
			want: "trusted comment must be",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.contract
			if c == nil {
				c = contract
			}
			r := tc.ref
			if r == "" {
				r = ref
			}
			err := verifyContractSignature(keyID, pub, c, []byte(tc.file), r)
			if err == nil {
				t.Fatal("verifyContractSignature: no error")
			}
			if !errors.Is(err, ErrContractSignature) {
				t.Fatalf("verifyContractSignature: %v, want ErrContractSignature", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("verifyContractSignature: %v, want error containing %q", err, tc.want)
			}
		})
	}
}

func TestEmbeddedPublicKey(t *testing.T) {
	keyID, pub, err := parsePublicKey(umonoPublicKey)
	if err != nil {
		t.Fatalf("parsePublicKey: %v", err)
	}
	if len(pub) != 32 {
		t.Fatalf("public key length = %d, want 32", len(pub))
	}
	reversed := make([]byte, len(keyID))
	for i := range keyID {
		reversed[i] = keyID[len(keyID)-1-i]
	}
	if got := strings.ToUpper(hex.EncodeToString(reversed)); got != "FF93F835DAA967E3" {
		t.Fatalf("key id = %s, want FF93F835DAA967E3", got)
	}
}

func TestVerifyContractSignatureUsesEmbeddedKey(t *testing.T) {
	keyID, _, priv := newTestKey(t)
	contract := []byte("name = \"umono/page\"\nversion = \"0.1\"\n")
	const ref = "umono/page@v0.1"
	file := signFile("ED", keyID, priv, contract, "contract "+ref)

	err := VerifyContractSignature(contract, []byte(file), ref)
	if err == nil {
		t.Fatal("VerifyContractSignature: no error")
	}
	if !errors.Is(err, ErrContractSignature) {
		t.Fatalf("VerifyContractSignature: %v, want ErrContractSignature", err)
	}
	if !strings.Contains(err.Error(), "key id does not match") {
		t.Fatalf("VerifyContractSignature: %v, want error containing %q", err, "key id does not match")
	}
}

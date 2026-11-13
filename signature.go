package themecore

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/blake2b"
)

// umonoPublicKey is the second line of umono.pub, the minisign public key
// that signs the published contracts.
const umonoPublicKey = "RWTjZ6naNfiT/6JEdZFEcmm1K2n/xSsuLy9zcIaZ37Rxo2XVh1R3vs1d"

// ErrContractSignature is the error every error of VerifyContractSignature
// wraps.
var ErrContractSignature = errors.New("contract signature")

// VerifyContractSignature verifies sig, the bytes of a .minisig file, against
// contract, the bytes of contract.toml, and ref, the reference the package
// declares (umono/page@v0.1), with the embedded public key. It does not touch
// the network, the file system or the clock.
func VerifyContractSignature(contract, sig []byte, ref string) error {
	keyID, pub, err := parsePublicKey(umonoPublicKey)
	if err != nil {
		panic(err)
	}
	return verifyContractSignature(keyID, pub, contract, sig, ref)
}

// parsePublicKey decodes s, the base64 of the 42 bytes of a minisign public
// key: the algorithm "Ed", the key id and the Ed25519 public key.
func parsePublicKey(s string) (keyID [8]byte, pub ed25519.PublicKey, err error) {
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return keyID, nil, fmt.Errorf("public key: invalid base64: %w", err)
	}
	if len(raw) != 42 {
		return keyID, nil, fmt.Errorf("public key: must be 42 bytes, is %d", len(raw))
	}
	if string(raw[0:2]) != "Ed" {
		return keyID, nil, fmt.Errorf("public key: algorithm must be Ed, is %q", raw[0:2])
	}
	copy(keyID[:], raw[2:10])
	return keyID, ed25519.PublicKey(raw[10:42]), nil
}

// verifyContractSignature verifies a minisign signature file against contract
// and ref with the given public key.
func verifyContractSignature(keyID [8]byte, pub ed25519.PublicKey, contract, sig []byte, ref string) error {
	lines := strings.Split(strings.TrimSuffix(string(sig), "\n"), "\n")
	if len(lines) != 4 {
		return fmt.Errorf("%w: must have 4 lines, has %d", ErrContractSignature, len(lines))
	}
	if !strings.HasPrefix(lines[0], "untrusted comment: ") {
		return fmt.Errorf("%w: line 1 must start with %q", ErrContractSignature, "untrusted comment: ")
	}
	if !strings.HasPrefix(lines[2], "trusted comment: ") {
		return fmt.Errorf("%w: line 3 must start with %q", ErrContractSignature, "trusted comment: ")
	}
	trusted := strings.TrimPrefix(lines[2], "trusted comment: ")

	raw, err := base64.StdEncoding.DecodeString(lines[1])
	if err != nil {
		return fmt.Errorf("%w: line 2: invalid base64: %v", ErrContractSignature, err)
	}
	if len(raw) != 74 {
		return fmt.Errorf("%w: line 2: must be 74 bytes, is %d", ErrContractSignature, len(raw))
	}
	if string(raw[0:2]) != "ED" {
		return fmt.Errorf("%w: line 2: algorithm must be ED, is %q", ErrContractSignature, raw[0:2])
	}
	if !bytes.Equal(raw[2:10], keyID[:]) {
		return fmt.Errorf("%w: line 2: key id does not match the public key", ErrContractSignature)
	}
	signature := raw[10:74]
	hash := blake2b.Sum512(contract)
	if !ed25519.Verify(pub, hash[:], signature) {
		return fmt.Errorf("%w: line 2: signature does not match the contract", ErrContractSignature)
	}

	global, err := base64.StdEncoding.DecodeString(lines[3])
	if err != nil {
		return fmt.Errorf("%w: line 4: invalid base64: %v", ErrContractSignature, err)
	}
	if len(global) != 64 {
		return fmt.Errorf("%w: line 4: must be 64 bytes, is %d", ErrContractSignature, len(global))
	}
	message := make([]byte, 0, len(signature)+len(trusted))
	message = append(message, signature...)
	message = append(message, trusted...)
	if !ed25519.Verify(pub, message, global) {
		return fmt.Errorf("%w: line 4: global signature does not match", ErrContractSignature)
	}

	if trusted != "contract "+ref {
		return fmt.Errorf("%w: trusted comment must be %q", ErrContractSignature, "contract "+ref)
	}
	return nil
}

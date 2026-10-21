package imageref

import (
	_ "crypto/sha256"
	"fmt"
	"strings"

	"github.com/distribution/reference"
)

func Validate(image string) error {
	if image == "" || len(image) > 512 || image != strings.TrimSpace(image) {
		return fmt.Errorf("invalid image reference")
	}
	if _, err := reference.ParseNormalizedNamed(image); err != nil {
		return fmt.Errorf("invalid image reference: %w", err)
	}
	return nil
}

func ValidDigest(d string) bool {
	if len(d) != 71 || !strings.HasPrefix(d, "sha256:") {
		return false
	}
	for _, c := range d[7:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// MatchDigest binds a pinned request to the root descriptor before import.
func MatchDigest(image, root string) error {
	if !ValidDigest(root) {
		return fmt.Errorf("invalid root digest")
	}
	if err := Validate(image); err != nil {
		return err
	}
	ref, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return err
	}
	if pinned, ok := ref.(reference.Digested); ok && pinned.Digest().String() != root {
		return fmt.Errorf("requested digest does not match root descriptor")
	}
	return nil
}

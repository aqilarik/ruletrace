package ruletrace

import (
	"crypto/sha1"
	"encoding/hex"
)

// Fingerprint is a derived, engine-side identifier for a canonical expression string.
// This is useful for caching / de-dup / mapping “the same condition” across rules.
func Fingerprint(expr string) string {
	sum := sha1.Sum([]byte(expr))
	return hex.EncodeToString(sum[:16])
}

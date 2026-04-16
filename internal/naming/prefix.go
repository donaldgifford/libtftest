// Package naming provides deterministic, parallel-safe resource name prefixes.
package naming

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"
	"testing"
	"time"
)

// Prefix returns a 10-char lowercase string suitable for embedding in AWS
// resource names to guarantee parallel safety. The format is "ltt-" followed
// by 6 hex chars derived from a hash of the test name, process ID, and
// nanosecond timestamp.
//
// The prefix is deterministic within a single test invocation (same test name
// + pid + time) but unique across parallel runs.
func Prefix(tb testing.TB) string {
	tb.Helper()

	h := sha256.New()
	h.Write([]byte(tb.Name()))
	h.Write([]byte(":"))
	h.Write([]byte(strconv.Itoa(os.Getpid())))
	h.Write([]byte(":"))
	h.Write([]byte(strconv.FormatInt(time.Now().UnixNano(), 10)))
	sum := h.Sum(nil)

	return "ltt-" + hex.EncodeToString(sum)[:6]
}

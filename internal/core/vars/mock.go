package vars

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

// MockValue returns a deterministic/simple generated value for a supported
// {{$name}} mock keyword. Unsupported keywords report ok=false so callers can
// preserve the original placeholder.
func MockValue(name string) (string, bool) {
	switch name {
	case "guid", "randomUUID":
		return uuidv4(), true
	case "timestamp":
		return strconv.FormatInt(time.Now().Unix(), 10), true
	case "isoTimestamp":
		return time.Now().UTC().Format(time.RFC3339Nano), true
	case "randomInt":
		return strconv.FormatInt(randomInt(1000), 10), true
	case "randomBoolean":
		if randomInt(2) == 0 {
			return "false", true
		}
		return "true", true
	// common aliases/placeholders - minimal set for compatibility without faker dep
	case "randomEmail":
		return "user@example.com", true
	case "randomFirstName":
		return "Bruno", true
	case "randomLastName":
		return "Dog", true
	case "randomFullName":
		return "Bruno Dog", true
	}
	return "", false
}

func uuidv4() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32])
}

func randomInt(n int64) int64 {
	if n <= 0 {
		return 0
	}
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0
	}
	v := int64(b[0])<<56 | int64(b[1])<<48 | int64(b[2])<<40 | int64(b[3])<<32 |
		int64(b[4])<<24 | int64(b[5])<<16 | int64(b[6])<<8 | int64(b[7])
	if v < 0 {
		v = -v
	}
	return v % n
}

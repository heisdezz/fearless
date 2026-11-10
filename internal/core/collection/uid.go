package collection

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
)

// GenerateUID generates a deterministic, stable UID for an item based on its path.
// This matches Bruno's path-derived UID scheme so items survive reloads across restarts.
func GenerateUID(path string) string {
	cleaned := filepath.Clean(strings.TrimSpace(path))
	if cleaned == "" || cleaned == "." {
		return ""
	}
	hash := sha256.Sum256([]byte(cleaned))
	// 16-byte (32 hex characters) prefix of SHA-256
	return hex.EncodeToString(hash[:16])
}

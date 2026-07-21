package lockfile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func hashValue(value any) string {
	encoded, _ := json.Marshal(value)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}
